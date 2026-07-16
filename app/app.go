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
	subProjectRunner *machine.SubProjectRunner
	shellPool        *machine.ShellSessionPool
	localShellPool   *machine.LocalShellPool
	shellAuxPool     *machine.ShellAuxPool
	tunnelMgr        *machine.TunnelManager
	transfers        *shellTransferStore
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
		println("?????????:", err.Error())
		sessionManager, _ = data.NewSessionManager(data.NewSessionID())
	}

	configManager := data.NewConfigManager("", sessionManager)
	logManager := data.NewLogManager(data.DefaultLogPathTilde)

	app := &App{
		outputChannel:  make(chan string, 1000),
		outputIngress:  make(chan string, 1000),
		configManager:  configManager,
		sessionManager: sessionManager,
		logManager:     logManager,
		shellHistory:   data.NewShellHistoryManager(),
		shellPool:      machine.NewShellSessionPool(),
		localShellPool: machine.NewLocalShellPool(),
		shellAuxPool:   machine.NewShellAuxPool(),
		tunnelMgr:       machine.NewTunnelManager(),
		shellCwds:      make(map[string]string),
	}
	app.refreshLogSettings()
	app.applyProxyFromConfig()
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

	// ?????????????????????
	if _, err := a.configManager.LoadConfig(); err != nil {
		if os.IsNotExist(err) {
			println("??????????????")
			data.CreateDefaultConfig("config.yaml")
			if _, loadErr := a.configManager.LoadConfig(); loadErr != nil {
				println("??????????:", loadErr.Error())
			} else {
				println("??????????")
			}
		} else {
			println("????????:", err.Error())
		}
	} else {
		println("????????")
	}
	a.applyWindowTheme(a.GetThemeSettings().Mode)
}

// DomReady is called after front-end resources have been loaded
func (a *App) DomReady(ctx context.Context) {
	// Add your action here
}

// BeforeClose ???????????????????????????????
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
	return machine.RemoteConfigNameWithResolver(sessionID, a.machineConfigExists)
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

// ConfirmQuit ???????????????
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

// GetConfig ????
func (a *App) GetConfig() (*define.Root, error) {
	return a.configManager.LoadConfig()
}

// GetConfigForRefresh ??????????????????
func (a *App) GetConfigForRefresh() (*define.Root, error) {
	return a.configManager.LoadConfigForRefresh()
}

// SaveConfig ????
func (a *App) SaveConfig(root *define.Root) error {
	return a.configManager.SaveConfig(root)
}

// ExecuteSubProject ?? SubProject??? Shell ?????
func (a *App) ExecuteSubProject(projectName, subProjectName string) error {
	a.executionMutex.Lock()
	defer a.executionMutex.Unlock()

	// ????????????????? SubProject ??
	if _, err := a.configManager.LoadConfigForRefresh(); err != nil {
		return err
	}

	// ??????????
	a.ClearOutput()

	// ???? SubProject
	go func() {
		success := true
		summary := "????"
		if a.logEnabled {
			if _, err := a.logManager.StartSession(projectName, subProjectName); err != nil {
				a.pushOutput(fmt.Sprintf("????????: %s", err.Error()))
			}
		}

		if err := a.subProjectRunner.ExecuteSubProject(projectName, subProjectName, a.outputWriter()); err != nil {
			a.pushOutput(fmt.Sprintf("????: %s", err.Error()))
			success = false
			summary = err.Error()
		}

		if a.logEnabled {
			a.logManager.FinishSession(success, summary)
		}
	}()

	return nil
}

// ExecuteCommand ???? (???????????????)
func (a *App) ExecuteCommand(projectName, subProjectName, commandName string) error {
	// ??????????????????????????? SubProject
	return a.ExecuteSubProject(projectName, subProjectName)
}

// StopSubProject ?? SubProject
func (a *App) StopSubProject(projectName, subProjectName string) error {
	return a.subProjectRunner.StopSubProject(projectName, subProjectName)
}

// StopCommand ???? (??????)
func (a *App) StopCommand(projectName, subProjectName, commandName string) error {
	// ??????????? SubProject
	return a.StopSubProject(projectName, subProjectName)
}

// StopAllSubProjects ???? SubProjects
func (a *App) StopAllSubProjects() error {
	// ????????
	status := a.subProjectRunner.GetExecutionStatus()
	if status.IsRunning {
		return a.StopSubProject(status.ProjectName, status.SubProjectName)
	}
	return nil
}

// StopAllCommands ?????? (??????)
func (a *App) StopAllCommands() {
	a.StopAllSubProjects()
}

// GetOutput ????
func (a *App) GetOutput() []string {
	var output []string

	// ???????????
	for {
		select {
		case msg := <-a.outputChannel:
			output = append(output, msg)
		default:
			return output
		}
	}
}

// ClearOutput ????
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

// GetSubProjectStatus ?? SubProject ??
func (a *App) GetSubProjectStatus() *define.SubProjectStatus {
	return a.subProjectRunner.GetExecutionStatus()
}

