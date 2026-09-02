//go:build darwin

package app

/*
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

void flashdockHideDesktopWindow(void);
void flashdockShowDesktopWindow(void);
void flashdockSetTrayEnabled(int on);
void flashdockSetTrayIconPNG(const unsigned char *data, int len);
void flashdockEnsureDesktopHook(void);
*/
import "C"

import (
	"unsafe"
)

func nativeEnsureWindowHook() {
	C.flashdockEnsureDesktopHook()
}

func nativeApplyWindowOpacity(opacity float64) {
	nativeSetWindowAlpha(opacity)
}

func nativeHideMainWindow(a *App) {
	C.flashdockHideDesktopWindow()
	wailsHide(a)
}

func nativeShowMainWindow(a *App) {
	wailsShow(a)
	C.flashdockShowDesktopWindow()
}

func nativeMainWindowVisible() bool {
	return nativeWindowIsVisible()
}

func nativeSetTrayEnabled(on bool) {
	flag := C.int(0)
	if on {
		flag = 1
	}
	C.flashdockSetTrayEnabled(flag)
	if on && len(embeddedDefaultAppIcon) > 0 {
		C.flashdockSetTrayIconPNG(
			(*C.uchar)(unsafe.Pointer(&embeddedDefaultAppIcon[0])),
			C.int(len(embeddedDefaultAppIcon)),
		)
	}
}

//export flashdockGoShow
func flashdockGoShow() {
	if desktopApp != nil {
		desktopApp.ShowMainWindow()
	}
}

//export flashdockGoQuit
func flashdockGoQuit() {
	desktopQuitFromTray()
}

//export flashdockGoMinimizeToTray
func flashdockGoMinimizeToTray() C.int {
	if desktopApp != nil && desktopApp.minimizeToTrayEnabled() {
		return 1
	}
	return 0
}
