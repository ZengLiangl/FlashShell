package app

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"FlashDock/cmds"
	"FlashDock/data"
	"FlashDock/define"
	"FlashDock/machine"

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
	shellHistory     *data.ShellHistoryManager
	shellCmdHistory  *data.ShellCommandHistoryManager
	subProjectRunner *machine.SubProjectRunner
	shellPool        *machine.ShellSessionPool
	localShellPool   *machine.LocalShellPool
	shellAuxPool     *machine.ShellAuxPool
	tunnelMgr        *machine.TunnelManager
	transfers        *shellTransferStore
	externalEdits    *externalEditStore
	outputChannel    chan string
	outputIngress    chan string
	executionMutex   sync.RWMutex
	shellCwdMu       sync.RWMutex
	shellCwds        map[string]string
	logEnabled       bool
	quitMu           sync.Mutex
	allowQuit        bool
}

// NewApp creates a new App application struct
func NewApp(sessionID string) *App {
	sessionManager, err := data.NewSessionManager(sessionID)
	if err != nil {
		println("创建会话管理器失败:", err.Error())
		sessionManager, _ = data.NewSessionManager(data.NewSessionID())
	}

	configManager := data.NewConfigManager("", sessionManager)
	logManager := data.NewLogManager(data.DefaultLogPathTilde)

	app := &App{
		outputChannel:   make(chan string, 1000),
		outputIngress:   make(chan string, 1000),
		configManager:   configManager,
		sessionManager:  sessionManager,
		logManager:      logManager,
		shellHistory:    data.NewShellHistoryManager(),
		shellCmdHistory: data.NewShellCommandHistoryManager(),
		shellPool:       machine.NewShellSessionPool(),
		localShellPool:  machine.NewLocalShellPool(),
		shellAuxPool:    machine.NewShellAuxPool(),
		tunnelMgr:       machine.NewTunnelManager(),
		externalEdits:   newExternalEditStore(),
		shellCwds:       make(map[string]string),
	}
	app.refreshLogSettings()
	app.applyProxyFromConfig()
	app.applySSHHandshakeFromConfig()
	app.applyShellCommandHistoryMaxFromConfig()
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
	define.SetHostKeyCallback(data.GlobalHostKeyManager().Callback())
	cmds.SetTransferReporter(a.reportTaskTransfer)
	a.setupSubProjectRunner()

	// 将更新后的机器配置重新保存到全局配置文件中
	if _, err := a.configManager.LoadConfig(); err != nil {
		if os.IsNotExist(err) {
			println("配置文件不存在，创建默认配置")
			defaultPath := data.DefaultConfigPath()
			if createErr := data.CreateDefaultConfig(defaultPath); createErr != nil {
				println("创建默认配置失败:", createErr.Error())
			} else if switchErr := a.configManager.SwitchConfigFile(defaultPath); switchErr != nil {
				println("加载默认配置文件失败:", switchErr.Error())
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

// BeforeClose 关闭窗口前触发；首次拦截并弹框确认，确认后再次关闭才真正退出。
func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	a.quitMu.Lock()
	allow := a.allowQuit
	a.quitMu.Unlock()
	if !allow {
		if a.ctx != nil {
			wailsRuntime.EventsEmit(a.ctx, "app:confirm-quit")
		}
		return true
	}
	a.cleanupBeforeQuit()
	return false
}

func (a *App) machineConfigExists(name string) bool {
	name = strings.TrimSpace(name)
	return name != "" && a.configManager.GetMachine(name) != nil
}

func (a *App) remoteConfigName(sessionID string) string {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" || machine.IsLocalShellID(sessionID) {
		return sessionID
	}
	names := make([]string, 0)
	for _, m := range a.GetMachines() {
		if strings.TrimSpace(m.Name) != "" {
			names = append(names, m.Name)
		}
	}
	return machine.RemoteConfigNameForKnown(sessionID, names)
}

func (a *App) cleanupBeforeQuit() {
	a.StopAllSubProjects()
	for _, session := range a.shellPool.ListSessions() {
		_ = a.shellPool.Disconnect(session.MachineName, a.shellHandlerFor(session.MachineName))
	}
	if a.localShellPool != nil {
		a.localShellPool.DisconnectAll(a.shellHandlerFor)
	}
	a.shellAuxPool.DisconnectAll()
	if a.tunnelMgr != nil {
		a.tunnelMgr.StopAll()
	}
}

// ConfirmQuit 用户确认退出后调用，关闭应用。
func (a *App) ConfirmQuit() {
	a.quitMu.Lock()
	a.allowQuit = true
	a.quitMu.Unlock()
	if a.ctx != nil {
		wailsRuntime.Quit(a.ctx)
	}
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
		OnCwd: func(cwd string) {
			a.pushShellCwd(machineName, cwd)
		},
		OnStatus: func(_ *define.ShellStatus) {
			go a.emitShellSessions()
		},
		OnClose: func() {
			if machineName != "" {
				if machine.IsLocalShellID(machineName) {
					if a.localShellPool != nil {
						a.localShellPool.RemoveSession(machineName)
					}
				} else {
					configName := a.remoteConfigName(machineName)
					a.shellPool.RemoveSession(machineName)
					if !a.shellPool.HasConnectedConfig(configName) {
						_ = a.shellAuxPool.Disconnect(configName)
						a.stopMachineTunnels(configName)
					}
				}
				a.clearShellCwd(machineName)
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

func (a *App) pushShellCwd(machineName, cwd string) {
	if clean, ok := machine.SanitizePtyCwd(cwd); ok {
		cwd = clean
	} else {
		return
	}
	if a.ctx == nil || machineName == "" || cwd == "" {
		return
	}
	a.shellCwdMu.Lock()
	if prev := a.shellCwds[machineName]; prev == cwd {
		a.shellCwdMu.Unlock()
		return
	}
	a.shellCwds[machineName] = cwd
	a.shellCwdMu.Unlock()
	wailsRuntime.EventsEmit(a.ctx, "shell:cwd", map[string]interface{}{
		"machineName": machineName,
		"cwd":         cwd,
	})
}

func (a *App) clearShellCwd(machineName string) {
	if machineName == "" {
		return
	}
	a.shellCwdMu.Lock()
	delete(a.shellCwds, machineName)
	a.shellCwdMu.Unlock()
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
		wailsRuntime.EventsEmit(a.ctx, "shell:status", a.listAllShellSessions())
	}
}

func (a *App) listAllShellSessions() []define.ShellStatus {
	remote := a.shellPool.ListSessions()
	var local []define.ShellStatus
	if a.localShellPool != nil {
		local = a.localShellPool.ListSessions()
	}
	if len(local) == 0 {
		return remote
	}
	out := make([]define.ShellStatus, 0, len(remote)+len(local))
	out = append(out, remote...)
	out = append(out, local...)
	return out
}

// GetConfig 获取配置
func (a *App) GetConfig() (*define.Root, error) {
	return a.configManager.LoadConfig()
}

// GetProjectSummaries 首页列表用：仅返回项目摘要，不含 steps
func (a *App) GetProjectSummaries() ([]define.ProjectSummary, error) {
	root, err := a.configManager.LoadConfig()
	if err != nil {
		return nil, err
	}
	out := make([]define.ProjectSummary, 0, len(root.Projects))
	for _, p := range root.Projects {
		out = append(out, define.ProjectSummary{
			Name:            p.Name,
			Description:     p.Description,
			SubProjectCount: len(p.SubProjects),
		})
	}
	return out, nil
}

// GetProject 按名称获取完整项目（含 subprojects/commands/steps）
func (a *App) GetProject(name string) (*define.Project, error) {
	root, err := a.configManager.LoadConfig()
	if err != nil {
		return nil, err
	}
	for i := range root.Projects {
		if root.Projects[i].Name == name {
			p := root.Projects[i]
			return &p, nil
		}
	}
	return nil, fmt.Errorf("未找到项目: %s", name)
}

// GetConfigForRefresh 获取配置（用于刷新，不更新全局配置）
func (a *App) GetConfigForRefresh() (*define.Root, error) {
	return a.configManager.LoadConfigForRefresh()
}

// SaveConfig 保存配置
func (a *App) SaveConfig(root *define.Root) error {
	return a.configManager.SaveConfig(root)
}

// ExecuteSubProject 执行 SubProject（可与 Shell 会话并行）
func (a *App) ExecuteSubProject(projectName, subProjectName string) error {
	a.executionMutex.Lock()
	defer a.executionMutex.Unlock()

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
	// 异步执行 SubProject
	return a.ExecuteSubProject(projectName, subProjectName)
}

// StopSubProject 停止 SubProject
func (a *App) StopSubProject(projectName, subProjectName string) error {
	return a.subProjectRunner.StopSubProject(projectName, subProjectName)
}

// StopCommand 停止命令 (保持向后兼容)
func (a *App) StopCommand(projectName, subProjectName, commandName string) error {
	// 非阻塞读取所有可用输出 SubProject
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
func (a *App) TestMachineConnection(machineID string) error {
	machineConfig := a.configManager.GetMachineFromGlobal(machineID)
	if machineConfig == nil {
		return fmt.Errorf("未找到机器配置: %s", machineID)
	}

	sshClient := machine.NewSSHClient(machineConfig, a.configManager.GetWorkPathVars())
	return sshClient.TestConnection()
}

// TestMachineDraftConnection 用表单中的连接信息测试，无需先保存到配置
func (a *App) TestMachineDraftConnection(m define.Machine, sensitive define.SensitiveData) error {
	host := strings.TrimSpace(sensitive.Host)
	user := strings.TrimSpace(sensitive.User)
	if host == "" {
		return fmt.Errorf("请填写主机地址")
	}
	if user == "" {
		return fmt.Errorf("请填写用户名")
	}
	if sensitive.Port <= 0 {
		sensitive.Port = 22
	}
	sensitive.Host = host
	sensitive.User = user
	if strings.TrimSpace(m.Name) == "" {
		m.Name = "draft-test"
	}
	if err := m.SetSensitiveData(&sensitive); err != nil {
		return fmt.Errorf("准备连接信息失败: %w", err)
	}
	sshClient := machine.NewSSHClient(&m, a.configManager.GetWorkPathVars())
	return sshClient.TestConnection()
}

// 返回副本并填充 host/port/user（优先 list hint，避免列表全量解密）。
func (a *App) GetMachines() []define.Machine {
	src := a.configManager.GetAllMachinesFromGlobal()
	dirty := false
	for i := range src {
		if changed, _ := src[i].EnsureListHints(); changed {
			dirty = true
		}
	}
	if dirty {
		if err := a.configManager.SaveGlobalConfigMachines(src); err != nil {
			a.pushOutput(fmt.Sprintf("迁移机器列表 hint 失败: %s", err.Error()))
		}
	}
	out := make([]define.Machine, len(src))
	for i := range src {
		out[i] = src[i]
		if !out[i].ApplyListFieldsForDisplay() {
			if s, err := src[i].GetSensitiveData(); err == nil && s != nil {
				out[i].Host = s.Host
				out[i].Port = s.Port
				if out[i].Port <= 0 {
					out[i].Port = 22
				}
				out[i].User = s.User
			}
		}
	}
	return out
}

// GetMachineGroups 获取机器分组列表
func (a *App) GetMachineGroups() []string {
	return a.configManager.GetMachineGroups()
}

// AddMachineGroup 添加机器分组
func (a *App) AddMachineGroup(name string) error {
	return a.configManager.AddMachineGroup(name)
}

// RenameMachineGroup 重命名机器分组
func (a *App) RenameMachineGroup(oldName, newName string) error {
	return a.configManager.RenameMachineGroup(oldName, newName)
}

// DeleteMachineGroup 删除机器分组
func (a *App) DeleteMachineGroup(name string) error {
	return a.configManager.DeleteMachineGroup(name)
}

// UpdateMachineGroup 仅更新机器所属分组（保留凭证等其它字段）
func (a *App) UpdateMachineGroup(machineID, group string) error {
	return a.configManager.UpdateMachineGroup(machineID, group)
}

// AddMachine 添加机器配置（到全局配置）
func (a *App) AddMachine(machine define.Machine) error {
	machine.EnsureID()
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

// UpdateMachine 更新机器配置（在全局配置中，按 ID）
func (a *App) UpdateMachine(machineID string, machine define.Machine) error {
	existing := a.configManager.GetMachineFromGlobal(machineID)
	if existing == nil {
		return fmt.Errorf("未找到机器配置: %s", machineID)
	}
	machine.ID = machineID
	if machine.EncryptedData == "" {
		machine.EncryptedData = existing.EncryptedData
	}
	return a.configManager.AddMachineToGlobal(&machine)
}

// UpdateMachineWithEvent 更新机器配置（带事件通知）
func (a *App) UpdateMachineWithEvent(machineID string, machine define.Machine) error {
	if err := a.UpdateMachine(machineID, machine); err != nil {
		a.emitOperationEvent(define.OpTypeMachineConfig, fmt.Sprintf("添加机器配置失败: %s", err.Error()), define.MsgTypeError, false, nil)
		return err
	}

	a.emitOperationEvent(define.OpTypeMachineConfig, fmt.Sprintf("成功添加机器配置: %s", machine.Name), define.MsgTypeSuccess, false, map[string]interface{}{
		"machineId":   machine.ID,
		"machineName": machine.Name,
	})
	return nil
}

// DeleteMachine 删除机器配置（从全局配置，按 ID）
func (a *App) DeleteMachine(machineID string) error {
	return a.configManager.RemoveMachineFromGlobal(machineID)
}

// DeleteMachineWithEvent 删除机器配置（带事件通知）
func (a *App) DeleteMachineWithEvent(machineID string) error {
	machine := a.configManager.GetMachineFromGlobal(machineID)
	err := a.configManager.RemoveMachineFromGlobal(machineID)
	if err != nil {
		a.emitOperationEvent(define.OpTypeMachineConfig, fmt.Sprintf("添加机器配置失败: %s", err.Error()), define.MsgTypeError, false, nil)
		return err
	}

	name := machineID
	if machine != nil {
		name = machine.Name
	}
	a.emitOperationEvent(define.OpTypeMachineConfig, fmt.Sprintf("成功删除机器配置: %s", name), define.MsgTypeSuccess, false, map[string]interface{}{
		"machineId": machineID,
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
		fmt.Printf("打开配置文件失败: %v\n", err)
		a.emitOperationEvent(define.OpTypeSwitchConfig, fmt.Sprintf("停止运行中的项目时出错: %v", err), define.MsgTypeWarning, false, nil)
	}

	// 清空输出
	a.ClearOutput()

	// 切换配置文件
	if err := a.configManager.SwitchConfigFile(configPath); err != nil {
		a.emitOperationEvent(define.OpTypeSwitchConfig, fmt.Sprintf("%v", err.Error()), define.MsgTypeError, true, nil)
		return fmt.Errorf("准备连接信息失败: %w", err)
	}

	// 重新创建 SubProjectRunner
	a.setupSubProjectRunner()

	// 发送事件到前端打开工作路径配置对话框
	if a.ctx != nil {
		fmt.Printf("发送 config:changed 事件，路径: %s\n", configPath)
		wailsRuntime.EventsEmit(a.ctx, "config:changed", map[string]interface{}{
			"configPath": configPath,
			"timestamp":  time.Now().Unix(),
		})
		fmt.Println("事件发送完成")
	}

	return nil
}

// SetMachineSensitiveData 设置机器敏感数据
func (a *App) SetMachineSensitiveData(machineID string, sensitiveData define.SensitiveData) error {
	machine := a.configManager.GetMachineFromGlobal(machineID)
	if machine == nil {
		return fmt.Errorf("未找到机器: %s", machineID)
	}

	// 设置敏感数据并加密
	if err := machine.SetSensitiveData(&sensitiveData); err != nil {
		return fmt.Errorf("准备连接信息失败: %w", err)
	}
	// 将更新后的机器配置重新保存到全局配置文件中
	return a.configManager.AddMachineToGlobal(machine)
}

// GetMachineSensitiveData 获取机器敏感数据
func (a *App) GetMachineSensitiveData(machineID string) (*define.SensitiveData, error) {
	machine := a.configManager.GetMachineFromGlobal(machineID)
	if machine == nil {
		return nil, fmt.Errorf("未找到机器: %s", machineID)
	}

	return machine.GetSensitiveData()
}

// ClearMachineSensitiveData 清除机器敏感数据缓存
func (a *App) ClearMachineSensitiveData(machineID string) error {
	machine := a.configManager.GetMachineFromGlobal(machineID)
	if machine == nil {
		return fmt.Errorf("未找到机器: %s", machineID)
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

// SelectXshellFile 选择单个 Xshell 会话文件
func (a *App) SelectXshellFile() (string, error) {
	paths, err := a.pickImportSources("选择 Xshell 文件或文件夹", []wailsRuntime.FileFilter{
		{DisplayName: "Xshell 会话 (*.xsh)", Pattern: "*.xsh"},
	})
	if err != nil || len(paths) == 0 {
		return "", err
	}
	return paths[0], nil
}

// SelectXshellFolder 选择 Xshell 会话文件夹
func (a *App) SelectXshellFolder() (string, error) {
	dirPath, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择 Xshell 会话文件夹",
	})
	if err != nil {
		return "", fmt.Errorf("选择文件夹失败: %w", err)
	}
	return dirPath, nil
}

func (a *App) pickImportSources(title string, filters []wailsRuntime.FileFilter) ([]string, error) {
	files, err := wailsRuntime.OpenMultipleFilesDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title:   title,
		Filters: filters,
	})
	if err != nil {
		return nil, fmt.Errorf("选择文件失败: %w", err)
	}
	// 使用 Wails 原生文件对话框（支持多选）
	return files, nil
}

// ImportXshellPick 选择并导入 Xshell 配置（支持多文件或文件夹）
func (a *App) ImportXshellPick(accountID, group string) (*data.MachineImportResult, error) {
	paths, err := a.pickImportSources("选择 Xshell 文件或文件夹", []wailsRuntime.FileFilter{
		{DisplayName: "Xshell 会话 (*.xsh)", Pattern: "*.xsh"},
	})
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, nil
	}
	return a.configManager.ImportXshell(paths, accountID, group)
}

// ImportFinalShellPick 选择并导入 FinalShell 配置（支持多文件或文件夹）
func (a *App) ImportFinalShellPick(accountID, group string) (*data.MachineImportResult, error) {
	// Pattern 使用 macOS UTType 声明，否则无法选择 *.json 等
	// 使用 *_connect_config.json 这类通配会在 Wails OpenFileDialog 中
	// 若 UTType 为 nil 会触发 insertObject: object cannot be nil
	paths, err := a.pickImportSources("选择 FinalShell 文件或文件夹", []wailsRuntime.FileFilter{
		{DisplayName: "FinalShell (*_connect_config.json)", Pattern: "*.json"},
	})
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, nil
	}
	return a.configManager.ImportFinalShell(paths, accountID, group)
}

// ImportXshellFromFile 从路径导入 Xshell
func (a *App) ImportXshellFromFile(filePath, accountID, group string) (*data.MachineImportResult, error) {
	if filePath == "" {
		return nil, fmt.Errorf("未选择文件")
	}
	return a.configManager.ImportXshell([]string{filePath}, accountID, group)
}

// ImportXshellFromFolder 从文件夹导入 Xshell
func (a *App) ImportXshellFromFolder(dirPath, accountID, group string) (*data.MachineImportResult, error) {
	if dirPath == "" {
		return nil, fmt.Errorf("未选择文件夹")
	}
	return a.configManager.ImportXshell([]string{dirPath}, accountID, group)
}

// GetGlobalAccounts 获取全局 SSH 帐号
func (a *App) GetGlobalAccounts() []data.GlobalAccountDTO {
	return a.configManager.GetGlobalAccounts()
}

// SaveGlobalAccounts 保存全局 SSH 帐号
func (a *App) SaveGlobalAccounts(accounts []data.GlobalAccount) error {
	return a.configManager.SaveGlobalAccounts(accounts)
}

// SaveGlobalAccountsFromDTO 保存全局 SSH 帐号（前端明文密码）
func (a *App) SaveGlobalAccountsFromDTO(accounts []data.GlobalAccountDTO) error {
	stored := make([]data.GlobalAccount, 0, len(accounts))
	for _, dto := range accounts {
		account := data.GlobalAccount{
			ID:   dto.ID,
			Name: dto.Name,
			User: dto.User,
		}
		account.EnsureID()
		if err := account.SetPassword(dto.Password); err != nil {
			return err
		}
		stored = append(stored, account)
	}
	return a.configManager.SaveGlobalAccounts(stored)
}

// CreateMachine 创建机器并保存连接信息
func (a *App) CreateMachine(machine define.Machine, sensitiveData define.SensitiveData) (string, error) {
	machine.EnsureID()
	if err := machine.SetSensitiveData(&sensitiveData); err != nil {
		return "", fmt.Errorf("设置敏感数据失败: %w", err)
	}
	if err := a.configManager.AddMachineToGlobal(&machine); err != nil {
		return "", err
	}
	return machine.ID, nil
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
		fmt.Println("错误: ctx 为 nil，无法发送事件")
		a.emitOperationEvent(define.OpTypeMachineConfig, "无法打开机器配置：ctx 为 nil", define.MsgTypeError, false, nil)
	}
}

// RefreshAll 全局刷新功能
func (a *App) RefreshConfigMenu() error {
	// 发送事件到前端打开工作路径配置对话框
	// 确保 subProjectRunner 使用最新的 configManager 引用
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
	// 发送事件到前端打开工作路径配置对话框
	// 确保 subProjectRunner 使用最新的 configManager 引用
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
		wailsRuntime.WindowSetTitle(a.ctx, "FlashDock")
	}
	return nil
}

// CreateApplicationMenu 创建应用程序菜单的公共方法
func (a *App) CreateApplicationMenu() *menu.Menu {
	appMenu := menu.NewMenu()

	// 清空输出
	fileMenu := appMenu.AddSubmenu("文件")
	fileMenu.AddText("新建窗口", keys.CmdOrCtrl("n"), func(_ *menu.CallbackData) {
		NewWindow()
	})

	fileMenu.AddSeparator()
	// 获取当前执行状态
	configMenu := appMenu.AddSubmenu("设置")
	// 清空输出
	configFileMenu := appMenu.AddSubmenu("配置文件")
	// 新任务开始前清空终端
	configFiles, err := a.GetConfigFiles()
	if err != nil {
		// 为每个配置文件添加菜单项
		configFileMenu.AddText("刷新配置列表", keys.CmdOrCtrl("r"), func(_ *menu.CallbackData) {
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
			// 非阻塞读取所有可用输出
			fileName := getFileName(configFile)
			// 创建菜单项
			_ = configFileMenu.AddRadio(fileName, configFile == currentConfig, nil, func(data *menu.CallbackData) {
				// 切换配置文件
				switchConfigFile(a, configFile)
			})
		}

		// 记录错误但不阻止切换
		configFileMenu.AddSeparator()
		configFileMenu.AddText("刷新配置列表", keys.CmdOrCtrl("r"), func(_ *menu.CallbackData) {
			a.RefreshConfigMenuWithEvent()
		})
		configFileMenu.AddText("打开全局配置", nil, func(_ *menu.CallbackData) {
			// 记录错误但不阻止切换 GlobalConfigManager
			globalConfigPath := a.configManager.GetGlobalConfigPath()
			if globalConfigPath != "" {
				OpenCurrentConfig(globalConfigPath)
			}
		})

		configFileMenu.AddText("打开全局配置", nil, func(_ *menu.CallbackData) {
			a.OpenCurrentConfigWithEvent()
		})
	}

	configMenu.AddText("机器配置", keys.CmdOrCtrl("m"), func(_ *menu.CallbackData) {
		// 设置敏感数据并加密
		a.OpenMachineConfig()
	})

	configMenu.AddText("连接管理器", keys.CmdOrCtrl("e"), func(_ *menu.CallbackData) {
		a.OpenConnectionManager()
	})

	configMenu.AddText("环境变量", keys.CmdOrCtrl("u"), func(_ *menu.CallbackData) {
		// 非阻塞读取所有可用输出
		a.OpenWorkPathConfig()
	})

	configMenu.AddSeparator()
	configMenu.AddText("系统设置", keys.CmdOrCtrl(","), func(_ *menu.CallbackData) {
		a.OpenSystemSettings()
	})

	// 清空输出
	helpMenu := appMenu.AddSubmenu("帮助")
	helpMenu.AddText("关于", nil, func(_ *menu.CallbackData) {
		// 切换配置文件
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
		// 显示关于信息
		println("启动新窗口失败:", err.Error())
	} else {
		println("成功切换到配置文件:", configFile)
		// 显示关于信息
		// 为每个配置文件添加菜单项，点击时调用 SwitchConfigFileWithEvent 切换配置
	}
}

// NewWindow 供前端调用的新建窗口入口
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
		fmt.Println("发送打开机器配置事件")
		return
	}

	// 获取当前执行状态
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

	// 获取当前执行状态
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
		a.emitOperationEvent(define.OpTypeOpenConfig, "没有找到当前配置文件", define.MsgTypeWarning, false, nil)
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
		a.emitOperationEvent(define.OpTypeEnvConfig, fmt.Sprintf("添加环境变量失败: %s", err.Error()), define.MsgTypeError, false, nil)
		return err
	}

	a.emitOperationEvent(define.OpTypeEnvConfig, fmt.Sprintf("成功添加环境变量: %s", key), define.MsgTypeSuccess, false, nil)
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
		a.emitOperationEvent(define.OpTypeEnvConfig, fmt.Sprintf("添加环境变量失败: %s", err.Error()), define.MsgTypeError, false, nil)
		return err
	}

	a.emitOperationEvent(define.OpTypeEnvConfig, fmt.Sprintf("成功添加环境变量: %s", key), define.MsgTypeSuccess, false, nil)
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
		fmt.Println("错误: ctx 为 nil，无法发送事件")
		a.emitOperationEvent(define.OpTypeEnvConfig, "无法打开环境变量：ctx 为 nil", define.MsgTypeError, false, nil)
	}
}

