//go:build !windows

package app

import "os/exec"

func configureWindowsUpdateCommand(_ *exec.Cmd) {}
