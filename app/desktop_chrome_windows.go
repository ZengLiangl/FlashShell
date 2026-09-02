//go:build windows

package app

import (
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	wmDestroy       = 0x0002
	wmRButtonUp     = 0x0205
	wmLButtonUp     = 0x0202
	wmLButtonDblClk = 0x0203
	wmTrayCallback  = 0x8000 + 32
	trayUID         = 1

	nimAdd     = 0
	nimModify  = 1
	nimDelete  = 2
	nifMessage = 0x01
	nifIcon    = 0x02
	nifTip     = 0x04

	wsExLayered    = 0x00080000
	lwaAlpha       = 0x00000002
	swHideCmd      = 0
	swShowCmd      = 5
	swRestoreCmd   = 9
	mfString       = 0x0000
	tpmRightButton = 0x0002
	tpmReturnCmd   = 0x0100

	idTrayShow = 1
	idTrayQuit = 2

	gwlpWndProc = ^uintptr(3)  // -4
	gwlExStyle  = ^uintptr(19) // -20
)

type notifyIconDataW struct {
	CbSize           uint32
	Hwnd             windows.HWND
	UID              uint32
	Flags            uint32
	CallbackMessage  uint32
	HIcon            windows.Handle
	Tip              [128]uint16
}

type point struct {
	X, Y int32
}

var (
	procSetWindowLongPtrW = user32.NewProc("SetWindowLongPtrW")
	procCallWindowProcW   = user32.NewProc("CallWindowProcW")
	procGetWindowLongW    = user32.NewProc("GetWindowLongW")
	procSetWindowLongW    = user32.NewProc("SetWindowLongW")
	procSetLayeredAttr    = user32.NewProc("SetLayeredWindowAttributes")
	procSetForeground     = user32.NewProc("SetForegroundWindow")
	procGetCursorPos      = user32.NewProc("GetCursorPos")
	procCreatePopupMenu   = user32.NewProc("CreatePopupMenu")
	procAppendMenuW       = user32.NewProc("AppendMenuW")
	procTrackPopupMenu    = user32.NewProc("TrackPopupMenu")
	procDestroyMenu       = user32.NewProc("DestroyMenu")
	procSetMenuDefault    = user32.NewProc("SetMenuDefaultItem")
	procShellNotifyIconW  = shell32.NewProc("Shell_NotifyIconW")

	desktopHookOnce    sync.Once
	desktopWndProcCB   uintptr
	desktopOldProc     uintptr
	desktopHookHWND    windows.HWND
	desktopChromeMu    sync.Mutex
	trayAdded          bool
)

func nativeEnsureWindowHook() {
	hwnd := findMainWindowHWND()
	if hwnd == 0 {
		return
	}
	desktopHookOnce.Do(func() {
		desktopWndProcCB = syscall.NewCallback(desktopWndProc)
	})
	desktopChromeMu.Lock()
	defer desktopChromeMu.Unlock()
	if desktopHookHWND == hwnd && desktopOldProc != 0 {
		return
	}
	r, _, _ := procSetWindowLongPtrW.Call(uintptr(hwnd), gwlpWndProc, desktopWndProcCB)
	if r == 0 {
		return
	}
	desktopOldProc = r
	desktopHookHWND = hwnd
}

func desktopWndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	switch msg {
	case wmTrayCallback:
		switch lparam {
		case wmLButtonUp, wmLButtonDblClk:
			if desktopApp != nil {
				desktopApp.ShowMainWindow()
			}
		case wmRButtonUp:
			showTrayMenu(windows.HWND(hwnd))
		}
		return 0
	case wmDestroy:
		nativeRemoveTrayLocked()
	}
	r, _, _ := procCallWindowProcW.Call(desktopOldProc, hwnd, msg, wparam, lparam)
	return r
}