// OpenConnectionManager 打开 Shell 连接管理器
func (a *App) OpenConnectionManager() {
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "open:connection-manager", map[string]interface{}{
			"timestamp": time.Now().Unix(),
		})
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
	fmt.Println("错误: ctx 为 nil，无法发送事件")
}

// emitOperationEvent 发送操作事件到前端
func (a *App) emitOperationEvent(eventType, message, messageType string, needReload bool, data any) {
	if a.ctx == nil {
		fmt.Printf("错误: ctx 为 nil，无法发送事件 %s\n", eventType)
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
	cfg, err := a.configManager.GetGlobalConfig()
	if err != nil {
		return nil, err
	}
	if cfg != nil {
		cfg.ShellMonitorIntervalMs = normalizeShellMonitorIntervalMs(cfg.ShellMonitorIntervalMs)
		cfg.SSHHandshakeTimeoutSec = normalizeSSHHandshakeTimeoutSec(cfg.SSHHandshakeTimeoutSec)
		cfg.ShellTerminalScrollback = normalizeShellTerminalScrollback(cfg.ShellTerminalScrollback)
		cfg.TaskOutputMaxLines = normalizeTaskOutputMaxLines(cfg.TaskOutputMaxLines)
		cfg.ShellCommandHistoryMax = data.NormalizeShellCommandHistoryMax(cfg.ShellCommandHistoryMax)
		normalizeProxySettings(&cfg.ProxySettings)
		if cfg.ShellLogHighlight == nil {
			v := true
			cfg.ShellLogHighlight = &v
		}
		cfg.ShellLogHighlightColors = data.NormalizeShellLogHighlightColors(cfg.ShellLogHighlightColors)
		cfg.ShellLogHighlightDisabled = data.NormalizeShellLogHighlightDisabled(cfg.ShellLogHighlightDisabled)
	}
	return cfg, nil
}

// GetShortcutSettings 获取快捷键配置（~/.flashdock/shortcuts.json）
func (a *App) GetShortcutSettings() (data.ShortcutSettings, error) {
	return data.LoadShortcutSettings()
}

// SaveShortcutSettings 保存快捷键配置到 JSON，并通知前端刷新
func (a *App) SaveShortcutSettings(settings data.ShortcutSettings) error {
	if err := data.SaveShortcutSettings(settings); err != nil {
		return err
	}
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "shortcuts:changed", settings)
	}
	return nil
}

