//go:build !windows

package app

func runWindowsApplyUpdateFromArgs(_ []string) bool { return false }

func maybeRelaunchNormalizedWindowsPortable(_ []string) bool { return false }
