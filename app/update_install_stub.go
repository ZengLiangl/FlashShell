//go:build !windows

package app

import "fmt"

func resolveWindowsUpdateTarget() (string, error) {
	return "", fmt.Errorf("仅 Windows 支持解析更新目标路径")
}

func windowsUpdateScriptEnv(_, _, _, _ string, _ int) []string {
	return nil
}
