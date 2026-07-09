package app

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"quick-cmd/data"
	"quick-cmd/define"
	"quick-cmd/machine"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx              context.Context
	configManager    *data.ConfigManager
	sessionManager   *data.SessionManager
	logManager       *data.LogManager
	subProjectRunner *machine.SubProjectRunner
	shellPool        *machine.ShellSessionPool
	outputChannel    chan string
	outputIngress    chan string
	executionMutex   sync.RWMutex
	logEnabled       bool
}

// NewApp creates a new App application struct
func NewApp(sessionID string) *App {
	sessionManager, err := data.NewSessionManager(sessionID)
	if err != nil {
		println("创建会话管理器失败:", err.Error())
		sessionManager, _ = data.NewSessionManager(data.NewSessionID())
	}

	configManager := data.NewConfigManager("", sessionManager)
	logManager := data.NewLogManager("~/.cmd-config/logs")

	app := &App{
		outputChannel:  make(chan string, 1000),
		outputIngress:  make(chan string, 1000),
		configManager:  configManager,
		sessionManager: sessionManager,
		logManager:     logManager,
		shellPool:      machine.NewShellSessionPool(),
	}
	app.refreshLogSettings()
	go app.outputEventLoop()
	return app
}

func (a *App) refreshLogSettings() {
	globalConfig, err := a.configManager.GetGlobalConfig()
	if err != nil || globalConfig == nil {
		a.logEnabled = false
		return
	}
	a.logEnabled = globalConfig.LogSettings.Enabled
	if globalConfig.LogSettings.Path != "" {
		a.logManager.SetBasePath(globalConfig.LogSettings.Path)
	}
}

// Startup is called when the app starts up
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.setupSubProjectRunner()

	// 尝试加载配置文件，如果不存在则创建默认配置
	if _, err := a.configManager.LoadConfig(); err != nil {
		if os.IsNotExist(err) {
			println("配置文件不存在，创建默认配置")
			data.CreateDefaultConfig("config.yaml")
			if _, loadErr := a.configManager.LoadConfig(); loadErr != nil {
				println("加载默认配置文件失败:", loadErr.Error())
			} else {
				println("默认配置文件加载成功")
			}
		} else {
			println("加载配置文件失败:", err.Error())
		}
	} else {
		println("配置文件加载成功")
	}
	a.applyWindowTheme(a.GetThemeSettings().Mode)
}

// DomReady is called after front-end resources have been loaded
func (a *App) DomReady(ctx context.Context) {
	// Add your action here
}

// BeforeClose is called when the application is about to quit
func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	a.StopAllSubProjects()
	for _, session := range a.shellPool.ListSessions() {
		_ = a.shellPool.Disconnect(session.MachineName, a.shellHandlerFor(session.MachineName))
	}
	return false
}

// Shutdown is called during application termination
func (a *App) Shutdown(ctx context.Context) {
	close(a.outputIngress)
}

func (a *App) setupSubProjectRunner() {
	a.subProjectRunner = machine.NewSubProjectRunner(a.configManager)
	a.subProjectRunner.SetStatusChangeHandler(a.emitExecutionStatus)
}

func (a *App) outputEventLoop() {
	for msg := range a.outputIngress {
		select {
		case a.outputChannel <- msg:
		default:
		}
		if a.logEnabled {
			a.logManager.WriteLine(msg)
		}
		if a.ctx != nil {
			wailsRuntime.EventsEmit(a.ctx, "output:line", msg)
		}
	}
}

func (a *App) outputWriter() chan<- string {
	return a.outputIngress
}

func (a *App) pushOutput(msg string) {
	select {
	case a.outputIngress <- msg:
	default:
	}
}

func (a *App) emitOutputClear() {
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "output:clear", nil)
	}
}

func (a *App) emitExecutionStatus(status *define.SubProjectStatus) {
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "execution:status", status)
	}
}

func (a *App) shellHandlerFor(machineName string) machine.ShellOutputHandler {
	return machine.ShellOutputHandler{
		OnLine: func(line string) {
			a.pushShellOutput(machineName, line)
		},
		OnData: func(data []byte) {
			a.pushShellData(machineName, data)
		},
		OnStatus: func(_ *define.ShellStatus) {
			go a.emitShellSessions()
		},
		OnClose: func() {
			if machineName != "" {
				a.shellPool.RemoveSession(machineName)
			}
			go a.emitShellSessions()
		},
	}
}

func (a *App) pushShellData(machineName string, data []byte) {
	if a.ctx != nil && len(data) > 0 {
		wailsRuntime.EventsEmit(a.ctx, "shell:data", map[string]interface{}{
			"machineName": machineName,
			"data":        base64.StdEncoding.EncodeToString(data),
		})
	}
}

