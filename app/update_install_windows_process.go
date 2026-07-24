//go:build windows

package app

import (
	"os/exec"
	"syscall"
)

const (
	windowsCreateNoWindow         = 0x08000000
	windowsCreateBreakawayFromJob = 0x01000000
	windowsCreateNewProcessGroup  = 0x00000200
)

func configureWindowsUpdateCommand(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
		// 脱离控制台与 Job，避免宿主退出时把更新器一并杀掉
		CreationFlags: windowsCreateNoWindow | windowsCreateBreakawayFromJob | windowsCreateNewProcessGroup,
	}
}
