package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const shellSessionRestoreFile = "shell_session_restore.json"

// ShellSessionRestoreTab 待恢复的 Shell 标签页
type ShellSessionRestoreTab struct {
	SessionID  string `json:"sessionId"`
	ConfigName string `json:"configName"`
	Kind       string `json:"kind"` // local | remote
	TabLabel   string `json:"tabLabel"`
	LastCwd    string `json:"lastCwd,omitempty"`
}

// ShellSessionRestoreState 窗口级 Shell 会话恢复状态
type ShellSessionRestoreState struct {
	Tabs []ShellSessionRestoreTab `json:"tabs"`
}

func shellSessionRestorePath() (string, error) {
	home, err := ConfigHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, shellSessionRestoreFile), nil
}

// LoadShellSessionRestore 读取待恢复标签页
func LoadShellSessionRestore() ([]ShellSessionRestoreTab, error) {
	path, err := shellSessionRestorePath()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var state ShellSessionRestoreState
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, fmt.Errorf("解析 %s 失败: %w", shellSessionRestoreFile, err)
	}
	out := make([]ShellSessionRestoreTab, 0, len(state.Tabs))
	for _, tab := range state.Tabs {
		tab.SessionID = strings.TrimSpace(tab.SessionID)
		tab.ConfigName = strings.TrimSpace(tab.ConfigName)
		tab.Kind = strings.TrimSpace(tab.Kind)
		tab.TabLabel = strings.TrimSpace(tab.TabLabel)
		tab.LastCwd = strings.TrimSpace(tab.LastCwd)
		if tab.Kind == "" {
			if tab.SessionID == "local" || strings.HasPrefix(tab.SessionID, "local-") {
				tab.Kind = "local"
			} else {
				tab.Kind = "remote"
			}
		}
		if tab.Kind == "local" {
			if tab.SessionID == "" {
				tab.SessionID = "local"
			}
			if tab.ConfigName == "" {
				tab.ConfigName = tab.SessionID
			}
		} else if tab.ConfigName == "" && tab.SessionID != "" {
			tab.ConfigName = tab.SessionID
		}
		if tab.SessionID == "" && tab.ConfigName == "" {
			continue
		}
		out = append(out, tab)
	}
	return out, nil
}

// SaveShellSessionRestore 保存待恢复标签页
func SaveShellSessionRestore(tabs []ShellSessionRestoreTab) error {
	path, err := shellSessionRestorePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	clean := make([]ShellSessionRestoreTab, 0, len(tabs))
	for _, tab := range tabs {
		tab.SessionID = strings.TrimSpace(tab.SessionID)
		tab.ConfigName = strings.TrimSpace(tab.ConfigName)
		tab.Kind = strings.TrimSpace(tab.Kind)
		tab.TabLabel = strings.TrimSpace(tab.TabLabel)
		tab.LastCwd = strings.TrimSpace(tab.LastCwd)
		if tab.SessionID == "" && tab.ConfigName == "" {
			continue
		}
		clean = append(clean, tab)
	}
	state := ShellSessionRestoreState{Tabs: clean}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0644)
}