func (a *App) pushShellOutput(machineName, msg string) {
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "shell:line", map[string]interface{}{
			"machineName": machineName,
			"line":        msg,
		})
	}
}

func (a *App) emitShellClear(machineName string) {
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "shell:clear", map[string]interface{}{
			"machineName": machineName,
		})
	}
}

func (a *App) emitShellSessions() {
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "shell:status", a.shellPool.ListSessions())
	}
}

// GetConfig 获取配置
func (a *App) GetConfig() (*define.Root, error) {
	return a.configManager.LoadConfig()
}

// GetConfigForRefresh 获取配置（用于刷新，不更新全局配置）
func (a *App) GetConfigForRefresh() (*define.Root, error) {
	return a.configManager.LoadConfigForRefresh()
}

// SaveConfig 保存配置
func (a *App) SaveConfig(root *define.Root) error {
	return a.configManager.SaveConfig(root)
}

// ExecuteSubProject 执行 SubProject
func (a *App) ExecuteSubProject(projectName, subProjectName string) error {
	a.executionMutex.Lock()
	defer a.executionMutex.Unlock()

	if a.shellPool.IsAnyConnected() {
		return fmt.Errorf("Shell 模式已连接，请先断开后再执行任务")
	}

	// 在执行前刷新配置，确保读取到最新的 SubProject 定义
	if _, err := a.configManager.LoadConfigForRefresh(); err != nil {
		return err
	}

	// 新任务开始前清空终端
	a.ClearOutput()

	// 异步执行 SubProject
	go func() {
		success := true
		summary := "执行完成"
		if a.logEnabled {
			if _, err := a.logManager.StartSession(projectName, subProjectName); err != nil {
				a.pushOutput(fmt.Sprintf("日志落盘启动失败: %s", err.Error()))
			}
		}

		if err := a.subProjectRunner.ExecuteSubProject(projectName, subProjectName, a.outputWriter()); err != nil {
			a.pushOutput(fmt.Sprintf("执行失败: %s", err.Error()))
			success = false
			summary = err.Error()
		}

		if a.logEnabled {
			a.logManager.FinishSession(success, summary)
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
	for {
		select {
		case <-a.outputIngress:
		default:
			goto drainedIngress
		}
	}
drainedIngress:
	for {
		select {
		case <-a.outputChannel:
		default:
			goto drainedChannel
		}
	}
drainedChannel:
	a.emitOutputClear()
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

	sshClient := machine.NewSSHClient(machineConfig, a.configManager.GetWorkPathVars())
	return sshClient.TestConnection()
}

// GetMachines 获取所有机器配置（从全局配置）
func (a *App) GetMachines() []define.Machine {
	return a.configManager.GetAllMachinesFromGlobal()
}

// AddMachine 添加机器配置（到全局配置）
func (a *App) AddMachine(machine define.Machine) error {
	return a.configManager.AddMachineToGlobal(&machine)
}

// AddMachineWithEvent 添加机器配置（带事件通知）
func (a *App) AddMachineWithEvent(machine define.Machine) error {
	err := a.configManager.AddMachineToGlobal(&machine)
	if err != nil {
		a.emitOperationEvent(define.OpTypeMachineConfig, fmt.Sprintf("添加机器配置失败: %s", err.Error()), define.MsgTypeError, false, nil)
		return err
	}

	a.emitOperationEvent(define.OpTypeMachineConfig, fmt.Sprintf("成功添加机器配置: %s", machine.Name), define.MsgTypeSuccess, false, map[string]interface{}{
		"machineName": machine.Name,
	})
	return nil
}

// UpdateMachine 更新机器配置（在全局配置中）
func (a *App) UpdateMachine(machineName string, machine define.Machine) error {
	// 先删除旧配置
	if err := a.configManager.RemoveMachineFromGlobal(machineName); err != nil {
		return fmt.Errorf("删除旧配置失败: %w", err)
	}

	// 添加新配置
	return a.configManager.AddMachineToGlobal(&machine)
}

// UpdateMachineWithEvent 更新机器配置（带事件通知）
func (a *App) UpdateMachineWithEvent(machineName string, machine define.Machine) error {
	// 先删除旧配置
	if err := a.configManager.RemoveMachineFromGlobal(machineName); err != nil {
		a.emitOperationEvent(define.OpTypeMachineConfig, fmt.Sprintf("删除旧配置失败: %s", err.Error()), define.MsgTypeError, false, nil)
		return fmt.Errorf("删除旧配置失败: %w", err)
	}

	// 添加新配置
	if err := a.configManager.AddMachineToGlobal(&machine); err != nil {
		a.emitOperationEvent(define.OpTypeMachineConfig, fmt.Sprintf("更新机器配置失败: %s", err.Error()), define.MsgTypeError, false, nil)
		return err
	}

	a.emitOperationEvent(define.OpTypeMachineConfig, fmt.Sprintf("成功更新机器配置: %s", machine.Name), define.MsgTypeSuccess, false, map[string]interface{}{
		"machineName": machine.Name,
		"oldName":     machineName,
	})
	return nil
}

// DeleteMachine 删除机器配置（从全局配置）
func (a *App) DeleteMachine(machineName string) error {
	return a.configManager.RemoveMachineFromGlobal(machineName)
}

// DeleteMachineWithEvent 删除机器配置（带事件通知）
func (a *App) DeleteMachineWithEvent(machineName string) error {
	err := a.configManager.RemoveMachineFromGlobal(machineName)
	if err != nil {
		a.emitOperationEvent(define.OpTypeMachineConfig, fmt.Sprintf("删除机器配置失败: %s", err.Error()), define.MsgTypeError, false, nil)
		return err
	}

	a.emitOperationEvent(define.OpTypeMachineConfig, fmt.Sprintf("成功删除机器配置: %s", machineName), define.MsgTypeSuccess, false, map[string]interface{}{
		"machineName": machineName,
	})
	return nil
}

// GetGlobalConfig 获取全局配置
func (a *App) GetGlobalConfig() (*data.GlobalConfig, error) {
	return a.configManager.GetGlobalConfig()
}

// GetGlobalConfigForRefresh 获取全局配置（用于刷新，从文件重新读取）
func (a *App) GetGlobalConfigForRefresh() (*data.GlobalConfig, error) {
	globalConfig, err := a.configManager.GetGlobalConfigForRefresh()
	if err != nil {
		return nil, err
	}
	a.UpdateApplicationMenu()
	return globalConfig, nil
}

// SaveGlobalConfig 保存全局配置
func (a *App) SaveGlobalConfig(config *data.GlobalConfig) error {
	return a.configManager.SaveGlobalConfig(config)
}

// GetConfigFiles 获取所有配置文件列表
func (a *App) GetConfigFiles() ([]string, error) {
	return a.configManager.GetConfigFiles()
}

// SwitchConfigFileWithEvent 切换配置文件（带事件通知）
func (a *App) SwitchConfigFileWithEvent(configPath string) error {
	// 停止所有正在运行的 SubProjects
	if err := a.StopAllSubProjects(); err != nil {
		// 记录错误但不阻止切换
		fmt.Printf("停止运行中的项目时出错: %v\n", err)
		a.emitOperationEvent(define.OpTypeSwitchConfig, fmt.Sprintf("停止运行中的项目时出错: %v", err), define.MsgTypeWarning, false, nil)
	}

	// 清空输出
	a.ClearOutput()

	// 切换配置文件
	if err := a.configManager.SwitchConfigFile(configPath); err != nil {
		a.emitOperationEvent(define.OpTypeSwitchConfig, fmt.Sprintf("%v", err.Error()), define.MsgTypeError, true, nil)
		return fmt.Errorf("切换配置文件失败: %w", err)
	}

	// 重新创建 SubProjectRunner
	a.setupSubProjectRunner()

	// 发送事件到前端通知配置文件已切换（保持向后兼容）
	if a.ctx != nil {
		fmt.Printf("发送 config:changed 事件，配置文件: %s\n", configPath)
		wailsRuntime.EventsEmit(a.ctx, "config:changed", map[string]interface{}{
			"configPath": configPath,
			"timestamp":  time.Now().Unix(),
		})
		fmt.Println("事件发送完成")
	}

	return nil
}

// SetMachineSensitiveData 设置机器敏感数据
func (a *App) SetMachineSensitiveData(machineName string, sensitiveData define.SensitiveData) error {
	machine := a.configManager.GetMachineFromGlobal(machineName)
	if machine == nil {
		return fmt.Errorf("未找到机器: %s", machineName)
	}

	// 设置敏感数据并加密
	if err := machine.SetSensitiveData(&sensitiveData); err != nil {
		return fmt.Errorf("设置敏感数据失败: %w", err)
	}
	// 将更新后的机器配置重新保存到全局配置文件中
	return a.configManager.AddMachineToGlobal(machine)
}

// GetMachineSensitiveData 获取机器敏感数据
func (a *App) GetMachineSensitiveData(machineName string) (*define.SensitiveData, error) {
	machine := a.configManager.GetMachineFromGlobal(machineName)
	if machine == nil {
		return nil, fmt.Errorf("未找到机器: %s", machineName)
	}

	return machine.GetSensitiveData()
}

// ClearMachineSensitiveData 清除机器敏感数据缓存
func (a *App) ClearMachineSensitiveData(machineName string) error {
	machine := a.configManager.GetMachineFromGlobal(machineName)
	if machine == nil {
		return fmt.Errorf("未找到机器: %s", machineName)
	}

	machine.ClearSensitiveData()
	return nil
}

// SelectKeyFile 选择密钥文件
func (a *App) SelectKeyFile() (string, error) {
	filePath, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title:           "选择SSH密钥文件",
		ShowHiddenFiles: true,
	})
	if err != nil {
		return "", fmt.Errorf("选择文件失败: %w", err)
	}

	return filePath, nil
}

