package mcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"
)

func (s *Service) listServers(ctx context.Context) (any, error) {
	out := make([]map[string]any, 0)
	tok, hasTok := s.activeToken(ctx)
	if s.cfg != nil {
		for _, m := range s.cfg.GetAllMachinesFromGlobal() {
			if !s.serverMCPEnabled(&m) {
				continue
			}
			if hasTok && !tok.SeesServer(m.Name) && !tok.SeesServer(m.ID) {
				continue
			}
			d := s.displayMachine(m)
			tags := d.Tags
			if tags == nil {
				tags = []string{}
			}
			port := d.Port
			if port <= 0 {
				port = 22
			}
			out = append(out, map[string]any{
				"alias":     d.Name,
				"host":      d.Host,
				"port":      port,
				"username":  d.User,
				"tags":      tags,
				"aiPolicy":  s.policyOf(&m),
				"allowSudo": m.AIAllowSudo,
			})
		}
	}
	// Reeve 契约：list_servers 返回 JSON 数组
	return out, nil
}

func (s *Service) handleSSHExec(ctx context.Context, a SshExecArgs) (any, error) {
	to := clampTimeout(a.TimeoutSecs, 30, 1, 600)
	res, err := s.execSSH(a.Server, a.Command, to)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *Service) handleSSHExecMulti(ctx context.Context, a SshExecMultiArgs) (any, error) {
	if len(a.Servers) == 0 || len(a.Servers) > 50 {
		return nil, wrapErr("[denied]", "servers 数量须为 1..=50")
	}
	type one struct {
		Server string `json:"server"`
		ExecResult
		Error string `json:"error,omitempty"`
	}
	out := make([]one, len(a.Servers))
	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup
	var mu sync.Mutex
	sum := map[string]int{"ok": 0, "error": 0, "denied": 0, "blocked": 0, "timeout": 0}
	for i, sv := range a.Servers {
		wg.Add(1)
		go func(i int, sv string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			item := one{Server: sv}
			if _, _, err := s.gate(ctx, "ssh_exec", sv, a.Command, a); err != nil {
				item.Error = err.Error()
				mu.Lock()
				sum[classifyDecision(err.Error())]++
				if _, ok := sum[classifyDecision(err.Error())]; !ok {
					sum["error"]++
				}
				mu.Unlock()
				out[i] = item
				return
			}
			res, err := s.execSSH(sv, a.Command, clampTimeout(a.TimeoutSecs, 30, 1, 600))
			if err != nil {
				item.Error = err.Error()
				mu.Lock()
				sum[classifyDecision(err.Error())]++
				mu.Unlock()
			} else {
				item.ExecResult = res
				mu.Lock()
				if res.ExitCode == 0 {
					sum["ok"]++
				} else {
					sum["error"]++
				}
				mu.Unlock()
			}
			out[i] = item
		}(i, sv)
	}
	wg.Wait()
	return map[string]any{"results": out, "summary": sum}, nil
}

func (s *Service) handleSSHExecScript(ctx context.Context, a SshExecScriptArgs) (any, error) {
	interp := "bash"
	if a.Interpreter != nil && strings.TrimSpace(*a.Interpreter) != "" {
		interp = strings.TrimSpace(*a.Interpreter)
	}
	switch interp {
	case "bash", "sh", "python3", "python":
	default:
		return nil, wrapErr("[denied]", "interpreter 仅支持 bash/sh/python3/python")
	}
	b64 := base64.StdEncoding.EncodeToString([]byte(a.Script))
	var cmd string
	if interp == "python" || interp == "python3" {
		cmd = fmt.Sprintf("%s -c \"import base64,os; exec(base64.b64decode('%s'))\"", interp, b64)
	} else {
		cmd = fmt.Sprintf("%s -lc 'echo %s | base64 -d | %s'", interp, b64, interp)
	}
	res, err := s.execSSH(a.Server, cmd, clampTimeout(a.TimeoutSecs, 60, 1, 600))
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *Service) handleSystemInfo(_ context.Context, a ServerOnly) (any, error) {
	res, err := s.execSSH(a.Server, "uname -a; hostnamectl; uptime", 30*time.Second)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *Service) handleDiskUsage(_ context.Context, a DiskUsageArgs) (any, error) {
	cmd := "df -hT"
	if a.Path != nil && strings.TrimSpace(*a.Path) != "" {
		p := strings.TrimSpace(*a.Path)
		if !safePathChars(p) {
			return nil, wrapErr("[denied]", "path 含非法字符")
		}
		cmd = "df -hT -- " + shellQuote(p)
	}
	return s.execSSH(a.Server, cmd, 30*time.Second)
}

