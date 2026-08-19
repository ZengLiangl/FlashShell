package data

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

const (
	// AppLogDirName 运行日志子目录（位于 ConfigHomeDir 下）
	AppLogDirName = "logs"
	// AppLogFilePrefix 按天日志文件名前缀
	AppLogFilePrefix = "flashshell-"
	// AppLogFileSuffix 按天日志文件名后缀
	AppLogFileSuffix = ".log"
	// AppLogRetentionDays 自动清理保留天数
	AppLogRetentionDays = 5
	appLogHeartbeatEvery = 30 * time.Second
)

var (
	appLogMu       sync.Mutex
	appLogFile     *os.File
	appLogPath     string
	appLogWriter   io.Writer
	origStdout     *os.File
	origStderr     *os.File
	logPipeW       *os.File
	logPipeDone    chan struct{}
	heartbeatStop  chan struct{}
	heartbeatDone  chan struct{}
)

// bestEffortWriter 写入失败时忽略错误，避免控制台句柄无效拖垮 MultiWriter。
type bestEffortWriter struct {
	w io.Writer
}

func (b bestEffortWriter) Write(p []byte) (int, error) {
	if b.w == nil {
		return len(p), nil
	}
	_, _ = b.w.Write(p)
	return len(p), nil
}

type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

// DefaultAppLogDir 返回全局配置目录下的 logs 目录。
func DefaultAppLogDir() (string, error) {
	configHome, err := ConfigHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configHome, AppLogDirName), nil
}

// AppLogPathForDay 返回指定日期的日志文件路径。
func AppLogPathForDay(day time.Time) (string, error) {
	dir, err := DefaultAppLogDir()
	if err != nil {
		return "", err
	}
	name := AppLogFilePrefix + day.Format("2006-01-02") + AppLogFileSuffix
	return filepath.Join(dir, name), nil
}

// CurrentAppLogPath 返回当前已打开的日志路径；未初始化时为空。
func CurrentAppLogPath() string {
	appLogMu.Lock()
	defer appLogMu.Unlock()
	return appLogPath
}

// InitAppLog 在配置目录 logs/ 下按天打开日志，将 stdout/stderr 镜像到文件，并清理过期文件。
// 路径固定，不提供自定义；失败时不影响应用继续启动。
func InitAppLog() (string, error) {
	appLogMu.Lock()
	defer appLogMu.Unlock()
	if appLogFile != nil {
		return appLogPath, nil
	}

	dir, err := DefaultAppLogDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("创建日志目录失败: %w", err)
	}

	now := time.Now()
	path := filepath.Join(dir, AppLogFilePrefix+now.Format("2006-01-02")+AppLogFileSuffix)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("打开日志文件失败: %w", err)
	}

	_ = pruneOldAppLogs(dir, now, AppLogRetentionDays)

	fileWriter := &lockedWriter{w: f}
	origStdout = os.Stdout
	origStderr = os.Stderr
	mirror := io.MultiWriter(fileWriter, bestEffortWriter{origStdout})

	pr, pw, err := os.Pipe()
	if err != nil {
		_ = f.Close()
		return "", fmt.Errorf("创建日志管道失败: %w", err)
	}
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(mirror, pr)
		_ = pr.Close()
		close(done)
	}()

	os.Stdout = pw
	os.Stderr = pw
	logPipeW = pw
	logPipeDone = done
	appLogFile = f
	appLogPath = path
	appLogWriter = fileWriter

	// 运行时 fatal / 未捕获 panic 额外落到同一文件（复制 fd，关闭日志文件后仍可用）
	_ = debug.SetCrashOutput(f, debug.CrashOptions{})

	stamp := now.Format("2006-01-02 15:04:05")
	_, _ = fmt.Fprintf(appLogWriter, "\n======== FlashShell 启动 %s pid=%d log=%s ========\n", stamp, os.Getpid(), path)
	_ = f.Sync()

	heartbeatStop = make(chan struct{})
	heartbeatDone = make(chan struct{})
	go runAppLogHeartbeat(heartbeatStop, heartbeatDone)

	return path, nil
}

func runAppLogHeartbeat(stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(appLogHeartbeatEvery)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			AppLogf("heartbeat alive")
		}
	}
}

// CloseAppLog 写入关闭记录并关闭日志文件。
func CloseAppLog() {
	appLogMu.Lock()
	stop := heartbeatStop
	done := heartbeatDone
	heartbeatStop = nil
	heartbeatDone = nil
	appLogMu.Unlock()
	if stop != nil {
		close(stop)
		<-done
	}

	appLogMu.Lock()
	defer appLogMu.Unlock()
	if appLogFile == nil {
		return
	}
	stamp := time.Now().Format("2006-01-02 15:04:05")
	_, _ = fmt.Fprintf(appLogFile, "======== FlashShell 退出 %s ========\n", stamp)
	_ = appLogFile.Sync()

	if logPipeW != nil {
		_ = logPipeW.Close()
		logPipeW = nil
	}
	if logPipeDone != nil {
		<-logPipeDone
		logPipeDone = nil
	}
	_ = debug.SetCrashOutput(nil, debug.CrashOptions{})
	_ = appLogFile.Close()
	appLogFile = nil
	appLogWriter = nil
	appLogPath = ""
	if origStdout != nil {
		os.Stdout = origStdout
		origStdout = nil
	}
	if origStderr != nil {
		os.Stderr = origStderr
		origStderr = nil
	}
}

// AppLogf 写入一条带时间戳的应用日志，并尽量立刻刷盘（便于闪退后仍能读到）。
func AppLogf(format string, args ...interface{}) {
	appLogMu.Lock()
	w := appLogWriter
	f := appLogFile
	appLogMu.Unlock()
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	stamp := time.Now().Format("2006-01-02 15:04:05")
	_, _ = fmt.Fprintf(w, "%s %s\n", stamp, msg)
	if f != nil {
		_ = f.Sync()
	}
}

// LogPanic 将 recover 到的 panic 与堆栈写入日志。
func LogPanic(recovered interface{}) {
	appLogMu.Lock()
	w := appLogWriter
	f := appLogFile
	if w == nil && f != nil {
		w = f
	}
	appLogMu.Unlock()
	if w == nil {
		return
	}
	stamp := time.Now().Format("2006-01-02 15:04:05")
	_, _ = fmt.Fprintf(w, "%s PANIC: %v\n%s\n", stamp, recovered, debug.Stack())
	if f != nil {
		_ = f.Sync()
	}
}

func pruneOldAppLogs(dir string, now time.Time, keepDays int) error {
	if keepDays <= 0 {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	// 保留最近 keepDays 个自然日（含当天），更早的按文件名日期删除
	oldestKeep := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).
		AddDate(0, 0, -(keepDays - 1))
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		name := ent.Name()
		day, ok := parseAppLogFileDay(name)
		if !ok {
			continue
		}
		if day.Before(oldestKeep) {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
	return nil
}

func parseAppLogFileDay(name string) (time.Time, bool) {
	if !strings.HasPrefix(name, AppLogFilePrefix) || !strings.HasSuffix(name, AppLogFileSuffix) {
		return time.Time{}, false
	}
	mid := strings.TrimSuffix(strings.TrimPrefix(name, AppLogFilePrefix), AppLogFileSuffix)
	day, err := time.ParseInLocation("2006-01-02", mid, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return day, true
}
