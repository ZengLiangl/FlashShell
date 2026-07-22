package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ShortcutBinding 单条快捷键绑定
type ShortcutBinding struct {
	Key      string `json:"key"`
	UseMod   bool   `json:"useMod"`
	UseShift bool   `json:"useShift,omitempty"`
}

// ShellSnippet 终端命令片段（可绑定快捷键；可选择是否直接执行）
type ShellSnippet struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Command string         `json:"command"`
	Scope   string         `json:"scope,omitempty"` // global 或机器配置名
	Binding *KeyMapBinding `json:"binding,omitempty"`
	// Execute 为 true 时发送到终端后追加换行并执行；false 仅插入文本
	Execute bool `json:"execute"`
}

// ShortcutSettings 可自定义系统快捷键（独立 JSON 文件）
type ShortcutSettings struct {
	NewWindow         ShortcutBinding `json:"newWindow"`
	MachineConfig     ShortcutBinding `json:"machineConfig"`
	ConnectionManager ShortcutBinding `json:"connectionManager"`
	EnvVars           ShortcutBinding `json:"envVars"`
	SystemSettings    ShortcutBinding `json:"systemSettings"`
	RefreshConfig     ShortcutBinding `json:"refreshConfig"`
	Find              ShortcutBinding `json:"find"`
	Copy              ShortcutBinding `json:"copy"`
	Paste             ShortcutBinding `json:"paste"`
	ClearOutput       ShortcutBinding `json:"clearOutput"`
	CommandPalette    ShortcutBinding `json:"commandPalette"`
	Snippets          []ShellSnippet  `json:"snippets,omitempty"`
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
		Paste:             ShortcutBinding{Key: "v", UseMod: true},
		ClearOutput:       ShortcutBinding{Key: "k", UseMod: true},
		CommandPalette:    ShortcutBinding{Key: "p", UseMod: true, UseShift: true},
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
	if s.Paste.Key == "" {
		s.Paste = def.Paste
	}
	if s.ClearOutput.Key == "" {
		s.ClearOutput = def.ClearOutput
	}
	if s.CommandPalette.Key == "" {
		s.CommandPalette = def.CommandPalette
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
			s := DefaultShortcutSettings()
			if migrateKeyMapsIntoSnippets(&s) {
				_ = SaveShortcutSettings(s)
			}
			return s, nil
		}
		return DefaultShortcutSettings(), err
	}
	var s ShortcutSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return DefaultShortcutSettings(), fmt.Errorf("解析快捷键配置失败: %w", err)
	}
	fillShortcutDefaults(&s)
	if migrateKeyMapsIntoSnippets(&s) {
		_ = SaveShortcutSettings(s)
	}
	return s, nil
}

// migrateKeyMapsIntoSnippets 将旧版 keymaps.json 并入命令片段（一次性）
func migrateKeyMapsIntoSnippets(s *ShortcutSettings) bool {
	if s == nil {
		return false
	}
	km, err := LoadKeyMapSettings()
	if err != nil || len(km.Entries) == 0 {
		return false
	}

	existingIDs := make(map[string]struct{}, len(s.Snippets))
	for _, sn := range s.Snippets {
		if sn.ID != "" {
			existingIDs[sn.ID] = struct{}{}
		}
	}

	changed := false
	for _, e := range km.Entries {
		id := e.ID
		if id == "" {
			id = fmt.Sprintf("km-migrated-%d", len(s.Snippets)+1)
		}
		if _, ok := existingIDs[id]; ok {
			continue
		}
		cmd := e.SendString
		execute := false
		if strings.HasSuffix(cmd, `\n`) {
			cmd = strings.TrimSuffix(cmd, `\n`)
			execute = true
		} else if strings.HasSuffix(cmd, "\n") {
			cmd = strings.TrimSuffix(cmd, "\n")
			execute = true
		}
		name := strings.TrimSpace(e.Name)
		if name == "" {
			name = "按键映射"
		}
		binding := e.Binding
		sn := ShellSnippet{
			ID:      id,
			Name:    name,
			Command: cmd,
			Scope:   "global",
			Execute: execute,
		}
		if binding.Key != "" {
			b := binding
			sn.Binding = &b
		}
		s.Snippets = append(s.Snippets, sn)
		existingIDs[id] = struct{}{}
		changed = true
	}
	if !changed {
		return false
	}
	_ = SaveKeyMapSettings(DefaultKeyMapSettings())
	return true
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
