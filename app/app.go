package app

import (
	"context"
	"fmt"
	"os"
	"sync"

	"quick-cmd/data"
	"quick-cmd/define"
	"quick-cmd/machine"
)

// App struct
type App struct {
	ctx           context.Context
	configManager *data.ConfigManager
	runners       map[string]define.Runner
	runnerMutex   sync.RWMutex
	outputChannel chan string
	isRunning     bool
	currentCmd    string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		runners:       make(map[string]define.Runner),
		outputChannel: make(chan string, 1000),
	}
}

// Startup is called when the app starts up
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.configManager = data.NewConfigManager("config.yaml")

	// 尝试加载配置文件，如果不存在则创建默认配置
	if _, err := a.configManager.LoadConfig(); err != nil {
		if os.IsNotExist(err) {
			println("配置文件不存在，创建默认配置")
			data.CreateDefaultConfig("config.yaml")
			a.configManager.LoadConfig()
		} else {
			println("加载配置文件失败:", err.Error())
		}
	} else {
		println("配置文件加载成功")
	}
}

// DomReady is called after front-end resources have been loaded
func (a *App) DomReady(ctx context.Context) {
	// Add your action here
}

// BeforeClose is called when the application is about to quit
func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	a.StopAllCommands()
	return false
}

// Shutdown is called during application termination
func (a *App) Shutdown(ctx context.Context) {
	close(a.outputChannel)
}

// GetConfig 获取配置
func (a *App) GetConfig() (*define.Root, error) {
	return a.configManager.LoadConfig()
}

// SaveConfig 保存配置
func (a *App) SaveConfig(root *define.Root) error {
	return a.configManager.SaveConfig(root)
}

// ExecuteCommand 执行命令
func (a *App) ExecuteCommand(projectName, subProjectName, commandName string) error {
	root := a.configManager.GetRoot()
	if root == nil {
		return fmt.Errorf("配置未加载")
	}

	// 查找命令
	var targetCmd *define.Command
	var project *define.Project

	for _, p := range root.Projects {
		if p.Name == projectName {
			project = &p
			for _, sp := range p.SubProjects {
				if sp.Name == subProjectName {
					for _, cmd := range sp.Commands {
						if cmd.Name == commandName {
							targetCmd = &cmd
							break
						}
					}
					break
				}
			}
			break
		}
	}

	if targetCmd == nil {
		return fmt.Errorf("未找到命令: %s/%s/%s", projectName, subProjectName, commandName)
	}

	// 创建执行器
	var runner define.Runner

	switch targetCmd.Type {
	case "batch":
		workDir := project.WorkDir
		if targetCmd.WorkDir != "" {
			workDir = targetCmd.WorkDir
		}
		runner = machine.NewLocalRunner(workDir)
	case "remote":
		if targetCmd.Machine == "" {
			return fmt.Errorf("远程命令未指定机器")
		}
		machineConfig := a.configManager.GetMachine(targetCmd.Machine)
		if machineConfig == nil {
			return fmt.Errorf("未找到机器配置: %s", targetCmd.Machine)
		}
		sshClient := machine.NewSSHClient(machineConfig)
		if err := sshClient.Connect(); err != nil {
			return fmt.Errorf("连接远程机器失败: %w", err)
		}
		runner = sshClient
	default:
		return fmt.Errorf("不支持的命令类型: %s", targetCmd.Type)
	}

	// 存储执行器
	runnerKey := fmt.Sprintf("%s/%s/%s", projectName, subProjectName, commandName)
	a.runnerMutex.Lock()
	a.runners[runnerKey] = runner
	a.isRunning = true
	a.currentCmd = runnerKey
	a.runnerMutex.Unlock()

	// 异步执行命令
	go func() {
		defer func() {
			a.runnerMutex.Lock()
			delete(a.runners, runnerKey)
			a.isRunning = false
			a.currentCmd = ""
			a.runnerMutex.Unlock()
		}()

		if err := runner.Execute(*targetCmd, a.outputChannel); err != nil {
			a.outputChannel <- fmt.Sprintf("执行失败: %s", err.Error())
		}
	}()

	return nil
}

// StopCommand 停止命令
func (a *App) StopCommand(projectName, subProjectName, commandName string) error {
	runnerKey := fmt.Sprintf("%s/%s/%s", projectName, subProjectName, commandName)

	a.runnerMutex.RLock()
	runner, exists := a.runners[runnerKey]
	a.runnerMutex.RUnlock()

	if !exists {
		return fmt.Errorf("命令未在运行")
	}

	return runner.Stop()
}

// StopAllCommands 停止所有命令
func (a *App) StopAllCommands() {
	a.runnerMutex.RLock()
	runners := make([]define.Runner, 0, len(a.runners))
	for _, runner := range a.runners {
		runners = append(runners, runner)
	}
	a.runnerMutex.RUnlock()

	for _, runner := range runners {
		runner.Stop()
	}
}

// GetOutput 获取输出
func (a *App) GetOutput() []string {
	var output []string

	// 非阻塞读取所有可用输出
	for {
		select {
		case msg := <-a.outputChannel:
			output = append(output, msg)
		default:
			return output
		}
	}
}

// ClearOutput 清空输出
func (a *App) ClearOutput() {
	// 清空通道中的所有消息
	for {
		select {
		case <-a.outputChannel:
		default:
			return
		}
	}
}

// GetStatus 获取状态
func (a *App) GetStatus() *define.CommandStatus {
	a.runnerMutex.RLock()
	defer a.runnerMutex.RUnlock()

	return &define.CommandStatus{
		IsRunning: a.isRunning,
		Command:   a.currentCmd,
	}
}

// TestMachineConnection 测试机器连接
func (a *App) TestMachineConnection(machineName string) error {
	machineConfig := a.configManager.GetMachine(machineName)
	if machineConfig == nil {
		return fmt.Errorf("未找到机器配置: %s", machineName)
	}

	sshClient := machine.NewSSHClient(machineConfig)
	return sshClient.TestConnection()
}

// GetMachines 获取所有机器配置
func (a *App) GetMachines() []define.Machine {
	root := a.configManager.GetRoot()
	if root == nil {
		return nil
	}
	return root.Machines
}

// AddMachine 添加机器配置
func (a *App) AddMachine(machine define.Machine) error {
	root := a.configManager.GetRoot()
	if root == nil {
		return fmt.Errorf("配置未加载")
	}

	root.Machines = append(root.Machines, machine)
	return a.configManager.SaveConfig(root)
}

// UpdateMachine 更新机器配置
func (a *App) UpdateMachine(machineName string, machine define.Machine) error {
	root := a.configManager.GetRoot()
	if root == nil {
		return fmt.Errorf("配置未加载")
	}

	for i, m := range root.Machines {
		if m.Name == machineName {
			root.Machines[i] = machine
			return a.configManager.SaveConfig(root)
		}
	}

	return fmt.Errorf("未找到机器: %s", machineName)
}

// DeleteMachine 删除机器配置
func (a *App) DeleteMachine(machineName string) error {
	root := a.configManager.GetRoot()
	if root == nil {
		return fmt.Errorf("配置未加载")
	}

	for i, m := range root.Machines {
		if m.Name == machineName {
			root.Machines = append(root.Machines[:i], root.Machines[i+1:]...)
			return a.configManager.SaveConfig(root)
		}
	}

	return fmt.Errorf("未找到机器: %s", machineName)
}
