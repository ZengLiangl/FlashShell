package app

import (
	"FlashDock/data"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var desktopApp *App

func (a *App) bindDesktopChrome() {
	desktopApp = a
}

func (a *App) applyDesktopChromeFromConfig() {
	if a == nil || a.configManager == nil {
		return
	}
	cfg, err := a.configManager.GetGlobalConfig()
	if err != nil || cfg == nil {
		return
	}
	a.applyDesktopChrome(cfg)
}

func (a *App) applyDesktopChrome(cfg *data.GlobalConfig) {
	if cfg == nil {
		return
	}
	nativeEnsureWindowHook()
	nativeApplyWindowOpacity(data.NormalizeWindowOpacity(cfg.WindowOpacity))
	nativeSetTrayEnabled(cfg.CloseToTray || cfg.MinimizeToTray)
}

func (a *App) closeToTrayEnabled() bool {
	if a == nil || a.configManager == nil {
		return false
	}
	cfg, err := a.configManager.GetGlobalConfig()
	return err == nil && cfg != nil && cfg.CloseToTray
}

func (a *App) minimizeToTrayEnabled() bool {
	if a == nil || a.configManager == nil {
		return false
	}
	cfg, err := a.configManager.GetGlobalConfig()
	return err == nil && cfg != nil && cfg.MinimizeToTray
}

// HideMainWindow 隐藏主窗口（托盘 / 全局热键）。
func (a *App) HideMainWindow() {
	nativeHideMainWindow(a)
}

// ShowMainWindow 显示并前置主窗口。
func (a *App) ShowMainWindow() {
	nativeShowMainWindow(a)
}

// ToggleMainWindow 显示或隐藏主窗口。
func (a *App) ToggleMainWindow() {
	if nativeMainWindowVisible() {
		a.HideMainWindow()
		return
	}
	a.ShowMainWindow()
}

// ApplyWindowOpacity 立即设置主窗口透明度（不落盘）。
func (a *App) ApplyWindowOpacity(opacity float64) {
	nativeEnsureWindowHook()
	nativeApplyWindowOpacity(data.NormalizeWindowOpacity(opacity))
}

// MinimizeMainWindow 最小化；开启「最小化到托盘」时改为隐藏。
func (a *App) MinimizeMainWindow() {
	if a.minimizeToTrayEnabled() {
		a.HideMainWindow()
		return
	}
	if a.ctx != nil {
		wailsRuntime.WindowMinimise(a.ctx)
	}
}

func desktopQuitFromTray() {
	if desktopApp == nil {
		return
	}
	desktopApp.ConfirmQuit()
}

func wailsHide(a *App) {
	if a != nil && a.ctx != nil {
		wailsRuntime.WindowHide(a.ctx)
	}
}

func wailsShow(a *App) {
	if a != nil && a.ctx != nil {
		wailsRuntime.WindowShow(a.ctx)
		wailsRuntime.WindowUnminimise(a.ctx)
	}
}
