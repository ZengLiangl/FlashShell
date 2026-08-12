//go:build windows

package app

import (
	"fmt"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// selectSystemApplicationPath Windows：文件对话框选 .exe（与 macOS 选应用逻辑隔离）
func (a *App) selectSystemApplicationPath() (string, error) {
	path, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title:            "选择应用程序",
		DefaultDirectory: `C:\Program Files`,
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "可执行文件 (*.exe)", Pattern: "*.exe"},
			{DisplayName: "所有文件", Pattern: "*"},
		},
	})
	if err != nil {
		return "", fmt.Errorf("选择应用程序失败: %w", err)
	}
	return path, nil
}