// OpenMachineConfig 打开机器配置对话框（供菜单调用）
func (a *App) OpenMachineConfig() {
	// 发送事件到前端打开机器配置对话框
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "open:machine-config", map[string]interface{}{
			"timestamp": time.Now().Unix(),
		})
		fmt.Println("发送打开机器配置事件")
	} else {
		fmt.Println("警告: ctx 为 nil，无法发送事件")
		a.emitOperationEvent(define.OpTypeMachineConfig, "无法发送事件，ctx 为 nil", define.MsgTypeError, false, nil)
	}
}

// RefreshAll 全局刷新功能
func (a *App) RefreshConfigMenu() error {
	// 刷新配置列表时，确保执行上下文也同步使用新配置
	// 否则 subProjectRunner 内部仍会持有旧的 configManager 引用。
	_ = a.StopAllSubProjects()
	a.ClearOutput()

	a.configManager = data.NewConfigManager("", a.sessionManager)
	a.setupSubProjectRunner()
	if a.ctx != nil {
		err := a.UpdateApplicationMenu()
		if err != nil {
			fmt.Printf("更新菜单失败: %v\n", err)
		} else {
			fmt.Println("菜单更新完成")
		}
	}
	return nil
}

// RefreshConfigMenuWithEvent 全局刷新功能（带事件通知）
func (a *App) RefreshConfigMenuWithEvent() error {
	// 刷新配置列表时，确保执行上下文也同步使用新配置
	// 否则 subProjectRunner 内部仍会持有旧的 configManager 引用。
	_ = a.StopAllSubProjects()
	a.ClearOutput()

	a.configManager = data.NewConfigManager("", a.sessionManager)
	a.setupSubProjectRunner()
	if a.ctx != nil {
		err := a.UpdateApplicationMenu()
		if err != nil {
			a.emitOperationEvent(define.OpTypeRefreshConfig, fmt.Sprintf("更新菜单失败: %v", err), define.MsgTypeError, true, nil)
			return err
		} else {
			fmt.Println("菜单更新完成")
		}
	}

	// 前端仅在 needReload 为 true 时会重载页面，从而刷新“实际上下文”视图。
	a.emitOperationEvent(define.OpTypeRefreshConfig, "配置列表刷新成功", define.MsgTypeSuccess, true, nil)
	return nil
}

