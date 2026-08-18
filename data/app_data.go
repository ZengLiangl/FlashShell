package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"FlashDock/define"
)

const appDataFileName = "app_data.json"

// AppDataFile ~/.flashshell/app_data.json：用顶层 key 区分原多个独立 JSON
type AppDataFile struct {
	KnownHosts          []KnownHostRecord           `json:"knownHosts"`
	ShellCommandHistory shellCmdHistoryFile         `json:"shellCommandHistory"`
	ShellHistory        []define.ShellHistoryRecord `json:"shellHistory"`
	Shortcuts           ShortcutSettings            `json:"shortcuts"`
}

var (
	appDataMu     sync.Mutex
	appDataCache  *AppDataFile
	appDataLoaded bool
)

func appDataPath() (string, error) {
	home, err := ConfigHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, appDataFileName), nil
}

func legacyAppDataPaths() (knownHosts, shellCmdHistory, shellHistory, shortcuts string, err error) {
	home, err := ConfigHomeDir()
	if err != nil {
		return "", "", "", "", err
	}
	return filepath.Join(home, "known_hosts.json"),
		filepath.Join(home, "shell_command_history.json"),
		filepath.Join(home, "shell_history.json"),
		filepath.Join(home, "shortcuts.json"),
		nil
}

func emptyAppData() *AppDataFile {
	return &AppDataFile{
		KnownHosts:          []KnownHostRecord{},
		ShellCommandHistory: shellCmdHistoryFile{ByScope: make(map[string][]string)},
		ShellHistory:        []define.ShellHistoryRecord{},
		Shortcuts:           DefaultShortcutSettings(),
	}
}

func readJSONFile(path string, dest any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dest)
}

func migrateLegacyIntoAppData(d *AppDataFile) (changed bool) {
	knownPath, cmdPath, histPath, shortcutPath, err := legacyAppDataPaths()
	if err != nil {
		return false
	}

	if _, err := os.Stat(knownPath); err == nil {
		var list []KnownHostRecord
		if readJSONFile(knownPath, &list) == nil && len(list) > 0 {
			d.KnownHosts = list
			changed = true
		}
	}
	if _, err := os.Stat(cmdPath); err == nil {
		var hist shellCmdHistoryFile
		if readJSONFile(cmdPath, &hist) == nil {
			if hist.ByScope == nil {
				hist.ByScope = make(map[string][]string)
			}
			d.ShellCommandHistory = hist
			changed = true
		}
	}
	if _, err := os.Stat(histPath); err == nil {
		var list []define.ShellHistoryRecord
		if readJSONFile(histPath, &list) == nil {
			d.ShellHistory = list
			changed = true
		}
	}
	if _, err := os.Stat(shortcutPath); err == nil {
		var s ShortcutSettings
		if readJSONFile(shortcutPath, &s) == nil {
			fillShortcutDefaults(&s)
			d.Shortcuts = s
			changed = true
		}
	}
	return changed
}

func removeLegacyAppDataFiles() {
	knownPath, cmdPath, histPath, shortcutPath, err := legacyAppDataPaths()
	if err != nil {
		return
	}
	for _, p := range []string{knownPath, cmdPath, histPath, shortcutPath} {
		_ = os.Remove(p)
	}
}

func loadAppDataLocked() (*AppDataFile, error) {
	if appDataLoaded && appDataCache != nil {
		return appDataCache, nil
	}
	path, err := appDataPath()
	if err != nil {
		return nil, err
	}
	d := emptyAppData()
	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		if migrateLegacyIntoAppData(d) {
			if saveErr := saveAppDataLocked(d); saveErr == nil {
				removeLegacyAppDataFiles()
			}
		}
		appDataCache = d
		appDataLoaded = true
		return d, nil
	}
	if err := json.Unmarshal(raw, d); err != nil {
		return nil, fmt.Errorf("解析 %s 失败: %w", appDataFileName, err)
	}
	if d.ShellCommandHistory.ByScope == nil {
		d.ShellCommandHistory.ByScope = make(map[string][]string)
	}
	if d.KnownHosts == nil {
		d.KnownHosts = []KnownHostRecord{}
	}
	if d.ShellHistory == nil {
		d.ShellHistory = []define.ShellHistoryRecord{}
	}
	fillShortcutDefaults(&d.Shortcuts)
	appDataCache = d
	appDataLoaded = true
	return d, nil
}

func saveAppDataLocked(d *AppDataFile) error {
	if d == nil {
		return fmt.Errorf("app data 为空")
	}
	if d.ShellCommandHistory.ByScope == nil {
		d.ShellCommandHistory.ByScope = make(map[string][]string)
	}
	path, err := appDataPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, raw, 0644); err != nil {
		return err
	}
	appDataCache = d
	appDataLoaded = true
	return nil
}

// loadAppDataSection 读取合并文件（供各子模块加载）
func loadAppDataSection() (*AppDataFile, error) {
	appDataMu.Lock()
	defer appDataMu.Unlock()
	return loadAppDataLocked()
}

func updateAppData(mutate func(*AppDataFile)) error {
	appDataMu.Lock()
	defer appDataMu.Unlock()
	d, err := loadAppDataLocked()
	if err != nil {
		return err
	}
	mutate(d)
	return saveAppDataLocked(d)
}
