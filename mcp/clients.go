package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const serverKey = "flashshell"

// ClientLink 单个 AI 客户端的接入状态
type ClientLink struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Desc         string `json:"desc"`
	Config       string `json:"config"`
	Linked       bool   `json:"linked"`
	ConfigPath   string `json:"configPath"`
	Installed    bool   `json:"installed"`    // 本机是否检测到客户端目录/配置
	GuidanceOK   bool   `json:"guidanceOk"`   // 规则已写入且版本匹配
	GuidancePath string `json:"guidancePath"` // 规则文件路径
}

// InstallOpts 一键接入参数（对齐 Reeve 接入弹窗）
type InstallOpts struct {
	TokenName string   `json:"tokenName"`
	Servers   []string `json:"servers"`
	CIDRs     []string `json:"cidrs"`
}

func (s *Service) stdioEntry(plain string) map[string]any {
	exe, _ := os.Executable()
	env := map[string]any{}
	if strings.TrimSpace(plain) != "" {
		env["FLASHSHELL_TOKEN"] = plain
	}
	if s.settings.HTTPPort > 0 {
		env["FLASHSHELL_RPC_PORT"] = fmt.Sprintf("%d", s.settings.HTTPPort)
	}
	entry := map[string]any{
		"command": exe,
		"args":    []string{"--mcp-stdio"},
	}
	if len(env) > 0 {
		entry["env"] = env
	}
	return entry
}

func (s *Service) httpEntry(plain string) map[string]any {
	headers := map[string]any{}
	if strings.TrimSpace(plain) != "" {
		headers["Authorization"] = "Bearer " + plain
	}
	return map[string]any{
		"url":     s.httpURL(),
		"headers": headers,
	}
}

func homeJoin(parts ...string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(append([]string{home}, parts...)...), nil
}

func claudeJSONPath() (string, error) { return homeJoin(".claude.json") }
func codexTOMLPath() (string, error)  { return homeJoin(".codex", "config.toml") }
func openCodeJSONPath() (string, error) {
	if p, err := homeJoin(".config", "opencode", "opencode.json"); err == nil {
		return p, nil
	}
	return homeJoin(".opencode", "opencode.json")
}

func pathExists(p string) bool {
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}

func clientDetected(id string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	switch id {
	case "claude-code":
		return pathExists(filepath.Join(home, ".claude.json")) || pathExists(filepath.Join(home, ".claude"))
	case "codex":
		return pathExists(filepath.Join(home, ".codex"))
	case "cursor":
		return pathExists(filepath.Join(home, ".cursor"))
	case "opencode":
		return pathExists(filepath.Join(home, ".config", "opencode")) || pathExists(filepath.Join(home, ".opencode"))
	default:
		return false
	}
}

func (s *Service) ListClientLinks() []ClientLink {
	out := make([]ClientLink, 0, 4)
	for _, id := range []string{"claude-code", "codex", "cursor", "opencode"} {
		out = append(out, s.clientLinkOf(id))
	}
	return out
}

func (s *Service) clientLinkOf(id string) ClientLink {
	var c ClientLink
	switch id {
	case "claude-code":
		p, _ := claudeJSONPath()
		c = ClientLink{
			ID: "claude-code", Name: "Claude Code", Desc: "Anthropic 官方 CLI · ~/.claude.json · stdio",
			Config: "stdio", ConfigPath: p, Linked: claudeLinked(),
		}
	case "codex":
		p, _ := codexTOMLPath()
		c = ClientLink{
			ID: "codex", Name: "Codex", Desc: "OpenAI Codex CLI · config.toml · stdio",
			Config: "stdio", ConfigPath: p, Linked: codexLinked(),
		}
	case "cursor":
		p, _ := cursorMCPPath()
		c = ClientLink{
			ID: "cursor", Name: "Cursor", Desc: "AI 编辑器 · mcp.json · Streamable HTTP",
			Config: "http", ConfigPath: p, Linked: cursorLinkedOK(),
		}
	case "opencode":
		p, _ := openCodeJSONPath()
		c = ClientLink{
			ID: "opencode", Name: "OpenCode", Desc: "OpenCode CLI · stdio",
			Config: "stdio", ConfigPath: p, Linked: openCodeLinked(),
		}
	default:
		return ClientLink{ID: id}
	}
	c.Installed = clientDetected(id)
	st := guidanceStatus(id)
	c.GuidanceOK = st.OK
	c.GuidancePath = st.Path
	return c
}