// UpdateApplicationMenu 通知前端刷新应用内菜单并更新窗口标题
func (a *App) UpdateApplicationMenu() error {
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "menu:refresh", nil)
	}
	globalConfig, _ := a.GetGlobalConfig()
	if globalConfig.WindowsName != "" {
		wailsRuntime.WindowSetTitle(a.ctx, globalConfig.WindowsName)
	} else {
		wailsRuntime.WindowSetTitle(a.ctx, "Quick Cmd")
	}
	return nil
}

// CreateApplicationMenu 创建应用程序菜单的公共方法
func (a *App) CreateApplicationMenu() *menu.Menu {
	appMenu := menu.NewMenu()

	// 文件菜单
	fileMenu := appMenu.AddSubmenu("文件")
	fileMenu.AddText("新建窗口", keys.CmdOrCtrl("n"), func(_ *menu.CallbackData) {
		NewWindow()
	})

	fileMenu.AddSeparator()
	// 添加机器配置菜单
	configMenu := appMenu.AddSubmenu("设置")
	// 配置菜单
	configFileMenu := appMenu.AddSubmenu("配置文件")
	// 动态加载配置文件列表
	configFiles, err := a.GetConfigFiles()
	if err != nil {
		// 如果获取失败，添加默认项
		configFileMenu.AddText("无法加载配置文件", keys.CmdOrCtrl("r"), func(_ *menu.CallbackData) {
			a.RefreshConfigMenuWithEvent()
		})
	} else {
		// 获取当前配置文件
		globalConfig, _ := a.GetGlobalConfig()
		currentConfig := a.configManager.GetConfigPath()
		if currentConfig == "" && globalConfig != nil {
			currentConfig = globalConfig.LastOpenedFile
		}

		// 为每个配置文件添加菜单项
		for _, configFile := range configFiles {
			// 获取文件名（去掉路径）
			fileName := getFileName(configFile)
			// 创建菜单项
			_ = configFileMenu.AddRadio(fileName, configFile == currentConfig, nil, func(data *menu.CallbackData) {
				// 切换配置文件
				switchConfigFile(a, configFile)
			})
		}

		// 添加分隔符和刷新选项
		configFileMenu.AddSeparator()
		configFileMenu.AddText("刷新配置列表", keys.CmdOrCtrl("r"), func(_ *menu.CallbackData) {
			a.RefreshConfigMenuWithEvent()
		})
		configFileMenu.AddText("打开全局配置", nil, func(_ *menu.CallbackData) {
			// 获取全局配置文件路径 GlobalConfigManager
			globalConfigPath := a.configManager.GetGlobalConfigPath()
			if globalConfigPath != "" {
				OpenCurrentConfig(globalConfigPath)
			}
		})

		configFileMenu.AddText("打开当前配置", nil, func(_ *menu.CallbackData) {
			a.OpenCurrentConfigWithEvent()
		})
	}

	configMenu.AddText("机器配置", keys.CmdOrCtrl("m"), func(_ *menu.CallbackData) {
		// 打开机器配置对话框
		a.OpenMachineConfig()
	})

	configMenu.AddText("环境变量", keys.CmdOrCtrl("e"), func(_ *menu.CallbackData) {
		// 打开环境变量配置对话框
		a.OpenWorkPathConfig()
	})

	configMenu.AddSeparator()
	configMenu.AddText("业务配置编辑", keys.CmdOrCtrl(","), func(_ *menu.CallbackData) {
		a.OpenConfigEditor()
	})
	configMenu.AddText("系统设置", nil, func(_ *menu.CallbackData) {
		a.OpenSystemSettings()
	})
	configMenu.AddText("执行历史", nil, func(_ *menu.CallbackData) {
		a.OpenExecutionHistory()
	})

	// 帮助菜单
	helpMenu := appMenu.AddSubmenu("帮助")
	helpMenu.AddText("关于", nil, func(_ *menu.CallbackData) {
		// 显示关于信息
		a.OpenAbout()
	})

	return appMenu
}

