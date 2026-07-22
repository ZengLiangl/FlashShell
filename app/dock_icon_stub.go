//go:build !darwin && !windows && !linux

package app

func setApplicationDockIconPNG(pngBytes []byte) {
	_ = pngBytes
}
