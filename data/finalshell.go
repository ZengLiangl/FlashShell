package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"FlashDock/define"

	"github.com/google/uuid"
)

const defaultFinalShellPassword = "123456"

type finalShellConfig struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	UserName string `json:"user_name"`
}

// ParseFinalShellFile 解析 FinalShell 连接配置文件
func ParseFinalShellFile(path string) (*finalShellConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg finalShellConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}
	if cfg.Name == "" {
		base := filepath.Base(path)
		cfg.Name = strings.TrimSuffix(base, "_connect_config.json")
	}
	if cfg.Host == "" {
		return nil, fmt.Errorf("未找到 host")
	}
	if cfg.Port <= 0 {
		cfg.Port = 22
	}
	return &cfg, nil
}

// ImportFinalShellFiles 批量导入 FinalShell 配置
func (gcm *GlobalConfigManager) ImportFinalShellFiles(paths []string, accountID string) (*MachineImportResult, error) {
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return nil, err
		}
	}

	files, err := CollectFilesFromPaths(paths, isFinalShellFile)
	if err != nil {
		return nil, err
	}

	result := &MachineImportResult{}
	accountUser, accountPassword := "", defaultFinalShellPassword
	if accountID != "" {
		user, password, err := gcm.GetGlobalAccountCredentials(accountID)
		if err != nil {
			return nil, err
		}
		accountUser, accountPassword = user, password
	}

	for _, path := range files {
		session, err := ParseFinalShellFile(path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", filepath.Base(path), err))
			result.Skipped++
			continue
		}

		user := session.UserName
		password := defaultFinalShellPassword
		if accountUser != "" {
			user = accountUser
			password = accountPassword
		}

		machine := gcm.findMachineByName(session.Name)
		if machine == nil {
			machine = &define.Machine{
				ID:   uuid.NewString(),
				Name: session.Name,
			}
		} else {
			machine.EnsureID()
		}

		sensitive := &define.SensitiveData{
			Host:     session.Host,
			Port:     session.Port,
			User:     user,
			Password: password,
		}
		if err := machine.SetSensitiveData(sensitive); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: 加密失败: %v", session.Name, err))
			result.Skipped++
			continue
		}

		if err := gcm.upsertMachine(machine); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", session.Name, err))
			result.Skipped++
			continue
		}
		result.Imported++
	}

	return result, nil
}
