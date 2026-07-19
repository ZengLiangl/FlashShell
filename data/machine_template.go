package data

import (
	"encoding/json"
	"fmt"
	"strings"

	"FlashDock/define"

	"github.com/google/uuid"
)

const redactedPassword = "__REDACTED__"

// MachineTemplateEntry 连接模板中的机器条目（密码脱敏）
type MachineTemplateEntry struct {
	Name        string             `json:"name"`
	Group       string             `json:"group,omitempty"`
	KeyFile     string             `json:"keyFile,omitempty"`
	ListHost    string             `json:"listHost,omitempty"`
	ListPort    int                `json:"listPort,omitempty"`
	ListUser    string             `json:"listUser,omitempty"`
	Tunnels     []define.SSHTunnel `json:"tunnels,omitempty"`
	Host        string             `json:"host,omitempty"`
	Port        int                `json:"port,omitempty"`
	User        string             `json:"user,omitempty"`
	Password    string             `json:"password,omitempty"`
	HasPassword bool               `json:"hasPassword,omitempty"`
}

// MachineTemplate 可导入导出的连接模板
type MachineTemplate struct {
	Version  string                 `json:"version"`
	Groups   []string               `json:"groups,omitempty"`
	Machines []MachineTemplateEntry `json:"machines"`
}

// ExportMachineTemplate 导出机器列表为模板（密码脱敏）
func ExportMachineTemplate(machines []define.Machine, groups []string) ([]byte, error) {
	tpl := MachineTemplate{
		Version:  "1",
		Groups:   groups,
		Machines: make([]MachineTemplateEntry, 0, len(machines)),
	}
	for _, m := range machines {
		entry := MachineTemplateEntry{
			Name:     m.Name,
			Group:    m.Group,
			KeyFile:  m.KeyFile,
			ListHost: m.ListHost,
			ListPort: m.ListPort,
			ListUser: m.ListUser,
			Tunnels:  m.Tunnels,
		}
		if s, err := m.GetSensitiveData(); err == nil && s != nil {
			entry.Host = s.Host
			entry.Port = s.Port
			entry.User = s.User
			if s.Password != "" {
				entry.Password = redactedPassword
				entry.HasPassword = true
			}
		}
		if entry.ListHost == "" {
			entry.ListHost = entry.Host
		}
		if entry.ListPort == 0 {
			entry.ListPort = entry.Port
		}
		if entry.ListUser == "" {
			entry.ListUser = entry.User
		}
		tpl.Machines = append(tpl.Machines, entry)
	}
	return json.MarshalIndent(tpl, "", "  ")
}

// ImportMachineTemplateResult 导入结果
type ImportMachineTemplateResult struct {
	Added   int `json:"added"`
	Updated int `json:"updated"`
	Skipped int `json:"skipped"`
}

// ImportMachineTemplate 导入模板到全局配置（merge: 同名更新，否则新增）
func ImportMachineTemplate(data []byte, merge bool, existing []define.Machine) ([]define.Machine, ImportMachineTemplateResult, error) {
	var tpl MachineTemplate
	if err := json.Unmarshal(data, &tpl); err != nil {
		return nil, ImportMachineTemplateResult{}, fmt.Errorf("解析模板失败: %w", err)
	}
	if len(tpl.Machines) == 0 {
		return existing, ImportMachineTemplateResult{}, fmt.Errorf("模板中没有机器")
	}

	byName := make(map[string]int)
	for i, m := range existing {
		byName[m.Name] = i
	}

	result := ImportMachineTemplateResult{}
	out := make([]define.Machine, len(existing))
	copy(out, existing)

	for _, entry := range tpl.Machines {
		name := strings.TrimSpace(entry.Name)
		if name == "" {
			result.Skipped++
			continue
		}
		host := strings.TrimSpace(entry.Host)
		if host == "" {
			host = strings.TrimSpace(entry.ListHost)
		}
		port := entry.Port
		if port <= 0 {
			port = entry.ListPort
		}
		if port <= 0 {
			port = 22
		}
		user := strings.TrimSpace(entry.User)
		if user == "" {
			user = strings.TrimSpace(entry.ListUser)
		}

		m := define.Machine{
			Name:     name,
			Group:    entry.Group,
			KeyFile:  entry.KeyFile,
			ListHost: entry.ListHost,
			ListPort: entry.ListPort,
			ListUser: entry.ListUser,
			Tunnels:  entry.Tunnels,
		}
		if m.ListHost == "" {
			m.ListHost = host
		}
		if m.ListPort <= 0 {
			m.ListPort = port
		}
		if m.ListUser == "" {
			m.ListUser = user
		}

		password := ""
		if entry.Password != "" && entry.Password != redactedPassword {
			password = entry.Password
		}

		if idx, ok := byName[name]; ok {
			if !merge {
				result.Skipped++
				continue
			}
			m.ID = out[idx].ID
			m.EncryptedData = out[idx].EncryptedData
			if host != "" || user != "" || password != "" {
				prev, _ := out[idx].GetSensitiveData()
				if prev == nil {
					prev = &define.SensitiveData{}
				}
				if host != "" {
					prev.Host = host
				}
				if port > 0 {
					prev.Port = port
				}
				if user != "" {
					prev.User = user
				}
				if password != "" {
					prev.Password = password
				}
				_ = m.SetSensitiveData(prev)
			} else {
				m.EncryptedData = out[idx].EncryptedData
			}
			out[idx] = m
			result.Updated++
			continue
		}

		m.ID = uuid.NewString()
		sensitive := &define.SensitiveData{Host: host, Port: port, User: user, Password: password}
		if err := m.SetSensitiveData(sensitive); err != nil {
			return nil, result, fmt.Errorf("加密机器 %s 失败: %w", name, err)
		}
		out = append(out, m)
		byName[name] = len(out) - 1
		result.Added++
	}
	return out, result, nil
}
