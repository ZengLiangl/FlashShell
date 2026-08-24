//go:build !windows

package app

func nativeRestoreMaximisedWindow() bool {
	return false
}

func nativeWindowMaximisedState() (known, maximised bool) {
	return false, false
}

func nativeWindowIsMaximised() bool {
	return false
}