// GetStatus ???? (??????)
func (a *App) GetStatus() *define.CommandStatus {
	subStatus := a.subProjectRunner.GetExecutionStatus()

	// ????? CommandStatus ??
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

// TestMachineConnection ??????
func (a *App) TestMachineConnection(machineID string) error {
	machineConfig := a.configManager.GetMachineFromGlobal(machineID)
	if machineConfig == nil {
		return fmt.Errorf("???????: %s", machineID)
	}

	sshClient := machine.NewSSHClient(machineConfig, a.configManager.GetWorkPathVars())
	return sshClient.TestConnection()
}

// TestMachineDraftConnection ????????????????????
func (a *App) TestMachineDraftConnection(m define.Machine, sensitive define.SensitiveData) error {
	host := strings.TrimSpace(sensitive.Host)
	user := strings.TrimSpace(sensitive.User)
	if host == "" {
		return fmt.Errorf("???????")
	}
	if user == "" {
		return fmt.Errorf("??????")
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
		return fmt.Errorf("????????: %w", err)
	}
	sshClient := machine.NewSSHClient(&m, a.configManager.GetWorkPathVars())
	return sshClient.TestConnection()
}

// GetMachines ???????????????
// ??????? host/port/user????????? IP ??????
func (a *App) GetMachines() []define.Machine {
	src := a.configManager.GetAllMachinesFromGlobal()
	out := make([]define.Machine, len(src))
	for i := range src {
		out[i] = src[i]
		if s, err := src[i].GetSensitiveData(); err == nil && s != nil {
			out[i].Host = s.Host
			out[i].Port = s.Port
			out[i].User = s.User
		}
	}
	return out
}

// GetMachineGroups ????????
func (a *App) GetMachineGroups() []string {
	return a.configManager.GetMachineGroups()
}

// AddMachineGroup ??????
func (a *App) AddMachineGroup(name string) error {
	return a.configManager.AddMachineGroup(name)
}

// RenameMachineGroup ???????
func (a *App) RenameMachineGroup(oldName, newName string) error {
	return a.configManager.RenameMachineGroup(oldName, newName)
}

// DeleteMachineGroup ??????
func (a *App) DeleteMachineGroup(name string) error {
	return a.configManager.DeleteMachineGroup(name)
}

// UpdateMachineGroup ????????????????????
func (a *App) UpdateMachineGroup(machineID, group string) error {
	return a.configManager.UpdateMachineGroup(machineID, group)
}

// AddMachine ?????????????
func (a *App) AddMachine(machine define.Machine) error {
	machine.EnsureID()
	return a.configManager.AddMachineToGlobal(&machine)
}

// AddMachineWithEvent ?????????????
func (a *App) AddMachineWithEvent(machine define.Machine) error {
	err := a.configManager.AddMachineToGlobal(&machine)
	if err != nil {
		a.emitOperationEvent(define.OpTypeMachineConfig, fmt.Sprintf("????????: %s", err.Error()), define.MsgTypeError, false, nil)
		return err
	}

	a.emitOperationEvent(define.OpTypeMachineConfig, fmt.Sprintf("????????: %s", machine.Name), define.MsgTypeSuccess, false, map[string]interface{}{
		"machineName": machine.Name,
	})
	return nil
}

// UpdateMachine ??????????????? ID?
func (a *App) UpdateMachine(machineID string, machine define.Machine) error {
	existing := a.configManager.GetMachineFromGlobal(machineID)
	if existing == nil {
		return fmt.Errorf("???????: %s", machineID)
	}
	machine.ID = machineID
	if machine.EncryptedData == "" {
		machine.EncryptedData = existing.EncryptedData
	}
	return a.configManager.AddMachineToGlobal(&machine)
}

// UpdateMachineWithEvent ?????????????
func (a *App) UpdateMachineWithEvent(machineID string, machine define.Machine) error {
	if err := a.UpdateMachine(machineID, machine); err != nil {
		a.emitOperationEvent(define.OpTypeMachineConfig, fmt.Sprintf("????????: %s", err.Error()), define.MsgTypeError, false, nil)
		return err
	}

	a.emitOperationEvent(define.OpTypeMachineConfig, fmt.Sprintf("????????: %s", machine.Name), define.MsgTypeSuccess, false, map[string]interface{}{
		"machineId":   machine.ID,
		"machineName": machine.Name,
	})
	return nil
}

// DeleteMachine ?????????????? ID?
func (a *App) DeleteMachine(machineID string) error {
	return a.configManager.RemoveMachineFromGlobal(machineID)
}

// DeleteMachineWithEvent ?????????????
func (a *App) DeleteMachineWithEvent(machineID string) error {
	machine := a.configManager.GetMachineFromGlobal(machineID)
	err := a.configManager.RemoveMachineFromGlobal(machineID)
	if err != nil {
		a.emitOperationEvent(define.OpTypeMachineConfig, fmt.Sprintf("????????: %s", err.Error()), define.MsgTypeError, false, nil)
		return err
	}

	name := machineID
	if machine != nil {
		name = machine.Name
	}
	a.emitOperationEvent(define.OpTypeMachineConfig, fmt.Sprintf("????????: %s", name), define.MsgTypeSuccess, false, map[string]interface{}{
		"machineId": machineID,
	})
	return nil
}

// GetGlobalConfig ??????
func (a *App) GetGlobalConfig() (*data.GlobalConfig, error) {
	return a.configManager.GetGlobalConfig()
}

// GetGlobalConfigForRefresh ????????????????????
func (a *App) GetGlobalConfigForRefresh() (*data.GlobalConfig, error) {
	globalConfig, err := a.configManager.GetGlobalConfigForRefresh()
	if err != nil {
		return nil, err
	}
	a.UpdateApplicationMenu()
	return globalConfig, nil
}

// SaveGlobalConfig ??????
func (a *App) SaveGlobalConfig(config *data.GlobalConfig) error {
	return a.configManager.SaveGlobalConfig(config)
}

// GetConfigFiles ??????????
func (a *App) GetConfigFiles() ([]string, error) {
	return a.configManager.GetConfigFiles()
}

// SwitchConfigFileWithEvent ?????????????
func (a *App) SwitchConfigFileWithEvent(configPath string) error {
	// ????????? SubProjects
	if err := a.StopAllSubProjects(); err != nil {
		// ??????????
		fmt.Printf("???????????: %v\n", err)
		a.emitOperationEvent(define.OpTypeSwitchConfig, fmt.Sprintf("???????????: %v", err), define.MsgTypeWarning, false, nil)
	}

	// ????
	a.ClearOutput()

	// ??????
	if err := a.configManager.SwitchConfigFile(configPath); err != nil {
		a.emitOperationEvent(define.OpTypeSwitchConfig, fmt.Sprintf("%v", err.Error()), define.MsgTypeError, true, nil)
		return fmt.Errorf("????????: %w", err)
	}

	// ???? SubProjectRunner
	a.setupSubProjectRunner()

	// ????????????????????????
	if a.ctx != nil {
		fmt.Printf("?? config:changed ???????: %s\n", configPath)
		wailsRuntime.EventsEmit(a.ctx, "config:changed", map[string]interface{}{
			"configPath": configPath,
			"timestamp":  time.Now().Unix(),
		})
		fmt.Println("??????")
	}

	return nil
}

// SetMachineSensitiveData ????????
func (a *App) SetMachineSensitiveData(machineID string, sensitiveData define.SensitiveData) error {
	machine := a.configManager.GetMachineFromGlobal(machineID)
	if machine == nil {
		return fmt.Errorf("?????: %s", machineID)
	}

	// ?????????
	if err := machine.SetSensitiveData(&sensitiveData); err != nil {
		return fmt.Errorf("????????: %w", err)
	}
	// ?????????????????????
	return a.configManager.AddMachineToGlobal(machine)
}

// GetMachineSensitiveData ????????
func (a *App) GetMachineSensitiveData(machineID string) (*define.SensitiveData, error) {
	machine := a.configManager.GetMachineFromGlobal(machineID)
	if machine == nil {
		return nil, fmt.Errorf("?????: %s", machineID)
	}

	return machine.GetSensitiveData()
}

// ClearMachineSensitiveData ??????????
func (a *App) ClearMachineSensitiveData(machineID string) error {
	machine := a.configManager.GetMachineFromGlobal(machineID)
	if machine == nil {
		return fmt.Errorf("?????: %s", machineID)
	}

	machine.ClearSensitiveData()
	return nil
}

// SelectKeyFile ??????
func (a *App) SelectKeyFile() (string, error) {
	filePath, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title:           "??SSH????",
		ShowHiddenFiles: true,
	})
	if err != nil {
		return "", fmt.Errorf("??????: %w", err)
	}

	return filePath, nil
}

