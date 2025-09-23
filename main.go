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
	app := app.NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "Quick Cmd",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.Startup,
		OnDomReady:       app.DomReady,
		OnBeforeClose:    app.BeforeClose,
		OnShutdown:       app.Shutdown,
		Fullscreen:       false,
		MinWidth:         800,
		MinHeight:        600,
		// 绑定后端结构体到前端
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