func (s *Service) InstallClient(id string) (IssuedToken, error) {
	return s.InstallClientWith(id, InstallOpts{})
}

func (s *Service) InstallClientWith(id string, opts InstallOpts) (IssuedToken, error) {
	name := strings.TrimSpace(opts.TokenName)
	if name == "" {
		name = s.clientLinkOf(id).Name + " on " + time.Now().Format("Jan 2")
	}
	clientID := id
	switch id {
	case "claude-code":
		clientID = "claude-code"
	case "codex", "cursor", "opencode":
		clientID = id
	default:
		return IssuedToken{}, fmt.Errorf("未知客户端: %s", id)
	}
	issued, err := s.tokens.IssueForClient(IssueOpts{
		Name: name, Client: clientID, Servers: opts.Servers, CIDRs: opts.CIDRs,
	})
	if err != nil {
		return IssuedToken{}, err
	}
	var ierr error
	switch id {
	case "claude-code":
		ierr = s.installClaude(issued.Plaintext)
	case "codex":
		ierr = s.installCodex(issued.Plaintext)
	case "cursor":
		ierr = s.installCursor(issued.Plaintext)
	case "opencode":
		ierr = s.installOpenCode(issued.Plaintext)
	}
	if ierr != nil {
		_, _ = s.tokens.Revoke(issued.ID)
		return IssuedToken{}, ierr
	}
	s.audit.Append(AuditEntry{
		Source: "FlashShell UI", Caller: "human", Tool: "token_issue", Module: "token",
		Params: mustJSON(map[string]any{"client": clientID, "name": name, "servers": opts.Servers, "cidrs": opts.CIDRs}),
		Result: "ok", Decision: "auto", Reason: "一键接入签发 scoped token",
	})
	return issued, nil
}

func (s *Service) UninstallClient(id string) error {
	switch id {
	case "claude-code":
		return s.uninstallClaude()
	case "codex":
		return s.uninstallCodex()
	case "cursor":
		return s.UninstallCursor()
	case "opencode":
		return s.uninstallOpenCode()
	default:
		return fmt.Errorf("未知客户端: %s", id)
	}
}

func (s *Service) RefreshClient(id string) (IssuedToken, error) {
	// 刷新 = 重新签发并写配置；尽量保留原 token 的可见服务器 / CIDR
	opts := InstallOpts{TokenName: s.clientLinkOf(id).Name + " refresh"}
	for _, t := range s.tokens.List() {
		if strings.EqualFold(t.Client, id) {
			opts.Servers = append([]string{}, t.Servers...)
			opts.CIDRs = append([]string{}, t.CIDRs...)
			if strings.TrimSpace(t.Name) != "" {
				opts.TokenName = t.Name
			}
			break
		}
	}
	return s.InstallClientWith(id, opts)
}

func cursorLinkedOK() bool {
	_, ok := cursorLinked()
	return ok
}

func claudeLinked() bool {
	path, err := claudeJSONPath()
	if err != nil {
		return false
	}
	root, err := readJSONFile(path)
	if err != nil {
		return false
	}
	servers, _ := root["mcpServers"].(map[string]any)
	if servers == nil {
		return false
	}
	_, ok := servers[serverKey]
	return ok
}

func (s *Service) installClaude(plain string) error {
	path, err := claudeJSONPath()
	if err != nil {
		return err
	}
	root, err := readJSONFile(path)
	if err != nil {
		return err
	}
	servers, _ := root["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
		root["mcpServers"] = servers
	}
	servers[serverKey] = s.stdioEntry(plain)
	return writeJSONFile(path, root)
}

