package data

import (
	"os"
	"path/filepath"
	"sync"
)

const (
	// ConfigHomeDirName 全局数据目录名（位于用户主目录下）
	ConfigHomeDirName = ".flashdock"
	// LegacyConfigHomeDirName 旧版目录，启动时若新目录不存在则自动迁移
	LegacyConfigHomeDirName = ".cmd-config"
	// DefaultConfigFileName 默认业务配置文件名
	DefaultConfigFileName = "config.yaml"
	// DefaultLogPathTilde 默认日志目录（带 ~）
	DefaultLogPathTilde = "~/.flashdock/logs"
)

// DefaultConfigPath 返回 ~/.flashdock/config.yaml（与 global_config.yaml 同目录）
func DefaultConfigPath() string {
	configHome, err := ConfigHomeDir()
	if err != nil {
		return DefaultConfigFileName
	}
	return filepath.Join(configHome, DefaultConfigFileName)
}

var migrateConfigHomeOnce sync.Once

// ConfigHomeDir 返回 ~/.flashdock，并在需要时从 ~/.cmd-config 一次性迁移。
func ConfigHomeDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	newDir := filepath.Join(homeDir, ConfigHomeDirName)
	oldDir := filepath.Join(homeDir, LegacyConfigHomeDirName)
	migrateConfigHomeOnce.Do(func() {
		migrateConfigHome(oldDir, newDir)
	})
	return newDir, nil
}

func migrateConfigHome(oldDir, newDir string) {
	if _, err := os.Stat(newDir); err == nil {
		return
	}
	if _, err := os.Stat(oldDir); err != nil {
		return
	}
	_ = os.Rename(oldDir, newDir)
}
