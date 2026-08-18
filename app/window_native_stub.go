//go:build !windows

package app

func nativeRestoreMaximisedWindow() bool {
	return false
}

func nativeWindowIsMaximised() bool {
	return false
}
