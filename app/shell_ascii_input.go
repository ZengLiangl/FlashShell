package app

import (
	"FlashDock/data"
	"FlashDock/inputmethod"
)

// ShellAsciiInputEnter Shell 终端获得焦点时临时关闭中文组词。
func (a *App) ShellAsciiInputEnter() {
	if !a.shellAsciiInputEnabled() {
		return
	}
	inputmethod.Enter()
}

// ShellAsciiInputLeave 失焦或离开 Shell 时恢复进入前的输入法状态。
func (a *App) ShellAsciiInputLeave() {
	inputmethod.Leave()
}

func (a *App) shellAsciiInputEnabled() bool {
	if a == nil || a.configManager == nil {
		return true
	}
	cfg, err := a.configManager.GetGlobalConfig()
	if err != nil {
		return true
	}
	return data.ShellAsciiInputEnabled(cfg)
}
