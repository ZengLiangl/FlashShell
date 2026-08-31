package app

import (
	"fmt"
	"os"

	"FlashDock/data"
	"FlashDock/define"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// GetMachineGroupDefaults 获取全部分组默认配置
func (a *App) GetMachineGroupDefaults() []data.MachineGroupDefaults {
	return a.configManager.GetMachineGroupDefaults()
}

// SaveMachineGroupDefaults 保存分组默认配置
func (a *App) SaveMachineGroupDefaults(def data.MachineGroupDefaults) error {
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	return a.configManager.SaveMachineGroupDefaults(def)
}

// ImportOpenSSHConfigPick 选择并导入 OpenSSH config
func (a *App) ImportOpenSSHConfigPick(accountID, group string) (*data.OpenSSHImportResult, error) {
	if err := a.requireUnlocked(); err != nil {
		return nil, err
	}
	path, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择 OpenSSH config",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "OpenSSH config", Pattern: "config"},
			{DisplayName: "所有文件", Pattern: "*"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("选择文件失败: %w", err)
	}
	if path == "" {
		return nil, nil
	}
	return a.configManager.ImportOpenSSHConfig(path, accountID, group)
}

// ImportOpenSSHConfigDefault 从默认 ~/.ssh/config 导入机器
func (a *App) ImportOpenSSHConfigDefault(accountID, group string) (*data.OpenSSHImportResult, error) {
	if err := a.requireUnlocked(); err != nil {
		return nil, err
	}
	path, err := data.DefaultOpenSSHConfigPath()
	if err != nil {
		return nil, fmt.Errorf("获取默认 SSH 配置路径失败: %w", err)
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("未找到本地 SSH 配置：%s", path)
		}
		return nil, fmt.Errorf("读取本地 SSH 配置失败: %w", err)
	}
	return a.configManager.ImportOpenSSHConfig(path, accountID, group)
}

// ImportMachinesCSVPick 选择并导入 CSV 机器列表
func (a *App) ImportMachinesCSVPick() (*data.MachineImportResult, error) {
	if err := a.requireUnlocked(); err != nil {
		return nil, err
	}
	path, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择 CSV 文件",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "CSV (*.csv)", Pattern: "*.csv"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("选择文件失败: %w", err)
	}
	if path == "" {
		return nil, nil
	}
	return a.configManager.ImportMachinesCSV(path)
}

// ExportMachinesCSVPick 选择路径并导出机器 CSV
func (a *App) ExportMachinesCSVPick() (string, error) {
	path, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "导出机器列表 CSV",
		DefaultFilename: "flashdock-machines.csv",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "CSV (*.csv)", Pattern: "*.csv"},
		},
	})
	if err != nil {
		return "", fmt.Errorf("选择保存路径失败: %w", err)
	}
	if path == "" {
		return "", nil
	}
	if err := a.configManager.ExportMachinesCSV(path); err != nil {
		return "", err
	}
	return path, nil
}

// ImportPuttyPick 选择并导入 PuTTY 注册表导出（*.reg）
func (a *App) ImportPuttyPick(accountID, group string) (*data.MachineImportResult, error) {
	if err := a.requireUnlocked(); err != nil {
		return nil, err
	}
	paths, err := a.pickImportSources("选择 PuTTY 注册表文件", []wailsRuntime.FileFilter{
		{DisplayName: "PuTTY 注册表 (*.reg)", Pattern: "*.reg"},
	})
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, nil
	}
	return a.configManager.ImportPutty(paths, accountID, group)
}

// ImportMobaXtermPick 选择并导入 MobaXterm 会话文件
func (a *App) ImportMobaXtermPick(accountID, group string) (*data.MachineImportResult, error) {
	if err := a.requireUnlocked(); err != nil {
		return nil, err
	}
	paths, err := a.pickImportSources("选择 MobaXterm 会话文件", []wailsRuntime.FileFilter{
		{DisplayName: "MobaXterm (*.mxtsessions)", Pattern: "*.mxtsessions"},
		{DisplayName: "INI (*.ini)", Pattern: "*.ini"},
		{DisplayName: "所有文件", Pattern: "*"},
	})
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, nil
	}
	return a.configManager.ImportMobaXterm(paths, accountID, group)
}

// ImportSecureCRTPick 选择 SecureCRT Sessions 文件夹并导入
func (a *App) ImportSecureCRTPick(accountID, group string) (*data.MachineImportResult, error) {
	if err := a.requireUnlocked(); err != nil {
		return nil, err
	}
	dirPath, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择 SecureCRT Sessions 文件夹",
	})
	if err != nil {
		return nil, fmt.Errorf("选择文件夹失败: %w", err)
	}
	if dirPath == "" {
		return nil, nil
	}
	return a.configManager.ImportSecureCRT([]string{dirPath}, accountID, group)
}

func (a *App) machineForConnect(machine *define.Machine) (*define.Machine, error) {
	if err := a.requireUnlocked(); err != nil {
		return nil, err
	}
	return a.configManager.MachineForConnect(machine)
}
