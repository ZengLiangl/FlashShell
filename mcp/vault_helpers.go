package mcp

import "strings"

func pubGet(it VaultItem, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(it.Public[k]); v != "" {
			return v
		}
	}
	return ""
}

func nz(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
