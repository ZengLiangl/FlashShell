//go:build windows

package app

import (
	"golang.org/x/sys/windows/registry"
)

func collectPlatformFonts(names map[string]struct{}) {
	readFontsKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`, names)
	readFontsKey(registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`, names)
}

func readFontsKey(root registry.Key, path string, names map[string]struct{}) {
	k, err := registry.OpenKey(root, path, registry.READ)
	if err != nil {
		return
	}
	defer k.Close()
	values, err := k.ReadValueNames(0)
	if err != nil {
		return
	}
	for _, v := range values {
		addFontName(names, v)
	}
}
