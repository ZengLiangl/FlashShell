package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"FlashDock/define"
)

// ShellHistoryManager 连接历史
type ShellHistoryManager struct {
	mu       sync.Mutex
	filePath string
	records  []define.ShellHistoryRecord
}

const maxShellHistoryRecords = 500

// NewShellHistoryManager 创建历史管理器
func NewShellHistoryManager() *ShellHistoryManager {
	path := "shell_history.json"
	if configHome, err := ConfigHomeDir(); err == nil {
		path = filepath.Join(configHome, "shell_history.json")
	}
	m := &ShellHistoryManager{filePath: path}
	_ = m.load()
	return m
}

func (m *ShellHistoryManager) load() error {
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			m.records = nil
			return nil
		}
		return err
	}
	var records []define.ShellHistoryRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return err
	}
	m.records = records
	return nil
}

func (m *ShellHistoryManager) save() error {
	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m.records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.filePath, data, 0644)
}

// List 按最近连接时间倒序
func (m *ShellHistoryManager) List() []define.ShellHistoryRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]define.ShellHistoryRecord, len(m.records))
	copy(out, m.records)
	sort.Slice(out, func(i, j int) bool {
		return out[i].LastConnectedAt > out[j].LastConnectedAt
	})
	return out
}

// RecordConnect 记录一次成功连接
func (m *ShellHistoryManager) RecordConnect(machine *define.Machine, host string, port int, user string) error {
	if machine == nil {
		return fmt.Errorf("machine is nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().Unix()
	for i := range m.records {
		if m.records[i].MachineID != "" && m.records[i].MachineID == machine.ID {
			m.records[i].MachineName = machine.Name
			m.records[i].Host = host
			m.records[i].Port = port
			m.records[i].User = user
			m.records[i].LastConnectedAt = now
			m.records[i].ConnectCount++
			return m.save()
		}
		if m.records[i].MachineID == "" && m.records[i].MachineName == machine.Name {
			m.records[i].MachineID = machine.ID
			m.records[i].MachineName = machine.Name
			m.records[i].Host = host
			m.records[i].Port = port
			m.records[i].User = user
			m.records[i].LastConnectedAt = now
			m.records[i].ConnectCount++
			return m.save()
		}
	}

	m.records = append(m.records, define.ShellHistoryRecord{
		MachineID:       machine.ID,
		MachineName:     machine.Name,
		Host:            host,
		Port:            port,
		User:            user,
		LastConnectedAt: now,
		ConnectCount:    1,
	})
	if len(m.records) > maxShellHistoryRecords {
		sort.Slice(m.records, func(i, j int) bool {
			return m.records[i].LastConnectedAt < m.records[j].LastConnectedAt
		})
		m.records = m.records[len(m.records)-maxShellHistoryRecords:]
	}
	return m.save()
}

// Clear 清空历史
func (m *ShellHistoryManager) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records = nil
	return m.save()
}

// Remove 删除一条
func (m *ShellHistoryManager) Remove(machineID, machineName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	filtered := m.records[:0]
	for _, r := range m.records {
		if machineID != "" && r.MachineID == machineID {
			continue
		}
		if machineID == "" && r.MachineName == machineName {
			continue
		}
		filtered = append(filtered, r)
	}
	m.records = filtered
	return m.save()
}
