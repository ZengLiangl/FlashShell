package app

import (
	"time"

	"FlashDock/define"
	"FlashDock/machine"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var shellReconnectDelays = []time.Duration{
	2 * time.Second,
	5 * time.Second,
	10 * time.Second,
}

func (a *App) initExecutionFeatures() {
	a.executionBootstrapOnce.Do(func() {
		machine.SetMachineResolver(func(name string) *define.Machine {
			if a.configManager == nil {
				return nil
			}
			return a.configManager.GetMachine(name)
		})
	})
	a.rebuildSubProjectRunner()
}

func (a *App) rebuildSubProjectRunner() {
	a.subProjectRunner = machine.NewSubProjectRunner(a.configManager)
	a.subProjectRunner.SetShellClientProvider(a.sshShareProvider())
	a.subProjectRunner.SetStatusChangeHandler(a.emitExecutionStatus)
	a.subProjectRunner.SetRemoteFailureHandler(a.emitRemoteFailureOpenShell)
}

func (a *App) emitRemoteFailureOpenShell(info machine.RemoteFailureInfo) {
	if a.ctx == nil || info.MachineName == "" {
		return
	}
	wailsRuntime.EventsEmit(a.ctx, "execution:open-shell", map[string]interface{}{
		"machineName": info.MachineName,
		"workdir":     info.WorkDir,
		"commandName": info.CommandName,
		"error":       info.Error,
	})
}

func (a *App) shellAutoReconnectEnabled() bool {
	return a.GetThemeSettings().ShellAutoReconnect
}

func (a *App) markShellUserDisconnect(sessionID string) {
	if sessionID == "" {
		return
	}
	a.shellDisconnectMu.Lock()
	if a.shellUserDisconnect == nil {
		a.shellUserDisconnect = make(map[string]bool)
	}
	a.shellUserDisconnect[sessionID] = true
	a.shellDisconnectMu.Unlock()
}

func (a *App) consumeShellUserDisconnect(sessionID string) bool {
	a.shellDisconnectMu.Lock()
	defer a.shellDisconnectMu.Unlock()
	if a.shellUserDisconnect == nil {
		return false
	}
	v := a.shellUserDisconnect[sessionID]
	delete(a.shellUserDisconnect, sessionID)
	return v
}

func (a *App) scheduleShellAutoReconnect(sessionID string) {
	if !a.shellAutoReconnectEnabled() || sessionID == "" || machine.IsLocalShellID(sessionID) {
		return
	}
	a.shellReconnectMu.Lock()
	if a.shellReconnecting == nil {
		a.shellReconnecting = make(map[string]bool)
	}
	if a.shellReconnecting[sessionID] {
		a.shellReconnectMu.Unlock()
		return
	}
	a.shellReconnecting[sessionID] = true
	a.shellReconnectMu.Unlock()

	go func() {
		defer func() {
			a.shellReconnectMu.Lock()
			delete(a.shellReconnecting, sessionID)
			a.shellReconnectMu.Unlock()
		}()
		configName := a.remoteConfigName(sessionID)
		for attempt, delay := range shellReconnectDelays {
			if a.consumeShellUserDisconnect(sessionID) {
				return
			}
			if a.ctx != nil {
				wailsRuntime.EventsEmit(a.ctx, "shell:reconnecting", map[string]interface{}{
					"machineName": sessionID,
					"attempt":     attempt + 1,
					"maxAttempts": len(shellReconnectDelays),
					"delaySec":    int(delay / time.Second),
				})
			}
			time.Sleep(delay)
			if a.consumeShellUserDisconnect(sessionID) {
				return
			}
			if _, err := a.ReconnectShell(sessionID); err == nil {
				if a.ctx != nil {
					wailsRuntime.EventsEmit(a.ctx, "shell:reconnected", map[string]interface{}{
						"machineName": sessionID,
						"configName":  configName,
					})
				}
				return
			}
		}
		if a.ctx != nil {
			wailsRuntime.EventsEmit(a.ctx, "shell:reconnect-failed", map[string]interface{}{
				"machineName": sessionID,
			})
		}
	}()
}

// DryRunSubProject 干跑 SubProject：仅展开将执行的步骤
func (a *App) DryRunSubProject(projectName, subProjectName string) ([]machine.DryRunLine, error) {
	if _, err := a.configManager.LoadConfigForRefresh(); err != nil {
		return nil, err
	}
	a.ClearOutput()
	lines, err := a.subProjectRunner.DryRunSubProject(projectName, subProjectName, a.outputWriter())
	if err != nil {
		a.pushOutput("干跑失败: " + err.Error())
	}
	return lines, err
}

// DryRunLine 干跑结果行（Wails 绑定用别名）
type DryRunLine = machine.DryRunLine
