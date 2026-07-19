package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"FlashDock/data"
	"FlashDock/define"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ExportMachineTemplate 导出连接模板（密码脱敏）
func (a *App) ExportMachineTemplate() (string, error) {
	machines := a.GetMachines()
	groups := a.GetMachineGroups()
	raw, err := data.ExportMachineTemplate(machines, groups)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// ExportMachineTemplateToFile 导出模板到用户选择的路径
func (a *App) ExportMachineTemplateToFile() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("应用未就绪")
	}
	path, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "导出连接模板",
		DefaultFilename: "flashdock-machines.json",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "JSON", Pattern: "*.json"},
		},
	})
	if err != nil || path == "" {
		return "", err
	}
	content, err := a.ExportMachineTemplate()
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}
	return path, nil
}

// ImportMachineTemplateFromFile 从文件导入连接模板
func (a *App) ImportMachineTemplateFromFile(merge bool) (data.ImportMachineTemplateResult, error) {
	if a.ctx == nil {
		return data.ImportMachineTemplateResult{}, fmt.Errorf("应用未就绪")
	}
	path, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "导入连接模板",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "JSON", Pattern: "*.json"},
		},
	})
	if err != nil || path == "" {
		return data.ImportMachineTemplateResult{}, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return data.ImportMachineTemplateResult{}, fmt.Errorf("读取文件失败: %w", err)
	}
	return a.ImportMachineTemplate(string(raw), merge)
}

// ImportMachineTemplate 导入连接模板 JSON
func (a *App) ImportMachineTemplate(jsonData string, merge bool) (data.ImportMachineTemplateResult, error) {
	existing := a.configManager.GetAllMachinesFromGlobal()
	updated, result, err := data.ImportMachineTemplate([]byte(jsonData), merge, existing)
	if err != nil {
		return result, err
	}
	if err := a.configManager.SaveGlobalConfigMachines(updated); err != nil {
		return result, fmt.Errorf("保存机器配置失败: %w", err)
	}
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "config:changed", nil)
	}
	return result, nil
}

// ListLocalFiles 列出本地目录
func (a *App) ListLocalFiles(dirPath string, showHidden bool) ([]define.LocalFileEntry, error) {
	dirPath = strings.TrimSpace(dirPath)
	if dirPath == "" || dirPath == "~" {
		var err error
		dirPath, err = os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("无法获取用户目录: %w", err)
		}
	}
	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, fmt.Errorf("路径不存在: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("不是目录")
	}
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	out := make([]define.LocalFileEntry, 0, len(entries))
	for _, ent := range entries {
		name := ent.Name()
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(dirPath, name)
		fi, err := ent.Info()
		if err != nil {
			continue
		}
		mode := fi.Mode()
		entry := define.LocalFileEntry{
			Name:    name,
			Path:    full,
			IsDir:   fi.IsDir(),
			Size:    fi.Size(),
			ModTime: fi.ModTime().Unix(),
			Mode:    mode.String(),
		}
		if fi.IsDir() {
			entry.Type = "目录"
		} else {
			entry.Type = "文件"
		}
		out = append(out, entry)
	}
	return out, nil
}

// GetLocalHomeDir 返回本机用户主目录
func (a *App) GetLocalHomeDir() (string, error) {
	return os.UserHomeDir()
}
