//go:build windows

package machine

import (
	"os/exec"
	"syscall"
)

func applyNoConsoleWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
}
