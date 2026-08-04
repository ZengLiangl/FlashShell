package data

import (
	"fmt"
	"strings"
)

const DefaultMachineGroupName = "默认分组"

// GetMachineGroups 返回已登记分组（合并机器上已用分组），默认分组不写入列表也始终可用
func (gcm *GlobalConfigManager) GetMachineGroups() []string {
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return []string{}
		}
	}
	seen := map[string]bool{}
	out := make([]string, 0)
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || name == DefaultMachineGroupName {
			return
		}
		if seen[name] {
			return
		}
		seen[name] = true
		out = append(out, name)
	}
	for _, g := range gcm.config.MachineGroups {
		add(g)
	}
	for _, m := range gcm.config.Machines {
		add(m.Group)
	}
	return out
}

// AddMachineGroup 添加自定义分组
func (gcm *GlobalConfigManager) AddMachineGroup(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("分组名称不能为空")
	}
	if name == DefaultMachineGroupName {
		return fmt.Errorf("「%s」为系统默认分组，无需添加", DefaultMachineGroupName)
	}
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return err
		}
	}
	for _, g := range gcm.GetMachineGroups() {
		if g == name {
			return fmt.Errorf("分组已存在: %s", name)
		}
	}
	gcm.config.MachineGroups = append(gcm.config.MachineGroups, name)
	return gcm.SaveGlobalConfig(gcm.config)
}

// RenameMachineGroup 重命名分组并更新机器归属
func (gcm *GlobalConfigManager) RenameMachineGroup(oldName, newName string) error {
	oldName = strings.TrimSpace(oldName)
	newName = strings.TrimSpace(newName)
	if oldName == "" || newName == "" {
		return fmt.Errorf("分组名称不能为空")
	}
	if oldName == DefaultMachineGroupName {
		return fmt.Errorf("不能重命名默认分组")
	}
	if newName == DefaultMachineGroupName {
		return fmt.Errorf("不能使用默认分组名称")
	}
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return err
		}
	}
	found := false
	groups := make([]string, 0, len(gcm.config.MachineGroups))
	for _, g := range gcm.config.MachineGroups {
		g = strings.TrimSpace(g)
		if g == oldName {
			found = true
			groups = append(groups, newName)
			continue
		}
		if g == newName {
			return fmt.Errorf("目标分组已存在: %s", newName)
		}
		if g != "" {
			groups = append(groups, g)
		}
	}
	// 即便未在 MachineGroups 登记，只要机器在用也允许重命名
	for i := range gcm.config.Machines {
		if strings.TrimSpace(gcm.config.Machines[i].Group) == oldName {
			found = true
			gcm.config.Machines[i].Group = newName
		}
	}
	if !found {
		return fmt.Errorf("分组不存在: %s", oldName)
	}
	gcm.config.MachineGroups = groups
	// 确保新名在登记列表中
	hasNew := false
	for _, g := range gcm.config.MachineGroups {
		if g == newName {
			hasNew = true
			break
		}
	}
	if !hasNew {
		gcm.config.MachineGroups = append(gcm.config.MachineGroups, newName)
	}
	return gcm.SaveGlobalConfig(gcm.config)
}

// DeleteMachineGroup 删除分组；所属机器归入默认分组（清空 group 字段）
func (gcm *GlobalConfigManager) DeleteMachineGroup(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("分组名称不能为空")
	}
	if name == DefaultMachineGroupName {
		return fmt.Errorf("不能删除默认分组")
	}
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return err
		}
	}
	groups := make([]string, 0, len(gcm.config.MachineGroups))
	found := false
	for _, g := range gcm.config.MachineGroups {
		g = strings.TrimSpace(g)
		if g == name {
			found = true
			continue
		}
		if g != "" {
			groups = append(groups, g)
		}
	}
	for i := range gcm.config.Machines {
		if strings.TrimSpace(gcm.config.Machines[i].Group) == name {
			found = true
			gcm.config.Machines[i].Group = ""
		}
	}
	if !found {
		return fmt.Errorf("分组不存在: %s", name)
	}
	gcm.config.MachineGroups = groups
	return gcm.SaveGlobalConfig(gcm.config)
}

// EnsureMachineGroupRegistered 若分组非空且未登记则写入配置（不单独保存，由调用方保存）
func (gcm *GlobalConfigManager) EnsureMachineGroupRegistered(name string) {
	name = strings.TrimSpace(name)
	if name == "" || name == DefaultMachineGroupName || gcm.config == nil {
		return
	}
	for _, g := range gcm.config.MachineGroups {
		if strings.TrimSpace(g) == name {
			return
		}
	}
	gcm.config.MachineGroups = append(gcm.config.MachineGroups, name)
}

// UpdateMachineGroup 仅更新机器所属分组，保留加密凭证等其它字段。
// group 为空或「默认分组」时归入默认分组（清空 group 字段）。
func (gcm *GlobalConfigManager) UpdateMachineGroup(machineID, group string) error {
	machineID = strings.TrimSpace(machineID)
	if machineID == "" {
		return fmt.Errorf("机器 ID 不能为空")
	}
	group = strings.TrimSpace(group)
	if group == DefaultMachineGroupName {
		group = ""
	}
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return err
		}
	}
	for i := range gcm.config.Machines {
		if gcm.config.Machines[i].ID != machineID {
			continue
		}
		if strings.TrimSpace(gcm.config.Machines[i].Group) == group {
			return nil
		}
		gcm.config.Machines[i].Group = group
		gcm.EnsureMachineGroupRegistered(group)
		return gcm.SaveGlobalConfig(gcm.config)
	}
	return fmt.Errorf("机器配置 '%s' 不存在", machineID)
}

// UpdateMachineShellMonitorOpen 仅更新 Shell 左侧监控栏展开状态，保留其它字段。
// machineKey 可为机器 ID 或名称。
func (gcm *GlobalConfigManager) UpdateMachineShellMonitorOpen(machineKey string, open bool) error {
	machineKey = strings.TrimSpace(machineKey)
	if machineKey == "" {
		return fmt.Errorf("机器 ID 或名称不能为空")
	}
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return err
		}
	}
	for i := range gcm.config.Machines {
		m := &gcm.config.Machines[i]
		if m.ID != machineKey && m.Name != machineKey {
			continue
		}
		if m.IsShellMonitorOpen() == open {
			return nil
		}
		m.SetShellMonitorOpen(open)
		return gcm.SaveGlobalConfig(gcm.config)
	}
	return fmt.Errorf("机器配置 '%s' 不存在", machineKey)
}

// UpdateMachinePinned 仅更新机器置顶状态，保留其它字段。
// machineKey 可为机器 ID 或名称。
func (gcm *GlobalConfigManager) UpdateMachinePinned(machineKey string, pinned bool) error {
	machineKey = strings.TrimSpace(machineKey)
	if machineKey == "" {
		return fmt.Errorf("机器 ID 或名称不能为空")
	}
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return err
		}
	}
	for i := range gcm.config.Machines {
		m := &gcm.config.Machines[i]
		if m.ID != machineKey && m.Name != machineKey {
			continue
		}
		if m.Pinned == pinned {
			return nil
		}
		m.Pinned = pinned
		return gcm.SaveGlobalConfig(gcm.config)
	}
	return fmt.Errorf("机器配置 '%s' 不存在", machineKey)
}
