package machine

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"unicode/utf8"

	"quick-cmd/define"
	"quick-cmd/utils"
)

// LocalRunner 本地命令执行器
type LocalRunner struct {
	workDir  string
	workVars map[string]string
	cmd      *exec.Cmd
}

// NewLocalRunner 创建本地执行器
func NewLocalRunner(workDir string, workVars map[string]string) *LocalRunner {
	return &LocalRunner{
		workDir:  workDir,
		workVars: workVars,
	}
}

// Execute 执行命令
func (lr *LocalRunner) Execute(cmd define.Command, output chan<- string, onStepStart func(step string), onStepComplete func()) error {
	workDir := lr.workDir
	if cmd.WorkDir != "" {
		workDir = cmd.WorkDir
	}

	return executeSteps(cmd.Steps, output, onStepStart, onStepComplete, func(command string, out chan<- string) error {
		return lr.executeStep(command, workDir, out)
	})
}

// sendOutput is non-blocking to avoid goroutine buildup when output is very chatty.
func (lr *LocalRunner) sendOutput(output chan<- string, msg string) {
	utils.SendOutput(output, msg)
}

// executeStep 执行单个命令步骤
func (lr *LocalRunner) executeStep(command, workDir string, output chan<- string) error {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
		applyNoConsoleWindow(cmd)
	} else {
		cmd = exec.Command("bash", "-c", command)
	}

	if workDir != "" {
		cmd.Dir = workDir
		lr.sendOutput(output, fmt.Sprintf("工作目录: %s", workDir))
	}

	env := os.Environ()
	env = append(env, "LANG=en_US.UTF-8")
	env = append(env, "LC_ALL=en_US.UTF-8")
	env = append(env, "LC_CTYPE=en_US.UTF-8")
	env = append(env, "TERM=xterm-256color")
	env = append(env, "COLORTERM=truecolor")
	env = append(env, "FORCE_COLOR=1")
	env = append(env, "CLICOLOR=1")
	env = append(env, "CLICOLOR_FORCE=1")
	cmd.Env = env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("获取stdout管道失败: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("获取stderr管道失败: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动命令失败: %w", err)
	}

	lr.cmd = cmd

	go lr.readOutput(stdout, output, "STDOUT")
	go lr.readOutput(stderr, output, "STDERR")

	if err := cmd.Wait(); err != nil {
		lr.sendOutput(output, fmt.Sprintf("命令执行失败: %s", err.Error()))
		return err
	}

	lr.cmd = nil
	return nil
}

// readOutput 读取命令输出，保留 ANSI 转义序列
func (lr *LocalRunner) readOutput(pipe io.Reader, output chan<- string, prefix string) {
	reader := bufio.NewReader(pipe)

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			lr.sendOutput(output, fmt.Sprintf("[%s] 读取输出错误: %s", prefix, err.Error()))
			break
		}

		if line != "" {
			line = strings.TrimRight(line, "\r\n")

			if !utf8.ValidString(line) {
				line = strings.ToValidUTF8(line, "�")
			}

			if line != "" {
				if prefix == "STDERR" {
					lr.sendOutput(output, fmt.Sprintf("[%s] %s", prefix, line))
				} else {
					lr.sendOutput(output, line)
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
		if runtime.GOOS == "windows" {
			pid := lr.cmd.Process.Pid
			killCmd := exec.Command("taskkill", "/T", "/F", "/PID", strconv.FormatInt(int64(pid), 10))
			return killCmd.Run()
		}
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
