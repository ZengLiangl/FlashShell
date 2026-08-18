//go:build windows

package app

const (
	swRestore = 9
)

var (
	procShowWindow = user32.NewProc("ShowWindow")
	procIsZoomed   = user32.NewProc("IsZoomed")
)

func nativeMainHWND() uintptr {
	return uintptr(findMainWindowHWND())
}

func nativeRestoreMaximisedWindow() bool {
	hwnd := nativeMainHWND()
	if hwnd == 0 {
		return false
	}
	procShowWindow.Call(hwnd, swRestore)
	return true
}

func nativeWindowIsMaximised() bool {
	hwnd := nativeMainHWND()
	if hwnd == 0 {
		return false
	}
	r, _, _ := procIsZoomed.Call(hwnd)
	return r != 0
}