// SelectXshellFile ???? Xshell ????
func (a *App) SelectXshellFile() (string, error) {
	paths, err := a.pickImportSources("?? Xshell ??????", []wailsRuntime.FileFilter{
		{DisplayName: "Xshell ?? (*.xsh)", Pattern: "*.xsh"},
	})
	if err != nil || len(paths) == 0 {
		return "", err
	}
	return paths[0], nil
}

// SelectXshellFolder ?? Xshell ?????
func (a *App) SelectXshellFolder() (string, error) {
	dirPath, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "?? Xshell ?????",
	})
	if err != nil {
		return "", fmt.Errorf("???????: %w", err)
	}
	return dirPath, nil
}

func (a *App) pickImportSources(title string, filters []wailsRuntime.FileFilter) ([]string, error) {
	files, err := wailsRuntime.OpenMultipleFilesDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title:   title,
		Filters: filters,
	})
	if err != nil {
		return nil, fmt.Errorf("??????: %w", err)
	}
	// ??? Wails ????????????????
	return files, nil
}

// ImportXshellPick ????? Xshell ?????????????
func (a *App) ImportXshellPick(accountID, group string) (*data.MachineImportResult, error) {
	paths, err := a.pickImportSources("?? Xshell ??????", []wailsRuntime.FileFilter{
		{DisplayName: "Xshell ?? (*.xsh)", Pattern: "*.xsh"},
	})
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, nil
	}
	return a.configManager.ImportXshell(paths, accountID, group)
}

