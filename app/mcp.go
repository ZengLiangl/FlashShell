package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"FlashDock/data"
	"FlashDock/machine"
	"FlashDock/mcp"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) startMCP() {
	if a.mcpSvc != nil {
		return
	}
	a.mcpSvc = mcp.New(a.configManager)
	a.mcpSvc.SetSSHShare(func(configName string) *machine.SSHClient {
		if a.shellPool == nil {
			return nil
		}
		return a.shellPool.SharedClientForConfig(configName)
	})
	if a.subProjectRunner != nil {
		a.subProjectRunner.SetShellClientProvider(a.sshShareProvider())
	}
	a.mcpSvc.SetEmitter(func(event string, payload any) {
		if a.ctx != nil {
			wailsRuntime.EventsEmit(a.ctx, event, payload)
		}
	})
	settings := a.mcpSvc.GetSettings()
	if !settings.AutoStart {
		// 未开自动开启：启动时不拉起 HTTP（忽略历史 enabled:true）
		if settings.Enabled {
			settings.Enabled = false
			_ = a.mcpSvc.UpdateSettings(settings)
		}
		return
	}
	if !settings.Enabled {
		settings.Enabled = true
		if err := a.mcpSvc.UpdateSettings(settings); err != nil {
			data.AppLogf("MCP 自动开启失败: %v", err)
		}
		return
	}
	if err := a.mcpSvc.StartHTTP(); err != nil {
		data.AppLogf("MCP 启动失败: %v", err)
	}
	a.emitMCPStatusChanged()
}

func (a *App) stopMCP() {
	if a.mcpSvc != nil {
		a.mcpSvc.Stop()
	}
}

// GetMCPStatus MCP 接入状态
func (a *App) GetMCPStatus() mcp.Status {
	if a.mcpSvc == nil {
		return mcp.Status{}
	}
	return a.mcpSvc.GetStatus()
}

// GetMCPSettings 读取 MCP 设置
func (a *App) GetMCPSettings() mcp.Settings {
	if a.mcpSvc == nil {
		return mcp.Settings{}
	}
	return a.mcpSvc.GetSettings()
}

// SaveMCPSettings 保存 MCP 设置（可能重启 HTTP 端口）
func (a *App) SaveMCPSettings(in mcp.Settings) error {
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	if a.mcpSvc == nil {
		return nil
	}
	err := a.mcpSvc.UpdateSettings(in)
	a.emitMCPStatusChanged()
	return err
}

func (a *App) emitMCPStatusChanged() {
	if a.ctx == nil || a.mcpSvc == nil {
		return
	}
	wailsRuntime.EventsEmit(a.ctx, "mcp:status-changed", a.mcpSvc.GetStatus())
}

// GetMCPSnippets 手动接入 JSON 片段
func (a *App) GetMCPSnippets() mcp.ClientSnippet {
	if a.mcpSvc == nil {
		return mcp.ClientSnippet{}
	}
	return a.mcpSvc.Snippets()
}

// ListMCPTokens 作用域 token 列表
func (a *App) ListMCPTokens() []mcp.Token {
	if a.mcpSvc == nil {
		return nil
	}
	return a.mcpSvc.ListTokens()
}

// GenerateMCPToken 手动签发 scoped token（明文仅返回一次）
func (a *App) GenerateMCPToken(name, client string) (mcp.Token, error) {
	if err := a.requireUnlocked(); err != nil {
		return mcp.Token{}, err
	}
	if a.mcpSvc == nil {
		return mcp.Token{}, nil
	}
	return a.mcpSvc.GenerateToken(name, client)
}

// IssueMCPToken 签发带可见服务器与 CIDR 的 scoped token（明文仅返回一次）
func (a *App) IssueMCPToken(opts mcp.IssueOpts) (mcp.Token, error) {
	if err := a.requireUnlocked(); err != nil {
		return mcp.Token{}, err
	}
	if a.mcpSvc == nil {
		return mcp.Token{}, nil
	}
	return a.mcpSvc.IssueToken(opts)
}

// ListMCPServerAliases 供接入向导勾选可见服务器
func (a *App) ListMCPServerAliases() []string {
	if a.mcpSvc == nil {
		return nil
	}
	return a.mcpSvc.ListServerAliases()
}

// WriteMCPGuidance 向 Claude/Codex/OpenCode 规则文件注入操作指引
func (a *App) WriteMCPGuidance(clientID string) (mcp.GuidanceStatus, error) {
	if a.mcpSvc == nil {
		return mcp.GuidanceStatus{}, nil
	}
	return a.mcpSvc.WriteGuidance(clientID)
}