// getFileName 获取文件名（去掉路径）
func getFileName(filePath string) string {
	if filePath == "" {
		return ""
	}
	// 简单的路径分割，支持 Unix 和 Windows 路径
	for i := len(filePath) - 1; i >= 0; i-- {
		if filePath[i] == '/' || filePath[i] == '\\' {
			return filePath[i+1:]
		}
	}
	return filePath
}

// switchConfigFile 切换配置文件
func switchConfigFile(appInstance *App, configFile string) {
	err := appInstance.SwitchConfigFileWithEvent(configFile)
	if err != nil {
		// 错误已经通过事件发送，这里不需要额外处理
		println("切换配置文件失败:", err.Error())
	} else {
		println("成功切换到配置文件:", configFile)
		// 配置文件切换成功后，前端会通过事件监听自动刷新
		// 这里不需要额外的操作，因为 SwitchConfigFileWithEvent 已经发送了事件
	}
}

// NewWindow 创建新窗口（通过启动新进程实现，独立会话）
func NewWindow() {
	execPath, err := os.Executable()
	if err != nil {
		println("启动新窗口失败:", err.Error())
		return
	}
	sessionID := data.NewSessionID()
	cmd := exec.Command(execPath, "-session="+sessionID)
	if err := cmd.Start(); err != nil {
		println("启动新窗口失败:", err.Error())
	}
}

// NewWindow 供前端调用的新建窗口入口
func (a *App) NewWindow() {
	NewWindow()
}

// GetCurrentConfigPath 获取当前业务配置文件路径
func (a *App) GetCurrentConfigPath() string {
	currentConfig := a.configManager.GetConfigPath()
	if currentConfig == "" {
		globalConfig, _ := a.GetGlobalConfig()
		if globalConfig != nil {
			currentConfig = globalConfig.LastOpenedFile
		}
	}
	return currentConfig
}

// OpenCurrentConfig 打开当前配置文件（供菜单调用）
func OpenCurrentConfig(lastOpenedFile string) {
	if lastOpenedFile == "" {
		fmt.Println("没有找到当前配置文件")
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(lastOpenedFile); os.IsNotExist(err) {
		fmt.Printf("配置文件不存在: %s\n", lastOpenedFile)
		return
	}

	// 使用系统默认程序打开配置文件
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("open", lastOpenedFile)
	case "windows": // Windows
		cmd = exec.Command("cmd", "/c", "start", "", lastOpenedFile)
	case "linux": // Linux
		cmd = exec.Command("xdg-open", lastOpenedFile)
	default:
		fmt.Printf("不支持的操作系统: %s\n", runtime.GOOS)
		return
	}

	err := cmd.Run()
	if err != nil {
		fmt.Printf("打开配置文件失败: %v\n", err)
	} else {
		fmt.Printf("成功打开配置文件: %s\n", lastOpenedFile)
	}
}

