package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// SessionState 窗口会话状态（各窗口独立）
type SessionState struct {
	SessionID      string `json:"sessionId"`
	LastOpenedFile string `json:"lastOpenedFile"`
	Theme          string `json:"theme"`
	TerminalPreset string `json:"terminalPreset"`
}

// SessionManager 多窗口会话管理
type SessionManager struct {
	baseDir string
	state   *SessionState
}

// NewSessionManager 创建会话管理器
func NewSessionManager(sessionID string) (*SessionManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("获取用户主目录失败: %w", err)
	}

	baseDir := filepath.Join(homeDir, ".cmd-config", "sessions")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}

	if sessionID == "" {
		sessionID = uuid.NewString()
	}

	sm := &SessionManager{
		baseDir: baseDir,
		state: &SessionState{
			SessionID: sessionID,
		},
	}

	if err := sm.load(); err != nil {
		return nil, err
	}
	return sm, nil
}

// GetSessionID 返回当前会话 ID
func (sm *SessionManager) GetSessionID() string {
	return sm.state.SessionID
}

// GetState 返回会话状态副本
func (sm *SessionManager) GetState() SessionState {
	return *sm.state
}

// GetLastOpenedFile 获取会话级最后打开的配置文件
func (sm *SessionManager) GetLastOpenedFile() string {
	return sm.state.LastOpenedFile
}

// SetLastOpenedFile 设置会话级最后打开的配置文件
func (sm *SessionManager) SetLastOpenedFile(path string) error {
	if sm.state.LastOpenedFile == path {
		return nil
	}
	sm.state.LastOpenedFile = path
	return sm.save()
}

// SetTheme 设置主题
func (sm *SessionManager) SetTheme(theme, terminalPreset string) error {
	sm.state.Theme = theme
	sm.state.TerminalPreset = terminalPreset
	return sm.save()
}

func (sm *SessionManager) sessionPath() string {
	return filepath.Join(sm.baseDir, sm.state.SessionID+".json")
}

func (sm *SessionManager) load() error {
	data, err := os.ReadFile(sm.sessionPath())
	if err != nil {
		if os.IsNotExist(err) {
			return sm.save()
		}
		return err
	}

	var state SessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}
	state.SessionID = sm.state.SessionID
	sm.state = &state
	return nil
}

func (sm *SessionManager) save() error {
	data, err := json.MarshalIndent(sm.state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sm.sessionPath(), data, 0644)
}

// NewSessionID 生成新会话 ID
func NewSessionID() string {
	return uuid.NewString()
}
