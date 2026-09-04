package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"FlashDock/data"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// New 创建 MCP 服务（不启动 HTTP）
func New(cfg *data.ConfigManager) *Service {
	return newService(cfg)
}

// RunStdio 以 stdio 运行 MCP（给 Claude Code / Codex / OpenCode；Cursor 走 HTTP）
func RunStdio() error {
	cm := data.NewConfigManager("", nil)
	_, _ = cm.LoadConfig()
	s := newService(cm)
	return s.mcp.Run(context.Background(), &mcpsdk.StdioTransport{})
}

// HasStdioFlag 是否以 MCP stdio 模式启动
func HasStdioFlag(args []string) bool {
	for _, a := range args {
		if a == "--mcp-stdio" || a == "-mcp-stdio" {
			return true
		}
	}
	return false
}

func (s *Service) ListTokens() []Token { return s.tokens.List() }

func (s *Service) IssueToken(opts IssueOpts) (IssuedToken, error) {
	issued, err := s.tokens.Issue(opts)
	if err != nil {
		return IssuedToken{}, err
	}
	s.audit.Append(AuditEntry{
		Source: "FlashShell UI", Caller: "human", Tool: "token_issue", Module: "token",
		Params: mustJSON(map[string]any{"name": opts.Name, "client": opts.Client, "servers": opts.Servers, "cidrs": opts.CIDRs}),
		Result: "ok", Decision: "auto", Reason: "手动签发 scoped token",
	})
	return issued, nil
}

func (s *Service) GenerateToken(name, client string) (IssuedToken, error) {
	return s.IssueToken(IssueOpts{Name: name, Client: client})
}

func (s *Service) UpdateToken(opts UpdateTokenOpts) (Token, error) {
	tok, err := s.tokens.Update(opts)
	if err != nil {
		return Token{}, err
	}
	s.audit.Append(AuditEntry{
		Source: "FlashShell UI", Caller: "human", Tool: "token_update", Module: "token",
		Params: mustJSON(map[string]any{"id": opts.ID, "name": tok.Name, "servers": tok.Servers, "cidrs": tok.CIDRs}),
		Result: "ok", Decision: "auto", Reason: "更新 scoped token 作用域",
	})
	return tok, nil
}

func (s *Service) RevokeToken(id string) error {
	removed, err := s.tokens.Revoke(id)
	if err != nil {
		return err
	}
	s.audit.Append(AuditEntry{
		Source: "FlashShell UI", Caller: "human", Tool: "token_revoked", Module: "token",
		Params: mustJSON(map[string]any{"id": removed.ID, "name": removed.Name, "client": removed.Client}),
		Result: "revoked", Decision: "auto", Reason: "删除 scoped token",
	})
	return nil
}

func (s *Service) ClearTokens() error {
	n := len(s.tokens.List())
	if err := s.tokens.Clear(); err != nil {
		return err
	}
	s.audit.Append(AuditEntry{
		Source: "FlashShell UI", Caller: "human", Tool: "token_revoked", Module: "token",
		Params: mustJSON(map[string]any{"count": n}),
		Result: "cleared", Decision: "auto", Reason: "清空全部 scoped token",
	})
	return nil
}

func (s *Service) ListServerAliases() []string { return s.allAliases() }

func (s *Service) GetToken(id string) (Token, bool) { return s.tokens.Get(id) }

func (s *Service) QueryAudit(f AuditFilter) []AuditEntry { return s.audit.Query(f) }

func (s *Service) AuditStats() AuditStats {
	st := s.audit.Stats()
	// 待审批以审批队列实时数为准（阻塞中尚未落审计的请求）
	pendingLive := len(s.approvals.List())
	if pendingLive > st.Pending {
		st.Pending = pendingLive
	}
	return st
}

func (s *Service) AuditMeta() AuditMeta { return s.audit.Meta() }

func (s *Service) ClearAudit() error { return s.audit.Clear() }

func (s *Service) DeleteAuditIDs(ids []string) error { return s.audit.DeleteIDs(ids) }

func (s *Service) ExportAuditJSONL(path string, f AuditFilter) error {
	return s.audit.ExportJSONL(path, f)
}

func (s *Service) ExportAuditCSV(path string, f AuditFilter) error {
	return s.audit.ExportCSV(path, f)
}

func (s *Service) ListApprovals() []ApprovalItem { return s.approvals.List() }