// OpenCurrentConfigWithEvent 打开当前配置文件（带事件通知）
func (a *App) OpenCurrentConfigWithEvent() {
	globalConfig, err := a.GetGlobalConfig()
	if err != nil {
		a.emitOperationEvent(define.OpTypeOpenConfig, fmt.Sprintf("获取全局配置失败: %s", err.Error()), define.MsgTypeError, false, nil)
		return
	}

	lastOpenedFile := globalConfig.LastOpenedFile
	if lastOpenedFile == "" {
		a.emitOperationEvent(define.OpTypeOpenConfig, "没有找到当前配置文件", define.MsgTypeWarning, false, nil)
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(lastOpenedFile); os.IsNotExist(err) {
		a.emitOperationEvent(define.OpTypeOpenConfig, fmt.Sprintf("配置文件不存在: %s", lastOpenedFile), define.MsgTypeError, false, nil)
		return
	}

	// 使用系统默认程序打开配置文件
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("open", lastOpenedFile)
	case "windows": // Windows
		cmd = exec.Command("cmd", "/c", "start", "", lastOpenedFile)
	case "linux": // Linux
		cmd = exec.Command("xdg-open", lastOpenedFile)
	default:
		a.emitOperationEvent(define.OpTypeOpenConfig, fmt.Sprintf("不支持的操作系统: %s", runtime.GOOS), define.MsgTypeError, false, nil)
		return
	}

	err = cmd.Run()
	if err != nil {
		a.emitOperationEvent(define.OpTypeOpenConfig, fmt.Sprintf("打开配置文件失败: %v", err), define.MsgTypeError, false, nil)
		return
	}

	a.emitOperationEvent(define.OpTypeOpenConfig, fmt.Sprintf("成功打开配置文件: %s", lastOpenedFile), define.MsgTypeSuccess, false, nil)
}

// OpenGlobalConfigWithEvent 打开全局配置文件（带事件通知）
func (a *App) OpenGlobalConfigWithEvent() {
	globalConfigPath := a.configManager.GetGlobalConfigPath()
	if globalConfigPath == "" {
		a.emitOperationEvent(define.OpTypeOpenConfig, "没有找到全局配置文件", define.MsgTypeWarning, false, nil)
		return
	}

	if _, err := os.Stat(globalConfigPath); os.IsNotExist(err) {
		a.emitOperationEvent(define.OpTypeOpenConfig, fmt.Sprintf("全局配置文件不存在: %s", globalConfigPath), define.MsgTypeError, false, nil)
		return
	}

	if err := openWithSystemApp(globalConfigPath); err != nil {
		a.emitOperationEvent(define.OpTypeOpenConfig, fmt.Sprintf("打开全局配置文件失败: %v", err), define.MsgTypeError, false, nil)
		return
	}

	a.emitOperationEvent(define.OpTypeOpenConfig, fmt.Sprintf("成功打开全局配置文件: %s", globalConfigPath), define.MsgTypeSuccess, false, nil)
}

// GetWorkPaths 获取所有工作路径
func (a *App) GetWorkPaths() map[string]string {
	return a.configManager.GetAllWorkPathsFromGlobal()
}

// AddWorkPath 添加工作路径
func (a *App) AddWorkPath(key, value string) error {
	return a.configManager.AddWorkPathToGlobal(key, value)
}

// AddWorkPathWithEvent 添加工作路径（带事件通知）
func (a *App) AddWorkPathWithEvent(key, value string) error {
	err := a.configManager.AddWorkPathToGlobal(key, value)
	if err != nil {
		a.emitOperationEvent(define.OpTypeEnvConfig, fmt.Sprintf("添加环境变量失败: %s", err.Error()), define.MsgTypeError, false, nil)
		return err
	}

	a.emitOperationEvent(define.OpTypeEnvConfig, fmt.Sprintf("成功添加环境变量: %s", key), define.MsgTypeSuccess, false, nil)
	return nil
}

// UpdateWorkPath 更新工作路径
func (a *App) UpdateWorkPath(key, value string) error {
	return a.configManager.UpdateWorkPathInGlobal(key, value)
}

// UpdateWorkPathWithEvent 更新工作路径（带事件通知）
func (a *App) UpdateWorkPathWithEvent(key, value string) error {
	err := a.configManager.UpdateWorkPathInGlobal(key, value)
	if err != nil {
		a.emitOperationEvent(define.OpTypeEnvConfig, fmt.Sprintf("更新环境变量失败: %s", err.Error()), define.MsgTypeError, false, nil)
		return err
	}

	a.emitOperationEvent(define.OpTypeEnvConfig, fmt.Sprintf("成功更新环境变量: %s", key), define.MsgTypeSuccess, false, nil)
	return nil
}

// DeleteWorkPath 删除工作路径
func (a *App) DeleteWorkPath(key string) error {
	return a.configManager.RemoveWorkPathFromGlobal(key)
}

