package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// KeyMapBinding 按键映射绑定（支持 Ctrl/Cmd、Alt、Shift）
type KeyMapBinding struct {
	Key      string `json:"key"`
	UseMod   bool   `json:"useMod"`
	UseAlt   bool   `json:"useAlt,omitempty"`
	UseShift bool   `json:"useShift,omitempty"`
}

// KeyMapEntry 单条按键映射（组合键 → 向终端发送字符串）
type KeyMapEntry struct {
	ID         string        `json:"id"`
	Enabled    bool          `json:"enabled"`
	Name       string        `json:"name,omitempty"`
	Binding    KeyMapBinding `json:"binding"`
	Action     string        `json:"action"` // 目前仅 sendString
	SendString string        `json:"sendString"`
}

// KeyMapSettings 按键映射配置（独立 JSON 文件）
type KeyMapSettings struct {
	Entries []KeyMapEntry `json:"entries"`
}

// DefaultKeyMapSettings 默认无映射
func DefaultKeyMapSettings() KeyMapSettings {
	return KeyMapSettings{Entries: []KeyMapEntry{}}
}

func keyMapSettingsPath() (string, error) {
	configHome, err := ConfigHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}
	return filepath.Join(configHome, "keymaps.json"), nil
}

func normalizeKeyMapSettings(s *KeyMapSettings) {
	if s.Entries == nil {
		s.Entries = []KeyMapEntry{}
	}
	for i := range s.Entries {
		e := &s.Entries[i]
		if e.ID == "" {
			e.ID = fmt.Sprintf("km-%d", i+1)
		}
		if e.Action == "" {
			e.Action = "sendString"
		}
		e.Binding.Key = strings.TrimSpace(e.Binding.Key)
	}
}

// LoadKeyMapSettings 从 ~/.flashshell/keymaps.json 加载
func LoadKeyMapSettings() (KeyMapSettings, error) {
	path, err := keyMapSettingsPath()
	if err != nil {
		return DefaultKeyMapSettings(), err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultKeyMapSettings(), nil
		}
		return DefaultKeyMapSettings(), err
	}
	var s KeyMapSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return DefaultKeyMapSettings(), fmt.Errorf("解析按键映射配置失败: %w", err)
	}
	normalizeKeyMapSettings(&s)
	return s, nil
}

// SaveKeyMapSettings 保存到 ~/.flashshell/keymaps.json
func SaveKeyMapSettings(s KeyMapSettings) error {
	normalizeKeyMapSettings(&s)
	path, err := keyMapSettingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化按键映射配置失败: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("保存按键映射配置失败: %w", err)
	}
	return nil
}
