package app

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"unicode"
)

// SystemFontInfo 系统字体信息
type SystemFontInfo struct {
	Family string `json:"family"`
	Mono   bool   `json:"mono"`
}

// ListSystemFonts 读取系统字体目录 / 注册表中的可用字体
func (a *App) ListSystemFonts() ([]SystemFontInfo, error) {
	return listSystemFonts()
}

func listSystemFonts() ([]SystemFontInfo, error) {
	names := map[string]struct{}{}
	collectPlatformFonts(names)
	fromRegistry := len(names)

	for _, dir := range systemFontDirs() {
		// Windows 已有注册表时，跳过系统 Fonts 目录的文件名扫描（噪声大）
		if runtime.GOOS == "windows" && fromRegistry > 0 {
			lower := strings.ToLower(dir)
			if strings.HasSuffix(lower, `\windows\fonts`) {
				continue
			}
		}
		collectFontsFromDir(dir, names)
	}

	out := make([]SystemFontInfo, 0, len(names))
	for name := range names {
		name = strings.TrimSpace(name)
		if name == "" || strings.HasPrefix(name, "@") {
			continue
		}
		out = append(out, SystemFontInfo{
			Family: name,
			Mono:   looksLikeMonoFont(name),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Mono != out[j].Mono {
			return out[i].Mono
		}
		return strings.ToLower(out[i].Family) < strings.ToLower(out[j].Family)
	})
	return out, nil
}

func systemFontDirs() []string {
	var dirs []string
	switch runtime.GOOS {
	case "windows":
		windir := os.Getenv("WINDIR")
		if windir == "" {
			windir = `C:\Windows`
		}
		dirs = append(dirs, filepath.Join(windir, "Fonts"))
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			dirs = append(dirs, filepath.Join(local, "Microsoft", "Windows", "Fonts"))
		}
	case "darwin":
		dirs = append(dirs,
			"/System/Library/Fonts",
			"/Library/Fonts",
		)
		if home, err := os.UserHomeDir(); err == nil {
			dirs = append(dirs, filepath.Join(home, "Library", "Fonts"))
		}
	default:
		dirs = append(dirs,
			"/usr/share/fonts",
			"/usr/local/share/fonts",
		)
		if home, err := os.UserHomeDir(); err == nil {
			dirs = append(dirs,
				filepath.Join(home, ".fonts"),
				filepath.Join(home, ".local", "share", "fonts"),
			)
		}
	}
	return dirs
}

func collectFontsFromDir(dir string, names map[string]struct{}) {
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			// 避免过深遍历导致卡顿
			if path != dir {
				rel, relErr := filepath.Rel(dir, path)
				if relErr == nil && strings.Count(rel, string(os.PathSeparator)) > 2 {
					return filepath.SkipDir
				}
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".ttf" && ext != ".otf" && ext != ".ttc" && ext != ".otc" {
			return nil
		}
		base := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		base = strings.ReplaceAll(base, "_", " ")
		base = strings.ReplaceAll(base, "-", " ")
		base = cleanFontLabel(base)
		if base != "" {
			names[base] = struct{}{}
		}
		return nil
	})
}

func cleanFontLabel(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// 去掉常见后缀噪音
	lower := strings.ToLower(s)
	for _, suf := range []string{" regular", " bold", " italic", " light", " medium", " black", " thin", " demibold", " semibold"} {
		if strings.HasSuffix(lower, suf) && len(s) > len(suf)+1 {
			s = strings.TrimSpace(s[:len(s)-len(suf)])
			lower = strings.ToLower(s)
		}
	}
	// 过滤纯数字或过短名
	if len(s) < 2 {
		return ""
	}
	hasLetter := false
	for _, r := range s {
		if unicode.IsLetter(r) {
			hasLetter = true
			break
		}
	}
	if !hasLetter {
		return ""
	}
	return s
}

func looksLikeMonoFont(name string) bool {
	n := strings.ToLower(name)
	keys := []string{
		"mono", "consolas", "courier", "menlo", "monaco", "fixedsys", "fixed",
		"iosevka", "fira code", "jetbrains", "cascadia", "source code", "sarasa",
		"hack", "inconsolata", "dejavu sans mono", "nerd font", "terminess",
		"ubuntu mono", "liberation mono", "anonymous pro", "pt mono",
		"droid sans mono", "roboto mono", "space mono", "ibm plex mono",
	}
	for _, k := range keys {
		if strings.Contains(n, k) {
			return true
		}
	}
	return false
}

func addFontName(names map[string]struct{}, raw string) {
	name := strings.TrimSpace(raw)
	if name == "" || strings.HasPrefix(name, "@") {
		return
	}
	// Windows 注册表常见：Arial (TrueType)
	for _, suf := range []string{
		" (TrueType)", " (OpenType)", " (All res)", " (VGA res)",
		" (Verdana)", " & ",
	} {
		if i := strings.Index(name, suf); i > 0 {
			name = name[:i]
		}
	}
	if i := strings.Index(name, " ("); i > 0 {
		name = name[:i]
	}
	name = strings.TrimSpace(name)
	if name != "" {
		names[name] = struct{}{}
	}
}
