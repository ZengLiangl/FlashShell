package machine

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

var sudoSFTPServerPaths = []string{
	"/usr/lib/openssh/sftp-server",
	"/usr/libexec/openssh/sftp-server",
	"/usr/lib/ssh/sftp-server",
	"/usr/libexec/sftp-server",
	"/usr/local/libexec/sftp-server",
	"/usr/local/lib/sftp-server",
}

// OpenSFTPClient 打开 SFTP；sudo=true 时通过 sudo 启动 sftp-server（需密码）
func OpenSFTPClient(sshClient *ssh.Client, sudo bool, password string) (*sftp.Client, error) {
	if sshClient == nil {
		return nil, fmt.Errorf("SSH客户端未连接")
	}
	opts := []sftp.ClientOption{
		sftp.UseConcurrentWrites(true),
		sftp.MaxConcurrentRequestsPerFile(64),
	}
	if !sudo {
		return sftp.NewClient(sshClient, opts...)
	}
	return openSudoSFTP(sshClient, password, opts...)
}

func openSudoSFTP(sshClient *ssh.Client, password string, opts ...sftp.ClientOption) (*sftp.Client, error) {
	password = strings.TrimSpace(password)
	if password == "" {
		return nil, fmt.Errorf("Sudo SFTP 需要密码认证（当前无可用密码）")
	}
	serverPath := probeSFTPServerPath(sshClient)
	session, err := sshClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("创建 sudo SFTP 会话失败: %w", err)
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		_ = session.Close()
		return nil, err
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		_ = session.Close()
		return nil, err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		_ = session.Close()
		return nil, err
	}

	const prompt = "SUDOPASSWORD:"
	const ready = "SFTPREADY"
	cmd := fmt.Sprintf(
		`sudo -S -p '%s' sh -c 'printf %s; exec %s -e'`,
		prompt, ready, shellQuotePath(serverPath),
	)
	if err := session.Start(cmd); err != nil {
		_ = session.Close()
		return nil, fmt.Errorf("启动 sudo sftp-server 失败: %w", err)
	}

	var (
		mu          sync.Mutex
		stderrAcc   string
		fedPassword bool
	)
	go func() {
		buf := make([]byte, 512)
		for {
			n, e := stderr.Read(buf)
			if n > 0 {
				mu.Lock()
				stderrAcc += string(buf[:n])
				needFeed := !fedPassword && strings.Contains(stderrAcc, prompt)
				if needFeed {
					fedPassword = true
				}
				mu.Unlock()
				if needFeed {
					_, _ = io.WriteString(stdin, password+"\n")
				}
			}
			if e != nil {
				return
			}
		}
	}()

	readyBuf := make([]byte, 0, 128)
	tmp := make([]byte, 64)
	deadline := time.Now().Add(20 * time.Second)
	for !bytes.Contains(readyBuf, []byte(ready)) {
		if time.Now().After(deadline) {
			_ = session.Close()
			mu.Lock()
			msg := strings.TrimSpace(stderrAcc)
			mu.Unlock()
			if msg != "" {
				return nil, fmt.Errorf("Sudo SFTP 握手超时: %s", msg)
			}
			return nil, fmt.Errorf("Sudo SFTP 握手超时（请确认密码与 sudo 权限）")
		}
		n, rerr := stdout.Read(tmp)
		if n > 0 {
			readyBuf = append(readyBuf, tmp[:n]...)
			if len(readyBuf) > 512 {
				readyBuf = readyBuf[len(readyBuf)-128:]
			}
			continue
		}
		if rerr != nil {
			_ = session.Close()
			mu.Lock()
			msg := strings.TrimSpace(stderrAcc)
			mu.Unlock()
			if msg != "" {
				return nil, fmt.Errorf("Sudo SFTP 失败: %s", msg)
			}
			return nil, fmt.Errorf("Sudo SFTP 失败: %w", rerr)
		}
	}

	idx := bytes.Index(readyBuf, []byte(ready))
	leftover := append([]byte(nil), readyBuf[idx+len(ready):]...)
	// 给 sftp-server 一点启动时间，避免过早 INIT
	time.Sleep(200 * time.Millisecond)
	rw := &sudoSFTPConn{
		session: session,
		stdin:   stdin,
		stdout:  io.MultiReader(bytes.NewReader(leftover), stdout),
	}
	client, err := sftp.NewClientPipe(rw, rw, opts...)
	if err != nil {
		_ = session.Close()
		return nil, fmt.Errorf("Sudo SFTP 初始化失败: %w", err)
	}
	return client, nil
}

type sudoSFTPConn struct {
	session *ssh.Session
	stdin   io.WriteCloser
	stdout  io.Reader
}

func (c *sudoSFTPConn) Read(p []byte) (int, error)  { return c.stdout.Read(p) }
func (c *sudoSFTPConn) Write(p []byte) (int, error) { return c.stdin.Write(p) }
func (c *sudoSFTPConn) Close() error {
	_ = c.stdin.Close()
	return c.session.Close()
}

func probeSFTPServerPath(sshClient *ssh.Client) string {
	for _, p := range sudoSFTPServerPaths {
		session, err := sshClient.NewSession()
		if err != nil {
			continue
		}
		err = session.Run("test -x " + shellQuotePath(p))
		_ = session.Close()
		if err == nil {
			return p
		}
	}
	return "/usr/lib/openssh/sftp-server"
}
