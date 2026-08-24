package app

import (
	"fmt"
	"strings"
	"time"

	"FlashDock/data"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var dockIconRetryDelays = []time.Duration{
	50 * time.Millisecond,
	250 * time.Millisecond,
	800 * time.Millisecond,
}

const (
	defaultWindowWidth  = 1200
	defaultWindowHeight = 768
)

// ListAppIconPresets 返回 Dock 图标预设（含预览 data URL）
func (a *App) ListAppIconPresets() []AppIconPresetInfo {
	return listAppIconPresets()
}

// PickCustomAppIcon 选择图片并保存到 ~/.flashshell/icons/custom.png，返回预设 id "custom"
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
	if err := a.persistAndApplyAppIcon("custom"); err != nil {
		a.applyAppIcon("custom")
		return "custom", err
	}
	return "custom", nil
}

// ApplyAppIconPreset 立即切换 Dock/任务栏图标并写入全局配置，避免只改表单、未点保存就重启丢失。
func (a *App) ApplyAppIconPreset(preset string) error {
	return a.persistAndApplyAppIcon(preset)
}

func (a *App) persistAppIconPreset(preset string) (string, error) {
	preset = resolveAppIconPreset(preset)
	if a.configManager == nil {
		return "", fmt.Errorf("配置管理器未初始化")
	}
	cfg, err := a.configManager.GetGlobalConfig()
	if err != nil {
		return "", err
	}
	if cfg == nil {
		return "", fmt.Errorf("全局配置未加载")
	}
	cfg.AppIconPreset = preset
	if err := a.configManager.SaveGlobalConfig(cfg); err != nil {
		return "", err
	}
	return preset, nil
}

func (a *App) persistAndApplyAppIcon(preset string) error {
	preset, err := a.persistAppIconPreset(preset)
	if err != nil {
		return err
	}
	a.applyAppIcon(preset)
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "app-icon:changed", preset)
	}
	return nil
}

func (a *App) applyWindowTitle(windowsName string) {
	if a.ctx == nil {
		return
	}
	wailsRuntime.WindowSetTitle(a.ctx, NormalizeWindowTitle(windowsName))
}

func NormalizeWindowTitle(name string) string {
	return data.NormalizeWindowsName(name)
}

func (a *App) applyAppIcon(preset string) {
	preset = resolveAppIconPreset(preset)
	pngBytes, err := presetPNGBytes(preset)
	if err != nil || len(pngBytes) == 0 {
		return
	}
	copied := append([]byte(nil), pngBytes...)
	setApplicationDockIconPNG(copied)
	persistFinderAppIcon(copied, preset == "default")
	go retryApplicationDockIcon(copied)
}

func retryApplicationDockIcon(pngBytes []byte) {
	for _, d := range dockIconRetryDelays {
		time.Sleep(d)
		setApplicationDockIconPNG(pngBytes)
	}
}

func (a *App) applyAppBranding(cfg *data.GlobalConfig) {
	if cfg == nil {
		a.applyWindowTitle(ProductName)
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
		a.rememberNormalWindowBounds()
		wailsRuntime.WindowMaximise(a.ctx)
		a.markChromeMaximised(true)
		return
	}
	if a.windowIsEffectivelyMaximised() {
		a.restoreNormalWindow()
	}
}

func (a *App) applyStartupFullscreenFromConfig() {
	cfg, err := a.configManager.GetGlobalConfig()
	if err != nil || cfg == nil || !cfg.StartupFullscreen {
		return
	}
	a.applyStartupFullscreen(true)
}

// ToggleWindowMaximised 顶栏空白双击：在最大化与还原之间切换。
// 与「启动时全屏」相同，使用窗口最大化而非系统独占全屏。
func (a *App) ToggleWindowMaximised() {
	if a.ctx == nil {
		return
	}
	a.winRestoreMu.Lock()
	defer a.winRestoreMu.Unlock()
	if time.Since(a.lastWinToggleAt) < 220*time.Millisecond {
		return
	}
	a.lastWinToggleAt = time.Now()
	if wailsRuntime.WindowIsFullscreen(a.ctx) {
		wailsRuntime.WindowUnfullscreen(a.ctx)
	}
	if a.windowIsEffectivelyMaximised() {
		a.restoreNormalWindowLocked()
		return
	}
	a.rememberNormalWindowBoundsLocked()
	wailsRuntime.WindowMaximise(a.ctx)
	a.chromeMaximised = true
}