// SaveSystemSettings 保存系统设置
func (a *App) SaveSystemSettings(config *data.GlobalConfig) error {
	a.normalizeThemeSettings(&config.ThemeSettings)
	normalizeProxySettings(&config.ProxySettings)
	config.ShellMonitorIntervalMs = normalizeShellMonitorIntervalMs(config.ShellMonitorIntervalMs)
	config.SSHHandshakeTimeoutSec = normalizeSSHHandshakeTimeoutSec(config.SSHHandshakeTimeoutSec)
	config.ShellTerminalScrollback = normalizeShellTerminalScrollback(config.ShellTerminalScrollback)
	config.TaskOutputMaxLines = normalizeTaskOutputMaxLines(config.TaskOutputMaxLines)
	config.ShellCommandHistoryMax = data.NormalizeShellCommandHistoryMax(config.ShellCommandHistoryMax)
	config.ShellMonitorIntervalSec = 0
	config.ShellLogHighlightColors = data.NormalizeShellLogHighlightColors(config.ShellLogHighlightColors)
	config.ShellLogHighlightDisabled = data.NormalizeShellLogHighlightDisabled(config.ShellLogHighlightDisabled)
	if err := a.configManager.SaveGlobalConfig(config); err != nil {
		return err
	}
	a.refreshLogSettings()
	a.applyProxySettings(config.ProxySettings)
	a.applySSHHandshakeTimeout(config.SSHHandshakeTimeoutSec)
	a.applyShellCommandHistoryMax(config.ShellCommandHistoryMax)
	if a.sessionManager != nil && config.ThemeSettings.Mode != "" {
		_ = a.sessionManager.SetTheme(config.ThemeSettings.Mode, config.ThemeSettings.TerminalPreset)
	}
	a.applyWindowTheme(config.ThemeSettings.Mode)
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "theme:changed", config.ThemeSettings)
		wailsRuntime.EventsEmit(a.ctx, "system-settings:changed", map[string]any{
			"shellMonitorIntervalMs":    config.ShellMonitorIntervalMs,
			"sshHandshakeTimeoutSec":    config.SSHHandshakeTimeoutSec,
			"shellTerminalScrollback":   config.ShellTerminalScrollback,
			"taskOutputMaxLines":        config.TaskOutputMaxLines,
			"shellCommandHistoryMax":    config.ShellCommandHistoryMax,
			"shellLogHighlight":         data.ShellLogHighlightEnabled(config),
			"shellLogHighlightColors":   config.ShellLogHighlightColors,
			"shellLogHighlightDisabled": config.ShellLogHighlightDisabled,
			"proxySettings":             config.ProxySettings,
		})
	}
	return nil
}