// ImportFinalShellPick ????? FinalShell ?????????????
func (a *App) ImportFinalShellPick(accountID, group string) (*data.MachineImportResult, error) {
	// Pattern ??? macOS UTType ????????? *.json??
	// ?? *_connect_config.json ?????? Wails OpenFileDialog ?
	// ? UTType ?? nil ????insertObject: object cannot be nil?
	paths, err := a.pickImportSources("?? FinalShell ??????", []wailsRuntime.FileFilter{
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

// ImportXshellFromFile ????? Xshell
func (a *App) ImportXshellFromFile(filePath, accountID, group string) (*data.MachineImportResult, error) {
	if filePath == "" {
		return nil, fmt.Errorf("?????")
	}
	return a.configManager.ImportXshell([]string{filePath}, accountID, group)
}

// ImportXshellFromFolder ?????? Xshell
func (a *App) ImportXshellFromFolder(dirPath, accountID, group string) (*data.MachineImportResult, error) {
	if dirPath == "" {
		return nil, fmt.Errorf("??????")
	}
	return a.configManager.ImportXshell([]string{dirPath}, accountID, group)
}

// GetGlobalAccounts ???? SSH ??
func (a *App) GetGlobalAccounts() []data.GlobalAccountDTO {
	return a.configManager.GetGlobalAccounts()
}

// SaveGlobalAccounts ???? SSH ??
func (a *App) SaveGlobalAccounts(accounts []data.GlobalAccount) error {
	return a.configManager.SaveGlobalAccounts(accounts)
}

// SaveGlobalAccountsFromDTO ???? SSH ??????????
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

// CreateMachine ???????????
func (a *App) CreateMachine(machine define.Machine, sensitiveData define.SensitiveData) (string, error) {
	machine.EnsureID()
	if err := machine.SetSensitiveData(&sensitiveData); err != nil {
		return "", fmt.Errorf("????????: %w", err)
	}
	if err := a.configManager.AddMachineToGlobal(&machine); err != nil {
		return "", err
	}
	return machine.ID, nil
}

// OpenMachineConfig ????????????????
func (a *App) OpenMachineConfig() {
	// ????????????????
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "open:machine-config", map[string]interface{}{
			"timestamp": time.Now().Unix(),
		})
		fmt.Println("??????????")
	} else {
		fmt.Println("??: ctx ? nil???????")
		a.emitOperationEvent(define.OpTypeMachineConfig, "???????ctx ? nil", define.MsgTypeError, false, nil)
	}
}

// RefreshAll ??????
func (a *App) RefreshConfigMenu() error {
	// ???????????????????????
	// ?? subProjectRunner ???????? configManager ???
	_ = a.StopAllSubProjects()
	a.ClearOutput()

	a.configManager = data.NewConfigManager("", a.sessionManager)
	a.setupSubProjectRunner()
	if a.ctx != nil {
		err := a.UpdateApplicationMenu()
		if err != nil {
			fmt.Printf("??????: %v\n", err)
		} else {
			fmt.Println("??????")
		}
	}
	return nil
}

// RefreshConfigMenuWithEvent ?????????????
func (a *App) RefreshConfigMenuWithEvent() error {
	// ???????????????????????
	// ?? subProjectRunner ???????? configManager ???
	_ = a.StopAllSubProjects()
	a.ClearOutput()

	a.configManager = data.NewConfigManager("", a.sessionManager)
	a.setupSubProjectRunner()
	if a.ctx != nil {
		err := a.UpdateApplicationMenu()
		if err != nil {
			a.emitOperationEvent(define.OpTypeRefreshConfig, fmt.Sprintf("??????: %v", err), define.MsgTypeError, true, nil)
			return err
		} else {
			fmt.Println("??????")
		}
	}

	// ???? needReload ? true ?????????????????????
	a.emitOperationEvent(define.OpTypeRefreshConfig, "????????", define.MsgTypeSuccess, true, nil)
	return nil
}

// UpdateApplicationMenu ??????????????????
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

// CreateApplicationMenu ?????????????
func (a *App) CreateApplicationMenu() *menu.Menu {
	appMenu := menu.NewMenu()

	// ????
	fileMenu := appMenu.AddSubmenu("??")
	fileMenu.AddText("????", keys.CmdOrCtrl("n"), func(_ *menu.CallbackData) {
		NewWindow()
	})

	fileMenu.AddSeparator()
	// ????????
	configMenu := appMenu.AddSubmenu("??")
	// ????
	configFileMenu := appMenu.AddSubmenu("????")
	// ??????????
	configFiles, err := a.GetConfigFiles()
	if err != nil {
		// ????????????
		configFileMenu.AddText("????????", keys.CmdOrCtrl("r"), func(_ *menu.CallbackData) {
			a.RefreshConfigMenuWithEvent()
		})
	} else {
		// ????????
		globalConfig, _ := a.GetGlobalConfig()
		currentConfig := a.configManager.GetConfigPath()
		if currentConfig == "" && globalConfig != nil {
			currentConfig = globalConfig.LastOpenedFile
		}

		// ????????????
		for _, configFile := range configFiles {
			// ???????????
			fileName := getFileName(configFile)
			// ?????
			_ = configFileMenu.AddRadio(fileName, configFile == currentConfig, nil, func(data *menu.CallbackData) {
				// ??????
				switchConfigFile(a, configFile)
			})
		}

		// ??????????
		configFileMenu.AddSeparator()
		configFileMenu.AddText("??????", keys.CmdOrCtrl("r"), func(_ *menu.CallbackData) {
			a.RefreshConfigMenuWithEvent()
		})
		configFileMenu.AddText("??????", nil, func(_ *menu.CallbackData) {
			// ?????????? GlobalConfigManager
			globalConfigPath := a.configManager.GetGlobalConfigPath()
			if globalConfigPath != "" {
				OpenCurrentConfig(globalConfigPath)
			}
		})

		configFileMenu.AddText("??????", nil, func(_ *menu.CallbackData) {
			a.OpenCurrentConfigWithEvent()
		})
	}

	configMenu.AddText("????", keys.CmdOrCtrl("m"), func(_ *menu.CallbackData) {
		// ?????????
		a.OpenMachineConfig()
	})

	configMenu.AddText("?????", keys.CmdOrCtrl("e"), func(_ *menu.CallbackData) {
		a.OpenConnectionManager()
	})

	configMenu.AddText("????", keys.CmdOrCtrl("u"), func(_ *menu.CallbackData) {
		// ???????????
		a.OpenWorkPathConfig()
	})

	configMenu.AddSeparator()
	// configMenu.AddText("??????", keys.CmdOrCtrl(","), func(_ *menu.CallbackData) {
	// 	a.OpenConfigEditor()
	// })
	configMenu.AddText("????", keys.CmdOrCtrl(","), func(_ *menu.CallbackData) {
		a.OpenSystemSettings()
	})

	// ????
	helpMenu := appMenu.AddSubmenu("??")
	helpMenu.AddText("??", nil, func(_ *menu.CallbackData) {
		// ??????
		a.OpenAbout()
	})

	return appMenu
}

// getFileName ???????????
func getFileName(filePath string) string {
	if filePath == "" {
		return ""
	}
	// ?????????? Unix ? Windows ??
	for i := len(filePath) - 1; i >= 0; i-- {
		if filePath[i] == '/' || filePath[i] == '\\' {
			return filePath[i+1:]
		}
	}
	return filePath
}

// switchConfigFile ??????
func switchConfigFile(appInstance *App, configFile string) {
	err := appInstance.SwitchConfigFileWithEvent(configFile)
	if err != nil {
		// ????????????????????
		println("????????:", err.Error())
	} else {
		println("?????????:", configFile)
		// ???????????????????????
		// ????????????? SwitchConfigFileWithEvent ???????
	}
}

// NewWindow ?????????????????????
func NewWindow() {
	execPath, err := os.Executable()
	if err != nil {
		println("???????:", err.Error())
		return
	}
	sessionID := data.NewSessionID()
	cmd := exec.Command(execPath, "-session="+sessionID)
	if err := cmd.Start(); err != nil {
		println("???????:", err.Error())
	}
}

// NewWindow ????????????
func (a *App) NewWindow() {
	NewWindow()
}

// GetCurrentConfigPath ????????????
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

// OpenCurrentConfig ???????????????
func OpenCurrentConfig(lastOpenedFile string) {
	if lastOpenedFile == "" {
		fmt.Println("??????????")
		return
	}

	// ????????
	if _, err := os.Stat(lastOpenedFile); os.IsNotExist(err) {
		fmt.Printf("???????: %s\n", lastOpenedFile)
		return
	}

	// ??????????????
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("open", lastOpenedFile)
	case "windows": // Windows
		cmd = exec.Command("cmd", "/c", "start", "", lastOpenedFile)
	case "linux": // Linux
		cmd = exec.Command("xdg-open", lastOpenedFile)
	default:
		fmt.Printf("????????: %s\n", runtime.GOOS)
		return
	}

	err := cmd.Run()
	if err != nil {
		fmt.Printf("????????: %v\n", err)
	} else {
		fmt.Printf("????????: %s\n", lastOpenedFile)
	}
}

