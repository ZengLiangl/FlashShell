package main

import (
	"embed"

	"quick-cmd/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// createMenu 创建应用菜单
func createMenu(appInstance *app.App) *menu.Menu {
	appMenu := menu.NewMenu()

	// 文件菜单
	fileMenu := appMenu.AddSubmenu("文件")

	// 配置菜单
	configMenu := fileMenu.AddSubmenu("配置文件")

	// 添加默认配置文件选项
	configMenu.AddText("config.yaml", nil, func(data *menu.CallbackData) {
		switchConfigFile(appInstance, "config.yaml")
	})

	configMenu.AddText("xyj.yaml", nil, func(data *menu.CallbackData) {
		switchConfigFile(appInstance, "xyj.yaml")
	})

	// 添加分隔符和刷新选项
	configMenu.AddSeparator()
	configMenu.AddText("刷新配置列表", keys.CmdOrCtrl("r"), func(_ *menu.CallbackData) {
		refreshConfigMenu(configMenu, appInstance)
	})

	// 添加退出菜单
	fileMenu.AddSeparator()
	fileMenu.AddText("退出", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		// 退出应用
	})

	// 帮助菜单
	helpMenu := appMenu.AddSubmenu("帮助")
	helpMenu.AddText("关于", nil, func(_ *menu.CallbackData) {
		// 显示关于信息
	})

	return appMenu
}

// getFileName 获取文件名（去掉路径）
func getFileName(filePath string) string {
	if filePath == "" {
		return ""
	}
	// 简单的路径分割，支持 Unix 和 Windows 路径
	for i := len(filePath) - 1; i >= 0; i-- {
		if filePath[i] == '/' || filePath[i] == '\\' {
			return filePath[i+1:]
		}
	}
	return filePath
}

// switchConfigFile 切换配置文件
func switchConfigFile(appInstance *app.App, configFile string) {
	err := appInstance.SwitchConfigFile(configFile)
	if err != nil {
		// 这里可以显示错误对话框，但为了简化，我们只打印错误
		println("切换配置文件失败:", err.Error())
	} else {
		println("成功切换到配置文件:", configFile)
		// 配置文件切换成功后，前端会通过事件监听自动刷新
		// 这里不需要额外的操作，因为 SwitchConfigFile 已经发送了事件
	}
}

// refreshConfigMenu 刷新配置菜单
func refreshConfigMenu(configMenu *menu.Menu, appInstance *app.App) {
	// 清空现有菜单项
	// 注意：Wails v2 的菜单 API 可能不支持动态修改
	// 这里只是示例，实际可能需要重新创建整个菜单
	println("刷新配置菜单")
}

// main is the entry point for the application
func main() {
	// Create an instance of the app structure
	appInstance := app.NewApp()

	// 创建菜单
	appMenu := createMenu(appInstance)

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "Quick Cmd",
		Width:  1024,
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
		MinWidth:         800,
		MinHeight:        600,
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
