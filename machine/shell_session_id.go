package machine

import (
	"fmt"
	"strconv"
	"strings"
)

const remoteSessionSep = "-"

// ConfigNameResolver 判断是否为已存在的机器配置名
type ConfigNameResolver func(name string) bool

// RemoteConfigName 从会话 ID 解析机器配置名（无配置表时使用启发式，可能误判含数字的配置名）
func RemoteConfigName(sessionID string) string {
	return RemoteConfigNameWithResolver(sessionID, nil)
}

// RemoteConfigNameWithResolver 结合配置名表解析；精确匹配优先，避免 va-test-66 被拆成 va-test
func RemoteConfigNameWithResolver(sessionID string, resolver ConfigNameResolver) string {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" || IsLocalShellID(sessionID) {
		return sessionID
	}
	if resolver != nil && resolver(sessionID) {
		return sessionID
	}
	if i := strings.LastIndex(sessionID, "#"); i > 0 {
		suffix := sessionID[i+1:]
		if n, err := strconv.Atoi(suffix); err == nil && n >= 2 {
			base := sessionID[:i]
			if resolver == nil || resolver(base) {
				return base
			}
		}
	}
	if base, idx := parseRemoteSuffix(sessionID, remoteSessionSep); idx >= 2 {
		if resolver == nil || resolver(base) {
			return base
		}
	}
	return sessionID
}

// RemoteSessionIndex 会话序号（首个为 1）
func RemoteSessionIndex(sessionID string) int {
	return RemoteSessionIndexForConfig(sessionID, "")
}

// RemoteSessionIndexForConfig 在已知配置名下解析会话序号
func RemoteSessionIndexForConfig(sessionID, configName string) int {
	sessionID = strings.TrimSpace(sessionID)
	configName = strings.TrimSpace(configName)
	if sessionID == "" || IsLocalShellID(sessionID) {
		return 1
	}
	if configName != "" {
		if sessionID == configName {
			return 1
		}
		prefix := configName + remoteSessionSep
		if strings.HasPrefix(sessionID, prefix) {
			suffix := sessionID[len(prefix):]
			if n, err := strconv.Atoi(suffix); err == nil && n >= 2 {
				return n
			}
		}
		hashPrefix := configName + "#"
		if strings.HasPrefix(sessionID, hashPrefix) {
			suffix := sessionID[len(hashPrefix):]
			if n, err := strconv.Atoi(suffix); err == nil && n >= 2 {
				return n
			}
		}
		return 1
	}
	if i := strings.LastIndex(sessionID, "#"); i > 0 {
		suffix := sessionID[i+1:]
		if n, err := strconv.Atoi(suffix); err == nil && n >= 2 {
			return n
		}
	}
	if _, idx := parseRemoteSuffix(sessionID, remoteSessionSep); idx >= 2 {
		return idx
	}
	return 1
}

func parseRemoteSuffix(sessionID, sep string) (base string, index int) {
	i := strings.LastIndex(sessionID, sep)
	if i <= 0 {
		return sessionID, 1
	}
	suffix := sessionID[i+1:]
	n, err := strconv.Atoi(suffix)
	if err != nil || n < 2 {
		return sessionID, 1
	}
	return sessionID[:i], n
}

// FormatRemoteSessionID 生成远程会话 ID（index<=1 时等于配置名）
func FormatRemoteSessionID(configName string, index int) string {
	configName = strings.TrimSpace(configName)
	if index <= 1 {
		return configName
	}
	return fmt.Sprintf("%s%s%d", configName, remoteSessionSep, index)
}

// ShellTabLabel 生成 Tab 显示名
func ShellTabLabel(sessionID, configName, kind string) string {
	if kind == ShellKindLocal || IsLocalShellID(sessionID) {
		if sessionID == LocalShellIDPrefix {
			return "本机"
		}
		n := strings.TrimPrefix(sessionID, LocalShellIDPrefix+"-")
		if n != "" && n != sessionID {
			return "本机 " + n
		}
		return "本机"
	}
	base := configName
	if base == "" {
		base = RemoteConfigName(sessionID)
	}
	idx := RemoteSessionIndexForConfig(sessionID, base)
	if idx <= 1 {
		return base
	}
	return fmt.Sprintf("%s-%d", base, idx)
}
