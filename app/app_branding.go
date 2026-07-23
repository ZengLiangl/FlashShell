package app

import (
	"fmt"
	"strings"

	"FlashDock/data"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ListAppIconPresets 返回 Dock 图标预设（含预览 data URL）
func (a *App) ListAppIconPresets() []AppIconPresetInfo {
	return listAppIconPresets()
}

// PickCustomAppIcon 选择图片并保存到 ~/.flashdock/icons/custom.png，返回预设 id "custom"
func (a *App) PickCustomAppIcon() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("应用未就绪")
	}
	path, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择 Dock 图标",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "图片", Pattern: "*.png;*.jpg;*.jpeg;*.PNG;*.JPG;*.JPEG"},
		},
	})
	if err != nil {
		return "", err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	if err := saveCustomAppIconFromFile(path); err != nil {
		return "", err
	}
	a.applyAppIcon("custom")
	return "custom", nil
}

func (a *App) applyWindowTitle(windowsName string) {
	if a.ctx == nil {
		return
	}
	title := strings.TrimSpace(windowsName)
	if title == "" {
		title = "FlashDock"
	}
	wailsRuntime.WindowSetTitle(a.ctx, title)
}

func (a *App) applyAppIcon(preset string) {
	preset = resolveAppIconPreset(preset)
	pngBytes, err := presetPNGBytes(preset)
	if err != nil || len(pngBytes) == 0 {
		return
	}
	setApplicationDockIconPNG(pngBytes)
}

func (a *App) applyAppBranding(cfg *data.GlobalConfig) {
	if cfg == nil {
		a.applyWindowTitle("FlashDock")
		a.applyAppIcon("default")
		return
	}
	cfg.AppIconPreset = resolveAppIconPreset(cfg.AppIconPreset)
	a.applyWindowTitle(cfg.WindowsName)
	a.applyAppIcon(cfg.AppIconPreset)
}

func (a *App) applyAppBrandingFromConfig() {
	cfg, err := a.configManager.GetGlobalConfig()
	if err != nil || cfg == nil {
		a.applyAppBranding(nil)
		return
	}
	a.applyAppBranding(cfg)
}

// applyStartupFullscreen 将「启动时全屏」落实为窗口最大化（保留标题栏），
// 而非系统独占全屏（macOS Spaces / Windows 无边框全屏）。
func (a *App) applyStartupFullscreen(enabled bool) {
	if a.ctx == nil {
		return
	}
	// 旧版本曾用 WindowFullscreen，开启最大化前先退出独占全屏
	if wailsRuntime.WindowIsFullscreen(a.ctx) {
		wailsRuntime.WindowUnfullscreen(a.ctx)
	}
	if enabled {
		wailsRuntime.WindowMaximise(a.ctx)
		return
	}
	if wailsRuntime.WindowIsMaximised(a.ctx) {
		wailsRuntime.WindowUnmaximise(a.ctx)
	}
}

func (a *App) applyStartupFullscreenFromConfig() {
	cfg, err := a.configManager.GetGlobalConfig()
	if err != nil || cfg == nil || !cfg.StartupFullscreen {
		return
	}
	a.applyStartupFullscreen(true)
}
