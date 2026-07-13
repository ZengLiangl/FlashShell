package app

import (
	"fmt"
	"path"
	"strings"
)

// NormalizeRemoteAbs 规范化远端绝对路径（去掉末尾 /，根除外）。
func NormalizeRemoteAbs(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if p != "/" {
		p = strings.TrimSuffix(p, "/")
	}
	return p
}

// PathJoinRemote 在 base 下拼接相对路径。
func PathJoinRemote(base, rel string) string {
	base = NormalizeRemoteAbs(base)
	rel = strings.TrimSpace(rel)
	rel = strings.TrimPrefix(rel, "./")
	for len(rel) > 1 && strings.HasSuffix(rel, "/") {
		rel = strings.TrimSuffix(rel, "/")
	}
	if rel == "." || rel == "" {
		return base
	}
	if strings.HasPrefix(rel, "/") {
		return NormalizeRemoteAbs(rel)
	}
	return path.Clean(path.Join(base, rel))
}

// ResolveRemotePath 纯函数：基于 base / home 解析 cd 目标，不访问网络。
// home 用于 ~ 与空 base 的回退。
func ResolveRemotePath(base, target, home string) (string, error) {
	target = strings.TrimSpace(target)
	target = strings.Trim(target, `"'`)
	base = strings.TrimSpace(base)
	homeRaw := strings.TrimSpace(home)
	hasHome := homeRaw != ""
	homeNorm := NormalizeRemoteAbs(homeRaw)

	if target != "/" {
		target = strings.TrimRight(target, "/")
	}

	if target == "" || target == "." {
		if base != "" && base != "." {
			return NormalizeRemoteAbs(base), nil
		}
		if hasHome {
			return homeNorm, nil
		}
		return "/", nil
	}

	if strings.HasPrefix(target, "/") {
		return NormalizeRemoteAbs(target), nil
	}

	if target == "~" || strings.HasPrefix(target, "~/") {
		if !hasHome {
			return "", fmt.Errorf("home 未知")
		}
		if target == "~" {
			return homeNorm, nil
		}
		return PathJoinRemote(homeNorm, strings.TrimPrefix(target, "~/")), nil
	}

	if base == "" || base == "." {
		if !hasHome {
			return "", fmt.Errorf("base 与 home 均未知")
		}
		base = homeNorm
	}
	return PathJoinRemote(base, target), nil
}

// ChooseCdPath 目录存在则采用 resolved，否则保留 current。
func ChooseCdPath(current, resolved string, exists bool) string {
	current = NormalizeRemoteAbs(current)
	resolved = NormalizeRemoteAbs(resolved)
	if exists {
		return resolved
	}
	return current
}

// RemoteAncestorPaths 返回从根到 abs 的祖先链（含自身），用于目录树展开。
// 例: /root/app → ["/", "/root", "/root/app"]
func RemoteAncestorPaths(abs string) []string {
	abs = NormalizeRemoteAbs(abs)
	if abs == "/" {
		return []string{"/"}
	}
	parts := strings.Split(strings.Trim(abs, "/"), "/")
	out := make([]string, 0, len(parts)+1)
	out = append(out, "/")
	cur := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		cur += "/" + p
		out = append(out, cur)
	}
	return out
}
