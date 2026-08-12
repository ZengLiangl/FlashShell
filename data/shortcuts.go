package data

import (
	"fmt"
	"strings"
)

// ShortcutBinding 单条快捷键绑定
type ShortcutBinding struct {
	Key      string `json:"key"`
	UseMod   bool   `json:"useMod"`
	UseShift bool   `json:"useShift,omitempty"`
	UseAlt   bool   `json:"useAlt,omitempty"`
}

// ShellSnippet 终端命令片段（可绑定快捷键；可选择是否直接执行）
type ShellSnippet struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Command string         `json:"command"`
	Scope   string         `json:"scope,omitempty"` // global 或机器配置名
	Binding *KeyMapBinding `json:"binding,omitempty"`
	// Execute 为 true 时发送到终端后追加换行并执行；false 仅插入文本
	Execute bool `json:"execute"`
	// OnConnect 为 true 时，Shell 会话连接成功后自动插入/执行（作用域为 global 或匹配机器）
	OnConnect bool `json:"onConnect,omitempty"`
}

// ShortcutSettings 可自定义系统快捷键（独立 JSON 文件）
type ShortcutSettings struct {
	NewWindow         ShortcutBinding `json:"newWindow"`
	MachineConfig     ShortcutBinding `json:"machineConfig"`
	ConnectionManager ShortcutBinding `json:"connectionManager"`
	EnvVars           ShortcutBinding `json:"envVars"`
	SystemSettings    ShortcutBinding `json:"systemSettings"`
	RefreshConfig     ShortcutBinding `json:"refreshConfig"`
	Find              ShortcutBinding `json:"find"`
	Copy              ShortcutBinding `json:"copy"`
	Paste             ShortcutBinding `json:"paste"`
	ClearOutput       ShortcutBinding `json:"clearOutput"`
	CommandPalette    ShortcutBinding `json:"commandPalette"`
	// QuickSwitcher 已废弃（合并进 ConnectionManager）；保留字段以兼容旧配置
	QuickSwitcher     ShortcutBinding `json:"quickSwitcher,omitempty"`
	PaneZoom          ShortcutBinding `json:"paneZoom"`
	NextTab           ShortcutBinding `json:"nextTab"`
	PrevTab           ShortcutBinding `json:"prevTab"`
	CloseTab          ShortcutBinding `json:"closeTab"`
	ToggleBroadcast   ShortcutBinding `json:"toggleBroadcast"`
	OpenSftp          ShortcutBinding `json:"openSftp"`
	OpenLocalShell    ShortcutBinding `json:"openLocalShell"`
	SplitFocusLeft    ShortcutBinding `json:"splitFocusLeft"`
	SplitFocusRight   ShortcutBinding `json:"splitFocusRight"`
	SplitFocusUp      ShortcutBinding `json:"splitFocusUp"`
	SplitFocusDown    ShortcutBinding `json:"splitFocusDown"`
	Snippets          []ShellSnippet  `json:"snippets,omitempty"`
}

// DefaultShortcutSettings 默认快捷键
func DefaultShortcutSettings() ShortcutSettings {
	return ShortcutSettings{
		NewWindow:         ShortcutBinding{Key: "n", UseMod: true},
		MachineConfig:     ShortcutBinding{Key: "m", UseMod: true},
		ConnectionManager: ShortcutBinding{Key: "e", UseMod: true},
		EnvVars:           ShortcutBinding{Key: "u", UseMod: true},
		SystemSettings:    ShortcutBinding{Key: ",", UseMod: true},
		RefreshConfig:     ShortcutBinding{Key: "r", UseMod: true},
		Find:              ShortcutBinding{Key: "f", UseMod: true},
		Copy:              ShortcutBinding{Key: "c", UseMod: true},
		Paste:             ShortcutBinding{Key: "v", UseMod: true},
		ClearOutput:       ShortcutBinding{Key: "k", UseMod: true},
		CommandPalette:    ShortcutBinding{Key: "p", UseMod: true, UseShift: true},
		PaneZoom:          ShortcutBinding{Key: "z", UseMod: true, UseShift: true},
		NextTab:           ShortcutBinding{Key: "Tab", UseMod: true},
		PrevTab:           ShortcutBinding{Key: "Tab", UseMod: true, UseShift: true},
		CloseTab:          ShortcutBinding{Key: "w", UseMod: true},
		ToggleBroadcast:   ShortcutBinding{Key: "b", UseMod: true},
		OpenSftp:          ShortcutBinding{Key: "o", UseMod: true, UseShift: true},
		OpenLocalShell:    ShortcutBinding{Key: "l", UseMod: true},
		SplitFocusLeft:    ShortcutBinding{Key: "ArrowLeft", UseMod: true, UseAlt: true},
		SplitFocusRight:   ShortcutBinding{Key: "ArrowRight", UseMod: true, UseAlt: true},
		SplitFocusUp:      ShortcutBinding{Key: "ArrowUp", UseMod: true, UseAlt: true},
		SplitFocusDown:    ShortcutBinding{Key: "ArrowDown", UseMod: true, UseAlt: true},
	}
}

