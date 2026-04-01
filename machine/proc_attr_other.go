//go:build !windows

package machine

import "os/exec"

func applyNoConsoleWindow(cmd *exec.Cmd) {}
