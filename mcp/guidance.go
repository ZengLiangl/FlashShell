package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	guidanceVersion = "flashshell-guidance-v1"
	guidanceBegin   = "<!-- FLASHSHELL:GUIDANCE:BEGIN -->"
	guidanceEnd     = "<!-- FLASHSHELL:GUIDANCE:END -->"
)

type GuidanceStatus struct {
	OK      bool   `json:"ok"`
	Stale   bool   `json:"stale"`
	Path    string `json:"path"`
	Version string `json:"version"`
}

func guidancePath(clientID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch clientID {
	case "claude-code":
		return filepath.Join(home, ".claude", "CLAUDE.md"), nil
	case "codex":
		return filepath.Join(home, ".codex", "AGENTS.md"), nil
	case "opencode":
		return filepath.Join(home, ".config", "opencode", "AGENTS.md"), nil
	case "cursor":
		return "", fmt.Errorf("Cursor 请使用复制规则弹窗")
	default:
		return "", fmt.Errorf("不支持的客户端: %s", clientID)
	}
}

func guidanceBody(serverAliases []string) string {
	var b strings.Builder
	b.WriteString(guidanceBegin)
	b.WriteString("\n")
	b.WriteString("<!-- version: ")
	b.WriteString(guidanceVersion)
	b.WriteString(" -->\n\n")
	b.WriteString(guidanceMarkdown(serverAliases))
	b.WriteString("\n")
	b.WriteString(guidanceEnd)
	b.WriteString("\n")
	return b.String()
}

// guidanceMarkdown 纯规则正文（无 HTML 标记区），供 Cursor 复制粘贴。
func guidanceMarkdown(serverAliases []string) string {
	var b strings.Builder
	b.WriteString("# FlashShell MCP 操作规则\n\n")
	b.WriteString("- 远程运维 / SSH / 部署 / 查日志 / 装服务：**一律走 FlashShell MCP 工具**，不要裸调系统 `ssh` / `scp`。\n")
	b.WriteString("- 永远先 `list_servers`，`server` 参数只用返回的 **alias**。\n")
	b.WriteString("- 凭据明文你永远拿不到；输出里的 `[REDACTED:…]` 是脱敏占位，不要当密码用、不要反复重生成。\n")
	b.WriteString("- 遇到 `[approval]`：告知用户在 FlashShell 审批队列放行，**不要疯狂重试**。\n")
	b.WriteString("- `[denied]` / `[blocked]`：换合规命令或请用户调服务器 AI 档位；不要硬刚危险命令。\n")
	b.WriteString("- 装带密码的服务用 `install_app` / `install_with_secret`，不要 `ssh_exec` 里 `openssl rand` 造密码。\n")
	if len(serverAliases) > 0 {
		b.WriteString("\n当前可见服务器别名快照：`")
		b.WriteString(strings.Join(serverAliases, "`, `"))
		b.WriteString("`\n")
	}
	return b.String()
}

func guidanceStatus(clientID string) GuidanceStatus {
	path, err := guidancePath(clientID)
	if err != nil || path == "" {
		return GuidanceStatus{}
	}
	st := GuidanceStatus{Path: path, Version: guidanceVersion}
	b, err := os.ReadFile(path)
	if err != nil {
		st.Stale = true
		return st
	}
	text := string(b)
	if !strings.Contains(text, guidanceBegin) || !strings.Contains(text, guidanceEnd) {
		st.Stale = true
		return st
	}
	if !strings.Contains(text, "version: "+guidanceVersion) {
		st.Stale = true
		return st
	}
	st.OK = true
	return st
}

func (s *Service) GuidancePreview(clientID string) string {
	aliases := s.allAliases()
	// Cursor 走复制粘贴，不需要 BEGIN/END 标记区
	if clientID == "cursor" {
		return guidanceMarkdown(aliases)
	}
	return guidanceBody(aliases)
}

func (s *Service) allAliases() []string {
	var out []string
	if s.cfg == nil {
		return out
	}
	for _, m := range s.cfg.GetAllMachinesFromGlobal() {
		if n := strings.TrimSpace(m.Name); n != "" {
			out = append(out, n)
		}
	}
	return out
}

func (s *Service) WriteGuidance(clientID string) (GuidanceStatus, error) {
	if clientID == "cursor" {
		return GuidanceStatus{}, fmt.Errorf("Cursor 请复制规则文本手动粘贴")
	}
	path, err := guidancePath(clientID)
	if err != nil {
		return GuidanceStatus{}, err
	}
	body := guidanceBody(s.allAliases())
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return GuidanceStatus{}, err
	}
	var existing string
	if b, err := os.ReadFile(path); err == nil {
		existing = string(b)
		bak := path + ".flashshell.bak"
		_ = os.WriteFile(bak, b, 0644)
	}
	next := upsertGuidanceBlock(existing, body)
	tmp := path + ".tmp." + fmt.Sprintf("%d", time.Now().UnixNano())
	if err := os.WriteFile(tmp, []byte(next), 0644); err != nil {
		return GuidanceStatus{}, err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return GuidanceStatus{}, err
	}
	return guidanceStatus(clientID), nil
}

func upsertGuidanceBlock(existing, block string) string {
	start := strings.Index(existing, guidanceBegin)
	end := strings.Index(existing, guidanceEnd)
	if start >= 0 && end > start {
		end += len(guidanceEnd)
		// 吃掉 END 后紧跟换行
		for end < len(existing) && (existing[end] == '\n' || existing[end] == '\r') {
			end++
		}
		return existing[:start] + block + existing[end:]
	}
	if strings.TrimSpace(existing) == "" {
		return block
	}
	return strings.TrimRight(existing, "\n") + "\n\n" + block
}
