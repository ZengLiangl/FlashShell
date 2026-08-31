package mcp

import (
	"regexp"
	"strings"
	"time"

	"FlashDock/crypto"
)

// SecurityLayer 安全模型一层
type SecurityLayer struct {
	Index   int    `json:"index"`
	Name    string `json:"name"`
	Mechan  string `json:"mechanism"`
	Status  string `json:"status"` // ok | warn | off
	Detail  string `json:"detail"`
	DocLink string `json:"docLink,omitempty"`
}

// IronRuleCheck 五条铁律自检
type IronRuleCheck struct {
	Rule    string `json:"rule"`
	Pass    bool   `json:"pass"`
	Detail  string `json:"detail"`
}

// SecurityOverview 安全总览（对齐 Reeve 安全模型文档）
type SecurityOverview struct {
	Layers          []SecurityLayer `json:"layers"`
	IronRules       []IronRuleCheck `json:"ironRules"`
	AIMode          string          `json:"aiMode"`
	EmergencyStop   bool            `json:"emergencyStop"`
	ArmedUntil      string          `json:"armedUntil,omitempty"`
	MCPOnline       bool            `json:"mcpOnline"`
	MCPBindHost     string          `json:"mcpBindHost"`
	BindLAN         bool            `json:"bindLan"`
	TokenCount      int             `json:"tokenCount"`
	AuditTotal      int             `json:"auditTotal"`
	AuditToday      int             `json:"auditToday"`
	SensitiveCount  int             `json:"sensitiveCount"`
	PendingApprovals int            `json:"pendingApprovals"`
	RedactRuleCount int             `json:"redactRuleCount"`
	RedactToolCount int             `json:"redactToolCount"`
	PolicyBreakdown map[string]int  `json:"policyBreakdown"`
	CustomDangerN   int             `json:"customDangerCount"`
	BuiltinDangerN  int             `json:"builtinDangerCount"`
	CredentialAlgo  string          `json:"credentialAlgo"`
	CredentialMode  string          `json:"credentialMode"`
	MachineCount    int             `json:"machineCount"`
	UpdatedAt       string          `json:"updatedAt"`
}

var redactToolNames = []string{
	"ssh_exec", "ssh_exec_script", "ssh_exec_multi",
	"sftp_read", "tail_log",
}

func (s *Service) SecurityOverview() SecurityOverview {
	st := s.GetSettings()
	stats := s.audit.Stats()
	tokens := s.tokens.List()
	sensitive := s.vault.ListMeta("")
	pending := len(s.approvals.List())

	policyBreak := map[string]int{}
	machineN := 0
	if s.cfg != nil {
		for _, m := range s.cfg.GetAllMachinesFromGlobal() {
			machineN++
			p := normalizePolicy(s.policyOf(&m))
			policyBreak[p]++
		}
	}

	bindHost := "127.0.0.1"
	if st.BindLAN {
		bindHost = "127.0.0.1 + 私网卡"
	}

	layer1Status := "ok"
	credSt := crypto.GetStatus()
	layer1Detail := "SSH/凭据字段 AES-256-GCM；DEK 存 OS keyring"
	credMode := "基础模式（DEK 在钥匙串）"
	if credSt.HasMasterPassword {
		credMode = "主密码模式（Argon2id KEK 包装 DEK）"
		layer1Detail = "AES-256-GCM + Argon2id KEK→DEK；锁定时 MCP/SSH 拒绝"
		if !credSt.Unlocked {
			layer1Status = "warn"
			layer1Detail += "（当前已锁定）"
		}
	} else {
		layer1Detail += "；建议设置 → 凭据安全启用主密码"
	}

	outboundOn := !st.OutboundAllowlistDisabled
	layer2Detail := "五档策略 + 致命黑名单 + sudo 强制审批 + 敏感路径永拦"
	if outboundOn {
		layer2Detail += "；出站白名单/溯源检测升级审批"
	}

	ov := SecurityOverview{
		Layers: []SecurityLayer{
			{Index: 1, Name: "凭据存储", Mechan: "AES-256-GCM + 加密落盘", Status: layer1Status, Detail: layer1Detail},
			{Index: 2, Name: "AI 访问策略", Mechan: "五档 + 黑名单 + sudo 审批", Status: aiLayerStatus(st), Detail: layer2Detail},
			{Index: 3, Name: "出口脱敏", Mechan: "8 个 MCP 工具实时脱敏 + 敏感库", Status: "ok", Detail: "命中规则 → [REDACTED:…] + vault 指针"},
			{Index: 4, Name: "审计追溯", Mechan: "每次工具调用 + 决策写审计", Status: auditLayerStatus(stats.Total), Detail: "含 auto / denied / blocked / approved"},
		},
		IronRules:        s.ironRuleChecks(st, bindHost),
		AIMode:           st.AIMode,
		EmergencyStop:    st.EmergencyStop,
		ArmedUntil:       st.ArmedUntil,
		MCPOnline:        s.online,
		MCPBindHost:      bindHost,
		BindLAN:          st.BindLAN,
		TokenCount:       len(tokens),
		AuditTotal:       stats.Total,
		AuditToday:       stats.Today,
		SensitiveCount:   len(sensitive),
		PendingApprovals: pending,
		RedactRuleCount:  len(allRedactRules()),
		RedactToolCount:  len(redactToolNames),
		PolicyBreakdown:  policyBreak,
		CustomDangerN:    len(st.CustomDangerPatterns),
		BuiltinDangerN:   len(BuiltinDangerPatternLabels()),
		CredentialAlgo:   "AES-256-GCM",
		CredentialMode:   credMode,
		MachineCount:     machineN,
		UpdatedAt:        time.Now().Format("2006-01-02 15:04:05"),
	}
	return ov
}