// OpenCurrentConfigWithEvent ???????????????
func (a *App) OpenCurrentConfigWithEvent() {
	globalConfig, err := a.GetGlobalConfig()
	if err != nil {
		a.emitOperationEvent(define.OpTypeOpenConfig, fmt.Sprintf("????????: %s", err.Error()), define.MsgTypeError, false, nil)
		return
	}

	lastOpenedFile := globalConfig.LastOpenedFile
	if lastOpenedFile == "" {
		a.emitOperationEvent(define.OpTypeOpenConfig, "??????????", define.MsgTypeWarning, false, nil)
		return
	}

	// ????????
	if _, err := os.Stat(lastOpenedFile); os.IsNotExist(err) {
		a.emitOperationEvent(define.OpTypeOpenConfig, fmt.Sprintf("???????: %s", lastOpenedFile), define.MsgTypeError, false, nil)
		return
	}

	// ??????????????
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("open", lastOpenedFile)
	case "windows": // Windows
		cmd = exec.Command("cmd", "/c", "start", "", lastOpenedFile)
	case "linux": // Linux
		cmd = exec.Command("xdg-open", lastOpenedFile)
	default:
		a.emitOperationEvent(define.OpTypeOpenConfig, fmt.Sprintf("????????: %s", runtime.GOOS), define.MsgTypeError, false, nil)
		return
	}

	err = cmd.Run()
	if err != nil {
		a.emitOperationEvent(define.OpTypeOpenConfig, fmt.Sprintf("????????: %v", err), define.MsgTypeError, false, nil)
		return
	}

	a.emitOperationEvent(define.OpTypeOpenConfig, fmt.Sprintf("????????: %s", lastOpenedFile), define.MsgTypeSuccess, false, nil)
}

// OpenGlobalConfigWithEvent ???????????????
func (a *App) OpenGlobalConfigWithEvent() {
	globalConfigPath := a.configManager.GetGlobalConfigPath()
	if globalConfigPath == "" {
		a.emitOperationEvent(define.OpTypeOpenConfig, "??????????", define.MsgTypeWarning, false, nil)
		return
	}

	if _, err := os.Stat(globalConfigPath); os.IsNotExist(err) {
		a.emitOperationEvent(define.OpTypeOpenConfig, fmt.Sprintf("?????????: %s", globalConfigPath), define.MsgTypeError, false, nil)
		return
	}

	if err := openWithSystemApp(globalConfigPath); err != nil {
		a.emitOperationEvent(define.OpTypeOpenConfig, fmt.Sprintf("??????????: %v", err), define.MsgTypeError, false, nil)
		return
	}

	a.emitOperationEvent(define.OpTypeOpenConfig, fmt.Sprintf("??????????: %s", globalConfigPath), define.MsgTypeSuccess, false, nil)
}

// GetWorkPaths ????????
func (a *App) GetWorkPaths() map[string]string {
	return a.configManager.GetAllWorkPathsFromGlobal()
}

// AddWorkPath ??????
func (a *App) AddWorkPath(key, value string) error {
	return a.configManager.AddWorkPathToGlobal(key, value)
}

// AddWorkPathWithEvent ?????????????
func (a *App) AddWorkPathWithEvent(key, value string) error {
	err := a.configManager.AddWorkPathToGlobal(key, value)
	if err != nil {
		a.emitOperationEvent(define.OpTypeEnvConfig, fmt.Sprintf("????????: %s", err.Error()), define.MsgTypeError, false, nil)
		return err
	}

	a.emitOperationEvent(define.OpTypeEnvConfig, fmt.Sprintf("????????: %s", key), define.MsgTypeSuccess, false, nil)
	return nil
}

// UpdateWorkPath ??????
func (a *App) UpdateWorkPath(key, value string) error {
	return a.configManager.UpdateWorkPathInGlobal(key, value)
}

