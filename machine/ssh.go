package machine

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"quick-cmd/cmds"
	"quick-cmd/define"
	"quick-cmd/utils"
)

// SSHClient SSH客户端封装
type SSHClient struct {
	config        *define.Machine
	remoteMachine *define.RemoteMachine
	workVars      map[string]string
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

// Execute 执行命令
func (sc *SSHClient) Execute(cmd define.Command, output chan<- string, onStepStart func(step string), onStepComplete func()) error {
	if !sc.remoteMachine.IsConnected() {
		return fmt.Errorf("SSH客户端未连接")
	}

	return executeSteps(cmd.Steps, output, onStepStart, onStepComplete, func(command string, out chan<- string) error {
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
	defer session.Close()

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

// Stop 停止执行
func (sc *SSHClient) Stop() error {
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
	if err := sc.Connect(sc.config, false); err != nil {
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