func nativeApplyWindowOpacity(opacity float64) {
	hwnd := findMainWindowHWND()
	if hwnd == 0 {
		return
	}
	ex, _, _ := procGetWindowLongW.Call(uintptr(hwnd), gwlExStyle)
	if ex&wsExLayered == 0 {
		procSetWindowLongW.Call(uintptr(hwnd), gwlExStyle, ex|wsExLayered)
	}
	alpha := byte(255)
	if opacity < 0.995 {
		alpha = byte(opacity*255 + 0.5)
		if alpha < 102 {
			alpha = 102 // 约 40%
		}
	}
	procSetLayeredAttr.Call(uintptr(hwnd), 0, uintptr(alpha), lwaAlpha)
}

func nativeHideMainWindow(a *App) {
	hwnd := findMainWindowHWND()
	if hwnd != 0 {
		procShowWindow.Call(uintptr(hwnd), swHideCmd)
	}
	wailsHide(a)
}

func nativeShowMainWindow(a *App) {
	wailsShow(a)
	hwnd := findMainWindowHWND()
	if hwnd == 0 {
		return
	}
	procShowWindow.Call(uintptr(hwnd), swRestoreCmd)
	procShowWindow.Call(uintptr(hwnd), swShowCmd)
	procSetForeground.Call(uintptr(hwnd))
}

func nativeMainWindowVisible() bool {
	hwnd := findMainWindowHWND()
	if hwnd == 0 {
		return false
	}
	r, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
	return r != 0
}

func nativeSetTrayEnabled(on bool) {
	nativeEnsureWindowHook()
	desktopChromeMu.Lock()
	defer desktopChromeMu.Unlock()
	if on {
		nativeAddTrayLocked()
		return
	}
	nativeRemoveTrayLocked()
}

func nativeAddTrayLocked() {
	hwnd := findMainWindowHWND()
	if hwnd == 0 {
		return
	}
	icon := winIconSmall
	if icon == 0 {
		icon = winIconBig
	}
	var nid notifyIconDataW
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.Hwnd = hwnd
	nid.UID = trayUID
	nid.Flags = nifMessage | nifIcon | nifTip
	nid.CallbackMessage = wmTrayCallback
	nid.HIcon = icon
	copyUTF16(nid.Tip[:], "FlashShell")
	cmd := uintptr(nimAdd)
	if trayAdded {
		cmd = nimModify
	}
	r, _, _ := procShellNotifyIconW.Call(cmd, uintptr(unsafe.Pointer(&nid)))
	if r != 0 {
		trayAdded = true
	}
}

func nativeRemoveTrayLocked() {
	if !trayAdded {
		return
	}
	hwnd := findMainWindowHWND()
	if hwnd == 0 {
		hwnd = desktopHookHWND
	}
	var nid notifyIconDataW
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.Hwnd = hwnd
	nid.UID = trayUID
	procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
	trayAdded = false
}

func showTrayMenu(hwnd windows.HWND) {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer procDestroyMenu.Call(menu)
	appendMenu(menu, idTrayShow, "显示 FlashShell")
	appendMenu(menu, idTrayQuit, "退出")
	procSetMenuDefault.Call(menu, idTrayShow, 0)
	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetForeground.Call(uintptr(hwnd))
	cmd, _, _ := procTrackPopupMenu.Call(menu, tpmRightButton|tpmReturnCmd, uintptr(pt.X), uintptr(pt.Y), 0, uintptr(hwnd), 0)
	switch cmd {
	case idTrayShow:
		if desktopApp != nil {
			desktopApp.ShowMainWindow()
		}
	case idTrayQuit:
		desktopQuitFromTray()
	}
}

func appendMenu(menu uintptr, id uintptr, text string) {
	p, _ := windows.UTF16PtrFromString(text)
	procAppendMenuW.Call(menu, mfString, id, uintptr(unsafe.Pointer(p)))
}

func copyUTF16(dst []uint16, s string) {
	u, err := windows.UTF16FromString(s)
	if err != nil {
		return
	}
	n := len(u)
	if n > len(dst) {
		n = len(dst)
		u[n-1] = 0
	}
	copy(dst, u[:n])
}
