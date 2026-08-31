package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"FlashDock/data"
)

func pendingConnectPath() (string, error) {
	dir, err := data.ConfigHomeDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "pending-connect.txt"), nil
}

// OpenMachineInNewWindow 在新窗口中打开并自动连接指定机器
func (a *App) OpenMachineInNewWindow(machineName string) error {
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	machineName = strings.TrimSpace(machineName)
	if machineName == "" {
		return fmt.Errorf("机器名为空")
	}
	path, err := pendingConnectPath()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(machineName), 0o600); err != nil {
		return err
	}
	execPath, err := os.Executable()
	if err != nil {
		return err
	}
	sessionID := data.NewSessionID()
	cmd := exec.Command(execPath, "-session="+sessionID)
	return cmd.Start()
}

// ConsumePendingConnectMachine 读取并清除新窗口待连接机器名
func (a *App) ConsumePendingConnectMachine() string {
	path, err := pendingConnectPath()
	if err != nil {
		return ""
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	_ = os.Remove(path)
	return strings.TrimSpace(string(raw))
}