// UpdateWorkPathWithEvent ?????????????
func (a *App) UpdateWorkPathWithEvent(key, value string) error {
	err := a.configManager.UpdateWorkPathInGlobal(key, value)
	if err != nil {
		a.emitOperationEvent(define.OpTypeEnvConfig, fmt.Sprintf("????????: %s", err.Error()), define.MsgTypeError, false, nil)
		return err
	}

	a.emitOperationEvent(define.OpTypeEnvConfig, fmt.Sprintf("????????: %s", key), define.MsgTypeSuccess, false, nil)
	return nil
}

// DeleteWorkPath ??????
func (a *App) DeleteWorkPath(key string) error {
	return a.configManager.RemoveWorkPathFromGlobal(key)
}

// DeleteWorkPathWithEvent ?????????????
func (a *App) DeleteWorkPathWithEvent(key string) error {
	err := a.configManager.RemoveWorkPathFromGlobal(key)
	if err != nil {
		a.emitOperationEvent(define.OpTypeEnvConfig, fmt.Sprintf("????????: %s", err.Error()), define.MsgTypeError, false, nil)
		return err
	}

	a.emitOperationEvent(define.OpTypeEnvConfig, fmt.Sprintf("????????: %s", key), define.MsgTypeSuccess, false, nil)
	return nil
}

// OpenWorkPathConfig ??????????????????
func (a *App) OpenWorkPathConfig() {
	// ??????????????????
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "open:workpath-config", map[string]interface{}{
			"timestamp": time.Now().Unix(),
		})
		fmt.Println("????????????")
	} else {
		fmt.Println("??: ctx ? nil???????")
		a.emitOperationEvent(define.OpTypeEnvConfig, "???????ctx ? nil", define.MsgTypeError, false, nil)
	}
}

// OpenConnectionManager ?? Shell ?????
func (a *App) OpenConnectionManager() {
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "open:connection-manager", map[string]interface{}{
			"timestamp": time.Now().Unix(),
		})
	}
}

// OpenAbout ??????????????
func (a *App) OpenAbout() {
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "open:about", map[string]interface{}{
			"timestamp": time.Now().Unix(),
		})
		fmt.Println("???????????")
		return
	}
	fmt.Println("??: ctx ? nil?????????")
}

// emitOperationEvent ?????????
func (a *App) emitOperationEvent(eventType, message, messageType string, needReload bool, data any) {
	if a.ctx == nil {
		fmt.Printf("??: ctx ? nil??????? %s\n", eventType)
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
	fmt.Printf("??????: %s - %s (%s)\n", eventType, message, messageType)
}

// GetSessionInfo ??????????
func (a *App) GetSessionInfo() data.SessionState {
	if a.sessionManager == nil {
		return data.SessionState{}
	}
	return a.sessionManager.GetState()
}

// GetSystemSettings ??????
func (a *App) GetSystemSettings() (*data.GlobalConfig, error) {
	cfg, err := a.configManager.GetGlobalConfig()
	if err != nil {
		return nil, err
	}
	if cfg != nil {
		cfg.ShellMonitorIntervalMs = normalizeShellMonitorIntervalMs(cfg.ShellMonitorIntervalMs)
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

// GetShortcutSettings ????????~/.flashdock/shortcuts.json?
func (a *App) GetShortcutSettings() (data.ShortcutSettings, error) {
	return data.LoadShortcutSettings()
}

// SaveShortcutSettings ???????? JSON????????
func (a *App) SaveShortcutSettings(settings data.ShortcutSettings) error {
	if err := data.SaveShortcutSettings(settings); err != nil {
		return err
	}
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "shortcuts:changed", settings)
	}
	return nil
}

// SaveSystemSettings ??????
func (a *App) SaveSystemSettings(config *data.GlobalConfig) error {
	a.normalizeThemeSettings(&config.ThemeSettings)
	normalizeProxySettings(&config.ProxySettings)
	config.ShellMonitorIntervalMs = normalizeShellMonitorIntervalMs(config.ShellMonitorIntervalMs)
	config.ShellMonitorIntervalSec = 0
	config.ShellLogHighlightColors = data.NormalizeShellLogHighlightColors(config.ShellLogHighlightColors)
	config.ShellLogHighlightDisabled = data.NormalizeShellLogHighlightDisabled(config.ShellLogHighlightDisabled)
	if err := a.configManager.SaveGlobalConfig(config); err != nil {
		return err
	}
	a.refreshLogSettings()
	a.applyProxySettings(config.ProxySettings)
	if a.sessionManager != nil && config.ThemeSettings.Mode != "" {
		_ = a.sessionManager.SetTheme(config.ThemeSettings.Mode, config.ThemeSettings.TerminalPreset)
	}
	a.applyWindowTheme(config.ThemeSettings.Mode)
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "theme:changed", config.ThemeSettings)
		wailsRuntime.EventsEmit(a.ctx, "system-settings:changed", map[string]any{
			"shellMonitorIntervalMs":  config.ShellMonitorIntervalMs,
			"shellLogHighlight":        data.ShellLogHighlightEnabled(config),
			"shellLogHighlightColors":  config.ShellLogHighlightColors,
			"shellLogHighlightDisabled": config.ShellLogHighlightDisabled,
			"proxySettings":            config.ProxySettings,
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

// GetExecutionLogs ????????
func (a *App) GetExecutionLogs(limit int) ([]data.LogEntry, error) {
	return a.logManager.ListLogs(limit)
}

// ReadExecutionLog ????????
func (a *App) ReadExecutionLog(fileName string) (string, error) {
	return a.logManager.ReadLog(fileName)
}

// OpenExecutionLog ?????????????
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
	return fmt.Errorf("???????: %s", fileName)
}

// OpenConfigEditor ?????????
func (a *App) OpenConfigEditor() {
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "open:config-editor", map[string]interface{}{
			"timestamp": time.Now().Unix(),
		})
	}
}