func (a *App) rememberNormalWindowBounds() {
	a.winRestoreMu.Lock()
	defer a.winRestoreMu.Unlock()
	a.rememberNormalWindowBoundsLocked()
}

func (a *App) rememberNormalWindowBoundsLocked() {
	if a.windowIsEffectivelyMaximised() {
		return
	}
	w, h := wailsRuntime.WindowGetSize(a.ctx)
	x, y := wailsRuntime.WindowGetPosition(a.ctx)
	if w < 200 || h < 200 {
		return
	}
	a.winRestoreX, a.winRestoreY = x, y
	a.winRestoreW, a.winRestoreH = w, h
	a.winRestoreSaved = true
}

func (a *App) restoreNormalWindow() {
	a.winRestoreMu.Lock()
	defer a.winRestoreMu.Unlock()
	a.restoreNormalWindowLocked()
}

func (a *App) restoreNormalWindowLocked() {
	a.chromeMaximised = false
	// Wails 无边框 Windows 的 WindowUnmaximise 走 SW_SHOW，经常无法真正还原。
	_ = nativeRestoreMaximisedWindow()
	wailsRuntime.WindowUnmaximise(a.ctx)
	screenW, screenH := 0, 0
	if screens, err := wailsRuntime.ScreenGetAll(a.ctx); err == nil {
		for _, screen := range screens {
			if !screen.IsCurrent {
				continue
			}
			screenW, screenH = screenLogicalSize(screen)
			break
		}
	}
	x, y, w, h := chooseRestoreBounds(a.winRestoreSaved, a.winRestoreX, a.winRestoreY, a.winRestoreW, a.winRestoreH, screenW, screenH)
	wailsRuntime.WindowSetSize(a.ctx, w, h)
	wailsRuntime.WindowSetPosition(a.ctx, x, y)
}

func (a *App) markChromeMaximised(max bool) {
	a.winRestoreMu.Lock()
	a.chromeMaximised = max
	a.winRestoreMu.Unlock()
}

// WindowIsChromeMaximised 给窗口按钮用：含我们自己记下的最大化状态，
// 不单依赖 Wails 在无边框窗口上经常不准的 WindowIsMaximised。
func (a *App) WindowIsChromeMaximised() bool {
	if a.ctx == nil {
		return false
	}
	// 先查 native / Wails，再短持锁读标记，避免持锁回调 UI 线程导致挂起
	nativeKnown, nativeMax := nativeWindowMaximisedState()
	wailsMax := wailsRuntime.WindowIsMaximised(a.ctx)
	a.winRestoreMu.Lock()
	if nativeKnown {
		// 拖动最大化窗口会由系统还原，必须同步清掉内部标记，否则按钮仍显示「还原」
		a.chromeMaximised = nativeMax
	}
	ourMax := a.chromeMaximised
	a.winRestoreMu.Unlock()
	return shouldTreatAsMaximised(nativeKnown, nativeMax, wailsMax, ourMax)
}

// windowIsEffectivelyMaximised 仅在已持有 winRestoreMu 时调用。
// 不在锁内调用 Wails（可能派发到 UI 线程），避免与绑定调用互相等待。
func (a *App) windowIsEffectivelyMaximised() bool {
	nativeKnown, nativeMax := nativeWindowMaximisedState()
	return shouldTreatAsMaximised(nativeKnown, nativeMax, false, a.chromeMaximised)
}

func shouldTreatAsMaximised(nativeKnown, nativeMax, wailsMax, ourMax bool) bool {
	if nativeKnown {
		return nativeMax
	}
	return wailsMax || ourMax
}

func chooseRestoreBounds(saved bool, x, y, w, h, screenW, screenH int) (int, int, int, int) {
	if saved && w >= 200 && h >= 200 {
		return x, y, w, h
	}
	rw, rh := defaultWindowWidth, defaultWindowHeight
	rx, ry := 0, 0
	if screenW > rw && screenH > rh {
		rx = (screenW - rw) / 2
		ry = (screenH - rh) / 2
	}
	return rx, ry, rw, rh
}

func screenLogicalSize(screen wailsRuntime.Screen) (int, int) {
	w, h := screen.Size.Width, screen.Size.Height
	if w <= 0 || h <= 0 {
		w, h = screen.Width, screen.Height
	}
	return w, h
}
