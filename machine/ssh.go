package machine

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"

	"FlashDock/cmds"
	"FlashDock/data"
	"FlashDock/define"
	"FlashDock/utils"

	"golang.org/x/crypto/ssh"
)

// SSHClient SSH客户端封装
type SSHClient struct {
	config        *define.Machine
	remoteMachine *define.RemoteMachine
	workVars      map[string]string
	sessionMu     sync.Mutex
	activeSession *ssh.Session
}

// NewSSHClient 创建SSH客户端
func NewSSHClient(machine *define.Machine, workVars map[string]string) *SSHClient {
	remoteMachine := define.NewRemoteMachine()
	return &SSHClient{
		config:        machine,
		remoteMachine: remoteMachine,
		workVars:      workVars,
	}
}

// Connect 连接到远程机器
func (sc *SSHClient) Connect(machine *define.Machine, withSFTP bool) error {
	return sc.remoteMachine.Connect(machine, withSFTP)
}

// ConnectAutoTrustOnce 连接远程；遇未知主机密钥则会话级信任一次并自动重试（不弹框）
func (sc *SSHClient) ConnectAutoTrustOnce(machine *define.Machine, withSFTP bool) error {
	err := sc.Connect(machine, withSFTP)
	if err == nil {
		return nil
	}
	if data.TrustSessionIfUnknown(err) {
		return sc.Connect(machine, withSFTP)
	}
	return err
}

// Execute 执行命令
func (sc *SSHClient) Execute(cmd define.Command, output chan<- string, onStepStart func(step string), onStepComplete func(), shouldStop func() bool) error {
	if !sc.remoteMachine.IsConnected() {
		return fmt.Errorf("SSH客户端未连接")
	}

	return executeSteps(cmd.Steps, output, onStepStart, onStepComplete, shouldStop, func(command string, out chan<- string) error {
		return sc.executeStep(command, out)
	})
}

// executeStep 执行单个命令步骤
func (sc *SSHClient) executeStep(command string, output chan<- string) error {
	specialCmd, splitStr := getSpecialCmd(command)
	if specialCmd != nil {
		return specialCmd(sc.remoteMachine, splitStr, output)
	}

	session, err := sc.remoteMachine.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %w", err)
	}
	sc.setActiveSession(session)
	defer sc.clearActiveSession(session)

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("获取stdout管道失败: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("获取stderr管道失败: %w", err)
	}

	if err := session.Start(command); err != nil {
		return fmt.Errorf("启动命令失败: %w", err)
	}

	go sc.readOutput(stdout, output, "STDOUT")
	go sc.readOutput(stderr, output, "STDERR")

	if err := session.Wait(); err != nil {
		return err
	}
	return nil
}

func getSpecialCmd(command string) (func(*define.RemoteMachine, []string, chan<- string) error, []string) {
	compile := regexp.MustCompile("\\S+")
	allString := compile.FindAllString(command, -1)
	if len(allString) == 0 {
		return nil, nil
	}
	specialCmd := cmds.CmdManager.GetSpecialCmd(allString[0])
	return specialCmd, allString
}

// readOutput 读取命令输出，保留 ANSI 转义序列
func (sc *SSHClient) readOutput(reader io.Reader, output chan<- string, prefix string) {
	buf := make([]byte, 1024)
	var lineBuffer strings.Builder

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			text := convertToUTF8(buf[:n])

			for _, char := range text {
				if char == '\n' {
					line := lineBuffer.String()
					if line != "" {
						if prefix == "STDERR" {
							utils.SendOutput(output, fmt.Sprintf("[%s] %s", prefix, line))
						} else {
							utils.SendOutput(output, line)
						}
					}
					lineBuffer.Reset()
				} else if char != '\r' {
					lineBuffer.WriteRune(char)
				}
			}
		}
		if err != nil {
			if lineBuffer.Len() > 0 {
				line := lineBuffer.String()
				if prefix == "STDERR" {
					utils.SendOutput(output, fmt.Sprintf("[%s] %s", prefix, line))
				} else {
					utils.SendOutput(output, line)
				}
			}
			break
		}
	}
}

func (sc *SSHClient) setActiveSession(session *ssh.Session) {
	sc.sessionMu.Lock()
	sc.activeSession = session
	sc.sessionMu.Unlock()
}

func (sc *SSHClient) clearActiveSession(session *ssh.Session) {
	sc.sessionMu.Lock()
	if sc.activeSession == session {
		sc.activeSession = nil
	}
	sc.sessionMu.Unlock()
	session.Close()
}

// Stop 停止执行：关闭当前 SSH session 以中断远程命令
func (sc *SSHClient) Stop() error {
	sc.sessionMu.Lock()
	session := sc.activeSession
	sc.activeSession = nil
	sc.sessionMu.Unlock()
	if session != nil {
		return session.Close()
	}
	return nil
}

// Close 关闭连接
func (sc *SSHClient) Close() error {
	if sc.remoteMachine != nil {
		return sc.remoteMachine.Close()
	}
	return nil
}

// TestConnection 测试连接（未知主机密钥时自动会话信任并重试一次）
func (sc *SSHClient) TestConnection() error {
	if err := sc.ConnectAutoTrustOnce(sc.config, false); err != nil {
		return err
	}
	defer sc.Close()

	session, err := sc.remoteMachine.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	return session.Run("echo 'connection test'")
}

// IsConnected 检查是否已连接
func (sc *SSHClient) IsConnected() bool {
	return sc.remoteMachine.IsConnected()
}

// GetConnectionInfo 获取连接信息
func (sc *SSHClient) GetConnectionInfo() string {
	if sc.remoteMachine.SSHClient == nil {
		return "未连接"
	}

	conn := sc.remoteMachine.SSHClient.Conn
	if conn == nil {
		return "连接信息不可用"
	}

	localAddr := conn.LocalAddr()
	remoteAddr := conn.RemoteAddr()

	return fmt.Sprintf("本地: %s -> 远程: %s", localAddr.String(), remoteAddr.String())
}