// GetMCPGuidancePreview 预览规则文本（Cursor 复制用）
func (a *App) GetMCPGuidancePreview(clientID string) string {
	if a.mcpSvc == nil {
		return ""
	}
	return a.mcpSvc.GuidancePreview(clientID)
}

// RevokeMCPToken 撤销 token
func (a *App) RevokeMCPToken(id string) error {
	if a.mcpSvc == nil {
		return nil
	}
	return a.mcpSvc.RevokeToken(id)
}

// ClearMCPTokens 清空全部 scoped token
func (a *App) ClearMCPTokens() error {
	if a.mcpSvc == nil {
		return nil
	}
	return a.mcpSvc.ClearTokens()
}

// OpenMCPPath 在系统文件管理器中定位配置文件或目录
func (a *App) OpenMCPPath(path string) error {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return fmt.Errorf("路径为空")
	}
	openTarget := path
	if _, err := os.Stat(path); err != nil {
		parent := filepath.Dir(path)
		if _, e2 := os.Stat(parent); e2 != nil {
			return fmt.Errorf("路径不存在: %s", path)
		}
		openTarget = parent
	}
	switch runtime.GOOS {
	case "darwin":
		if openTarget == path {
			return exec.Command("open", "-R", path).Start()
		}
		return exec.Command("open", openTarget).Start()
	case "windows":
		return exec.Command("explorer", "/select,", path).Start()
	default:
		dir := openTarget
		if fi, err := os.Stat(openTarget); err == nil && !fi.IsDir() {
			dir = filepath.Dir(openTarget)
		}
		return exec.Command("xdg-open", dir).Start()
	}
}

// InstallCursorMCP 一键写入 ~/.cursor/mcp.json
func (a *App) InstallCursorMCP() error {
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	if a.mcpSvc == nil {
		return nil
	}
	return a.mcpSvc.InstallCursor()
}

// RefreshCursorMCP 刷新 Cursor 中的 FlashShell MCP 地址与 token
func (a *App) RefreshCursorMCP() error {
	if a.mcpSvc == nil {
		return nil
	}
	return a.mcpSvc.RefreshCursor()
}

// UninstallCursorMCP 从 Cursor 移除 FlashShell MCP
func (a *App) UninstallCursorMCP() error {
	if a.mcpSvc == nil {
		return nil
	}
	return a.mcpSvc.UninstallCursor()
}

// ListMCPClients 四个 AI 客户端接入状态
func (a *App) ListMCPClients() []mcp.ClientLink {
	if a.mcpSvc == nil {
		return nil
	}
	return a.mcpSvc.ListClientLinks()
}

// InstallMCPClient 一键接入（默认全服务器 + 127.0.0.1/32）；返回明文 Token 一次
func (a *App) InstallMCPClient(id string) (mcp.Token, error) {
	if err := a.requireUnlocked(); err != nil {
		return mcp.Token{}, err
	}
	if a.mcpSvc == nil {
		return mcp.Token{}, nil
	}
	return a.mcpSvc.InstallClient(id)
}

// InstallMCPClientWith 接入向导：Token 名 / 可见服务器 / CIDR
func (a *App) InstallMCPClientWith(id string, opts mcp.InstallOpts) (mcp.Token, error) {
	if err := a.requireUnlocked(); err != nil {
		return mcp.Token{}, err
	}
	if a.mcpSvc == nil {
		return mcp.Token{}, nil
	}
	return a.mcpSvc.InstallClientWith(id, opts)
}

// UninstallMCPClient 从对应客户端移除 FlashShell entry 并删专属 token
func (a *App) UninstallMCPClient(id string) error {
	if a.mcpSvc == nil {
		return nil
	}
	return a.mcpSvc.UninstallClient(id)
}

// RefreshMCPClient 重新签发并写入配置（旧明文不可恢复）
func (a *App) RefreshMCPClient(id string) (mcp.Token, error) {
	if a.mcpSvc == nil {
		return mcp.Token{}, nil
	}
	return a.mcpSvc.RefreshClient(id)
}

// QueryMCPAudit 查询审计日志
func (a *App) QueryMCPAudit(filter mcp.AuditFilter) []mcp.AuditEntry {
	if a.mcpSvc == nil {
		return nil
	}
	return a.mcpSvc.QueryAudit(filter)
}

// GetMCPAuditStats 审计汇总
func (a *App) GetMCPAuditStats() mcp.AuditStats {
	if a.mcpSvc == nil {
		return mcp.AuditStats{}
	}
	return a.mcpSvc.AuditStats()
}

