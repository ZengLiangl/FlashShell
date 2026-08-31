package app

import (
	"fmt"
	"strings"
	"sync"

	"FlashDock/data"
	"FlashDock/define"
	"FlashDock/machine"
)

type portForwardSSH struct {
	client *machine.SSHClient
}

type portForwardRuntimeStore struct {
	mu      sync.Mutex
	sshByID map[string]*portForwardSSH
}

func newPortForwardRuntimeStore() *portForwardRuntimeStore {
	return &portForwardRuntimeStore{sshByID: make(map[string]*portForwardSSH)}
}

func (s *portForwardRuntimeStore) set(id string, entry *portForwardSSH) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.sshByID[id]; ok && old != nil && old.client != nil {
		_ = old.client.Close()
	}
	if entry == nil {
		delete(s.sshByID, id)
		return
	}
	s.sshByID[id] = entry
}

func (s *portForwardRuntimeStore) get(id string) *portForwardSSH {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sshByID[id]
}

func (s *portForwardRuntimeStore) closeAll() {
	s.mu.Lock()
	ids := make([]string, 0, len(s.sshByID))
	for id := range s.sshByID {
		ids = append(ids, id)
	}
	s.mu.Unlock()
	for _, id := range ids {
		s.set(id, nil)
	}
}

func portForwardKey(ruleID string) string {
	return "_pf_" + strings.TrimSpace(ruleID)
}

// ListPortForwards 返回全部独立端口转发规则
func (a *App) ListPortForwards() []data.PortForwardRule {
	return data.GlobalPortForwardStore().List()
}

// SavePortForwards 保存独立端口转发规则
func (a *App) SavePortForwards(rules []data.PortForwardRule) error {
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	return data.GlobalPortForwardStore().SaveAll(rules)
}

func (a *App) sshClientForPortForward(machineName string, machineConfig *define.Machine) (*machine.SSHClient, bool, error) {
	if machineConfig == nil {
		return nil, false, fmt.Errorf("机器配置为空")
	}
	if id := a.shellPool.FirstSessionOfConfig(machineName); id != "" {
		if sm := a.shellPool.GetSession(id); sm != nil {
			if client := sm.SharedSSHClient(); client != nil && client.IsConnected() {
				return client, false, nil
			}
		}
	}
	prepared, err := a.configManager.MachineForConnect(machineConfig)
	if err != nil {
		return nil, false, err
	}
	client := machine.NewSSHClient(prepared, a.configManager.GetWorkPathVars())
	if err := client.ConnectAutoTrustOnce(prepared, false); err != nil {
		return nil, false, err
	}
	return client, true, nil
}

// StartPortForward 启动独立端口转发
func (a *App) StartPortForward(id string) error {
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("规则 ID 不能为空")
	}
	rule := data.GlobalPortForwardStore().Get(id)
	if rule == nil {
		return fmt.Errorf("未找到端口转发规则: %s", id)
	}
	if strings.TrimSpace(rule.MachineName) == "" {
		return fmt.Errorf("请配置关联机器")
	}
	machineConfig := a.configManager.GetMachine(rule.MachineName)
	if machineConfig == nil {
		return fmt.Errorf("未找到机器配置: %s", rule.MachineName)
	}
	client, owned, err := a.sshClientForPortForward(rule.MachineName, machineConfig)
	if err != nil {
		return err
	}
	if a.tunnelMgr == nil {
		if owned {
			_ = client.Close()
		}
		return fmt.Errorf("隧道管理器未初始化")
	}
	key := portForwardKey(id)
	a.tunnelMgr.StopAllFor(key)
	spec := rule.ToSSHTunnel()
	if err := a.tunnelMgr.AddTemporary(key, spec, client); err != nil {
		if owned {
			_ = client.Close()
		}
		return err
	}
	if owned {
		if a.portForwardSSH == nil {
			a.portForwardSSH = newPortForwardRuntimeStore()
		}
		a.portForwardSSH.set(id, &portForwardSSH{client: client})
	}
	return nil
}

// StopPortForward 停止独立端口转发
func (a *App) StopPortForward(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("规则 ID 不能为空")
	}
	key := portForwardKey(id)
	if a.tunnelMgr != nil {
		a.tunnelMgr.StopAllFor(key)
	}
	if a.portForwardSSH != nil {
		a.portForwardSSH.set(id, nil)
	}
	return nil
}

// StartAutoPortForwards 启动标记为自动启动的端口转发
func (a *App) StartAutoPortForwards() {
	for _, rule := range data.GlobalPortForwardStore().List() {
		if !rule.Enabled || !rule.AutoStart {
			continue
		}
		if err := a.StartPortForward(rule.ID); err != nil {
			fmt.Printf("自动启动端口转发失败(%s): %v\n", rule.Name, err)
		}
	}
}

// GetPortForwardStatus 返回独立端口转发运行状态
func (a *App) GetPortForwardStatus(id string) []define.SSHTunnelStatus {
	if a.tunnelMgr == nil {
		return nil
	}
	return a.tunnelMgr.StatusList(portForwardKey(strings.TrimSpace(id)))
}
