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

// nativeWindowMaximisedState 返回 (已知 HWND, 是否最大化)。
// HWND 可用时以 IsZoomed 为准，避免拖动还原后仍被内部标记误判为最大化。
func nativeWindowMaximisedState() (known, maximised bool) {
	hwnd := nativeMainHWND()
	if hwnd == 0 {
		return false, false
	}
	r, _, _ := procIsZoomed.Call(hwnd)
	return true, r != 0
}

func nativeWindowIsMaximised() bool {
	_, max := nativeWindowMaximisedState()
	return max
}
