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
	ctx              context.Context
	configManager    *data.ConfigManager
	subProjectRunner *machine.SubProjectRunner
	outputChannel    chan string
	executionMutex   sync.RWMutex
}

// NewApp creates a new App application struct
func NewApp() *App {
	app := &App{
		outputChannel: make(chan string, 1000),
	}
	return app
}

// Startup is called when the app starts up
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.configManager = data.NewConfigManager("config.yaml")

	// 创建 SubProjectRunner
	a.subProjectRunner = machine.NewSubProjectRunner(a.configManager)

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
	a.StopAllSubProjects()
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

// ExecuteSubProject 执行 SubProject
func (a *App) ExecuteSubProject(projectName, subProjectName string) error {
	a.executionMutex.Lock()
	defer a.executionMutex.Unlock()

	// 异步执行 SubProject
	go func() {
		if err := a.subProjectRunner.ExecuteSubProject(projectName, subProjectName, a.outputChannel); err != nil {
			a.outputChannel <- fmt.Sprintf("执行失败: %s", err.Error())
		}
	}()

	return nil
}

// ExecuteCommand 执行命令 (保持向后兼容，但现在不推荐使用)
func (a *App) ExecuteCommand(projectName, subProjectName, commandName string) error {
	// 为了向后兼容，我们仍然保留这个方法，但它现在会执行整个 SubProject
	return a.ExecuteSubProject(projectName, subProjectName)
}

// StopSubProject 停止 SubProject
func (a *App) StopSubProject(projectName, subProjectName string) error {
	return a.subProjectRunner.StopSubProject(projectName, subProjectName)
}

// StopCommand 停止命令 (保持向后兼容)
func (a *App) StopCommand(projectName, subProjectName, commandName string) error {
	// 为了向后兼容，停止整个 SubProject
	return a.StopSubProject(projectName, subProjectName)
}

// StopAllSubProjects 停止所有 SubProjects
func (a *App) StopAllSubProjects() error {
	// 获取当前执行状态
	status := a.subProjectRunner.GetExecutionStatus()
	if status.IsRunning {
		return a.StopSubProject(status.ProjectName, status.SubProjectName)
	}
	return nil
}

// StopAllCommands 停止所有命令 (保持向后兼容)
func (a *App) StopAllCommands() {
	a.StopAllSubProjects()
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

// GetSubProjectStatus 获取 SubProject 状态
func (a *App) GetSubProjectStatus() *define.SubProjectStatus {
	return a.subProjectRunner.GetExecutionStatus()
}

// GetStatus 获取状态 (保持向后兼容)
func (a *App) GetStatus() *define.CommandStatus {
	subStatus := a.subProjectRunner.GetExecutionStatus()

	// 转换为旧的 CommandStatus 格式
	command := ""
	if subStatus.IsRunning {
		command = fmt.Sprintf("%s/%s/%s", subStatus.ProjectName, subStatus.SubProjectName, subStatus.CurrentCommand)
	}

	return &define.CommandStatus{
		IsRunning: subStatus.IsRunning,
		Command:   command,
		Output:    subStatus.Output,
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