func (s *Service) PurgeAuditByRetention() (int, error) {
	days := s.GetSettings().AuditRetentionDays
	if days <= 0 {
		days = 90
	}
	n, err := s.audit.PurgeOlderThan(days)
	_, _ = s.ExpireSensitiveVault()
	return n, err
}

func (s *Service) ListSensitiveMeta() []map[string]any {
	if s.sensitive == nil {
		return []map[string]any{}
	}
	return s.sensitive.ListMeta()
}

// ListInstalledMeta 服务凭据台账（无明文）
func (s *Service) ListInstalledMeta(server string) []map[string]any {
	if s.vault == nil {
		return []map[string]any{}
	}
	return s.vault.ListMeta(server)
}

func (s *Service) DeleteInstalled(id string) error {
	if s.vault == nil {
		return fmt.Errorf("凭据库未初始化")
	}
	return s.vault.Delete(id)
}

func (s *Service) UpdateInstalledNotes(id, notes string) error {
	if s.vault == nil {
		return fmt.Errorf("凭据库未初始化")
	}
	return s.vault.UpdateNotes(id, notes)
}

func (s *Service) ExpireSensitiveVault() (int, error) {
	if s.sensitive == nil {
		return 0, nil
	}
	return s.sensitive.ExpireDue(time.Now())
}

func (s *Service) RevealSensitive(id string) (string, error) {
	if s.sensitive == nil {
		return "", fmt.Errorf("敏感库未初始化")
	}
	return s.sensitive.Reveal(id)
}

func (s *Service) DiscardSensitive(id string) error {
	if s.sensitive == nil {
		return fmt.Errorf("敏感库未初始化")
	}
	return s.sensitive.Discard(id)
}

// PromoteSensitiveOpts 敏感库转服务凭据
type PromoteSensitiveOpts struct {
	ID     string `json:"id"`
	Server string `json:"server"`
	Kind   string `json:"kind"`
	Label  string `json:"label"`
	Field  string `json:"field"` // 写入 secrets 的字段名，默认 password/value
}

func (s *Service) PromoteSensitive(opts PromoteSensitiveOpts) (map[string]any, error) {
	if s.sensitive == nil || s.vault == nil {
		return nil, fmt.Errorf("敏感库/凭据库未初始化")
	}
	id := strings.TrimSpace(opts.ID)
	plain, err := s.sensitive.Reveal(id)
	if err != nil {
		return nil, err
	}
	meta, _ := s.sensitive.FindMeta(id)
	server := strings.TrimSpace(opts.Server)
	if server == "" && meta != nil {
		server = strings.TrimSpace(fmt.Sprint(meta["server"]))
	}
	// 允许空 = 共用凭据
	if server != "" {
		if _, err := s.machineByAlias(server); err != nil {
			return nil, err
		}
	}
	kind := strings.TrimSpace(opts.Kind)
	if kind == "" && meta != nil {
		kind = fmt.Sprint(meta["kind"])
	}
	if kind == "" {
		kind = "credential"
	}
	label := strings.TrimSpace(opts.Label)
	if label == "" && meta != nil {
		label = fmt.Sprint(meta["label"])
	}
	if label == "" {
		label = id
	}
	field := strings.TrimSpace(opts.Field)
	if field == "" {
		if strings.Contains(strings.ToLower(kind), "token") {
			field = "token"
		} else {
			field = "password"
		}
	}
	saved, err := s.vault.Save(VaultItem{
		ServerAlias: server,
		Kind:        kind,
		Label:       label,
		Public:      map[string]string{"__tunnel_server_id": server, "fromSensitive": id},
		Secrets:     map[string]string{field: plain},
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id":          saved.ID,
		"label":       saved.Label,
		"kind":        saved.Kind,
		"serverAlias": saved.ServerAlias,
	}, nil
}

func (s *Service) ListRedactRules() []map[string]any {
	return ListRedactRulesMeta()
}

func (s *Service) SaveRedactRules(rules []UserRedactRule) error {
	return SaveUserRedactRules(rules)
}

func (s *Service) TestRedact(text string) []RedactHit {
	return TestRedactRules(text)
}

func (s *Service) ReloadRedact() {
	ReloadRedactRules()
}

func (s *Service) ListFalsePositiveSamples() []FalsePositiveSample {
	return ListFalsePositives()
}

func (s *Service) AddOutboundHost(host string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return fmt.Errorf("host 为空")
	}
	for _, h := range s.settings.OutboundHosts {
		if strings.EqualFold(h, host) {
			return nil
		}
	}
	s.settings.OutboundHosts = append(s.settings.OutboundHosts, host)
	return saveSettings(s.settings)
}
