//go:build !windows && !darwin

package app

func nativeEnsureWindowHook() {}

func nativeApplyWindowOpacity(_ float64) {}

func nativeHideMainWindow(a *App) {
	wailsHide(a)
}

func nativeShowMainWindow(a *App) {
	wailsShow(a)
}

func nativeMainWindowVisible() bool {
	return true
}

func nativeSetTrayEnabled(_ bool) {}
