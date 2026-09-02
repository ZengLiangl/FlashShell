package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"FlashDock/crypto"
	"FlashDock/data"
	"FlashDock/define"
	"FlashDock/machine"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Status MCP 运行状态（给前端）
type Status struct {
	Enabled        bool         `json:"enabled"`
	AutoStart      bool         `json:"autoStart"`
	Online         bool         `json:"online"`
	StdioOnline    bool         `json:"stdioOnline"`
	ObserverOnline bool         `json:"observerOnline"`
	HTTPURL        string       `json:"httpUrl"`
	HTTPPort       int          `json:"httpPort"`
	BindLAN        bool         `json:"bindLan"`
	LANURL         string       `json:"lanUrl"`
	LocalAddr      string       `json:"localAddr"`
	StdioPath      string       `json:"stdioPath"`
	CursorLinked   bool         `json:"cursorLinked"`
	TokenCount     int          `json:"tokenCount"`
	ToolCount      int          `json:"toolCount"`
	ServerCount    int          `json:"serverCount"`
	PendingCount   int          `json:"pendingCount"`
	DefaultPolicy  string       `json:"defaultPolicy"`
	StartedAt      string       `json:"startedAt"`
	Clients        []ClientLink `json:"clients"`
	Instructions   []string     `json:"instructions"`
}

// ClientSnippet 手动接入片段
type ClientSnippet struct {
	StdioJSON string `json:"stdioJson"`
	HTTPJSON  string `json:"httpJson"`
	TOML      string `json:"toml"`
	HTTPURL   string `json:"httpUrl"`
	HTTPAuth  string `json:"httpAuth"`
}

// Service FlashShell MCP 服务
type Service struct {
	mu        sync.Mutex
	cfg       *data.ConfigManager
	settings  Settings
	tokens    *TokenStore
	audit     *AuditLog
	vault     *Vault
	knowledge *Knowledge
	ledger    *Ledger
	approvals *ApprovalHub
	prov      *provenanceBuf
	httpSrv   *http.Server
	listener  net.Listener
	mcp       *mcpsdk.Server
	online    bool
	startedAt time.Time
	emit      func(event string, data any)

	shareSSH func(configName string) *machine.SSHClient
	sshMu    sync.Mutex
	ownedSSH map[string]*machine.SSHClient
	sshLocks sync.Map
}

func newService(cfg *data.ConfigManager) *Service {
	s := &Service{
		cfg:       cfg,
		settings:  loadSettings(),
		tokens:    loadTokens(),
		audit:     newAuditLog(),
		vault:     loadVault(),
		knowledge: newKnowledge(),
		ledger:    loadLedger(),
		approvals: newApprovalHub(),
		prov:      newProvenance(),
	}
	if _, err := s.tokens.EnsureClient("Cursor", "cursor"); err != nil {
		data.AppLogf("MCP 生成默认 token 失败: %v", err)
	}
	s.approvals.SetTimeoutHandler(func(item ApprovalItem) {
		s.audit.Append(AuditEntry{
			Source:   item.Source,
			Caller:   "approval-timeout",
			Tool:     item.Tool,
			Module:   toolModule(item.Tool),
			Server:   item.Server,
			Params:   clip(item.ParamsJSON, 4000),
			Result:   "审批 5 分钟超时，自动拒绝",
			Decision: "denied",
			Reason:   "denied_by_timeout",
		})
	})
	s.mcp = mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "flashshell",
		Version: appVersion(),
	}, &mcpsdk.ServerOptions{
		Instructions: serverInstructions,
	})
	s.registerTools()
	s.mcp.AddReceivingMiddleware(s.tokenScopeMiddleware())
	return s
}

func appVersion() string {
	return "1.1.33"
}

