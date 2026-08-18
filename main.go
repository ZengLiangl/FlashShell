package main

import (
	"embed"
	"flag"
	"os"
	"os/exec"
	"runtime"
	"strings"
	_ "syscall"

	"FlashDock/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

// main is the entry point for the application
func main() {
	// Windows：--apply-update / 版本号便携名归一，须在启动 UI 前处理并退出
	if app.HandleEarlyUpdateArgs(os.Args[1:]) {
		return
	}

	// parse run mode from command line
	runMode := flag.String("reg", "desk", "运行模式: desk(前台)/back(后台)")
	sessionID := flag.String("session", "", "窗口会话 ID")
	flag.Parse()

	app.ConsumeWindowsUpdateCleanupEnv()

	// daemonize if requested (like nohup), only on non-Windows
	if *runMode == "back" && os.Getenv("FLASHDOCK_DAEMONIZED") != "1" {
		exePath, err := os.Executable()
		if err == nil {
			childArgs := os.Args[1:]
			for i, a := range childArgs {
				if strings.HasPrefix(a, "-reg=") {
					childArgs[i] = "-reg=desk"
				}
			}

			stdoutFile, _ := os.OpenFile("/tmp/FlashShell.out", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			stderrFile, _ := os.OpenFile("/tmp/FlashShell.err", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

			cmd := exec.Command(exePath, childArgs...)
			cmd.Env = append(os.Environ(), "FLASHDOCK_DAEMONIZED=1")
			cmd.Stdin = nil
			if stdoutFile != nil {
				cmd.Stdout = stdoutFile
			}
			if stderrFile != nil {
				cmd.Stderr = stderrFile
			}
			if runtime.GOOS != "windows" {
				// cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
			}
			_ = cmd.Start()
		}
		os.Exit(0)
	}

	// Create an instance of the app structure
	appInstance := app.NewApp(*sessionID)

	// 获取窗口名称
	globalConfig, err := appInstance.GetGlobalConfig()
	windowsName := app.ProductName
	if err == nil && globalConfig != nil {
		windowsName = app.NormalizeWindowTitle(globalConfig.WindowsName)
	}

	// 按主题初始化窗口背景，避免标题栏/边缘与主题不一致
	bg := &options.RGBA{R: 255, G: 255, B: 255, A: 255}
	macAppearance := mac.NSAppearanceNameAqua
	if err == nil && globalConfig != nil && globalConfig.ThemeSettings.Mode == "dark" {
		bg = &options.RGBA{R: 20, G: 20, B: 20, A: 255}
		macAppearance = mac.NSAppearanceNameDarkAqua
	}

	// Create application with options
	err = wails.Run(&options.App{
		Title:  windowsName,
		Width:  1200,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: bg,
		OnStartup:        appInstance.Startup,
		OnDomReady:       appInstance.DomReady,
		OnBeforeClose:    appInstance.BeforeClose,
		OnShutdown:       appInstance.Shutdown,
		Fullscreen:       false,
		WindowStartState: options.Normal,
		// Windows 无边框，顶栏与内容合一；macOS 用 TitleBarHidden，勿开 Frameless
		Frameless: runtime.GOOS == "windows",
		MinWidth:  1200,
		MinHeight: 768,
		Menu:      nil,
		Mac: &mac.Options{
			// 隐藏系统标题文字行，内容顶到窗口最上沿；红绿灯垂直位置由 setTrafficLightPosition 对齐顶栏
			TitleBar:             mac.TitleBarHidden(),
			Appearance:           macAppearance,
			WebviewIsTransparent: true,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			Theme: func() windows.Theme {
				if err == nil && globalConfig != nil && globalConfig.ThemeSettings.Mode == "dark" {
					return windows.Dark
				}
				if err == nil && globalConfig != nil && globalConfig.ThemeSettings.Mode == "system" {
					return windows.SystemDefault
				}
				return windows.Light
			}(),
		},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: false, // Windows 上与 EnableFileDrop 同为 true 会导致拖放失效
			CSSDropProperty:    "--wails-drop-target",
			CSSDropValue:       "drop",
		},
		// 绑定后端结构体到前端
		Bind: []interface{}{
			appInstance,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
