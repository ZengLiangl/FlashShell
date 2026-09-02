package mcp

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// GlobalAIMode 全局 AI 总开关（凌驾于各服务器档位）
const (
	AIModeNormal    = "normal"    // 按各服务器档位
	AIModeArmed     = "armed"     // 限时统一放行（仍受黑名单/sudo）
	AIModeEmergency = "emergency" // 紧急停止（元工具除外）
)

// GateResult 策略引擎一次判定结果（对齐 Reeve 决策类型）
type GateResult struct {
	Allow        bool
	ViaApproval  bool
	Decision     string // auto | approved | denied | blocked | cancelled（失败时）
	Reason       string
	Prefix       string // [denied]/[blocked]/[approval]
	NeedsApprove bool
}

func (s *Service) evaluateGlobalAI() (ok bool, decision, reason string) {
	st := s.GetSettings()
	mode := strings.ToLower(strings.TrimSpace(st.AIMode))
	if mode == "" {
		mode = AIModeNormal
	}
	if st.EmergencyStop || mode == AIModeEmergency {
		return false, "denied", "全局紧急停止：拒绝所有非元工具 AI 调用"
	}
	if mode == AIModeArmed {
		if st.ArmedUntil == "" {
			return true, "auto", "全局限时放行（未设到期时间）"
		}
		until, err := time.Parse(time.RFC3339, st.ArmedUntil)
		if err != nil || time.Now().After(until) {
			return false, "denied", "全局限时放行已过期"
		}
		return true, "auto", "全局限时放行至 " + until.Format("2006-01-02 15:04:05")
	}
	return true, "", ""
}

// lethalCommandRules 永久拦截（任何档位，含 trusted）
var lethalCommandRules = []struct {
	re  *regexp.Regexp
	why string
}{
	{regexp.MustCompile(`(?i)rm\s+(-[a-zA-Z]*f[a-zA-Z]*|--force).*/(\s|$)`), "命中致命黑名单: rm -rf /"},
	{regexp.MustCompile(`(?i)rm\s+-[a-zA-Z]*r[a-zA-Z]*f[a-zA-Z]*\s+(/|/\*|/\s+\*)`), "命中致命黑名单: rm -rf /"},
	{regexp.MustCompile(`(?i)mkfs(\.\w+)?\s`), "命中致命黑名单: mkfs"},
	{regexp.MustCompile(`(?i)dd\s+.*\bof=/dev/`), "命中致命黑名单: dd of=/dev"},
	{regexp.MustCompile(`:\(\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;`), "命中致命黑名单: fork bomb"},
	{regexp.MustCompile(`(?i)\bDROP\s+(DATABASE|SCHEMA)\b`), "命中致命黑名单: DROP DATABASE"},
	{regexp.MustCompile(`(?i)\bFLUSHALL\b`), "命中致命黑名单: FLUSHALL"},
	{regexp.MustCompile(`(?i)docker\s+system\s+prune\s+-a\s+--volumes`), "命中致命黑名单: docker prune volumes"},
	{regexp.MustCompile(`(?i)\bkubeadm\s+reset\b`), "命中致命黑名单: kubeadm reset"},
	{regexp.MustCompile(`(?i)\bpg_resetwal\b`), "命中致命黑名单: pg_resetwal"},
}

// lethalBlocked 永久拦截（任何档位）
func lethalBlocked(cmd string) (bool, string) {
	s := strings.TrimSpace(cmd)
	if s == "" {
		return false, ""
	}
	for _, r := range lethalCommandRules {
		if r.re.MatchString(s) {
			return true, r.why
		}
	}
	if why := matchCustomDangerDetail(s); why != "" {
		return true, why
	}
	return false, ""
}

// severeNeedsApproval 严重命令 → 强制审批（非永久拦）
func severeNeedsApproval(cmd string) (bool, string) {
	s := strings.TrimSpace(cmd)
	rules := []struct {
		re  *regexp.Regexp
		why string
	}{
		{regexp.MustCompile(`(?i)\b(shutdown|poweroff|halt|reboot)\b`), "严重操作需审批: 关机/重启"},
		{regexp.MustCompile(`(?i)\bsystemctl\s+poweroff\b`), "严重操作需审批: systemctl poweroff"},
		{regexp.MustCompile(`(?i)\b1pctl\s+(uninstall|reset)\b`), "严重操作需审批: 1pctl"},
		{regexp.MustCompile(`(?i)\bSHUTDOWN\b`), "严重操作需审批: DB SHUTDOWN"},
	}
	for _, r := range rules {
		if r.re.MatchString(s) {
			return true, r.why
		}
	}
	return false, ""
}

