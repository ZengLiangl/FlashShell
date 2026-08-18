package data

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	// ConfigHomeDirName 全局数据目录名（位于用户主目录下）
	ConfigHomeDirName = ".flashshell"
	// LegacyConfigHomeDirName 上一版目录 ~/.flashdock，启动时复制到新目录后删除
	LegacyConfigHomeDirName = ".flashdock"
	// OldestConfigHomeDirName 更早的 ~/.cmd-config
	OldestConfigHomeDirName = ".cmd-config"
	// DefaultConfigFileName 默认业务配置文件名
	DefaultConfigFileName = "config.yaml"
	// ConfigHomeEnv 测试或特殊部署可设置此环境变量，覆盖默认 ~/.flashshell。
	// 设置后不会读写真实用户目录，也不会触发旧目录迁移。
	ConfigHomeEnv = "FLASHSHELL_CONFIG_HOME"
	// LegacyConfigHomeEnv 兼容旧测试与脚本里的 FLASHDOCK_CONFIG_HOME
	LegacyConfigHomeEnv = "FLASHDOCK_CONFIG_HOME"
)

// DefaultConfigPath 返回 ~/.flashshell/config.yaml（与 global_config.yaml 同目录）
func DefaultConfigPath() string {
	configHome, err := ConfigHomeDir()
	if err != nil {
		return DefaultConfigFileName
	}
	return filepath.Join(configHome, DefaultConfigFileName)
}

var migrateConfigHomeOnce sync.Once

func configHomeOverride() string {
	if v := strings.TrimSpace(os.Getenv(ConfigHomeEnv)); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv(LegacyConfigHomeEnv))
}

// ConfigHomeDir 返回配置数据目录。
// 优先使用 FLASHSHELL_CONFIG_HOME（或旧名 FLASHDOCK_CONFIG_HOME）；
// 否则为 ~/.flashshell，并在需要时从 ~/.flashdock / ~/.cmd-config 一次性迁移。
func ConfigHomeDir() (string, error) {
	if override := configHomeOverride(); override != "" {
		if err := os.MkdirAll(override, 0755); err != nil {
			return "", err
		}
		return override, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	newDir := filepath.Join(homeDir, ConfigHomeDirName)
	flashdockDir := filepath.Join(homeDir, LegacyConfigHomeDirName)
	cmdConfigDir := filepath.Join(homeDir, OldestConfigHomeDirName)
	migrateConfigHomeOnce.Do(func() {
		migrateConfigHome(flashdockDir, newDir)
		if _, err := os.Stat(newDir); err != nil {
			migrateConfigHome(cmdConfigDir, newDir)
		}
	})
	return newDir, nil
}

func migrateConfigHome(oldDir, newDir string) {
	oldInfo, err := os.Stat(oldDir)
	if err != nil || !oldInfo.IsDir() {
		return
	}
	if _, err := os.Stat(newDir); err == nil {
		_ = os.RemoveAll(oldDir)
		return
	}
	if err := copyDir(oldDir, newDir); err != nil {
		_ = os.RemoveAll(newDir)
		return
	}
	_ = os.RemoveAll(oldDir)
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		return copyFile(path, dest)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
