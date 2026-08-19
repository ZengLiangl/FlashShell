package data

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInitAppLogWritesAndPrunes(t *testing.T) {
	dir := IsolateConfigHome(t)
	t.Cleanup(func() {
		CloseAppLog()
		resetAppLogStateForTest()
	})

	logDir := filepath.Join(dir, AppLogDirName)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("mkdir logs: %v", err)
	}
	oldName := AppLogFilePrefix + time.Now().AddDate(0, 0, -AppLogRetentionDays-1).Format("2006-01-02") + AppLogFileSuffix
	oldPath := filepath.Join(logDir, oldName)
	if err := os.WriteFile(oldPath, []byte("old\n"), 0644); err != nil {
		t.Fatalf("write old log: %v", err)
	}
	keepName := AppLogFilePrefix + time.Now().AddDate(0, 0, -1).Format("2006-01-02") + AppLogFileSuffix
	keepPath := filepath.Join(logDir, keepName)
	if err := os.WriteFile(keepPath, []byte("keep\n"), 0644); err != nil {
		t.Fatalf("write keep log: %v", err)
	}

	path, err := InitAppLog()
	if err != nil {
		t.Fatalf("InitAppLog: %v", err)
	}
	if !strings.Contains(path, AppLogDirName) {
		t.Fatalf("log path not under logs/: %s", path)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("expected old log pruned, still exists: %v", err)
	}
	if _, err := os.Stat(keepPath); err != nil {
		t.Fatalf("expected recent log kept: %v", err)
	}

	AppLogf("hello %s", "world")
	fmt.Fprintf(os.Stdout, "from-stdout\n")
	// 给管道拷贝一点时间
	time.Sleep(50 * time.Millisecond)

	CloseAppLog()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "FlashShell 启动") {
		t.Fatalf("missing startup banner: %s", body)
	}
	if !strings.Contains(body, "hello world") {
		t.Fatalf("missing AppLogf line: %s", body)
	}
	if !strings.Contains(body, "from-stdout") {
		t.Fatalf("missing stdout mirror: %s", body)
	}
	if !strings.Contains(body, "FlashShell 退出") {
		t.Fatalf("missing shutdown banner: %s", body)
	}
}

func TestParseAppLogFileDay(t *testing.T) {
	day, ok := parseAppLogFileDay("flashshell-2026-08-19.log")
	if !ok {
		t.Fatal("expected parse ok")
	}
	if day.Format("2006-01-02") != "2026-08-19" {
		t.Fatalf("day=%v", day)
	}
	if _, ok := parseAppLogFileDay("other.log"); ok {
		t.Fatal("expected reject")
	}
}

func TestLogPanicWritesStack(t *testing.T) {
	_ = IsolateConfigHome(t)
	t.Cleanup(func() {
		CloseAppLog()
		resetAppLogStateForTest()
	})
	path, err := InitAppLog()
	if err != nil {
		t.Fatalf("InitAppLog: %v", err)
	}
	LogPanic("boom")
	CloseAppLog()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "PANIC: boom") {
		t.Fatalf("missing panic: %s", body)
	}
	if !strings.Contains(body, "goroutine") {
		t.Fatalf("missing stack: %s", body)
	}
}

func resetAppLogStateForTest() {
	appLogMu.Lock()
	defer appLogMu.Unlock()
	appLogFile = nil
	appLogPath = ""
	appLogWriter = nil
	origStdout = nil
	origStderr = nil
	logPipeW = nil
	logPipeDone = nil
	heartbeatStop = nil
	heartbeatDone = nil
}
