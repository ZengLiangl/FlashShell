package app

import "FlashDock/data"

// GetShellSessionRestore 读取待恢复的 Shell 标签页
func (a *App) GetShellSessionRestore() ([]data.ShellSessionRestoreTab, error) {
	return data.LoadShellSessionRestore()
}

// SaveShellSessionRestore 保存待恢复的 Shell 标签页
func (a *App) SaveShellSessionRestore(tabs []data.ShellSessionRestoreTab) error {
	return data.SaveShellSessionRestore(tabs)
}
