package main

import (
	"embed"

	"quick-cmd/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// main is the entry point for the application
func main() {
	// Create an instance of the app structure
	appInstance := app.NewApp()
	// 创建菜单
	appMenu := appInstance.CreateApplicationMenu()

	// 获取窗口名称
	globalConfig, err := appInstance.GetGlobalConfig()
	windowsName := "Quick Cmd" // 默认窗口名称
	if err == nil && globalConfig != nil {
		windowsName = globalConfig.WindowsName
	}

	// Create application with options
	err = wails.Run(&options.App{
		Title:  windowsName,
		Width:  1200,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        appInstance.Startup,
		OnDomReady:       appInstance.DomReady,
		OnBeforeClose:    appInstance.BeforeClose,
		OnShutdown:       appInstance.Shutdown,
		Fullscreen:       false,
		MinWidth:         1200,
		MinHeight:        768,
		Menu:             appMenu,
		// 绑定后端结构体到前端
		Bind: []interface{}{
			appInstance,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