// OpenSystemSettings ??????
func (a *App) OpenSystemSettings() {
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "open:system-settings", map[string]interface{}{
			"timestamp": time.Now().Unix(),
		})
	}
}

// OpenExecutionHistory ??????
func (a *App) OpenExecutionHistory() {
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "open:execution-history", map[string]interface{}{
			"timestamp": time.Now().Unix(),
		})
	}
}

// GetThemeSettings ???????????????????
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

// SaveThemeSettings ????????????
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
// ConnectShell ???? Shell ??????? ID???????name / name#2?
func (a *App) ConnectShell(configName string) (string, error) {
	if machine.IsLocalShellID(configName) {
		return "", fmt.Errorf("??? ConnectLocalShell ??????")
	}
	configName = strings.TrimSpace(configName)
	configName = a.remoteConfigName(configName)
	machineConfig := a.configManager.GetMachine(configName)
	if machineConfig == nil {
		return "", fmt.Errorf("???????: %s", configName)
	}

	sessionID, err := a.shellPool.Connect(machineConfig, a.configManager.GetWorkPathVars(), a.shellHandlerFor)
	if err != nil {
		return "", err
	}

	a.ensureShellAux(sessionID, machineConfig)
	if err := a.ensureMachineTunnels(machineConfig); err != nil {
		fmt.Printf("SSH ??????(%s): %v\n", configName, err)
	}

	if sensitive, sErr := machineConfig.GetSensitiveData(); sErr == nil && a.shellHistory != nil {
		_ = a.shellHistory.RecordConnect(machineConfig, sensitive.Host, sensitive.Port, sensitive.User)
	}

	a.emitShellSessions()
	return sessionID, nil
}


// ReconnectShell ??? ID ????????
func (a *App) ReconnectShell(sessionID string) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", fmt.Errorf("?? ID ??")
	}
	if machine.IsLocalShellID(sessionID) {
		return a.ConnectLocalShell(sessionID)
	}
	configName := a.remoteConfigName(sessionID)
	machineConfig := a.configManager.GetMachine(configName)
	if machineConfig == nil {
		return "", fmt.Errorf("???????: %s", configName)
	}
	if a.shellPool.IsConnected(sessionID) {
		a.ensureShellAux(sessionID, machineConfig)
		a.emitShellSessions()
		return sessionID, nil
	}
	if err := a.shellPool.ConnectID(sessionID, machineConfig, a.configManager.GetWorkPathVars(), a.shellHandlerFor(sessionID)); err != nil {
		return "", err
	}
	a.ensureShellAux(sessionID, machineConfig)
	if err := a.ensureMachineTunnels(machineConfig); err != nil {
		fmt.Printf("SSH ??????(%s): %v\n", configName, err)
	}
	a.emitShellSessions()
	return sessionID, nil
}
// ConnectLocalShell ??????????sessionID ????????? ID?????? ID ???
func (a *App) ConnectLocalShell(sessionID string) (string, error) {
	if a.localShellPool == nil {
		return "", fmt.Errorf("???????")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID != "" {
		if !machine.IsLocalShellID(sessionID) {
			return "", fmt.Errorf("?????? ID")
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
	return a.shellAuxPool.Get(a.resolveAuxKey(sessionOrConfig))
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
		fmt.Printf("??????(%s): %v\n", auxKey, auxErr)
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
// DisconnectShell ??????
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
		}
	}
	a.clearShellCwd(machineName)
	a.emitShellSessions()
	return err
}

// GetShellHistory ??????
func (a *App) GetShellHistory() []define.ShellHistoryRecord {
	if a.shellHistory == nil {
		return nil
	}
	return a.shellHistory.List()
}

// ClearShellHistory ??????
func (a *App) ClearShellHistory() error {
	if a.shellHistory == nil {
		return nil
	}
	return a.shellHistory.Clear()
}

// RemoveShellHistory ????????
func (a *App) RemoveShellHistory(machineID, machineName string) error {
	if a.shellHistory == nil {
		return nil
	}
	return a.shellHistory.Remove(machineID, machineName)
}

// GetShellMonitor ????????
func (a *App) GetShellMonitor(machineName string) *define.ShellMonitorSnapshot {
	aux, err := a.getShellAux(machineName)
	if err != nil {
		host := ""
		if m := a.configManager.GetMachine(machineName); m != nil {
			if s, e := m.GetSensitiveData(); e == nil {
				host = s.Host
			}
		}
		// ?????????????????????? UI ??????
		return &define.ShellMonitorSnapshot{
			MachineName: machineName,
			Host:        host,
			UpdatedAt:   time.Now().Unix(),
			TopMem:      []define.ShellProcessStat{},
		}
	}
	snap := aux.FetchMonitor()
	if m := a.configManager.GetMachine(machineName); m != nil {
		if s, e := m.GetSensitiveData(); e == nil && s.Host != "" {
			snap.Host = s.Host
		}
	}
	return snap
}

// ListShellFiles ??????
func (a *App) ListShellFiles(machineName, dirPath string, showHidden bool) ([]define.SftpEntry, error) {
	aux, err := a.getShellAux(machineName)
	if err != nil {
		return nil, err
	}
	return aux.ListDir(dirPath, showHidden)
}

// DeleteShellFile ?????????
func (a *App) DeleteShellFile(machineName, remotePath string) error {
	if strings.TrimSpace(remotePath) == "" || remotePath == "/" {
		return fmt.Errorf("????")
	}
	aux, err := a.getShellAux(machineName)
	if err != nil {
		return err
	}
	return aux.RemovePath(remotePath)
}

// GetShellRemoteHome ???? home?SFTP ?????
func (a *App) GetShellRemoteHome(machineName string) (string, error) {
	return a.getRemoteHome(machineName)
}

// GetShellRemotePwd ?????? pwd???????????? GetShellPtyCwd?
func (a *App) GetShellRemotePwd(machineName string) (string, error) {
	aux, err := a.getShellAux(machineName)
	if err != nil {
		return "", err
	}
	return aux.Pwd()
}

// GetShellPtyCwd ?? PTY ????????????? / home?
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
	return "", fmt.Errorf("PTY cwd ??")
}