func commandAllowedRegex(command string, allowlist []string) bool {
	cmd := strings.TrimSpace(command)
	if cmd == "" || len(allowlist) == 0 {
		return false
	}
	for _, a := range allowlist {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if strings.HasPrefix(cmd, a) {
			return true
		}
		if re, err := regexp.Compile(a); err == nil && re.MatchString(cmd) {
			return true
		}
	}
	return false
}

func policyDecide(policy, kind, command string, allowlist []string) GateResult {
	policy = normalizePolicy(policy)
	if kind == kindMeta {
		return GateResult{Allow: true, Decision: "auto", Reason: "元工具自动放行"}
	}
	switch policy {
	case PolicyDisabled:
		return GateResult{Decision: "denied", Prefix: "[denied]", Reason: "该服务器当前 AI 策略为 disabled"}
	case PolicyReadonly:
		if kind == kindMutating {
			return GateResult{Decision: "denied", Prefix: "[denied]", Reason: "readonly 档禁止改动型操作"}
		}
		return GateResult{Allow: true, Decision: "auto", Reason: "readonly 档只读工具自动放行"}
	case PolicyApproval:
		return GateResult{NeedsApprove: true, Prefix: "[approval]", Reason: "approval 档：该操作需人工审批"}
	case PolicyAllowlist:
		if commandAllowedRegex(command, allowlist) {
			return GateResult{Allow: true, Decision: "auto", Reason: "allowlist 正则命中，自动放行"}
		}
		return GateResult{NeedsApprove: true, Prefix: "[approval]", Reason: "allowlist 未命中，升级审批"}
	default: // trusted
		return GateResult{Allow: true, Decision: "auto", Reason: "trusted 档自动放行"}
	}
}

// ---------- 出站白名单 ----------

var urlInCmd = regexp.MustCompile(`(?i)https?://[^\s'"\\|;<>]+|` +
	`(?:curl|wget|fetch)\s+[^\n]*?\b((?:[a-z0-9-]+\.)+[a-z]{2,}(?::\d+)?(?:/[^\s]*)?)`)

var hostLike = regexp.MustCompile(`(?i)\b((?:\d{1,3}\.){3}\d{1,3}|(?:[a-z0-9-]+\.)+[a-z]{2,})(?::\d{2,5})?\b`)

func builtinOutboundHosts() []string {
	return []string{
		"debian.org", "ubuntu.com", "centos.org", "rocky.org", "alibaba.com", "aliyun.com",
		"tencent.com", "myhuaweicloud.com", "docker.com", "docker.io", "ghcr.io", "quay.io",
		"npmjs.org", "pypi.org", "golang.org", "proxy.golang.org", "maven.org", "apache.org",
		"github.com", "gitlab.com", "gitee.com", "npm.taobao.org", "npmmirror.com",
		"mirrors.ustc.edu.cn", "mirrors.tuna.tsinghua.edu.cn", "mirrors.aliyun.com",
		"letsencrypt.org", "cloudflare.com",
	}
}

func extractOutboundEndpoints(cmd string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(h string) {
		h = strings.ToLower(strings.TrimSpace(h))
		h = strings.TrimPrefix(h, "http://")
		h = strings.TrimPrefix(h, "https://")
		if i := strings.IndexAny(h, "/?#"); i >= 0 {
			h = h[:i]
		}
		if h == "" {
			return
		}
		if _, ok := seen[h]; ok {
			return
		}
		seen[h] = struct{}{}
		out = append(out, h)
	}
	for _, m := range urlInCmd.FindAllString(cmd, -1) {
		if u, err := url.Parse(m); err == nil && u.Host != "" {
			add(u.Host)
			continue
		}
		add(m)
	}
	// curl/wget 裸 host
	if regexp.MustCompile(`(?i)\b(curl|wget)\b`).MatchString(cmd) {
		for _, m := range hostLike.FindAllString(cmd, -1) {
			add(m)
		}
	}
	return out
}

