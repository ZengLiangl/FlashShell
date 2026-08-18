//go:build !darwin && !windows && !linux

package app

func setApplicationDockIconPNG(pngBytes []byte) {
	_ = pngBytes
}

func persistFinderAppIcon(pngBytes []byte, restoreDefault bool) {
	_ = pngBytes
	_ = restoreDefault
}
