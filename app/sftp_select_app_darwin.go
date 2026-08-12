//go:build darwin

package app

import (
	"fmt"
	"os/exec"
	"strings"
)

// selectSystemApplicationPath macOS：系统「选择应用程序」面板。
// 不用 Wails OpenFileDialog + *.app：UTType 对不上 application-bundle，应用会全部置灰。
func (a *App) selectSystemApplicationPath() (string, error) {
	cmd := exec.Command("osascript", "-e", `try
	POSIX path of (choose application as alias)
on error number -128
	return ""
end try`)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("选择应用程序失败: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