// GetMCPAuditMeta 审计筛选项（工具/模块/服务器/来源）
func (a *App) GetMCPAuditMeta() mcp.AuditMeta {
	if a.mcpSvc == nil {
		return mcp.AuditMeta{}
	}
	return a.mcpSvc.AuditMeta()
}

// ClearMCPAudit 清空全部审计日志
func (a *App) ClearMCPAudit() error {
	if a.mcpSvc == nil {
		return nil
	}
	return a.mcpSvc.ClearAudit()
}

// DeleteMCPAudit 按 ID 批量删除审计
func (a *App) DeleteMCPAudit(ids []string) error {
	if a.mcpSvc == nil {
		return nil
	}
	return a.mcpSvc.DeleteAuditIDs(ids)
}

// PurgeMCPAudit 按保留策略清理过期审计
func (a *App) PurgeMCPAudit() (int, error) {
	if a.mcpSvc == nil {
		return 0, nil
	}
	return a.mcpSvc.PurgeAuditByRetention()
}

// ListMCPSensitive 敏感库元数据（脱敏捕获，无明文）
func (a *App) ListMCPSensitive() []map[string]any {
	if a.mcpSvc == nil {
		return nil
	}
	return a.mcpSvc.ListSensitiveMeta()
}

// AddMCPOutboundHost 将 host 加入出站白名单
func (a *App) AddMCPOutboundHost(host string) error {
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	if a.mcpSvc == nil {
		return nil
	}
	return a.mcpSvc.AddOutboundHost(host)
}

// ExportMCPAudit 导出当前过滤条件下的审计日志（csv / jsonl）
func (a *App) ExportMCPAudit(format string, filter mcp.AuditFilter) (string, error) {
	if a.mcpSvc == nil {
		return "", fmt.Errorf("MCP 未启动")
	}
	if a.ctx == nil {
		return "", fmt.Errorf("应用未就绪")
	}
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "csv"
	}
	defName := "flashshell-audit." + format
	filters := []wailsRuntime.FileFilter{{DisplayName: "CSV", Pattern: "*.csv"}}
	if format == "jsonl" || format == "json" {
		format = "jsonl"
		defName = "flashshell-audit.jsonl"
		filters = []wailsRuntime.FileFilter{{DisplayName: "JSONL", Pattern: "*.jsonl"}}
	}
	path, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "导出审计日志",
		DefaultFilename: defName,
		Filters:         filters,
	})
	if err != nil || path == "" {
		return "", err
	}
	filter.Limit = 5000
	if format == "jsonl" {
		if err := a.mcpSvc.ExportAuditJSONL(path, filter); err != nil {
			return "", err
		}
		return path, nil
	}
	if err := a.mcpSvc.ExportAuditCSV(path, filter); err != nil {
		return "", err
	}
	return path, nil
}

// ListMCPCustomDangerPatterns 自定义危险黑名单正则
func (a *App) ListMCPCustomDangerPatterns() []string {
	if a.mcpSvc == nil {
		return nil
	}
	return a.mcpSvc.ListCustomDangerPatterns()
}

// SaveMCPCustomDangerPatterns 保存自定义危险黑名单
func (a *App) SaveMCPCustomDangerPatterns(patterns []string) error {
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	if a.mcpSvc == nil {
		return nil
	}
	return a.mcpSvc.SaveCustomDangerPatterns(patterns)
}

// ListMCPApprovals 待审批列表
func (a *App) ListMCPApprovals() []mcp.ApprovalItem {
	if a.mcpSvc == nil {
		return nil
	}
	return a.mcpSvc.ListApprovals()
}

// DecideMCPApproval 审批放行或拒绝；addOutboundHosts 为 true 时把违规出站 host 永久加入白名单（失败则不放行）
func (a *App) DecideMCPApproval(id string, allow bool, addOutboundHosts bool, rejectReason string) error {
	if a.mcpSvc == nil {
		return nil
	}
	return a.mcpSvc.DecideApproval(id, allow, addOutboundHosts, rejectReason)
}

// DecideMCPApprovalBatch 批量审批
func (a *App) DecideMCPApprovalBatch(ids []string, allow bool, rejectReason string) error {
	if a.mcpSvc == nil {
		return nil
	}
	return a.mcpSvc.DecideApprovalBatch(ids, allow, rejectReason)
}

// GetMCPApprovalContext 审批弹窗上下文：该服务器最近审计
func (a *App) GetMCPApprovalContext(server string) []mcp.AuditEntry {
	if a.mcpSvc == nil {
		return nil
	}
	return a.mcpSvc.ApprovalContext(server, 8)
}