func fillShortcutDefaults(s *ShortcutSettings) {
	def := DefaultShortcutSettings()
	if s.NewWindow.Key == "" {
		s.NewWindow = def.NewWindow
	}
	if s.MachineConfig.Key == "" {
		s.MachineConfig = def.MachineConfig
	}
	// 旧版默认 Mod+E 绑定环境变量；升级后改为连接管理器，环境变量迁到 Mod+U
	if s.ConnectionManager.Key == "" {
		s.ConnectionManager = def.ConnectionManager
		if s.EnvVars.Key == "" || s.EnvVars.Key == "e" {
			s.EnvVars = def.EnvVars
		}
	} else if s.EnvVars.Key == "" {
		s.EnvVars = def.EnvVars
	}
	if s.SystemSettings.Key == "" {
		s.SystemSettings = def.SystemSettings
	}
	if s.RefreshConfig.Key == "" {
		s.RefreshConfig = def.RefreshConfig
	}
	if s.Find.Key == "" {
		s.Find = def.Find
	}
	if s.Copy.Key == "" {
		s.Copy = def.Copy
	}
	if s.Paste.Key == "" {
		s.Paste = def.Paste
	}
	if s.ClearOutput.Key == "" {
		s.ClearOutput = def.ClearOutput
	}
	if s.CommandPalette.Key == "" {
		s.CommandPalette = def.CommandPalette
	}
	// 废弃 Ctrl+J 快速切换：清空旧绑定，避免残留
	s.QuickSwitcher = ShortcutBinding{}
	if s.PaneZoom.Key == "" {
		s.PaneZoom = def.PaneZoom
	}
	if s.NextTab.Key == "" {
		s.NextTab = def.NextTab
	}
	if s.PrevTab.Key == "" {
		s.PrevTab = def.PrevTab
	}
	if s.CloseTab.Key == "" {
		s.CloseTab = def.CloseTab
	}
	if s.ToggleBroadcast.Key == "" {
		s.ToggleBroadcast = def.ToggleBroadcast
	}
	if s.OpenSftp.Key == "" {
		s.OpenSftp = def.OpenSftp
	}
	if s.OpenLocalShell.Key == "" {
		s.OpenLocalShell = def.OpenLocalShell
	}
	if s.SplitFocusLeft.Key == "" {
		s.SplitFocusLeft = def.SplitFocusLeft
	}
	if s.SplitFocusRight.Key == "" {
		s.SplitFocusRight = def.SplitFocusRight
	}
	if s.SplitFocusUp.Key == "" {
		s.SplitFocusUp = def.SplitFocusUp
	}
	if s.SplitFocusDown.Key == "" {
		s.SplitFocusDown = def.SplitFocusDown
	}
}

// LoadShortcutSettings 从 ~/.flashdock/app_data.json 的 shortcuts 加载
func LoadShortcutSettings() (ShortcutSettings, error) {
	d, err := loadAppDataSection()
	if err != nil {
		return DefaultShortcutSettings(), err
	}
	s := d.Shortcuts
	fillShortcutDefaults(&s)
	if migrateKeyMapsIntoSnippets(&s) {
		_ = SaveShortcutSettings(s)
	}
	return s, nil
}

// migrateKeyMapsIntoSnippets 将旧版 keymaps.json 并入命令片段（一次性）
func migrateKeyMapsIntoSnippets(s *ShortcutSettings) bool {
	if s == nil {
		return false
	}
	km, err := LoadKeyMapSettings()
	if err != nil || len(km.Entries) == 0 {
		return false
	}

	existingIDs := make(map[string]struct{}, len(s.Snippets))
	for _, sn := range s.Snippets {
		if sn.ID != "" {
			existingIDs[sn.ID] = struct{}{}
		}
	}

	changed := false
	for _, e := range km.Entries {
		id := e.ID
		if id == "" {
			id = fmt.Sprintf("km-migrated-%d", len(s.Snippets)+1)
		}
		if _, ok := existingIDs[id]; ok {
			continue
		}
		cmd := e.SendString
		execute := false
		if strings.HasSuffix(cmd, `\n`) {
			cmd = strings.TrimSuffix(cmd, `\n`)
			execute = true
		} else if strings.HasSuffix(cmd, "\n") {
			cmd = strings.TrimSuffix(cmd, "\n")
			execute = true
		}
		name := strings.TrimSpace(e.Name)
		if name == "" {
			name = "按键映射"
		}
		binding := e.Binding
		sn := ShellSnippet{
			ID:      id,
			Name:    name,
			Command: cmd,
			Scope:   "global",
			Execute: execute,
		}
		if binding.Key != "" {
			b := binding
			sn.Binding = &b
		}
		s.Snippets = append(s.Snippets, sn)
		existingIDs[id] = struct{}{}
		changed = true
	}
	if !changed {
		return false
	}
	_ = SaveKeyMapSettings(DefaultKeyMapSettings())
	return true
}

// SaveShortcutSettings 保存到 ~/.flashdock/app_data.json 的 shortcuts
func SaveShortcutSettings(s ShortcutSettings) error {
	fillShortcutDefaults(&s)
	return updateAppData(func(d *AppDataFile) {
		d.Shortcuts = s
	})
}