// SyncShellCwd ???? cd ????? cwd?Enter ????????? shell ???
func (a *App) SyncShellCwd(machineName, cdLine string) (string, error) {
	cdLine = strings.TrimSpace(cdLine)
	if cdLine == "" {
		a.shellCwdMu.RLock()
		raw := a.shellCwds[machineName]
		a.shellCwdMu.RUnlock()
		if clean, ok := machine.SanitizePtyCwd(raw); ok {
			return clean, nil
		}
		return "", fmt.Errorf("PTY cwd ??")
	}
	if len(cdLine) < 2 || !strings.EqualFold(cdLine[:2], "cd") {
		return "", fmt.Errorf("? cd ??")
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

// ShellDirExists ?????????
func (a *App) ShellDirExists(machineName, dirPath string) (bool, error) {
	aux, err := a.getShellAux(machineName)
	if err != nil {
		return false, err
	}
	return aux.DirExists(dirPath)
}

// ResolveShellPath ?????????? ? ?? base?? base ?? home?
func (a *App) ResolveShellPath(machineName, basePath, target string) (string, error) {
	home, err := a.getRemoteHome(machineName)
	if err != nil {
		home = ""
	}
	return ResolveRemotePath(basePath, target, home)
}

// ApplyShellCd ?? cd ???? SFTP ???????????? current?
func (a *App) ApplyShellCd(machineName, current, target string) (string, error) {
	home, err := a.getRemoteHome(machineName)
	if err != nil {
		return "", fmt.Errorf("?? home ??: %w", err)
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
		return "", fmt.Errorf("?????? %s: %w", resolved, err)
	}
	return ChooseCdPath(current, resolved, exists), nil
}

// shellDirExistsReliable?Stat/ReadDir?????????????? cd ???
func (a *App) shellDirExistsReliable(machineName, dirPath string) (bool, error) {
	exists, err := a.ShellDirExists(machineName, dirPath)
	if err == nil {
		return exists, nil
	}
	// Stat ??????????????
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
		return fmt.Errorf("??? SSH ??")
	}
	return a.tunnelMgr.EnsureForMachine(machineConfig.Name, machineConfig.Tunnels, client)
}

func (a *App) stopMachineTunnels(configName string) {
	if a.tunnelMgr != nil {
		a.tunnelMgr.StopAllFor(configName)
	}
}

// GetShellTunnelStatus ????????
func (a *App) GetShellTunnelStatus(configName string) []define.SSHTunnelStatus {
	if a.tunnelMgr == nil {
		return nil
	}
	return a.tunnelMgr.StatusList(a.remoteConfigName(configName))
}

// BroadcastShellInput ?????????
func (a *App) BroadcastShellInput(sessionIDs []string, input string) error {
	a.executionMutex.RLock()
	defer a.executionMutex.RUnlock()
	if len(sessionIDs) == 0 {
		return fmt.Errorf("???????")
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
				err = fmt.Errorf("???????")
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


// SendShellInput ??? PTY Shell ????
func (a *App) SendShellInput(machineName, input string) error {
	a.executionMutex.RLock()
	defer a.executionMutex.RUnlock()
	if machine.IsLocalShellID(machineName) {
		if a.localShellPool == nil {
			return fmt.Errorf("???????")
		}
		return a.localShellPool.SendInput(machineName, input)
	}
	return a.shellPool.SendInput(machineName, input)
}

// SendShellInterrupt ??? PTY Shell ?? Ctrl+C
func (a *App) SendShellInterrupt(machineName string) error {
	a.executionMutex.RLock()
	defer a.executionMutex.RUnlock()
	if machine.IsLocalShellID(machineName) {
		if a.localShellPool == nil {
			return fmt.Errorf("???????")
		}
		return a.localShellPool.SendInterrupt(machineName)
	}
	return a.shellPool.SendInterrupt(machineName)
}

// ResizeShell ???? PTY ????
func (a *App) ResizeShell(machineName string, cols, rows int) error {
	a.executionMutex.RLock()
	defer a.executionMutex.RUnlock()
	if machine.IsLocalShellID(machineName) {
		if a.localShellPool == nil {
			return fmt.Errorf("???????")
		}
		return a.localShellPool.Resize(machineName, cols, rows)
	}
	return a.shellPool.Resize(machineName, cols, rows)
}

// ExecuteShellCommand ?????
func (a *App) ExecuteShellCommand(machineName, command string) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return fmt.Errorf("??????")
	}
	if !strings.HasSuffix(command, "\n") {
		command += "\n"
	}
	return a.SendShellInput(machineName, command)
}

// StopShellCommand ?????
func (a *App) StopShellCommand(machineName string) error {
	return a.SendShellInterrupt(machineName)
}

// GetShellSessions ???? Shell ????
func (a *App) GetShellSessions() []define.ShellStatus {
	return a.listAllShellSessions()
}

// GetShellStatus ????????????????
func (a *App) GetShellStatus() *define.ShellStatus {
	sessions := a.listAllShellSessions()
	if len(sessions) == 0 {
		return &define.ShellStatus{}
	}
	return &sessions[0]
}

// ClearShellOutput ????????
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
		return fmt.Errorf("????????: %s", runtime.GOOS)
	}
	return cmd.Run()
}