func (s *Service) uninstallClaude() error {
	path, err := claudeJSONPath()
	if err != nil {
		return err
	}
	root, err := readJSONFile(path)
	if err != nil {
		return err
	}
	servers, _ := root["mcpServers"].(map[string]any)
	if servers != nil {
		delete(servers, serverKey)
	}
	_ = s.tokens.RevokeByClient("claude-code")
	_ = s.tokens.RevokeByClient("claude")
	return writeJSONFile(path, root)
}

func openCodeLinked() bool {
	path, err := openCodeJSONPath()
	if err != nil {
		return false
	}
	root, err := readJSONFile(path)
	if err != nil {
		return false
	}
	mcp, _ := root["mcp"].(map[string]any)
	if mcp == nil {
		servers, _ := root["mcpServers"].(map[string]any)
		if servers == nil {
			return false
		}
		_, ok := servers[serverKey]
		return ok
	}
	_, ok := mcp[serverKey]
	return ok
}

func (s *Service) installOpenCode(plain string) error {
	path, err := openCodeJSONPath()
	if err != nil {
		return err
	}
	root, err := readJSONFile(path)
	if err != nil {
		return err
	}
	mcp, _ := root["mcp"].(map[string]any)
	if mcp == nil {
		mcp = map[string]any{}
		root["mcp"] = mcp
	}
	entry := s.stdioEntry(plain)
	entry["type"] = "local"
	mcp[serverKey] = entry
	return writeJSONFile(path, root)
}

func (s *Service) uninstallOpenCode() error {
	path, err := openCodeJSONPath()
	if err != nil {
		return err
	}
	root, err := readJSONFile(path)
	if err != nil {
		return err
	}
	if mcp, ok := root["mcp"].(map[string]any); ok {
		delete(mcp, serverKey)
	}
	if servers, ok := root["mcpServers"].(map[string]any); ok {
		delete(servers, serverKey)
	}
	_ = s.tokens.RevokeByClient("opencode")
	return writeJSONFile(path, root)
}

func codexLinked() bool {
	path, err := codexTOMLPath()
	if err != nil {
		return false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(b), "[mcp_servers."+serverKey+"]") ||
		strings.Contains(string(b), "[mcp_servers.\""+serverKey+"\"]")
}

func (s *Service) installCodex(plain string) error {
	path, err := codexTOMLPath()
	if err != nil {
		return err
	}
	exe, _ := os.Executable()
	exe = strings.ReplaceAll(exe, `\`, `\\`)
	var bld strings.Builder
	fmt.Fprintf(&bld, "\n[mcp_servers.%s]\ncommand = %q\nargs = [\"--mcp-stdio\"]\n", serverKey, exe)
	if strings.TrimSpace(plain) != "" {
		fmt.Fprintf(&bld, "env = { FLASHSHELL_TOKEN = %q, FLASHSHELL_RPC_PORT = %q }\n",
			plain, fmt.Sprintf("%d", s.settings.HTTPPort))
	}
	b, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	text := string(b)
	if codexLinked() {
		text = stripCodexBlock(text)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.TrimRight(text, "\n")+bld.String()), 0644)
}

func (s *Service) uninstallCodex() error {
	path, err := codexTOMLPath()
	if err != nil {
		return err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	text := stripCodexBlock(string(b))
	_ = s.tokens.RevokeByClient("codex")
	return os.WriteFile(path, []byte(text), 0644)
}

func stripCodexBlock(text string) string {
	lines := strings.Split(text, "\n")
	var out []string
	skip := false
	header := "[mcp_servers." + serverKey + "]"
	headerQ := "[mcp_servers.\"" + serverKey + "\"]"
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "[") && strings.HasSuffix(trim, "]") {
			skip = trim == header || trim == headerQ
			if skip {
				continue
			}
		}
		if skip {
			if trim == "" || strings.Contains(trim, "=") {
				continue
			}
		}
		out = append(out, line)
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n") + "\n"
}
