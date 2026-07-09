package data

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// LogEntry 执行日志索引项
type LogEntry struct {
	FileName    string `json:"fileName"`
	FullPath    string `json:"fullPath"`
	Project     string `json:"project"`
	SubProject  string `json:"subProject"`
	StartedAt   string `json:"startedAt"`
	Size        int64  `json:"size"`
	Success     bool   `json:"success"`
}

// LogManager 执行日志落盘管理
type LogManager struct {
	mu          sync.Mutex
	basePath    string
	currentFile *os.File
	currentMeta *LogEntry
}

// NewLogManager 创建日志管理器
func NewLogManager(basePath string) *LogManager {
	return &LogManager{basePath: expandPath(basePath)}
}

// SetBasePath 更新日志目录
func (lm *LogManager) SetBasePath(basePath string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.basePath = expandPath(basePath)
}

// StartSession 开始一次执行日志
func (lm *LogManager) StartSession(projectName, subProjectName string) (string, error) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if err := os.MkdirAll(lm.basePath, 0755); err != nil {
		return "", fmt.Errorf("创建日志目录失败: %w", err)
	}

	if lm.currentFile != nil {
		_ = lm.currentFile.Close()
		lm.currentFile = nil
	}

	ts := time.Now().Format("20060102-150405")
	safeProject := sanitizeFileName(projectName)
	safeSub := sanitizeFileName(subProjectName)
	fileName := fmt.Sprintf("%s_%s_%s.log", ts, safeProject, safeSub)
	fullPath := filepath.Join(lm.basePath, fileName)

	file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return "", fmt.Errorf("创建日志文件失败: %w", err)
	}

	header := fmt.Sprintf("=== Quick Cmd 执行日志 ===\n项目: %s/%s\n开始时间: %s\n\n",
		projectName, subProjectName, time.Now().Format(time.RFC3339))
	if _, err := file.WriteString(header); err != nil {
		_ = file.Close()
		return "", err
	}

	lm.currentFile = file
	lm.currentMeta = &LogEntry{
		FileName:   fileName,
		FullPath:   fullPath,
		Project:    projectName,
		SubProject: subProjectName,
		StartedAt:  time.Now().Format(time.RFC3339),
		Success:    true,
	}
	return fullPath, nil
}

// WriteLine 写入一行日志
func (lm *LogManager) WriteLine(line string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if lm.currentFile == nil {
		return
	}
	_, _ = lm.currentFile.WriteString(line + "\n")
}

// FinishSession 结束当前日志会话
func (lm *LogManager) FinishSession(success bool, summary string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if lm.currentFile == nil {
		return
	}

	if lm.currentMeta != nil {
		lm.currentMeta.Success = success
	}

	footer := fmt.Sprintf("\n=== 结束 ===\n结果: %s\n%s\n", boolToResult(success), summary)
	_, _ = lm.currentFile.WriteString(footer)
	_ = lm.currentFile.Close()
	lm.currentFile = nil
	lm.currentMeta = nil
}

// ListLogs 列出最近日志
func (lm *LogManager) ListLogs(limit int) ([]LogEntry, error) {
	if limit <= 0 {
		limit = 50
	}

	entries, err := os.ReadDir(lm.basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []LogEntry{}, nil
		}
		return nil, err
	}

	result := make([]LogEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		project, subProject := parseLogFileName(entry.Name())
		result = append(result, LogEntry{
			FileName:   entry.Name(),
			FullPath:   filepath.Join(lm.basePath, entry.Name()),
			Project:    project,
			SubProject: subProject,
			StartedAt:  info.ModTime().Format(time.RFC3339),
			Size:       info.Size(),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].StartedAt > result[j].StartedAt
	})

	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

// ReadLog 读取日志内容
func (lm *LogManager) ReadLog(fileName string) (string, error) {
	fullPath := filepath.Join(lm.basePath, filepath.Base(fileName))
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func sanitizeFileName(name string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", " ", "_")
	result := replacer.Replace(name)
	if result == "" {
		return "unknown"
	}
	return result
}

func parseLogFileName(fileName string) (project, subProject string) {
	base := strings.TrimSuffix(fileName, ".log")
	parts := strings.SplitN(base, "_", 3)
	if len(parts) == 3 {
		return parts[1], parts[2]
	}
	return "", base
}

func boolToResult(ok bool) string {
	if ok {
		return "成功"
	}
	return "失败"
}
