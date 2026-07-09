package main

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	_ "syscall"

	"quick-cmd/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// main is the entry point for the application
func main() {
	// parse run mode from command line
	runMode := flag.String("reg", "desk", "运行模式: desk(前台)/back(后台)")
	sessionID := flag.String("session", "", "窗口会话 ID")
	flag.Parse()

	// daemonize if requested (like nohup), only on non-Windows
	if *runMode == "back" && os.Getenv("QUICKCMD_DAEMONIZED") != "1" {
		exePath, err := os.Executable()
		if err == nil {
			childArgs := os.Args[1:]
			for i, a := range childArgs {
				if strings.HasPrefix(a, "-reg=") {
					childArgs[i] = "-reg=desk"
				}
			}

			stdoutFile, _ := os.OpenFile("/tmp/quick-cmd.out", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			stderrFile, _ := os.OpenFile("/tmp/quick-cmd.err", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

			cmd := exec.Command(exePath, childArgs...)
			cmd.Env = append(os.Environ(), "QUICKCMD_DAEMONIZED=1")
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
	windowsName := "Quick Cmd" // 默认窗口名称
	if err == nil && globalConfig != nil {
		windowsName = globalConfig.WindowsName
	}
	if *sessionID != "" && len(*sessionID) >= 8 {
		windowsName = fmt.Sprintf("%s [%s]", windowsName, (*sessionID)[:8])
	}

	// Create application with options
	err = wails.Run(&options.App{
		Title:  windowsName,
		Width:  1200,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnStartup:        appInstance.Startup,
		OnDomReady:       appInstance.DomReady,
		OnBeforeClose:    appInstance.BeforeClose,
		OnShutdown:       appInstance.Shutdown,
		Fullscreen:       false,
		MinWidth:         1200,
		MinHeight:        768,
		Menu:             nil,
		// 绑定后端结构体到前端
		Bind: []interface{}{
			appInstance,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
