package machine

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// LocalShellOption 可发现的本机 Shell
type LocalShellOption struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Command   string `json:"command"`
	IsDefault bool   `json:"isDefault"`
}

// ListLocalShellOptions 枚举本机可用 shell（Windows / Unix）
func ListLocalShellOptions() []LocalShellOption {
	if runtime.GOOS == "windows" {
		return listWindowsLocalShells()
	}
	return listUnixLocalShells()
}

func listWindowsLocalShells() []LocalShellOption {
	var out []LocalShellOption
	sys := os.Getenv("SystemRoot")
	if sys == "" {
		sys = `C:\Windows`
	}
	candidates := []struct {
		id, name, path string
	}{
		{"powershell", "Windows PowerShell", filepath.Join(sys, "System32", "WindowsPowerShell", "v1.0", "powershell.exe")},
		{"pwsh", "PowerShell 7+", findOnPath("pwsh.exe")},
		{"cmd", "CMD", filepath.Join(sys, "System32", "cmd.exe")},
		{"wsl", "WSL", findOnPath("wsl.exe")},
	}
	seen := map[string]bool{}
	for _, c := range candidates {
		path := strings.TrimSpace(c.path)
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err != nil {
			continue
		}
		key := strings.ToLower(path)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, LocalShellOption{
			ID:      c.id,
			Name:    c.name,
			Command: path,
		})
	}
	if len(out) == 0 {
		out = append(out, LocalShellOption{
			ID:      "powershell",
			Name:    "Windows PowerShell",
			Command: "powershell.exe",
		})
	}
	out[0].IsDefault = true
	return out
}

func listUnixLocalShells() []LocalShellOption {
	var out []LocalShellOption
	shellEnv := strings.TrimSpace(os.Getenv("SHELL"))
	candidates := []struct {
		id, name, path string
	}{
		{"bash", "Bash", "/bin/bash"},
		{"zsh", "Zsh", "/bin/zsh"},
		{"fish", "Fish", "/usr/bin/fish"},
		{"sh", "sh", "/bin/sh"},
	}
	if shellEnv != "" {
		base := filepath.Base(shellEnv)
		candidates = append([]struct{ id, name, path string }{
			{base, base, shellEnv},
		}, candidates...)
	}
	seen := map[string]bool{}
	for _, c := range candidates {
		if c.path == "" {
			continue
		}
		if _, err := os.Stat(c.path); err != nil {
			continue
		}
		key := c.path
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, LocalShellOption{ID: c.id, Name: c.name, Command: c.path})
	}
	if len(out) == 0 {
		out = append(out, LocalShellOption{ID: "sh", Name: "sh", Command: "/bin/sh"})
	}
	out[0].IsDefault = true
	return out
}

func findOnPath(name string) string {
	p, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return p
}