// DeleteWorkPathWithEvent 删除工作路径（带事件通知）
func (a *App) DeleteWorkPathWithEvent(key string) error {
	err := a.configManager.RemoveWorkPathFromGlobal(key)
	if err != nil {
		a.emitOperationEvent(define.OpTypeEnvConfig, fmt.Sprintf("删除环境变量失败: %s", err.Error()), define.MsgTypeError, false, nil)
		return err
	}

	a.emitOperationEvent(define.OpTypeEnvConfig, fmt.Sprintf("成功删除环境变量: %s", key), define.MsgTypeSuccess, false, nil)
	return nil
}

// OpenWorkPathConfig 打开工作路径配置对话框（供菜单调用）
func (a *App) OpenWorkPathConfig() {
	// 发送事件到前端打开工作路径配置对话框
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "open:workpath-config", map[string]interface{}{
			"timestamp": time.Now().Unix(),
		})
		fmt.Println("发送打开工作路径配置事件")
	} else {
		fmt.Println("警告: ctx 为 nil，无法发送事件")
		a.emitOperationEvent(define.OpTypeEnvConfig, "无法发送事件，ctx 为 nil", define.MsgTypeError, false, nil)
	}
}

// OpenAbout 打开关于对话框（供菜单调用）
func (a *App) OpenAbout() {
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "open:about", map[string]interface{}{
			"timestamp": time.Now().Unix(),
		})
		fmt.Println("发送打开关于对话框事件")
		return
	}
	fmt.Println("警告: ctx 为 nil，无法发送关于事件")
}

// emitOperationEvent 发送操作事件到前端
func (a *App) emitOperationEvent(eventType, message, messageType string, needReload bool, data any) {
	if a.ctx == nil {
		fmt.Printf("警告: ctx 为 nil，无法发送事件 %s\n", eventType)
		return
	}

	event := define.OperationEvent{
		Type:        eventType,
		NeedReload:  needReload,
		Message:     message,
		MessageType: messageType,
		Timestamp:   time.Now().Unix(),
		Data:        data,
	}

	wailsRuntime.EventsEmit(a.ctx, "operation:result", event)
	fmt.Printf("发送操作事件: %s - %s (%s)\n", eventType, message, messageType)
}

// GetSessionInfo 获取当前窗口会话信息
func (a *App) GetSessionInfo() data.SessionState {
	if a.sessionManager == nil {
		return data.SessionState{}
	}
	return a.sessionManager.GetState()
}

// GetSystemSettings 获取系统设置
func (a *App) GetSystemSettings() (*data.GlobalConfig, error) {
	return a.configManager.GetGlobalConfig()
}

// SaveSystemSettings 保存系统设置
func (a *App) SaveSystemSettings(config *data.GlobalConfig) error {
	if err := a.configManager.SaveGlobalConfig(config); err != nil {
		return err
	}
	a.refreshLogSettings()
	if a.sessionManager != nil && config.ThemeSettings.Mode != "" {
		_ = a.sessionManager.SetTheme(config.ThemeSettings.Mode, config.ThemeSettings.TerminalPreset)
	}
	return nil
}

// GetExecutionLogs 获取执行历史列表
func (a *App) GetExecutionLogs(limit int) ([]data.LogEntry, error) {
	return a.logManager.ListLogs(limit)
}

// ReadExecutionLog 读取执行日志内容
func (a *App) ReadExecutionLog(fileName string) (string, error) {
	return a.logManager.ReadLog(fileName)
}

// OpenExecutionLog 用系统默认程序打开日志文件
func (a *App) OpenExecutionLog(fileName string) error {
	logs, err := a.logManager.ListLogs(200)
	if err != nil {
		return err
	}
	for _, entry := range logs {
		if entry.FileName == fileName {
			return openWithSystemApp(entry.FullPath)
		}
	}
	return fmt.Errorf("未找到日志文件: %s", fileName)
}

// OpenConfigEditor 打开业务配置编辑器
func (a *App) OpenConfigEditor() {
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "open:config-editor", map[string]interface{}{
			"timestamp": time.Now().Unix(),
		})
	}
}

// OpenSystemSettings 打开系统设置
func (a *App) OpenSystemSettings() {
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "open:system-settings", map[string]interface{}{
			"timestamp": time.Now().Unix(),
		})
	}
}

// OpenExecutionHistory 打开执行历史
func (a *App) OpenExecutionHistory() {
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "open:execution-history", map[string]interface{}{
			"timestamp": time.Now().Unix(),
		})
	}
}