const serverInstructions = `你正连接到 FlashShell（本机桌面 Shell 工作台）的 MCP。把用户配置的 SSH 主机以受控方式暴露给你：策略引擎 + 审批队列 + 审计；你永远拿不到凭据明文。

工作流：
1. 永远先 list_servers，server 参数只用返回的 alias。
2. 看清 os：Windows 用 PowerShell/cmd，不要发 df/ss/systemctl。
3. 只读：system_info / disk_usage / port_check / service_status / tail_log / sftp_list / sftp_read。
4. 执行：ssh_exec（默认 30s）/ ssh_exec_script / ssh_exec_multi。
5. 写文件：sftp_write（现写小文本）或 sftp_upload（本地已有文件）。
6. 不会的命令先 evaluate_skills 再 get_skill。
7. 装带密码的服务必须 install_with_secret 或 install_app，不要 ssh_exec 生成密码。

错误前缀：[denied] 策略拒；[blocked] 危险黑名单；[approval] 进审批；[timeout] 超时；[notfound] 别名/工具不存在。
出口可能出现 [REDACTED:xxx]，这是脱敏占位，不是泄漏。`

// StartHTTP 在 127.0.0.1（可选局域网）启动 Streamable HTTP
func (s *Service) StartHTTP() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.settings.Enabled {
		return nil
	}
	if s.online {
		return nil
	}
	host := "127.0.0.1"
	if s.settings.BindLAN {
		host = "0.0.0.0"
	}
	addr := fmt.Sprintf("%s:%d", host, s.settings.HTTPPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		ln, err = net.Listen("tcp", host+":0")
		if err != nil {
			return err
		}
		s.settings.HTTPPort = ln.Addr().(*net.TCPAddr).Port
	}
	handler := mcpsdk.NewStreamableHTTPHandler(func(r *http.Request) *mcpsdk.Server {
		return s.mcp
	}, nil)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"name":"flashshell"}`))
	})
	mux.Handle("/mcp", s.authMiddleware(handler))
	s.httpSrv = &http.Server{Handler: mux}
	s.listener = ln
	s.online = true
	s.startedAt = time.Now()
	go func() {
		_, _ = s.PurgeAuditByRetention()
	}()
	go func() {
		_ = s.httpSrv.Serve(ln)
	}()
	_ = s.writeRuntime()
	return nil
}

func (s *Service) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		token := strings.TrimPrefix(auth, "Bearer ")
		token = strings.TrimSpace(token)
		if token == "" {
			token = strings.TrimSpace(r.URL.Query().Get("token"))
		}
		tok, ok := s.tokens.ValidFrom(token, r.RemoteAddr)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		src := tok.Name
		if src == "" {
			src = clientNameFromUA(r.UserAgent())
		}
		ctx := context.WithValue(r.Context(), ctxKeySource{}, src)
		ctx = context.WithValue(ctx, ctxKeyToken{}, tok)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type ctxKeySource struct{}
type ctxKeyToken struct{}

func clientNameFromUA(ua string) string {
	ua = strings.ToLower(ua)
	switch {
	case strings.Contains(ua, "cursor"):
		return "Cursor"
	case strings.Contains(ua, "claude"):
		return "Claude Code"
	case strings.Contains(ua, "codex"):
		return "Codex"
	case strings.Contains(ua, "opencode"):
		return "OpenCode"
	default:
		return "MCP Client"
	}
}

func sourceFromCtx(ctx context.Context) string {
	if ctx == nil {
		return "MCP"
	}
	if v, ok := ctx.Value(ctxKeySource{}).(string); ok && v != "" {
		return v
	}
	return "MCP"
}

func tokenFromCtx(ctx context.Context) (Token, bool) {
	if ctx == nil {
		return Token{}, false
	}
	t, ok := ctx.Value(ctxKeyToken{}).(Token)
	return t, ok
}

// activeToken 每次工具调用前从磁盘对齐后的 TokenStore 取最新作用域（避免 ctx 快照过期）。
func (s *Service) activeToken(ctx context.Context) (Token, bool) {
	tok, ok := tokenFromCtx(ctx)
	if !ok {
		return Token{}, false
	}
	if strings.TrimSpace(tok.ID) == "" {
		return tok, ok
	}
	if fresh, found := s.tokens.Get(tok.ID); found {
		return fresh, true
	}
	return tok, ok
}

func (s *Service) tokenScopeMiddleware() mcpsdk.Middleware {
	return func(next mcpsdk.MethodHandler) mcpsdk.MethodHandler {
		return func(ctx context.Context, method string, req mcpsdk.Request) (mcpsdk.Result, error) {
			if tok, ok := tokenFromCtx(ctx); ok && strings.TrimSpace(tok.ID) != "" {
				if fresh, found := s.tokens.Get(tok.ID); found {
					ctx = context.WithValue(ctx, ctxKeyToken{}, fresh)
				}
				return next(ctx, method, req)
			}
			plain := strings.TrimSpace(os.Getenv("FLASHSHELL_TOKEN"))
			if plain != "" {
				if tok, ok := s.tokens.ValidFrom(plain, "127.0.0.1"); ok {
					ctx = context.WithValue(ctx, ctxKeyToken{}, tok)
					src := tok.Name
					if src == "" {
						src = "stdio"
					}
					ctx = context.WithValue(ctx, ctxKeySource{}, src)
				}
			}
			return next(ctx, method, req)
		}
	}
}

func (s *Service) Stop() {
	s.closeOwnedSSH()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.httpSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = s.httpSrv.Shutdown(ctx)
		cancel()
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}
	s.online = false
	_ = os.Remove(join(mustHome(), runtimeFile))
}

func mustHome() string {
	d, _ := homeDir()
	return d
}

type runtimeInfo struct {
	Port      int    `json:"port"`
	PID       int    `json:"pid"`
	StartedAt string `json:"startedAt"`
	HTTPURL   string `json:"httpUrl"`
}

func (s *Service) writeRuntime() error {
	// 不落盘 Token 明文（只存 SHA256）；端口与 URL 供本机调试
	info := runtimeInfo{
		Port:      s.settings.HTTPPort,
		PID:       os.Getpid(),
		StartedAt: s.startedAt.Format(time.RFC3339),
		HTTPURL:   s.httpURL(),
	}
	b, _ := json.MarshalIndent(info, "", "  ")
	return os.WriteFile(join(mustHome(), runtimeFile), b, 0600)
}

func (s *Service) httpURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d/mcp", s.settings.HTTPPort)
}

func (s *Service) lanURL() string {
	ip := firstLANIP()
	if ip == "" {
		return ""
	}
	return fmt.Sprintf("http://%s:%d/mcp", ip, s.settings.HTTPPort)
}

func firstLANIP() string {
	ifs, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifs {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}
			return ip.String()
		}
	}
	return ""
}

func (s *Service) GetStatus() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, cursorOK := cursorLinked()
	exe, _ := os.Executable()
	serverCount := 0
	if s.cfg != nil {
		serverCount = len(s.cfg.GetAllMachinesFromGlobal())
	}
	return Status{
		Enabled:        s.settings.Enabled,
		AutoStart:      s.settings.AutoStart,
		Online:         s.online,
		StdioOnline:    s.online,
		ObserverOnline: s.online,
		HTTPURL:        s.httpURL(),
		HTTPPort:       s.settings.HTTPPort,
		BindLAN:        s.settings.BindLAN,
		LANURL:         s.lanURL(),
		LocalAddr:      "127.0.0.1",
		StdioPath:      exe,
		CursorLinked:   cursorOK,
		TokenCount:     len(s.tokens.List()),
		ToolCount:      35,
		ServerCount:    serverCount,
		PendingCount:   len(s.approvals.List()),
		DefaultPolicy:  s.settings.DefaultPolicy,
		StartedAt:      s.startedAt.Format("2006-01-02 15:04:05"),
		Clients:        s.ListClientLinks(),
	}
}

func (s *Service) SetEmitter(fn func(event string, data any)) {
	s.emit = fn
	s.approvals.SetEmitter(fn)
}

func (s *Service) UpdateSettings(in Settings) error {
	s.mu.Lock()
	needRestart := s.settings.HTTPPort != in.HTTPPort || s.settings.BindLAN != in.BindLAN || s.settings.Enabled != in.Enabled
	s.settings = in
	_ = saveSettings(in)
	s.mu.Unlock()
	if needRestart {
		s.Stop()
		if in.Enabled {
			return s.StartHTTP()
		}
	}
	return nil
}

func (s *Service) GetSettings() Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.settings
}

func (s *Service) Snippets() ClientSnippet {
	exe, _ := os.Executable()
	placeholder := "<在接入时复制的 Token>"
	stdio := map[string]any{
		"mcpServers": map[string]any{
			"flashshell": map[string]any{
				"command": exe,
				"args":    []string{"--mcp-stdio"},
				"env": map[string]string{
					"FLASHSHELL_TOKEN":    placeholder,
					"FLASHSHELL_RPC_PORT": fmt.Sprintf("%d", s.settings.HTTPPort),
				},
			},
		},
	}
	httpCfg := map[string]any{
		"mcpServers": map[string]any{
			"flashshell": map[string]any{
				"url": s.httpURL(),
				"headers": map[string]string{
					"Authorization": "Bearer " + placeholder,
				},
			},
		},
	}
	sb, _ := json.MarshalIndent(stdio, "", "  ")
	hb, _ := json.MarshalIndent(httpCfg, "", "  ")
	toml := fmt.Sprintf("[mcp_servers.flashshell]\ncommand = %q\nargs = [\"--mcp-stdio\"]\nenv = { FLASHSHELL_TOKEN = %q, FLASHSHELL_RPC_PORT = %q }\n",
		exe, placeholder, fmt.Sprintf("%d", s.settings.HTTPPort))
	return ClientSnippet{
		StdioJSON: string(sb),
		HTTPJSON:  string(hb),
		TOML:      toml,
		HTTPURL:   s.httpURL(),
		HTTPAuth:  "Authorization: Bearer " + placeholder,
	}
}

func (s *Service) policyOf(m *define.Machine) string {
	if m != nil && strings.TrimSpace(m.AIPolicy) != "" {
		return normalizePolicy(m.AIPolicy)
	}
	if m != nil {
		// 历史机器未写 aiPolicy 时默认 disabled，避免默认可操作
		return PolicyDisabled
	}
	return normalizePolicy(s.settings.DefaultPolicy)
}

func (s *Service) serverMCPEnabled(m *define.Machine) bool {
	return normalizePolicy(s.policyOf(m)) != PolicyDisabled
}

func (s *Service) gate(ctx context.Context, tool, server, preview string, params any) (viaApproval bool, reason string, err error) {
	kind := toolKinds[tool]
	if kind == "" {
		kind = kindMutating
	}
	if kind == kindMeta {
		return false, "元工具自动放行", nil
	}

	if crypto.IsLocked() {
		why := "凭据库已锁定，请先在 FlashShell 解锁"
		return false, why, wrapErr("[denied]", why)
	}

	// 全局紧急停止 / 限时放行过期
	if ok, dec, why := s.evaluateGlobalAI(); !ok {
		return false, why, wrapErr("[denied]", why)
	} else if dec == "auto" && strings.Contains(why, "限时放行") {
		// armed：跳过档位拒绝，仍走致命/sudo/出站
		reason = why
	}

	var policy string
	var allow []string
	var allowSudo bool
	if server != "" {
		if tok, ok := s.activeToken(ctx); ok && !tok.SeesServer(server) {
			why := "该 Token 不可见服务器 " + server
			return false, why, wrapErr("[denied]", why)
		}
		m, err := s.machineByAlias(server)
		if err != nil {
			return false, err.Error(), err
		}
		if !s.serverMCPEnabled(m) {
			why := "该服务器当前 AI 策略为 disabled"
			return false, why, wrapErr("[denied]", why)
		}
		policy = s.policyOf(m)
		allow = m.AIAllowlist
		allowSudo = m.AIAllowSudo
	} else {
		policy = normalizePolicy(s.settings.DefaultPolicy)
	}

	armedBypass := false
	st := s.GetSettings()
	if !st.EmergencyStop && strings.ToLower(st.AIMode) == AIModeArmed {
		if t, ok := st.ArmedUntilTime(); ok && time.Now().Before(t) {
			armedBypass = true
		} else if st.ArmedUntil == "" {
			armedBypass = true
		}
	}

	if hit, why := lethalBlocked(preview); hit {
		return false, why, wrapErr("[blocked]", why)
	}
	if hit, why := commandBlocked(preview); hit {
		return false, why, wrapErr("[blocked]", why)
	}
	if pathBlocked(preview) {
		why := fmt.Sprintf("命中敏感路径黑名单: %s", strings.TrimSpace(preview))
		return false, why, wrapErr("[blocked]", why)
	}

	needApprove := false
	approveWhy := ""
	var outboundBad []string

	if hit, why := severeNeedsApproval(preview); hit {
		needApprove = true
		approveWhy = why
	}
	if kind == kindMutating && containsSudo(preview) {
		if !allowSudo {
			why := "该服务器未开启 AI sudo（含 sudo 强制审批）"
			return false, why, wrapErr("[denied]", why)
		}
		needApprove = true
		if approveWhy == "" {
			approveWhy = "含 sudo，强制人工审批"
		}
	}
	if ok, why, bad := s.checkOutbound(preview); !ok {
		needApprove = true
		outboundBad = bad
		if approveWhy == "" {
			approveWhy = why
		} else {
			approveWhy = approveWhy + "; " + why
		}
	}
	if hit, why := s.prov.CheckCommand(preview); hit {
		needApprove = true
		if approveWhy == "" {
			approveWhy = why
		} else {
			approveWhy = approveWhy + "; " + why
		}
	}

	if !armedBypass {
		pd := policyDecide(policy, kind, preview, allow)
		if !pd.Allow && !pd.NeedsApprove {
			return false, pd.Reason, wrapErr(pd.Prefix, pd.Reason)
		}
		if pd.NeedsApprove {
			needApprove = true
			if approveWhy == "" {
				approveWhy = pd.Reason
			} else {
				approveWhy = approveWhy + "; " + pd.Reason
			}
		}
		if pd.Allow && !needApprove {
			return false, pd.Reason, nil
		}
	} else if !needApprove {
		return false, "全局限时放行", nil
	}

	if needApprove {
		isDanger := containsSudo(preview)
		if hit, _ := severeNeedsApproval(preview); hit {
			isDanger = true
		}
		ok, rejectReason, aerr := s.approvals.Request(ctx, tool, server, preview, mustJSON(params), sourceFromCtx(ctx), approveWhy, outboundBad, isDanger)
		if aerr != nil {
			msg := aerr.Error()
			if strings.Contains(msg, "超时") {
				return false, "denied_by_timeout", wrapErr("[denied]", "审批超时，已自动拒绝")
			}
			if strings.Contains(msg, "取消") {
				return false, approveWhy, wrapErr("[cancelled]", strings.TrimPrefix(msg, "[approval] "))
			}
			return false, approveWhy, aerr
		}
		if !ok {
			why := "用户拒绝了该操作"
			if rejectReason != "" {
				why = why + ": " + rejectReason
			}
			return false, why, wrapErr("[denied]", why)
		}
		return true, approveWhy + " → 人工已批准", nil
	}
	return false, reason, nil
}

func (s *Service) record(ctx context.Context, tool, server string, params any, result string, decision, reason string, started time.Time, err error) {
	if err != nil && (decision == "success" || decision == "auto" || decision == "approved" || decision == "") {
		decision = classifyDecision(err.Error())
		if reason == "" {
			reason = err.Error()
		}
		result = err.Error()
	}
	if decision == "success" || decision == "" {
		decision = "auto"
	}
	if decision == "pending" || decision == "approval" {
		decision = "cancelled"
	}
	// 只读类工具喂溯源缓冲（脱敏后）
	switch tool {
	case "sftp_read", "tail_log", "ssh_exec", "ssh_exec_script":
		s.prov.Append(redactText(result))
	}
	s.audit.Append(AuditEntry{
		Source:     sourceFromCtx(ctx),
		Caller:     sourceFromCtx(ctx),
		Tool:       tool,
		Module:     toolModule(tool),
		Server:     server,
		Params:     clip(mustJSON(params), 4000),
		Result:     clip(redactText(result), 8000),
		Decision:   decision,
		Reason:     clip(reason, 500),
		DurationMs: time.Since(started).Milliseconds(),
	})
}

func classifyDecision(msg string) string {
	switch {
	case strings.HasPrefix(msg, "[denied]"):
		return "denied"
	case strings.HasPrefix(msg, "[blocked]"):
		return "blocked"
	case strings.HasPrefix(msg, "[cancelled]"):
		return "cancelled"
	case strings.HasPrefix(msg, "[approval]"):
		return "cancelled"
	case strings.HasPrefix(msg, "[timeout]"):
		return "denied"
	case strings.HasPrefix(msg, "[notfound]"):
		return "denied"
	default:
		return "denied"
	}
}
