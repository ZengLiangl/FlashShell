//go:build !darwin && !windows

package app

import (
	"fmt"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// selectSystemApplicationPath Linux 等：文件对话框任选可执行文件
func (a *App) selectSystemApplicationPath() (string, error) {
	path, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title:            "选择应用程序",
		DefaultDirectory: "/usr/bin",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "所有文件", Pattern: "*"},
		},
	})
	if err != nil {
		return "", fmt.Errorf("选择应用程序失败: %w", err)
	}
	return path, nil
}
