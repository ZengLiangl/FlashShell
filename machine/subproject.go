package machine

import (
	"fmt"
	"sync"

	"quick-cmd/define"
)

// SubProjectRunner SubProject 执行器实现
type SubProjectRunner struct {
	configManager ConfigManagerInterface
	runners       map[string]define.Runner
	runnerMutex   sync.RWMutex
	currentStatus *define.SubProjectStatus
	statusMutex   sync.RWMutex
	stopChannel   chan bool
}

// ConfigManagerInterface 配置管理器接口
type ConfigManagerInterface interface {
	GetRoot() *define.Root
	GetMachine(name string) *define.Machine
}

// NewSubProjectRunner 创建 SubProject 执行器
func NewSubProjectRunner(configManager ConfigManagerInterface) *SubProjectRunner {
	return &SubProjectRunner{
		configManager: configManager,
		runners:       make(map[string]define.Runner),
		currentStatus: &define.SubProjectStatus{},
		stopChannel:   make(chan bool, 1),
	}
}

// ExecuteSubProject 执行 SubProject
func (spr *SubProjectRunner) ExecuteSubProject(projectName, subProjectName string, output chan<- string) error {
	// 获取配置
	root := spr.configManager.GetRoot()
	if root == nil {
		return fmt.Errorf("配置未加载")
	}

	// 查找 SubProject
	var subProject *define.SubProject
	var project *define.Project

	for _, p := range root.Projects {
		if p.Name == projectName {
			project = &p
			for _, sp := range p.SubProjects {
				if sp.Name == subProjectName {
					subProject = &sp
					break
				}
			}
			break
		}
	}

	if subProject == nil {
		return fmt.Errorf("未找到 SubProject: %s/%s", projectName, subProjectName)
	}

	// 初始化执行状态
	spr.statusMutex.Lock()
	spr.currentStatus = &define.SubProjectStatus{
		ProjectName:       projectName,
		SubProjectName:    subProjectName,
		IsRunning:         true,
		CurrentCommand:    "",
		CompletedCommands: 0,
		TotalCommands:     len(subProject.Commands),
	}
	spr.statusMutex.Unlock()

	output <- fmt.Sprintf("开始执行 SubProject: %s/%s", projectName, subProjectName)
	output <- fmt.Sprintf("总共 %d 个命令需要执行", len(subProject.Commands))

	// 创建执行上下文
	ctx := &define.ExecutionContext{
		ProjectName:    projectName,
		SubProjectName: subProjectName,
		Commands:       subProject.Commands,
		CurrentIndex:   0,
		OutputChannel:  output,
		WorkDir:        project.WorkDir,
	}

	// 按顺序执行所有 Commands
	for i, command := range subProject.Commands {
		// 检查是否需要停止
		select {
		case <-spr.stopChannel:
			output <- "执行已被用户停止"
			spr.updateStatus("", len(subProject.Commands), false)
			return fmt.Errorf("执行被用户停止")
		default:
		}

		// 更新当前执行状态
		spr.updateStatus(command.Name, i, true)

		output <- fmt.Sprintf("执行命令 %d/%d: %s", i+1, len(subProject.Commands), command.Name)
		output <- fmt.Sprintf("命令描述: %s", command.Description)
		output <- fmt.Sprintf("命令类型: %s", command.Type)

		// 执行当前命令
		if err := spr.executeCommand(command, ctx, output); err != nil {
			output <- fmt.Sprintf("命令执行失败: %s", err.Error())
			spr.updateStatus(command.Name, i, false)
			return fmt.Errorf("命令 '%s' 执行失败: %w", command.Name, err)
		}

		output <- fmt.Sprintf("命令 '%s' 执行完成", command.Name)

		// 更新完成状态
		spr.updateStatus(command.Name, i+1, true)
	}

	// 所有命令执行完成
	output <- fmt.Sprintf("SubProject '%s' 执行完成！", subProjectName)
	spr.updateStatus("", len(subProject.Commands), false)

	return nil
}

// executeCommand 执行单个命令
func (spr *SubProjectRunner) executeCommand(command define.Command, ctx *define.ExecutionContext, output chan<- string) error {
	var runner define.Runner
	var err error

	switch command.Type {
	case "batch":
		// 本地执行
		workDir := ctx.WorkDir
		if command.WorkDir != "" {
			workDir = command.WorkDir
		}
		runner = NewLocalRunner(workDir)

	case "remote":
		// 远程执行
		if command.Machine == "" {
			return fmt.Errorf("远程命令未指定机器")
		}

		machineConfig := spr.configManager.GetMachine(command.Machine)
		if machineConfig == nil {
			return fmt.Errorf("未找到机器配置: %s", command.Machine)
		}

		sshClient := NewSSHClient(machineConfig)
		if err := sshClient.Connect(); err != nil {
			return fmt.Errorf("连接远程机器失败: %w", err)
		}
		runner = sshClient

	default:
		return fmt.Errorf("不支持的命令类型: %s", command.Type)
	}

	// 存储执行器以便停止
	runnerKey := fmt.Sprintf("%s/%s/%s", ctx.ProjectName, ctx.SubProjectName, command.Name)
	spr.runnerMutex.Lock()
	spr.runners[runnerKey] = runner
	spr.runnerMutex.Unlock()

	// 执行命令
	err = runner.Execute(command, output)

	// 清理执行器
	spr.runnerMutex.Lock()
	delete(spr.runners, runnerKey)
	spr.runnerMutex.Unlock()

	return err
}

// StopSubProject 停止 SubProject 执行
func (spr *SubProjectRunner) StopSubProject(projectName, subProjectName string) error {
	// 发送停止信号
	select {
	case spr.stopChannel <- true:
	default:
		// 通道已满，说明已经有停止信号
	}

	// 停止所有正在运行的命令
	spr.runnerMutex.RLock()
	runners := make([]define.Runner, 0, len(spr.runners))
	for key, runner := range spr.runners {
		// 只停止当前 SubProject 的命令
		if fmt.Sprintf("%s/%s", projectName, subProjectName) == key[:len(fmt.Sprintf("%s/%s", projectName, subProjectName))] {
			runners = append(runners, runner)
		}
	}
	spr.runnerMutex.RUnlock()

	// 停止所有相关的执行器
	for _, runner := range runners {
		if err := runner.Stop(); err != nil {
			// 记录错误但继续停止其他执行器
			fmt.Printf("停止执行器时出错: %v\n", err)
		}
	}

	return nil
}

// GetExecutionStatus 获取执行状态
func (spr *SubProjectRunner) GetExecutionStatus() *define.SubProjectStatus {
	spr.statusMutex.RLock()
	defer spr.statusMutex.RUnlock()

	// 返回状态的副本
	status := *spr.currentStatus
	return &status
}

// updateStatus 更新执行状态
func (spr *SubProjectRunner) updateStatus(currentCommand string, completedCommands int, isRunning bool) {
	spr.statusMutex.Lock()
	defer spr.statusMutex.Unlock()

	spr.currentStatus.CurrentCommand = currentCommand
	spr.currentStatus.CompletedCommands = completedCommands
	spr.currentStatus.IsRunning = isRunning
}