func normalizeShellMonitorIntervalMs(ms int) int {
	if ms < 200 {
		return 1000
	}
	if ms > 60000 {
		return 60000
	}
	return ms
}

func normalizeSSHHandshakeTimeoutSec(sec int) int {
	if sec <= 0 {
		return 30
	}
	if sec < 5 {
		return 5
	}
	if sec > 300 {
		return 300
	}
	return sec
}

func normalizeShellTerminalScrollback(n int) int {
	if n <= 0 {
		return 2000
	}
	if n < 100 {
		return 100
	}
	if n > 100000 {
		return 100000
	}
	return n
}

func normalizeTaskOutputMaxLines(n int) int {
	if n <= 0 {
		return 1000
	}
	if n < 100 {
		return 100
	}
	if n > 100000 {
		return 100000
	}
	return n
}

func (a *App) applySSHHandshakeTimeout(sec int) {
	define.SetSSHHandshakeTimeout(time.Duration(normalizeSSHHandshakeTimeoutSec(sec)) * time.Second)
}

func (a *App) applySSHHandshakeFromConfig() {
	cfg, err := a.configManager.GetGlobalConfig()
	if err != nil || cfg == nil {
		a.applySSHHandshakeTimeout(30)
		return
	}
	a.applySSHHandshakeTimeout(cfg.SSHHandshakeTimeoutSec)
}

