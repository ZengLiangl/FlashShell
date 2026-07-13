package data

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MachineImportResult 机器配置导入结果
type MachineImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

// XshellImportResult 兼容旧名称
type XshellImportResult = MachineImportResult

func matchFileName(name string, match func(string) bool) bool {
	return match(strings.ToLower(name))
}

// CollectFilesFromPaths 从文件或文件夹路径收集匹配的配置文件
func CollectFilesFromPaths(paths []string, match func(fileName string) bool) ([]string, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("未选择文件或文件夹")
	}

	seen := make(map[string]struct{})
	result := make([]string, 0)

	addFile := func(path string) {
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		result = append(result, path)
	}

	for _, path := range paths {
		if path == "" {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			entries, err := os.ReadDir(path)
			if err != nil {
				return nil, err
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				if matchFileName(entry.Name(), match) {
					addFile(filepath.Join(path, entry.Name()))
				}
			}
			continue
		}
		if !matchFileName(filepath.Base(path), match) {
			return nil, fmt.Errorf("不支持的文件: %s", filepath.Base(path))
		}
		addFile(path)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("未找到可导入的配置文件")
	}
	return result, nil
}

func isXshellFile(name string) bool {
	return strings.HasSuffix(name, ".xsh")
}

func isFinalShellFile(name string) bool {
	return strings.HasSuffix(name, "_connect_config.json")
}
