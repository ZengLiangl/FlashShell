package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ShortcutBinding 单条快捷键绑定
type ShortcutBinding struct {
	Key    string `json:"key"`
	UseMod bool   `json:"useMod"`
}

// ShortcutSettings 可自定义系统快捷键（独立 JSON 文件）
type ShortcutSettings struct {
	NewWindow          ShortcutBinding `json:"newWindow"`
	MachineConfig      ShortcutBinding `json:"machineConfig"`
	ConnectionManager  ShortcutBinding `json:"connectionManager"`
	EnvVars            ShortcutBinding `json:"envVars"`
	SystemSettings     ShortcutBinding `json:"systemSettings"`
	RefreshConfig      ShortcutBinding `json:"refreshConfig"`
	Find               ShortcutBinding `json:"find"`
	Copy               ShortcutBinding `json:"copy"`
	ClearOutput        ShortcutBinding `json:"clearOutput"`
}

// DefaultShortcutSettings 默认快捷键
func DefaultShortcutSettings() ShortcutSettings {
	return ShortcutSettings{
		NewWindow:         ShortcutBinding{Key: "n", UseMod: true},
		MachineConfig:     ShortcutBinding{Key: "m", UseMod: true},
		ConnectionManager: ShortcutBinding{Key: "e", UseMod: true},
		EnvVars:           ShortcutBinding{Key: "u", UseMod: true},
		SystemSettings:    ShortcutBinding{Key: ",", UseMod: true},
		RefreshConfig:     ShortcutBinding{Key: "r", UseMod: true},
		Find:              ShortcutBinding{Key: "f", UseMod: true},
		Copy:              ShortcutBinding{Key: "c", UseMod: true},
		ClearOutput:       ShortcutBinding{Key: "k", UseMod: true},
	}
}

func shortcutSettingsPath() (string, error) {
	configHome, err := ConfigHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}
	return filepath.Join(configHome, "shortcuts.json"), nil
}

func fillShortcutDefaults(s *ShortcutSettings) {
	def := DefaultShortcutSettings()
	if s.NewWindow.Key == "" {
		s.NewWindow = def.NewWindow
	}
	if s.MachineConfig.Key == "" {
		s.MachineConfig = def.MachineConfig
	}
	// 旧版默认 Mod+E 绑定环境变量；升级后改为连接管理器，环境变量迁到 Mod+U
	if s.ConnectionManager.Key == "" {
		s.ConnectionManager = def.ConnectionManager
		if s.EnvVars.Key == "" || s.EnvVars.Key == "e" {
			s.EnvVars = def.EnvVars
		}
	} else if s.EnvVars.Key == "" {
		s.EnvVars = def.EnvVars
	}
	if s.SystemSettings.Key == "" {
		s.SystemSettings = def.SystemSettings
	}
	if s.RefreshConfig.Key == "" {
		s.RefreshConfig = def.RefreshConfig
	}
	if s.Find.Key == "" {
		s.Find = def.Find
	}
	if s.Copy.Key == "" {
		s.Copy = def.Copy
	}
	if s.ClearOutput.Key == "" {
		s.ClearOutput = def.ClearOutput
	}
}

// LoadShortcutSettings 从 ~/.flashdock/shortcuts.json 加载
func LoadShortcutSettings() (ShortcutSettings, error) {
	path, err := shortcutSettingsPath()
	if err != nil {
		return DefaultShortcutSettings(), err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultShortcutSettings(), nil
		}
		return DefaultShortcutSettings(), err
	}
	var s ShortcutSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return DefaultShortcutSettings(), fmt.Errorf("解析快捷键配置失败: %w", err)
	}
	fillShortcutDefaults(&s)
	return s, nil
}

// SaveShortcutSettings 保存到 ~/.flashdock/shortcuts.json
func SaveShortcutSettings(s ShortcutSettings) error {
	fillShortcutDefaults(&s)
	path, err := shortcutSettingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化快捷键配置失败: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("保存快捷键配置失败: %w", err)
	}
	return nil
}