func (a *App) applyShellCommandHistoryMax(max int) {
	if a.shellCmdHistory == nil {
		return
	}
	a.shellCmdHistory.SetMaxPerScope(max)
}

func (a *App) applyShellCommandHistoryMaxFromConfig() {
	cfg, err := a.configManager.GetGlobalConfig()
	if err != nil || cfg == nil {
		a.applyShellCommandHistoryMax(200)
		return
	}
	a.applyShellCommandHistoryMax(cfg.ShellCommandHistoryMax)
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
	settings := data.ThemeSettings{
		Mode: "light", TerminalPreset: "classic", UiAccent: "blue",
		UiFontFamily: "system", ShellFontFamily: "consolas",
		ShellFontSize: 13, ShellLineHeight: 1.2,
	}
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
	a.normalizeThemeSettings(&settings)
	return settings
}

func (a *App) normalizeThemeSettings(settings *data.ThemeSettings) {
	if settings.Mode == "" {
		settings.Mode = "light"
	}
	if settings.UiAccent == "" {
		settings.UiAccent = "blue"
	}
	if settings.TerminalPreset == "" {
		settings.TerminalPreset = "classic"
	}
	if settings.UiFontFamily == "" {
		settings.UiFontFamily = "system"
	}
	if settings.ShellFontFamily == "" {
		settings.ShellFontFamily = "consolas"
	}
	if settings.ShellFontSize <= 0 {
		settings.ShellFontSize = 13
	}
	if settings.ShellLineHeight <= 0 {
		settings.ShellLineHeight = 1.2
	}
}

