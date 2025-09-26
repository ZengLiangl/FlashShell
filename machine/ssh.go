package machine

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"quick-cmd/cmds"
	"quick-cmd/define"
)

// SSHClient SSH客户端封装
type SSHClient struct {
	config        *define.Machine
	remoteMachine *define.RemoteMachine
}

// NewSSHClient 创建SSH客户端
func NewSSHClient(machine *define.Machine) *SSHClient {
	remoteMachine := define.NewRemoteMachine()
	return &SSHClient{
		config:        machine,
		remoteMachine: remoteMachine,
	}
}

// Connect 连接到远程机器
func (sc *SSHClient) Connect(machine *define.Machine) error {
	// 使用 RemoteMachine 连接
	return sc.remoteMachine.Connect(machine)
}

// Execute 执行命令
func (sc *SSHClient) Execute(cmd define.Command, output chan<- string) error {
	if !sc.remoteMachine.IsConnected() {
		return fmt.Errorf("SSH客户端未连接")
	}
	for i, step := range cmd.Steps {
		output <- fmt.Sprintf("执行步骤 %d: %s", i+1, step)
		if err := sc.executeStep(step, output); err != nil {
			return fmt.Errorf("步骤 %d 执行失败: %w", i+1, err)
		}
	}

	return nil
}

// executeStep 执行单个命令步骤
func (sc *SSHClient) executeStep(command string, output chan<- string) error {
	// 判断是否在cmds包中
	specialCmd, splitStr := getSpecialCmd(command)
	if specialCmd != nil {
		return specialCmd(sc.remoteMachine, splitStr, output)
	}

	// 创建新的 SSH session
	session, err := sc.remoteMachine.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %w", err)
	}
	defer session.Close()
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
		return err
	}
	return nil
}

func getSpecialCmd(command string) (func(*define.RemoteMachine, []string, chan<- string) error, []string) {
	compile := regexp.MustCompile("\\S+")
	allString := compile.FindAllString(command, -1)
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
	// SSH session 不支持信号发送，这里暂时返回 nil
	// 如果需要停止功能，可以考虑使用 context 或其他机制
	return nil
}

// Close 关闭连接
func (sc *SSHClient) Close() error {
	if sc.remoteMachine != nil {
		return sc.remoteMachine.Close()
	}
	return nil
}

// TestConnection 测试连接
func (sc *SSHClient) TestConnection() error {
	if err := sc.Connect(sc.config); err != nil {
		return err
	}
	defer sc.Close()

	// 执行简单的测试命令
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
