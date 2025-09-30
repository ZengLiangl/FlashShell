package machine

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"unicode/utf8"

	"quick-cmd/define"
)

// LocalRunner 本地命令执行器
type LocalRunner struct {
	workDir string
	cmd     *exec.Cmd
}

// NewLocalRunner 创建本地执行器
func NewLocalRunner(workDir string) *LocalRunner {
	return &LocalRunner{
		workDir: workDir,
	}
}

// Execute 执行命令
func (lr *LocalRunner) Execute(cmd define.Command, output chan<- string, onStepStart func(step string), onStepComplete func()) error {
	// 设置工作目录
	workDir := lr.workDir
	if cmd.WorkDir != "" {
		workDir = cmd.WorkDir
	}

	for i, step := range cmd.Steps {
		// 通知步骤开始执行
		if onStepStart != nil {
			onStepStart(step)
		}

		output <- fmt.Sprintf("执行步骤 %d: %s", i+1, step)

		if err := lr.executeStep(step, workDir, output); err != nil {
			return fmt.Errorf("步骤 %d 执行失败: %w", i+1, err)
		}

		// 通知步骤执行完成
		if onStepComplete != nil {
			onStepComplete()
		}
	}

	return nil
}

// executeStep 执行单个命令步骤
func (lr *LocalRunner) executeStep(command, workDir string, output chan<- string) error {
	var cmd *exec.Cmd

	// 根据操作系统选择shell
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		// 在 macOS/Linux 上使用 bash 并设置正确的编码
		cmd = exec.Command("bash", "-c", command)
	}

	// 设置工作目录
	if workDir != "" {
		cmd.Dir = workDir
		output <- fmt.Sprintf("工作目录: %s", workDir)
	}

	// 设置环境变量，确保使用 UTF-8 编码并支持颜色输出
	env := os.Environ()
	// 设置多种可能的 UTF-8 编码环境变量
	env = append(env, "LANG=en_US.UTF-8")
	env = append(env, "LC_ALL=en_US.UTF-8")
	env = append(env, "LC_CTYPE=en_US.UTF-8")
	// 设置终端类型和颜色支持
	env = append(env, "TERM=xterm-256color")
	env = append(env, "COLORTERM=truecolor")
	env = append(env, "FORCE_COLOR=1")
	// 确保某些工具输出颜色
	env = append(env, "CLICOLOR=1")
	env = append(env, "CLICOLOR_FORCE=1")
	cmd.Env = env

	// 获取输出管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("获取stdout管道失败: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("获取stderr管道失败: %w", err)
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动命令失败: %w", err)
	}

	lr.cmd = cmd

	// 读取输出
	go lr.readOutput(stdout, output, "STDOUT")
	go lr.readOutput(stderr, output, "STDERR")

	// 等待命令完成
	if err := cmd.Wait(); err != nil {
		output <- fmt.Sprintf("命令执行失败: %s", err.Error())
		return err
	}

	lr.cmd = nil
	// output <- "命令执行完成"
	return nil
}

// readOutput 读取命令输出，保留 ANSI 转义序列
func (lr *LocalRunner) readOutput(pipe io.Reader, output chan<- string, prefix string) {
	reader := bufio.NewReader(pipe)

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			output <- fmt.Sprintf("[%s] 读取输出错误: %s", prefix, err.Error())
			break
		}

		if line != "" {
			// 移除行尾的换行符，但保留 ANSI 转义序列
			line = strings.TrimRight(line, "\r\n")

			// 确保文本是有效的 UTF-8，但保留 ANSI 转义序列
			if !utf8.ValidString(line) {
				// 如果不是有效的 UTF-8，尝试转换，但保留转义序列
				line = strings.ToValidUTF8(line, "�")
			}

			// 不过滤空行，因为可能包含只有 ANSI 转义序列的行
			if line != "" {
				// 添加前缀标识输出来源
				if prefix == "STDERR" {
					output <- fmt.Sprintf("[%s] %s", prefix, line)
				} else {
					output <- line
				}
			}
		}

		if err == io.EOF {
			break
		}
	}
}

// Stop 停止执行
func (lr *LocalRunner) Stop() error {
	if lr.cmd != nil && lr.cmd.Process != nil {
		return lr.cmd.Process.Kill()
	}
	return nil
}

// IsRunning 检查是否正在运行
func (lr *LocalRunner) IsRunning() bool {
	return lr.cmd != nil && lr.cmd.Process != nil
}

// GetWorkDir 获取工作目录
func (lr *LocalRunner) GetWorkDir() string {
	return lr.workDir
}

// SetWorkDir 设置工作目录
func (lr *LocalRunner) SetWorkDir(workDir string) {
	lr.workDir = workDir
}
