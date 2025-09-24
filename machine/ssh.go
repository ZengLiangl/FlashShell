package machine

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"quick-cmd/define"

	"golang.org/x/crypto/ssh"
)

// SSHClient SSH客户端封装
type SSHClient struct {
	config  *define.Machine
	client  *ssh.Client
	session *ssh.Session
}

// NewSSHClient 创建SSH客户端
func NewSSHClient(machine *define.Machine) *SSHClient {
	return &SSHClient{
		config: machine,
	}
}

// Connect 连接到远程机器
func (sc *SSHClient) Connect() error {
	// 获取敏感数据
	sensitiveData, err := sc.config.GetSensitiveData()
	if err != nil {
		return fmt.Errorf("获取敏感数据失败: %w", err)
	}

	var auth []ssh.AuthMethod

	// 密钥认证
	if sc.config.KeyFile != "" {
		key, err := sc.loadPrivateKey(sc.config.KeyFile)
		if err != nil {
			return fmt.Errorf("加载私钥失败: %w", err)
		}
		auth = append(auth, ssh.PublicKeys(key))
	}

	// 密码认证
	if sensitiveData.Password != "" {
		auth = append(auth, ssh.Password(sensitiveData.Password))
	}

	if len(auth) == 0 {
		return fmt.Errorf("未配置认证方式")
	}

	config := &ssh.ClientConfig{
		User:            sensitiveData.User,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应该验证主机密钥
		Timeout:         30 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", sensitiveData.Host, sensitiveData.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %w", err)
	}

	sc.client = client
	return nil
}

// Execute 执行命令
func (sc *SSHClient) Execute(cmd define.Command, output chan<- string) error {
	if sc.client == nil {
		return fmt.Errorf("SSH客户端未连接")
	}

	for _, step := range cmd.Steps {
		if err := sc.executeStep(step, output); err != nil {
			return err
		}
	}

	return nil
}

// executeStep 执行单个命令步骤
func (sc *SSHClient) executeStep(command string, output chan<- string) error {
	session, err := sc.client.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %w", err)
	}
	defer session.Close()

	// 设置伪终端以支持 ANSI 颜色
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // 禁用回显
		ssh.TTY_OP_ISPEED: 14400, // 输入速度 = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // 输出速度 = 14.4kbaud
	}

	if err := session.RequestPty("xterm-256color", 80, 24, modes); err != nil {
		// 如果请求 PTY 失败，继续执行但可能没有颜色支持
		output <- fmt.Sprintf("警告: 无法设置伪终端: %s", err.Error())
	}

	// 设置环境变量以支持颜色输出
	session.Setenv("TERM", "xterm-256color")
	session.Setenv("COLORTERM", "truecolor")
	session.Setenv("FORCE_COLOR", "1")

	// 设置输出管道
	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("获取stdout管道失败: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("获取stderr管道失败: %w", err)
	}

	// 启动命令
	if err := session.Start(command); err != nil {
		return fmt.Errorf("启动命令失败: %w", err)
	}

	// 读取输出
	go sc.readOutput(stdout, output, "STDOUT")
	go sc.readOutput(stderr, output, "STDERR")

	// 等待命令完成
	if err := session.Wait(); err != nil {
		// output <- fmt.Sprintf("命令执行失败: %s", err.Error())
		return err
	}

	return nil
}

// readOutput 读取命令输出，保留 ANSI 转义序列
func (sc *SSHClient) readOutput(reader io.Reader, output chan<- string, prefix string) {
	buf := make([]byte, 1024)
	var lineBuffer strings.Builder

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			text := convertToUTF8(buf[:n])

			// 逐字符处理，保留 ANSI 转义序列
			for _, char := range text {
				if char == '\n' {
					// 遇到换行符，输出当前行
					line := lineBuffer.String()
					if line != "" {
						if prefix == "STDERR" {
							output <- fmt.Sprintf("[%s] %s", prefix, line)
						} else {
							output <- line
						}
					}
					lineBuffer.Reset()
				} else if char != '\r' {
					// 添加字符到行缓冲区（跳过回车符）
					lineBuffer.WriteRune(char)
				}
			}
		}
		if err != nil {
			// 输出剩余的内容
			if lineBuffer.Len() > 0 {
				line := lineBuffer.String()
				if prefix == "STDERR" {
					output <- fmt.Sprintf("[%s] %s", prefix, line)
				} else {
					output <- line
				}
			}
			break
		}
	}
}

// Stop 停止执行
func (sc *SSHClient) Stop() error {
	if sc.session != nil {
		return sc.session.Signal(ssh.SIGTERM)
	}
	return nil
}

// Close 关闭连接
func (sc *SSHClient) Close() error {
	if sc.client != nil {
		return sc.client.Close()
	}
	return nil
}

// loadPrivateKey 加载私钥
func (sc *SSHClient) loadPrivateKey(keyPath string) (ssh.Signer, error) {
	// 展开路径
	if strings.HasPrefix(keyPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		keyPath = filepath.Join(homeDir, keyPath[2:])
	}

	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	return signer, nil
}

// TestConnection 测试连接
func (sc *SSHClient) TestConnection() error {
	if err := sc.Connect(); err != nil {
		return err
	}
	defer sc.Close()

	// 执行简单的测试命令
	session, err := sc.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	return session.Run("echo 'connection test'")
}

// IsConnected 检查是否已连接
func (sc *SSHClient) IsConnected() bool {
	if sc.client == nil {
		return false
	}

	// 尝试创建一个会话来测试连接
	session, err := sc.client.NewSession()
	if err != nil {
		return false
	}
	session.Close()
	return true
}

// GetConnectionInfo 获取连接信息
func (sc *SSHClient) GetConnectionInfo() string {
	if sc.client == nil {
		return "未连接"
	}

	conn := sc.client.Conn
	if conn == nil {
		return "连接信息不可用"
	}

	localAddr := conn.LocalAddr()
	remoteAddr := conn.RemoteAddr()

	return fmt.Sprintf("本地: %s -> 远程: %s", localAddr.String(), remoteAddr.String())
}