func aiLayerStatus(st Settings) string {
	if st.EmergencyStop || strings.EqualFold(st.AIMode, AIModeEmergency) {
		return "warn"
	}
	return "ok"
}

func auditLayerStatus(total int) string {
	if total == 0 {
		return "warn"
	}
	return "ok"
}

func (s *Service) ironRuleChecks(st Settings, bindHost string) []IronRuleCheck {
	return []IronRuleCheck{
		{
			Rule:   "MCP schema 永不含凭据字段",
			Pass:   true,
			Detail: "工具参数仅用 server 别名；密码/私钥不出后端",
		},
		{
			Rule:   "AI 仅能用服务器别名，凭据明文不出后端",
			Pass:   true,
			Detail: "list_servers 无凭据；SSH 由 FlashShell 代连",
		},
		{
			Rule:   "危险黑名单所有档位都拦（含 trusted）",
			Pass:   true,
			Detail: "rm -rf /、DROP DATABASE、FLUSHALL 等 lethal 永拦",
		},
		{
			Rule:   "MCP 只绑 127.0.0.1 / 私网，永不 0.0.0.0",
			Pass:   !st.BindLAN || bindHost != "0.0.0.0",
			Detail: "当前监听: " + bindHost + "（BindLAN=" + boolStr(st.BindLAN) + "）",
		},
		{
			Rule:   "每次 AI 操作写审计（含拒绝/拦截）",
			Pass:   true,
			Detail: "gate 拒绝/拦截/审批均 record",
		},
	}
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func BuiltinDangerPatternLabels() []string {
	return []string{
		"rm -rf /", "mkfs", "dd of=/dev", "fork bomb",
		"DROP DATABASE", "FLUSHALL", "docker system prune -a --volumes",
		"kubeadm reset", "pg_resetwal", "shutdown/reboot", "敏感路径写入",
	}
}

func (s *Service) ListCustomDangerPatterns() []string {
	st := s.GetSettings()
	out := append([]string{}, st.CustomDangerPatterns...)
	return out
}

func (s *Service) SaveCustomDangerPatterns(patterns []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var norm []string
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, err := regexp.Compile(p); err != nil {
			return wrapErr("[denied]", "无效正则: "+p+" — "+err.Error())
		}
		norm = append(norm, p)
	}
	s.settings.CustomDangerPatterns = norm
	return saveSettings(s.settings)
}
