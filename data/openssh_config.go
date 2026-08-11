package data

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"FlashDock/define"

	"github.com/google/uuid"
)

// OpenSSHImportResult OpenSSH config 导入结果
type OpenSSHImportResult struct {
	Added   int      `json:"added"`
	Updated int      `json:"updated"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors,omitempty"`
}

// ParsedOpenSSHHost 解析后的 OpenSSH Host 块
type ParsedOpenSSHHost struct {
	Name         string
	HostName     string
	Port         int
	User         string
	IdentityFile string
	ProxyJump    string
}

func hasWildcardPattern(s string) bool {
	return strings.ContainsAny(s, "*?")
}

// ParseOpenSSHConfig 解析 OpenSSH config 文件
func ParseOpenSSHConfig(path string) ([]ParsedOpenSSHHost, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var hosts []ParsedOpenSSHHost
	var current *ParsedOpenSSHHost
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "include ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.ToLower(fields[0])
		value := strings.Trim(strings.Join(fields[1:], " "), `"`)
		switch key {
		case "host":
			if current != nil && current.HostName != "" {
				hosts = append(hosts, *current)
			}
			patterns := strings.Fields(value)
			if len(patterns) == 0 {
				current = nil
				continue
			}
			name := patterns[0]
			skip := false
			for _, p := range patterns {
				if hasWildcardPattern(p) {
					skip = true
					break
				}
			}
			if skip {
				current = nil
				continue
			}
			current = &ParsedOpenSSHHost{Name: name, Port: 22}
		case "hostname":
			if current != nil {
				current.HostName = value
			}
		case "user":
			if current != nil {
				current.User = value
			}
		case "port":
			if current != nil {
				if p, err := strconv.Atoi(value); err == nil && p > 0 {
					current.Port = p
				}
			}
		case "identityfile":
			if current != nil && current.IdentityFile == "" {
				current.IdentityFile = expandPath(value)
			}
		case "proxyjump":
			if current != nil {
				current.ProxyJump = value
			}
		}
	}
	if current != nil && current.HostName != "" {
		hosts = append(hosts, *current)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return hosts, nil
}

// ImportOpenSSHConfig 从 OpenSSH config 导入机器
func (gcm *GlobalConfigManager) ImportOpenSSHConfig(path, accountID, group string) (*OpenSSHImportResult, error) {
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return nil, err
		}
	}
	hosts, err := ParseOpenSSHConfig(path)
	if err != nil {
		return nil, err
	}
	result := &OpenSSHImportResult{}
	group = strings.TrimSpace(group)
	accountUser, accountPassword := "", ""
	if accountID != "" {
		user, password, err := gcm.GetGlobalAccountCredentials(accountID)
		if err != nil {
			return nil, err
		}
		accountUser, accountPassword = user, password
	}

	for _, host := range hosts {
		if host.HostName == "" {
			result.Skipped++
			continue
		}
		name := strings.TrimSpace(host.Name)
		if name == "" {
			name = host.HostName
		}
		user := host.User
		password := ""
		if accountUser != "" {
			user = accountUser
			password = accountPassword
		}
		existing := gcm.findMachineByName(name)
		isUpdate := existing != nil
		machine := existing
		if machine == nil {
			machine = &define.Machine{
				ID:    uuid.NewString(),
				Name:  name,
				Group: group,
			}
		} else {
			machine.EnsureID()
			if group != "" {
				machine.Group = group
			}
		}
		if host.IdentityFile != "" {
			machine.KeyFile = host.IdentityFile
		}
		if host.ProxyJump != "" {
			machine.ProxyJump = host.ProxyJump
		}
		sensitive := &define.SensitiveData{
			Host:     host.HostName,
			Port:     host.Port,
			User:     user,
			Password: password,
		}
		if err := machine.SetSensitiveData(sensitive); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: 加密失败: %v", name, err))
			result.Skipped++
			continue
		}
		gcm.EnsureMachineGroupRegistered(machine.Group)
		if err := gcm.upsertMachine(machine); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", name, err))
			result.Skipped++
			continue
		}
		if isUpdate {
			result.Updated++
		} else {
			result.Added++
		}
	}
	return result, nil
}

// DefaultOpenSSHConfigPath 返回默认 ~/.ssh/config 路径
func DefaultOpenSSHConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ssh", "config"), nil
}
