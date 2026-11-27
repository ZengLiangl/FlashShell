package data

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"FlashDock/define"
)

// ShellHistoryManager 连接历史
type ShellHistoryManager struct {
	mu      sync.Mutex
	records []define.ShellHistoryRecord
}

const maxShellHistoryRecords = 500

// NewShellHistoryManager 创建历史管理器
func NewShellHistoryManager() *ShellHistoryManager {
	m := &ShellHistoryManager{}
	_ = m.load()
	return m
}

func (m *ShellHistoryManager) load() error {
	d, err := loadAppDataSection()
	if err != nil {
		return err
	}
	m.records = append([]define.ShellHistoryRecord(nil), d.ShellHistory...)
	return nil
}

func (m *ShellHistoryManager) save() error {
	records := append([]define.ShellHistoryRecord(nil), m.records...)
	return updateAppData(func(d *AppDataFile) {
		d.ShellHistory = records
	})
}

// List 按最近连接时间倒序。
// 同时兜底合并历史里残留的重复项（旧版本按 ID 匹配时，机器重建会留下同名记录）。
func (m *ShellHistoryManager) List() []define.ShellHistoryRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]define.ShellHistoryRecord, len(m.records))
	copy(out, m.records)
	sort.Slice(out, func(i, j int) bool {
		return out[i].LastConnectedAt > out[j].LastConnectedAt
	})
	return mergeShellHistoryDuplicates(out)
}

// mergeShellHistoryDuplicates 按机器名合并重复历史，保留最近一条并累加次数。
// 入参需已按时间倒序，合并结果保持该顺序。
func mergeShellHistoryDuplicates(records []define.ShellHistoryRecord) []define.ShellHistoryRecord {
	if len(records) < 2 {
		return records
	}
	byName := make(map[string]int, len(records))
	out := make([]define.ShellHistoryRecord, 0, len(records))
	for _, r := range records {
		key := r.MachineName
		if key == "" {
			key = r.MachineID
		}
		if idx, ok := byName[key]; ok && key != "" {
			out[idx].ConnectCount += r.ConnectCount
			if r.LastConnectedAt > out[idx].LastConnectedAt {
				out[idx].LastConnectedAt = r.LastConnectedAt
				out[idx].MachineID = r.MachineID
				out[idx].Host = r.Host
				out[idx].Port = r.Port
				out[idx].User = r.User
			}
			continue
		}
		if key != "" {
			byName[key] = len(out)
		}
		out = append(out, r)
	}
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

	// 同一台机器只保留一条历史。机器被删掉重建后 ID 会变，
	// 若只按 ID 匹配就会不断堆积同名记录（最近连接里出现多条一样的主机）。
	// 因此按「ID 或 名称」任一命中都视为同一台机器，命中多条时合并成一条。
	matched := -1
	for i := range m.records {
		sameID := m.records[i].MachineID != "" && m.records[i].MachineID == machine.ID
		sameName := m.records[i].MachineName == machine.Name
		if !sameID && !sameName {
			continue
		}
		if matched < 0 {
			matched = i
			continue
		}
		// 已经命中过：合并计数后丢弃这一条
		m.records[matched].ConnectCount += m.records[i].ConnectCount
		if m.records[i].LastConnectedAt > m.records[matched].LastConnectedAt {
			m.records[matched].LastConnectedAt = m.records[i].LastConnectedAt
		}
		m.records = append(m.records[:i], m.records[i+1:]...)
		i--
	}

	if matched >= 0 {
		m.records[matched].MachineID = machine.ID
		m.records[matched].MachineName = machine.Name
		m.records[matched].Host = host
		m.records[matched].Port = port
		m.records[matched].User = user
		m.records[matched].LastConnectedAt = now
		m.records[matched].ConnectCount++
		return m.save()
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
