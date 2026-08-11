package data

import (
	"fmt"
	"strings"
)

// GetMachineGroupDefaults 返回全部分组默认配置
func (gcm *GlobalConfigManager) GetMachineGroupDefaults() []MachineGroupDefaults {
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return []MachineGroupDefaults{}
		}
	}
	if len(gcm.config.MachineGroupDefaultsList) == 0 {
		return []MachineGroupDefaults{}
	}
	out := make([]MachineGroupDefaults, len(gcm.config.MachineGroupDefaultsList))
	copy(out, gcm.config.MachineGroupDefaultsList)
	return out
}

// GetMachineGroupDefaultsByName 按分组名获取默认配置
func (gcm *GlobalConfigManager) GetMachineGroupDefaultsByName(name string) *MachineGroupDefaults {
	name = strings.TrimSpace(name)
	if name == "" || name == DefaultMachineGroupName {
		name = ""
	}
	for _, item := range gcm.GetMachineGroupDefaults() {
		if strings.TrimSpace(item.Name) == name {
			copy := item
			return &copy
		}
	}
	return nil
}

// SaveMachineGroupDefaults 保存单条分组默认配置
func (gcm *GlobalConfigManager) SaveMachineGroupDefaults(def MachineGroupDefaults) error {
	def.Name = strings.TrimSpace(def.Name)
	if def.Name == "" {
		return fmt.Errorf("分组名称不能为空")
	}
	if def.Name == DefaultMachineGroupName {
		return fmt.Errorf("不能为默认分组设置默认配置")
	}
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return err
		}
	}
	found := false
	for i, item := range gcm.config.MachineGroupDefaultsList {
		if strings.TrimSpace(item.Name) == def.Name {
			gcm.config.MachineGroupDefaultsList[i] = def
			found = true
			break
		}
	}
	if !found {
		gcm.config.MachineGroupDefaultsList = append(gcm.config.MachineGroupDefaultsList, def)
	}
	gcm.EnsureMachineGroupRegistered(def.Name)
	return gcm.SaveGlobalConfig(gcm.config)
}

func (gcm *GlobalConfigManager) removeMachineGroupDefaults(name string) {
	if gcm.config == nil {
		return
	}
	name = strings.TrimSpace(name)
	list := make([]MachineGroupDefaults, 0, len(gcm.config.MachineGroupDefaultsList))
	for _, item := range gcm.config.MachineGroupDefaultsList {
		if strings.TrimSpace(item.Name) == name {
			continue
		}
		list = append(list, item)
	}
	gcm.config.MachineGroupDefaultsList = list
}

func (gcm *GlobalConfigManager) renameMachineGroupDefaults(oldName, newName string) {
	if gcm.config == nil {
		return
	}
	oldName = strings.TrimSpace(oldName)
	newName = strings.TrimSpace(newName)
	for i, item := range gcm.config.MachineGroupDefaultsList {
		if strings.TrimSpace(item.Name) == oldName {
			item.Name = newName
			gcm.config.MachineGroupDefaultsList[i] = item
			return
		}
	}
}
