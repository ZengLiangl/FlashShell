package app

import "FlashDock/machine"

func (a *App) sshShareProvider() machine.ShellClientProvider {
	return sshShareBundle{pool: a.shellPool, mcp: a.mcpSvc}
}

// sshShareBundle 任务模式与 MCP 共用已有 SSH：优先 Shell 会话，其次 MCP 已拨号连接。
type sshShareBundle struct {
	pool *machine.ShellSessionPool
	mcp  interface {
		OwnedClient(configName string) *machine.SSHClient
	}
}

func (b sshShareBundle) SharedClientForConfig(configName string) *machine.SSHClient {
	if b.pool != nil {
		if c := b.pool.SharedClientForConfig(configName); c != nil && c.IsConnected() {
			return c
		}
	}
	if b.mcp != nil {
		if c := b.mcp.OwnedClient(configName); c != nil && c.IsConnected() {
			return c
		}
	}
	return nil
}