// GetThemeSettings 获取当前主题设置（会话优先，其次全局）
func (a *App) GetThemeSettings() data.ThemeSettings {
	globalConfig, err := a.configManager.GetGlobalConfig()
	settings := data.ThemeSettings{Mode: "light", TerminalPreset: "classic"}
	if err == nil && globalConfig != nil {
		settings = globalConfig.ThemeSettings
	}
	if a.sessionManager != nil {
		state := a.sessionManager.GetState()
		if state.Theme != "" {
			settings.Mode = state.Theme
		}
		if state.TerminalPreset != "" {
			settings.TerminalPreset = state.TerminalPreset
		}
	}
	if settings.Mode == "" {
		settings.Mode = "light"
	}
	if settings.TerminalPreset == "" {
		settings.TerminalPreset = "classic"
	}
	return settings
}

// SaveThemeSettings 保存主题设置到会话与全局
func (a *App) SaveThemeSettings(settings data.ThemeSettings) error {
	if a.sessionManager != nil {
		if err := a.sessionManager.SetTheme(settings.Mode, settings.TerminalPreset); err != nil {
			return err
		}
	}
	globalConfig, err := a.configManager.GetGlobalConfig()
	if err != nil {
		return err
	}
	globalConfig.ThemeSettings = settings
	return a.configManager.SaveGlobalConfig(globalConfig)
}

func (a *App) applyWindowTheme(mode string) {
	if a.ctx == nil {
		return
	}
	switch mode {
	case "dark":
		wailsRuntime.WindowSetDarkTheme(a.ctx)
		wailsRuntime.WindowSetBackgroundColour(a.ctx, 20, 20, 20, 255)
	case "system":
		wailsRuntime.WindowSetSystemDefaultTheme(a.ctx)
	default:
		wailsRuntime.WindowSetLightTheme(a.ctx)
		wailsRuntime.WindowSetBackgroundColour(a.ctx, 255, 255, 255, 255)
	}
}

// ConnectShell 连接远程 Shell（支持多会话）
func (a *App) ConnectShell(machineName string) error {
	a.executionMutex.Lock()
	status := a.subProjectRunner.GetExecutionStatus()
	if status.IsRunning {
		a.executionMutex.Unlock()
		return fmt.Errorf("任务正在执行，请先停止后再使用 Shell")
	}
	a.executionMutex.Unlock()

	machineConfig := a.configManager.GetMachine(machineName)
	if machineConfig == nil {
		return fmt.Errorf("未找到机器配置: %s", machineName)
	}

	err := a.shellPool.Connect(machineName, machineConfig, a.configManager.GetWorkPathVars(), a.shellHandlerFor(machineName))
	if err == nil {
		a.emitShellSessions()
	}
	return err
}

// DisconnectShell 断开指定机器的 Shell
func (a *App) DisconnectShell(machineName string) error {
	a.executionMutex.Lock()
	defer a.executionMutex.Unlock()
	err := a.shellPool.Disconnect(machineName, a.shellHandlerFor(machineName))
	a.emitShellSessions()
	return err
}

// SendShellInput 向指定 PTY Shell 发送输入
func (a *App) SendShellInput(machineName, input string) error {
	a.executionMutex.RLock()
	defer a.executionMutex.RUnlock()
	return a.shellPool.SendInput(machineName, input)
}

// SendShellInterrupt 向指定 PTY Shell 发送 Ctrl+C
func (a *App) SendShellInterrupt(machineName string) error {
	a.executionMutex.RLock()
	defer a.executionMutex.RUnlock()
	return a.shellPool.SendInterrupt(machineName)
}

// ResizeShell 调整指定 PTY 终端尺寸
func (a *App) ResizeShell(machineName string, cols, rows int) error {
	a.executionMutex.RLock()
	defer a.executionMutex.RUnlock()
	return a.shellPool.Resize(machineName, cols, rows)
}

// ExecuteShellCommand 兼容旧接口
func (a *App) ExecuteShellCommand(machineName, command string) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return fmt.Errorf("命令不能为空")
	}
	if !strings.HasSuffix(command, "\n") {
		command += "\n"
	}
	return a.SendShellInput(machineName, command)
}

// StopShellCommand 兼容旧接口
func (a *App) StopShellCommand(machineName string) error {
	return a.SendShellInterrupt(machineName)
}

// GetShellSessions 获取所有 Shell 会话状态
func (a *App) GetShellSessions() []define.ShellStatus {
	return a.shellPool.ListSessions()
}

// GetShellStatus 兼容旧接口，返回首个会话或空状态
func (a *App) GetShellStatus() *define.ShellStatus {
	sessions := a.shellPool.ListSessions()
	if len(sessions) == 0 {
		return &define.ShellStatus{}
	}
	return &sessions[0]
}

// ClearShellOutput 清空指定终端显示
func (a *App) ClearShellOutput(machineName string) {
	a.emitShellClear(machineName)
}

func openWithSystemApp(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
	return cmd.Run()
}
