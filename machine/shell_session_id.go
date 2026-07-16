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

// RemoteConfigNameForKnown 结合已知配置名列表解析会话所属机器。
func RemoteConfigNameForKnown(sessionID string, knownConfigs []string) string {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" || IsLocalShellID(sessionID) {
		return sessionID
	}
	for _, cfg := range knownConfigs {
		cfg = strings.TrimSpace(cfg)
		if cfg == "" {
			continue
		}
		if sessionID == cfg {
			return cfg
		}
		if idx := remoteSessionIndexForConfig(sessionID, cfg); idx >= 2 {
			return cfg
		}
	}
	return remoteConfigNameLegacy(sessionID, nil)
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
	return remoteConfigNameLegacy(sessionID, resolver)
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
		if idx := remoteSessionIndexForConfig(sessionID, configName); idx >= 2 {
			return idx
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

func remoteSessionIndexForConfig(sessionID, configName string) int {
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

func remoteConfigNameLegacy(sessionID string, resolver ConfigNameResolver) string {
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

// FormatRemoteSessionID 生成远程会话 ID（slot 1 为配置名，之后为 配置名-序号）
func FormatRemoteSessionID(configName string, slot int) string {
	configName = strings.TrimSpace(configName)
	if slot <= 1 {
		return configName
	}
	return fmt.Sprintf("%s%s%d", configName, remoteSessionSep, slot)
}

// FormatLocalSessionID 生成本地终端会话 ID（local / local-2）
func FormatLocalSessionID(slot int) string {
	if slot <= 1 {
		return LocalShellIDPrefix
	}
	return fmt.Sprintf("%s-%d", LocalShellIDPrefix, slot)
}

// LocalSessionIndex 解析本地会话序号（首个为 1）
func LocalSessionIndex(sessionID string) int {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" || sessionID == LocalShellIDPrefix {
		return 1
	}
	prefix := LocalShellIDPrefix + "-"
	if strings.HasPrefix(sessionID, prefix) {
		suffix := sessionID[len(prefix):]
		if n, err := strconv.Atoi(suffix); err == nil && n >= 2 {
			return n
		}
	}
	return 1
}

// ShellTabLabel 生成 Tab 显示名
func ShellTabLabel(sessionID, configName, kind string) string {
	if kind == ShellKindLocal || IsLocalShellID(sessionID) {
		if sessionID == LocalShellIDPrefix {
			return "本机"
		}
		n := strings.TrimPrefix(sessionID, LocalShellIDPrefix+"-")
		if n != "" && n != sessionID {
			return "本机-" + n
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
