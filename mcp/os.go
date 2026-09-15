package mcp

import (
	"bytes"
	"strconv"
	"strings"
	"time"

	"FlashDock/define"
)

func (s *Service) rememberOS(alias, osName string) {
	osName = normalizeOSName(osName)
	if alias == "" || osName == "" {
		return
	}
	s.osGuess.Store(alias, osName)
}

func (s *Service) resolvedOS(m *define.Machine) string {
	if m == nil {
		return ""
	}
	if v, ok := s.osGuess.Load(m.Name); ok {
		if n, ok := v.(string); ok {
			if n = normalizeOSName(n); n != "" {
				return n
			}
		}
	}
	if m.ID != "" {
		if v, ok := s.osGuess.Load(m.ID); ok {
			if n, ok := v.(string); ok {
				if n = normalizeOSName(n); n != "" {
					return n
				}
			}
		}
	}
	return normalizeOSName(m.OS)
}

func normalizeOSName(raw string) string {
	u := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case u == "":
		return ""
	case strings.Contains(u, "win"):
		return "Windows"
	case strings.Contains(u, "darwin") || strings.Contains(u, "mac"):
		return "Darwin"
	default:
		s := strings.TrimSpace(raw)
		if s == "" {
			return ""
		}
		return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
	}
}

func (s *Service) aliasIsWindows(alias string) bool {
	m, err := s.machineByAlias(alias)
	if err != nil {
		if v, ok := s.osGuess.Load(alias); ok {
			n, _ := v.(string)
			return normalizeOSName(n) == "Windows"
		}
		return false
	}
	return s.resolvedOS(m) == "Windows"
}

func detectRemoteOS(rm *define.RemoteMachine) string {
	if rm == nil || !rm.IsConnected() {
		return ""
	}
	if out := sshQuick(rm, "uname -s", 8*time.Second); out != "" {
		return normalizeOSName(out)
	}
	if out := sshQuick(rm, "cmd /c ver", 8*time.Second); strings.Contains(strings.ToLower(out), "windows") {
		return "Windows"
	}
	return ""
}

func sshQuick(rm *define.RemoteMachine, cmd string, timeout time.Duration) string {
	if rm == nil || rm.SSHClient == nil {
		return ""
	}
	session, err := rm.NewSession()
	if err != nil {
		return ""
	}
	defer session.Close()
	var stdout bytes.Buffer
	session.Stdout = &stdout
	done := make(chan error, 1)
	go func() { done <- session.Run(cmd) }()
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case err := <-done:
		if err != nil {
			return ""
		}
		return strings.TrimSpace(stdout.String())
	case <-t.C:
		_ = session.Close()
		return ""
	}
}

func windowsSystemInfoCmd() string {
	return `powershell -NoProfile -Command "Get-CimInstance Win32_OperatingSystem | Select-Object Caption,Version,LastBootUpTime | Format-List; hostname"`
}

func windowsDiskUsageCmd() string {
	return `powershell -NoProfile -Command "Get-CimInstance Win32_LogicalDisk | Select-Object DeviceID,@{N='SizeGB';E={[math]::Round($_.Size/1GB,1)}},@{N='FreeGB';E={[math]::Round($_.FreeSpace/1GB,1)}} | Format-Table -AutoSize"`
}

func windowsPortCheckCmd(port int) string {
	return `powershell -NoProfile -Command "Get-NetTCPConnection -State Listen -LocalPort ` + strconv.Itoa(port) + ` -ErrorAction SilentlyContinue | Format-Table -AutoSize"`
}

func windowsServiceStatusCmd(name string) string {
	n := strings.ReplaceAll(name, "'", "''")
	return `powershell -NoProfile -Command "Get-Service -Name '` + n + `' -ErrorAction SilentlyContinue | Format-List"`
}

func windowsTailLogCmd(path string, lines int64) string {
	p := strings.ReplaceAll(path, "'", "''")
	return `powershell -NoProfile -Command "Get-Content -LiteralPath '` + p + `' -Tail ` + strconv.FormatInt(lines, 10) + `"`
}