// SaveThemeSettings 保存主题设置到会话与全局
func (a *App) SaveThemeSettings(settings data.ThemeSettings) error {
	a.normalizeThemeSettings(&settings)
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
	if err := a.configManager.SaveGlobalConfig(globalConfig); err != nil {
		return err
	}
	a.applyWindowTheme(settings.Mode)
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "theme:changed", settings)
	}
	return nil
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

func (a *App) onRemoteShellConnected(sessionID string, machineConfig *define.Machine, configName string) {
	if sessionID == "" || machineConfig == nil {
		return
	}
	a.ensureShellAux(sessionID, machineConfig)
	if err := a.ensureMachineTunnels(machineConfig); err != nil {
		fmt.Printf("SSH 隧道启动失败(%s): %v\n", configName, err)
	}
	if sensitive, sErr := machineConfig.GetSensitiveData(); sErr == nil && a.shellHistory != nil {
		_ = a.shellHistory.RecordConnect(machineConfig, sensitive.Host, sensitive.Port, sensitive.User)
	}
}

// ConnectShell 连接远程 Shell，返回新会话 ID（同机可多开：web1 / web1-2）
func (a *App) ConnectShell(configName string) (string, error) {
	if machine.IsLocalShellID(configName) {
		return "", fmt.Errorf("请使用 ConnectLocalShell 创建本地终端")
	}
	configName = strings.TrimSpace(configName)
	configName = a.remoteConfigName(configName)
	machineConfig := a.configManager.GetMachine(configName)
	if machineConfig == nil {
		return "", fmt.Errorf("未找到机器配置: %s", configName)
	}

	workVars := a.configManager.GetWorkPathVars()
	sessionID, err := a.shellPool.Connect(machineConfig, workVars, a.shellHandlerFor, func(id string, connectErr error) {
		if connectErr == nil {
			a.onRemoteShellConnected(id, machineConfig, configName)
		}
		a.emitShellSessions()
	})
	if err != nil {
		return "", err
	}
	// 立即推送「连接中」状态；拨号在后台进行，不阻塞本调用
	go a.emitShellSessions()
	return sessionID, nil
}

// ReconnectShell 按会话 ID 软断开后重连
func (a *App) ReconnectShell(sessionID string) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", fmt.Errorf("会话 ID 为空")
	}
	if machine.IsLocalShellID(sessionID) {
		return a.ConnectLocalShell(sessionID)
	}
	configName := a.remoteConfigName(sessionID)
	machineConfig := a.configManager.GetMachine(configName)
	if machineConfig == nil {
		return "", fmt.Errorf("未找到机器配置: %s", configName)
	}
	if a.shellPool.IsConnected(sessionID) {
		a.ensureShellAux(sessionID, machineConfig)
		a.emitShellSessions()
		return sessionID, nil
	}
	workVars := a.configManager.GetWorkPathVars()
	if err := a.shellPool.ConnectID(sessionID, machineConfig, workVars, a.shellHandlerFor(sessionID), func(_ string, connectErr error) {
		if connectErr == nil {
			a.onRemoteShellConnected(sessionID, machineConfig, configName)
		}
		a.emitShellSessions()
	}); err != nil {
		return "", err
	}
	go a.emitShellSessions()
	return sessionID, nil
}

