package data

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"FlashDock/define"

	"github.com/google/uuid"
)

const defaultXshellPassword = "123456"

// ParsedXshellSession 解析后的 Xshell 会话
type ParsedXshellSession struct {
	Name string
	Host string
	Port int
	User string
}

// ParseXshellFile 解析单个 .xsh 文件
func ParseXshellFile(path string) (*ParsedXshellSession, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	session, err := ParseXshellContent(data)
	if err != nil {
		return nil, err
	}
	base := filepath.Base(path)
	session.Name = strings.TrimSuffix(base, filepath.Ext(base))
	return session, nil
}

func normalizeXshellBytes(data []byte) []byte {
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
		utf16 := data[2:]
		buf := make([]byte, 0, len(utf16)/2)
		for i := 0; i+1 < len(utf16); i += 2 {
			buf = append(buf, utf16[i])
		}
		return buf
	}
	return data
}

// ParseXshellContent 解析 Xshell 会话内容
func ParseXshellContent(data []byte) (*ParsedXshellSession, error) {
	data = normalizeXshellBytes(data)
	section := ""
	values := map[string]string{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = line[1 : len(line)-1]
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		values[section+":"+key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	host := values["CONNECTION:Host"]
	if host == "" {
		return nil, fmt.Errorf("未找到 Host")
	}
	port := 22
	if portStr := values["CONNECTION:Port"]; portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
			port = p
		}
	}
	user := values["CONNECTION:AUTHENTICATION:UserName"]
	return &ParsedXshellSession{
		Host: host,
		Port: port,
		User: user,
	}, nil
}

// CollectXshellFiles 收集目录或路径列表中的 .xsh 文件
func CollectXshellFiles(paths []string) ([]string, error) {
	return CollectFilesFromPaths(paths, isXshellFile)
}

// CollectXshellFilesLegacy 兼容旧签名
func CollectXshellFilesLegacy(path string, isDirectory bool) ([]string, error) {
	if isDirectory {
		return CollectFilesFromPaths([]string{path}, isXshellFile)
	}
	return CollectFilesFromPaths([]string{path}, isXshellFile)
}

// ImportXshellFiles 批量导入 Xshell 会话到全局机器配置
func (gcm *GlobalConfigManager) ImportXshellFiles(paths []string, accountID string) (*MachineImportResult, error) {
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return nil, err
		}
	}

	files, err := CollectXshellFiles(paths)
	if err != nil {
		return nil, err
	}

	result := &MachineImportResult{}
	accountUser, accountPassword := "", defaultXshellPassword
	if accountID != "" {
		user, password, err := gcm.GetGlobalAccountCredentials(accountID)
		if err != nil {
			return nil, err
		}
		accountUser, accountPassword = user, password
	}

	for _, path := range files {
		session, err := ParseXshellFile(path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", filepath.Base(path), err))
			result.Skipped++
			continue
		}

		user := session.User
		password := defaultXshellPassword
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

func (gcm *GlobalConfigManager) findMachineByName(name string) *define.Machine {
	for i := range gcm.config.Machines {
		if gcm.config.Machines[i].Name == name {
			return &gcm.config.Machines[i]
		}
	}
	return nil
}

func (gcm *GlobalConfigManager) upsertMachine(machine *define.Machine) error {
	machine.EnsureID()
	for i, existing := range gcm.config.Machines {
		if existing.ID == machine.ID {
			gcm.config.Machines[i] = *machine
			return gcm.SaveGlobalConfig(gcm.config)
		}
	}
	gcm.config.Machines = append(gcm.config.Machines, *machine)
	return gcm.SaveGlobalConfig(gcm.config)
}