func isPrivateOrLocalHost(host string) bool {
	h := host
	if i := strings.Index(h, ":"); i >= 0 {
		h = h[:i]
	}
	if h == "localhost" || h == "127.0.0.1" || h == "::1" {
		return true
	}
	ip := net.ParseIP(h)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

func hostAllowed(host string, customs []string, enabled bool) bool {
	if !enabled {
		return true
	}
	h := strings.ToLower(host)
	if i := strings.Index(h, ":"); i >= 0 {
		h = h[:i]
	}
	if isPrivateOrLocalHost(h) {
		return true
	}
	// 裸 IP 永远不命中公网白名单
	if net.ParseIP(h) != nil {
		return false
	}
	check := append(append([]string{}, customs...), builtinOutboundHosts()...)
	for _, allow := range check {
		allow = strings.ToLower(strings.TrimSpace(allow))
		if allow == "" {
			continue
		}
		if u, err := url.Parse(allow); err == nil && u.Host != "" {
			allow = strings.ToLower(u.Host)
		}
		if allow == h || strings.HasSuffix(h, "."+allow) {
			return true
		}
	}
	return false
}

func (s *Service) checkOutbound(cmd string) (ok bool, reason string, bad []string) {
	st := s.GetSettings()
	enabled := st.OutboundAllowlistEnabled
	// 默认启用：未显式关闭即 true；零值 yaml 也可能是 false，用指针更好，这里用 "未配置当启用"
	// Settings 里用 *bool 复杂，约定：字段 OutboundAllowlistDisabled
	if st.OutboundAllowlistDisabled {
		return true, "", nil
	}
	_ = enabled
	eps := extractOutboundEndpoints(cmd)
	if len(eps) == 0 {
		return true, "", nil
	}
	var badList []string
	for _, ep := range eps {
		if !hostAllowed(ep, st.OutboundHosts, true) {
			badList = append(badList, ep)
		}
	}
	if len(badList) == 0 {
		return true, "", nil
	}
	return false, fmt.Sprintf("出站地址不在白名单，升级审批: %s", strings.Join(badList, ", ")), badList
}

// ---------- 溯源检测 ----------

type provenanceBuf struct {
	mu   sync.Mutex
	text string
	max  int
}

func newProvenance() *provenanceBuf {
	return &provenanceBuf{max: 256 * 1024}
}

func (p *provenanceBuf) Append(s string) {
	if p == nil || s == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.text += "\n" + s
	if len(p.text) > p.max {
		p.text = p.text[len(p.text)-p.max:]
	}
}

func (p *provenanceBuf) Clear() {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.text = ""
	p.mu.Unlock()
}

func (p *provenanceBuf) CheckCommand(cmd string) (hit bool, reason string) {
	if p == nil || strings.TrimSpace(cmd) == "" {
		return false, ""
	}
	p.mu.Lock()
	buf := p.text
	p.mu.Unlock()
	if buf == "" {
		return false, ""
	}
	// 公网地址出现在缓冲里又出现在命令里
	for _, ep := range extractOutboundEndpoints(cmd) {
		h := ep
		if i := strings.Index(h, ":"); i >= 0 {
			h = h[:i]
		}
		if isPrivateOrLocalHost(h) {
			continue
		}
		if strings.Contains(buf, h) || strings.Contains(buf, ep) {
			return true, fmt.Sprintf("命令要连接的地址 %s 正是 AI 刚从服务器读到的内容 —— 符合「读到被投毒的内容后照抄执行」的特征", ep)
		}
	}
	// ≥48 字符连续重合
	cmd = strings.TrimSpace(cmd)
	if len(cmd) >= 48 {
		for i := 0; i+48 <= len(cmd); i += 8 {
			chunk := cmd[i : i+48]
			if strings.Contains(buf, chunk) {
				return true, "命令与刚读到的内容存在 ≥48 字符连续重合，疑似照抄执行"
			}
		}
	}
	return false, ""
}