func (s *Service) handlePortCheck(_ context.Context, a PortCheckArgs) (any, error) {
	if a.Port < 1 || a.Port > 65535 {
		return nil, wrapErr("[denied]", "端口须为 1..=65535")
	}
	cmd := fmt.Sprintf("ss -tlnH sport = :%d || netstat -tln 2>/dev/null | grep ':%d '", a.Port, a.Port)
	return s.execSSH(a.Server, cmd, 20*time.Second)
}

func (s *Service) handleServiceStatus(_ context.Context, a ServiceStatusArgs) (any, error) {
	if !safeServiceName(a.Service) {
		return nil, wrapErr("[denied]", "service 名非法")
	}
	cmd := fmt.Sprintf("systemctl status %s --no-pager --lines=20", a.Service)
	return s.execSSH(a.Server, cmd, 20*time.Second)
}

func (s *Service) handleTailLog(_ context.Context, a TailLogArgs) (any, error) {
	if !safePathChars(a.Path) {
		return nil, wrapErr("[denied]", "path 含非法字符")
	}
	n := int64(200)
	if a.Lines != nil && *a.Lines > 0 {
		n = *a.Lines
	}
	if n > 5000 {
		n = 5000
	}
	cmd := fmt.Sprintf("tail -n %d -- %s", n, shellQuote(a.Path))
	return s.execSSH(a.Server, cmd, 30*time.Second)
}

func (s *Service) handleCertList(_ context.Context, a ServerOnly) (any, error) {
	cmd := `sh -c 'for d in /etc/letsencrypt/live /etc/ssl/certs /opt/flashshell/openresty/certs; do [ -d "$d" ] && echo "== $d" && ls -1 "$d" 2>/dev/null; done; (command -v openssl >/dev/null && echo PEM:; find /etc/letsencrypt/live -name cert.pem 2>/dev/null | while read f; do echo "$f"; openssl x509 -in "$f" -noout -subject -issuer -enddate 2>/dev/null; done)'`
	return s.execSSH(a.Server, cmd, 40*time.Second)
}

func (s *Service) handleRunRunbook(ctx context.Context, a NameOnly) (any, error) {
	body, err := s.knowledge.GetRunbook(a.Name)
	if err != nil {
		return nil, err
	}
	steps := parseRunbookSteps(body)
	var results []any
	for i, st := range steps {
		if _, _, err := s.gate(ctx, "ssh_exec", st.Server, st.Script, st); err != nil {
			return nil, fmt.Errorf("步骤 %d: %w", i+1, err)
		}
		res, err := s.execSSH(st.Server, st.Script, 60*time.Second)
		if err != nil {
			return nil, fmt.Errorf("步骤 %d: %w", i+1, err)
		}
		results = append(results, map[string]any{"step": i + 1, "server": st.Server, "result": res})
		if res.ExitCode != 0 {
			return map[string]any{"ok": false, "stoppedAt": i + 1, "results": results}, nil
		}
	}
	return map[string]any{"ok": true, "results": results}, nil
}

type rbStep struct {
	Server string
	Script string
}

func parseRunbookSteps(body string) []rbStep {
	var out []rbStep
	lines := strings.Split(body, "\n")
	in := false
	var buf []string
	server := ""
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") && !in {
			in = true
			server = ""
			rest := strings.TrimSpace(strings.TrimPrefix(trim, "```"))
			if i := strings.Index(rest, "on:"); i >= 0 {
				server = strings.TrimSpace(rest[i+3:])
				server = strings.Fields(server)[0]
			}
			buf = nil
			continue
		}
		if strings.HasPrefix(trim, "```") && in {
			in = false
			if server != "" && len(buf) > 0 {
				out = append(out, rbStep{Server: server, Script: strings.Join(buf, "\n")})
			}
			continue
		}
		if in {
			buf = append(buf, line)
		}
	}
	return out
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