// ConnectLocalShell 创建或重连本地终端。sessionID 为空则新建并返回新 ID；非空则按该 ID 重连。
func (a *App) ConnectLocalShell(sessionID string) (string, error) {
	if a.localShellPool == nil {
		return "", fmt.Errorf("本地终端不可用")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID != "" {
		if !machine.IsLocalShellID(sessionID) {
			return "", fmt.Errorf("非法本地会话 ID")
		}
		if err := a.localShellPool.ConnectID(sessionID, a.shellHandlerFor); err != nil {
			return "", err
		}
		a.emitShellSessions()
		return sessionID, nil
	}
	id, err := a.localShellPool.Connect(a.shellHandlerFor)
	if err != nil {
		return "", err
	}
	a.emitShellSessions()
	return id, nil
}
func (a *App) resolveAuxKey(sessionOrConfig string) string {
	if machine.IsLocalShellID(sessionOrConfig) {
		return sessionOrConfig
	}
	return a.remoteConfigName(sessionOrConfig)
}

func (a *App) getShellAux(sessionOrConfig string) (*machine.ShellAuxManager, error) {
	key := a.resolveAuxKey(sessionOrConfig)
	if aux, err := a.shellAuxPool.Get(key); err == nil {
		return aux, nil
	}
	// 辅助通道缺失时按当前 PTY 会话补挂载（连接竞态 / 初次 SFTP 失败等）
	a.ensureShellAuxFor(sessionOrConfig)
	return a.shellAuxPool.Get(key)
}

func (a *App) ensureShellAuxFor(sessionOrConfig string) {
	if machine.IsLocalShellID(sessionOrConfig) {
		return
	}
	cfgName := a.remoteConfigName(sessionOrConfig)
	machineConfig := a.configManager.GetMachine(cfgName)
	if machineConfig == nil {
		return
	}
	sessionID := strings.TrimSpace(sessionOrConfig)
	if !a.shellPool.IsConnected(sessionID) {
		sessionID = a.shellPool.FirstSessionOfConfig(cfgName)
	}
	if sessionID == "" {
		sessionID = cfgName
	}
	a.ensureShellAux(sessionID, machineConfig)
}

func (a *App) ensureShellAux(sessionID string, machineConfig *define.Machine) {
	if machineConfig == nil {
		return
	}
	auxKey := machineConfig.Name
	host := ""
	if s, err := machineConfig.GetSensitiveData(); err == nil && s != nil {
		host = s.Host
	}
	var ptyClient *machine.SSHClient
	if sm := a.shellPool.GetSession(sessionID); sm != nil {
		ptyClient = sm.SharedSSHClient()
	}
	if auxErr := a.shellAuxPool.EnsureAttached(auxKey, machineConfig, a.configManager.GetWorkPathVars(), ptyClient, host); auxErr != nil {
		fmt.Printf("辅助通道挂载失败(%s): %v\n", auxKey, auxErr)
		return
	}
	if aux, err := a.shellAuxPool.Get(auxKey); err == nil {
		_ = machine.UninstallShellCwdHook(aux)
	}
	a.seedShellCwdIfEmpty(sessionID)
}

func (a *App) seedShellCwdIfEmpty(machineName string) {
	a.shellCwdMu.RLock()
	has := a.shellCwds[machineName] != ""
	a.shellCwdMu.RUnlock()
	if has {
		return
	}
	if home, err := a.getRemoteHome(machineName); err == nil && home != "" {
		a.pushShellCwd(machineName, home)
	}
}

// DisconnectShell 断开指定机器的 Shell
func (a *App) DisconnectShell(machineName string) error {
	a.executionMutex.Lock()
	defer a.executionMutex.Unlock()
	var err error
	if machine.IsLocalShellID(machineName) {
		if a.localShellPool != nil {
			err = a.localShellPool.Disconnect(machineName, a.shellHandlerFor(machineName))
		}
	} else {
		configName := a.remoteConfigName(machineName)
		err = a.shellPool.Disconnect(machineName, a.shellHandlerFor(machineName))
		if !a.shellPool.HasConnectedConfig(configName) {
			_ = a.shellAuxPool.Disconnect(configName)
			a.stopMachineTunnels(configName)
			a.clearSessionHostKeyTrust(configName)
		}
		if a.externalEdits != nil {
			a.externalEdits.stopForMachine(machineName)
		}
	}
	a.clearShellCwd(machineName)
	a.emitShellSessions()
	return err
}

// GetShellHistory 获取连接历史
func (a *App) GetShellHistory() []define.ShellHistoryRecord {
	if a.shellHistory == nil {
		return nil
	}
	return a.shellHistory.List()
}

// ClearShellHistory 清空连接历史
func (a *App) ClearShellHistory() error {
	if a.shellHistory == nil {
		return nil
	}
	return a.shellHistory.Clear()
}

// RemoveShellHistory 删除一条连接历史
func (a *App) RemoveShellHistory(machineID, machineName string) error {
	if a.shellHistory == nil {
		return nil
	}
	return a.shellHistory.Remove(machineID, machineName)
}

// GetShellMonitor 获取机器监控快照；netIface 为空时使用默认网卡
func (a *App) GetShellMonitor(machineName, netIface string) *define.ShellMonitorSnapshot {
	aux, err := a.getShellAux(machineName)
	if err != nil {
		host := ""
		cfgName := a.remoteConfigName(machineName)
		if m := a.configManager.GetMachine(cfgName); m != nil {
			if s, e := m.GetSensitiveData(); e == nil {
				host = s.Host
			}
		}
		// 无辅助通道时仍返回空快照，避免监控 UI 报错
		return &define.ShellMonitorSnapshot{
			MachineName: machineName,
			Host:        host,
			UpdatedAt:   time.Now().Unix(),
			TopMem:      []define.ShellProcessStat{},
			NetIfaces:   []string{},
		}
	}
	snap := aux.FetchMonitor(netIface)
	if m := a.configManager.GetMachine(a.remoteConfigName(machineName)); m != nil {
		if s, e := m.GetSensitiveData(); e == nil && s.Host != "" {
			snap.Host = s.Host
		}
	}
	return snap
}

// GetShellSystemInfo 获取机器系统信息
func (a *App) GetShellSystemInfo(machineName string) *define.ShellSystemInfo {
	aux, err := a.getShellAux(machineName)
	if err != nil {
		host := ""
		cfgName := a.remoteConfigName(machineName)
		if m := a.configManager.GetMachine(cfgName); m != nil {
			if s, e := m.GetSensitiveData(); e == nil {
				host = s.Host
			}
		}
		return &define.ShellSystemInfo{
			MachineName: machineName,
			Host:        host,
			Error:       err.Error(),
		}
	}
	info := aux.FetchSystemInfo()
	if m := a.configManager.GetMachine(a.remoteConfigName(machineName)); m != nil {
		if s, e := m.GetSensitiveData(); e == nil && s.Host != "" {
			info.Host = s.Host
		}
	}
	return info
}

// ListShellFiles 列出远端目录
func (a *App) ListShellFiles(machineName, dirPath string, showHidden bool) ([]define.SftpEntry, error) {
	aux, err := a.getShellAux(machineName)
	if err != nil {
		return nil, err
	}
	return aux.ListDir(dirPath, showHidden)
}

// DeleteShellFile 删除远端文件或目录
func (a *App) DeleteShellFile(machineName, remotePath string) error {
	if strings.TrimSpace(remotePath) == "" || remotePath == "/" {
		return fmt.Errorf("非法路径")
	}
	aux, err := a.getShellAux(machineName)
	if err != nil {
		return err
	}
	return aux.RemovePath(remotePath)
}

// GetShellRemoteHome 远端登录 home（SFTP 初始目录）
func (a *App) GetShellRemoteHome(machineName string) (string, error) {
	return a.getRemoteHome(machineName)
}

// GetShellRemotePwd 获取辅助通道 pwd（兼容旧调用；新逻辑请用 GetShellPtyCwd）
func (a *App) GetShellRemotePwd(machineName string) (string, error) {
	aux, err := a.getShellAux(machineName)
	if err != nil {
		return "", err
	}
	return aux.Pwd()
}

// GetShellPtyCwd 获取 PTY 终端当前工作目录（内存缓存 / home）
func (a *App) GetShellPtyCwd(machineName string) (string, error) {
	a.shellCwdMu.RLock()
	raw := a.shellCwds[machineName]
	a.shellCwdMu.RUnlock()
	if clean, ok := machine.SanitizePtyCwd(raw); ok {
		return clean, nil
	}
	if home, err := a.getRemoteHome(machineName); err == nil && home != "" {
		return NormalizeRemoteAbs(home), nil
	}
	return "", fmt.Errorf("PTY cwd 未知")
}

// SyncShellCwd 根据终端 cd 命令行同步 cwd（Enter 后调用；不修改远端 shell 配置）
func (a *App) SyncShellCwd(machineName, cdLine string) (string, error) {
	cdLine = strings.TrimSpace(cdLine)
	if cdLine == "" {
		a.shellCwdMu.RLock()
		raw := a.shellCwds[machineName]
		a.shellCwdMu.RUnlock()
		if clean, ok := machine.SanitizePtyCwd(raw); ok {
			return clean, nil
		}
		return "", fmt.Errorf("PTY cwd 未知")
	}
	if len(cdLine) < 2 || !strings.EqualFold(cdLine[:2], "cd") {
		return "", fmt.Errorf("非 cd 命令")
	}
	target := strings.TrimSpace(cdLine[2:])
	home, err := a.getRemoteHome(machineName)
	if err != nil {
		home = ""
	}
	a.shellCwdMu.RLock()
	current := a.shellCwds[machineName]
	a.shellCwdMu.RUnlock()
	if strings.TrimSpace(current) == "" {
		current = home
	}
	current = NormalizeRemoteAbs(current)
	resolved, err := ResolveShellCdTarget(current, target, home)
	if err != nil {
		return "", err
	}
	resolved = NormalizeRemoteAbs(resolved)
	a.pushShellCwd(machineName, resolved)
	return resolved, nil
}

// ShellDirExists 远端路径是否为目录
func (a *App) ShellDirExists(machineName, dirPath string) (bool, error) {
	aux, err := a.getShellAux(machineName)
	if err != nil {
		return false, err
	}
	return aux.DirExists(dirPath)
}

// ResolveShellPath 规范化远端路径（相对 → 基于 base；空 base 回退 home）
func (a *App) ResolveShellPath(machineName, basePath, target string) (string, error) {
	home, err := a.getRemoteHome(machineName)
	if err != nil {
		home = ""
	}
	return ResolveRemotePath(basePath, target, home)
}

// ApplyShellCd 解析 cd 目标并用 SFTP 校验；目录不存在则返回原 current。
func (a *App) ApplyShellCd(machineName, current, target string) (string, error) {
	home, err := a.getRemoteHome(machineName)
	if err != nil {
		return "", fmt.Errorf("获取 home 失败: %w", err)
	}
	if strings.TrimSpace(current) == "" {
		current = home
	}
	current = NormalizeRemoteAbs(current)
	resolved, err := ResolveShellCdTarget(current, target, home)
	if err != nil {
		return "", err
	}
	if resolved == current {
		return current, nil
	}
	exists, err := a.shellDirExistsReliable(machineName, resolved)
	if err != nil {
		return "", fmt.Errorf("校验目录失败 %s: %w", resolved, err)
	}
	return ChooseCdPath(current, resolved, exists), nil
}

// shellDirExistsReliable：Stat/ReadDir 探测目录是否存在（供 cd 校验）
func (a *App) shellDirExistsReliable(machineName, dirPath string) (bool, error) {
	exists, err := a.ShellDirExists(machineName, dirPath)
	if err == nil {
		return exists, nil
	}
	// Stat 失败时再尝试列目录
	parent := path.Dir(dirPath)
	base := path.Base(dirPath)
	if dirPath == "/" || base == "." || base == "/" {
		return false, err
	}
	entries, listErr := a.ListShellFiles(machineName, parent, true)
	if listErr != nil {
		return false, err
	}
	for _, e := range entries {
		if e.Name == base && e.IsDir {
			return true, nil
		}
	}
	return false, nil
}

func (a *App) getRemoteHome(machineName string) (string, error) {
	aux, err := a.getShellAux(machineName)
	if err != nil {
		return "", err
	}
	home, err := aux.Home()
	if err != nil {
		return "", err
	}
	return NormalizeRemoteAbs(home), nil
}
func (a *App) ensureMachineTunnels(machineConfig *define.Machine) error {
	if a.tunnelMgr == nil || machineConfig == nil || len(machineConfig.Tunnels) == 0 {
		return nil
	}
	var client *machine.SSHClient
	if id := a.shellPool.FirstSessionOfConfig(machineConfig.Name); id != "" {
		if sm := a.shellPool.GetSession(id); sm != nil {
			client = sm.SharedSSHClient()
		}
	}
	if client == nil {
		return fmt.Errorf("未配置 SSH 隧道")
	}
	return a.tunnelMgr.EnsureForMachine(machineConfig.Name, machineConfig.Tunnels, client)
}

func (a *App) stopMachineTunnels(configName string) {
	if a.tunnelMgr != nil {
		a.tunnelMgr.StopAllFor(configName)
	}
}

// GetShellTunnelStatus 返回机器 SSH 隧道状态
func (a *App) GetShellTunnelStatus(configName string) []define.SSHTunnelStatus {
	if a.tunnelMgr == nil {
		return nil
	}
	return a.tunnelMgr.StatusList(a.remoteConfigName(configName))
}

// BroadcastShellInput 向多个会话广播输入
func (a *App) BroadcastShellInput(sessionIDs []string, input string) error {
	a.executionMutex.RLock()
	defer a.executionMutex.RUnlock()
	if len(sessionIDs) == 0 {
		return fmt.Errorf("请填写主机地址")
	}
	var firstErr error
	ok := 0
	for _, id := range sessionIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		var err error
		if machine.IsLocalShellID(id) {
			if a.localShellPool == nil {
				err = fmt.Errorf("本地终端不可用")
			} else {
				err = a.localShellPool.SendInput(id, input)
			}
		} else {
			err = a.shellPool.SendInput(id, input)
		}
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		ok++
	}
	if ok == 0 && firstErr != nil {
		return firstErr
	}
	return nil
}

