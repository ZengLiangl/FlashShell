//go:build !windows

package app

func collectPlatformFonts(names map[string]struct{}) {
	// 非 Windows：依赖 systemFontDirs 目录扫描
}
