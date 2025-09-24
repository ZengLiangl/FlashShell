package main

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"quick-cmd/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

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

// NewWindow 创建新窗口（通过启动新进程实现）
func NewWindow() {
	// 获取当前程序的路径
	execPath, err := os.Executable()
	if err != nil {
		println("启动新窗口失败:", err.Error())
		return
	}
	// 启动新的进程
	cmd := exec.Command(execPath)
	if err := cmd.Start(); err != nil {
		println("启动新窗口失败:", err.Error())
	}
}

// OpenCurrentConfig 打开当前配置文件（供菜单调用）
func OpenCurrentConfig(lastOpenedFile string) {

	if lastOpenedFile == "" {
		fmt.Println("没有找到当前配置文件")
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(lastOpenedFile); os.IsNotExist(err) {
		fmt.Printf("配置文件不存在: %s\n", lastOpenedFile)
		return
	}

	// 使用系统默认程序打开配置文件
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("open", lastOpenedFile)
	case "windows": // Windows
		cmd = exec.Command("cmd", "/c", "start", "", lastOpenedFile)
	case "linux": // Linux
		cmd = exec.Command("xdg-open", lastOpenedFile)
	default:
		fmt.Printf("不支持的操作系统: %s\n", runtime.GOOS)
		return
	}

	err := cmd.Run()
	if err != nil {
		fmt.Printf("打开配置文件失败: %v\n", err)
	} else {
		fmt.Printf("成功打开配置文件: %s\n", lastOpenedFile)
	}
}

// createMenu 创建应用菜单
func createMenu(appInstance *app.App) *menu.Menu {

	appMenu := menu.NewMenu()

	// 文件菜单
	fileMenu := appMenu.AddSubmenu("文件")
	fileMenu.AddText("新建窗口", keys.CmdOrCtrl("n"), func(_ *menu.CallbackData) {
		// 新建窗口
		NewWindow()
	})
	fileMenu.AddText("打开当前配置", nil, func(_ *menu.CallbackData) {
		// 打开当前配置
		newVar, _ := appInstance.GetGlobalConfig()
		OpenCurrentConfig(newVar.LastOpenedFile)
	})

	fileMenu.AddSeparator()
	// 添加退出菜单
	fileMenu.AddSeparator()
	// fileMenu.AddText("退出", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
	// 	// 退出应用
	// 	runtime.Quit(appInstance.GetCtx())
	// })

	// 添加机器配置菜单
	configMenu := appMenu.AddSubmenu("设置")

	// 配置菜单
	configFileMenu := appMenu.AddSubmenu("配置文件")
	// 动态加载配置文件列表
	configFiles, err := appInstance.GetConfigFiles()
	if err != nil {
		// 如果获取失败，添加默认项
		configFileMenu.AddText("无法加载配置文件", keys.CmdOrCtrl("r"), func(_ *menu.CallbackData) {
			appInstance.RefreshConfigMenu()
		})
	} else {
		// 获取当前配置文件
		globalConfig, _ := appInstance.GetGlobalConfig()
		currentConfig := ""
		if globalConfig != nil {
			currentConfig = globalConfig.LastOpenedFile
		}

		// 为每个配置文件添加菜单项
		for _, configFile := range configFiles {
			// 获取文件名（去掉路径）
			fileName := getFileName(configFile)
			// 创建菜单项
			_ = configFileMenu.AddRadio(fileName, configFile == currentConfig, nil, func(data *menu.CallbackData) {
				// 切换配置文件
				switchConfigFile(appInstance, configFile)
			})
		}

		// 添加分隔符和刷新选项
		configFileMenu.AddSeparator()
		configFileMenu.AddText("刷新配置列表", keys.CmdOrCtrl("r"), func(_ *menu.CallbackData) {
			appInstance.RefreshConfigMenu()
		})

	}
	configMenu.AddText("机器配置", keys.CmdOrCtrl("m"), func(_ *menu.CallbackData) {
		// 打开机器配置对话框
		appInstance.OpenMachineConfig()
	})

	configMenu.AddText("环境变量", keys.CmdOrCtrl("e"), func(_ *menu.CallbackData) {
		// 打开环境变量配置对话框
		appInstance.OpenWorkPathConfig()
	})

	// 帮助菜单
	helpMenu := appMenu.AddSubmenu("帮助")
	helpMenu.AddText("关于", nil, func(_ *menu.CallbackData) {
		// 显示关于信息
	})

	return appMenu
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