// SendShellInput 向指定 PTY Shell 发送输入
func (a *App) SendShellInput(machineName, input string) error {
	a.executionMutex.RLock()
	defer a.executionMutex.RUnlock()
	a.recordShellCommand(machineName, input)
	if machine.IsLocalShellID(machineName) {
		if a.localShellPool == nil {
			return fmt.Errorf("本地终端不可用")
		}
		return a.localShellPool.SendInput(machineName, input)
	}
	return a.shellPool.SendInput(machineName, input)
}

// SendShellInterrupt 向指定 PTY Shell 发送 Ctrl+C
func (a *App) SendShellInterrupt(machineName string) error {
	a.executionMutex.RLock()
	defer a.executionMutex.RUnlock()
	if machine.IsLocalShellID(machineName) {
		if a.localShellPool == nil {
			return fmt.Errorf("本地终端不可用")
		}
		return a.localShellPool.SendInterrupt(machineName)
	}
	return a.shellPool.SendInterrupt(machineName)
}

// ResizeShell 调整指定 PTY 终端尺寸
func (a *App) ResizeShell(machineName string, cols, rows int) error {
	a.executionMutex.RLock()
	defer a.executionMutex.RUnlock()
	if machine.IsLocalShellID(machineName) {
		if a.localShellPool == nil {
			return fmt.Errorf("本地终端不可用")
		}
		return a.localShellPool.Resize(machineName, cols, rows)
	}
	return a.shellPool.Resize(machineName, cols, rows)
}

// ExecuteShellCommand 兼容旧接口
func (a *App) ExecuteShellCommand(machineName, command string) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return fmt.Errorf("请填写用户名")
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
	return a.listAllShellSessions()
}

// GetShellStatus 兼容旧接口，返回首个会话或空状态
func (a *App) GetShellStatus() *define.ShellStatus {
	sessions := a.listAllShellSessions()
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
