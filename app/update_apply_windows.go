//go:build windows

package app

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func runWindowsApplyUpdateFromArgs(args []string) bool {
	cfg, ok := parseApplyUpdateArgs(args)
	if !ok {
		return false
	}
	code := runWindowsApplyUpdate(cfg)
	os.Exit(code)
	return true
}

func runWindowsApplyUpdate(cfg applyUpdateArgs) int {
	logPath := strings.TrimSpace(cfg.Log)
	if logPath == "" {
		logPath = filepath.Join(os.TempDir(), "flashdock-apply-update.log")
	}
	logf := func(format string, a ...interface{}) {
		line := fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, a...))
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return
		}
		_, _ = f.WriteString(line)
		_ = f.Close()
	}

	source, err := os.Executable()
	if err != nil {
		logf("resolve source failed: %v", err)
		return 1
	}
	if resolved, err := filepath.EvalSymlinks(source); err == nil {
		source = resolved
	}
	source = filepath.Clean(source)

	target := strings.TrimSpace(cfg.Target)
	if target == "" {
		logf("missing --update-target")
		return 1
	}
	target = filepath.Clean(target)

	logf("updater started (--apply-update)")
	logf("source=%s", source)
	logf("target=%s", target)
	logf("finalName=%s", windowsFinalExeName)
	logf("hostPid=%d", cfg.PID)

	if _, err := os.Stat(source); err != nil {
		logf("source file not found: %v", err)
		return 1
	}
	if _, err := os.Stat(target); err != nil {
		logf("target executable not found: %v", err)
		return 1
	}

	if err := waitForWindowsHostExit(cfg.PID, 90*time.Second, logf); err != nil {
		logf("%v", err)
		return 1
	}
	logf("host process exited")
	time.Sleep(3 * time.Second)
	logf("cooldown finished, starting file replace")

	if err := replaceWindowsExecutable(source, target, logf); err != nil {
		logf("replace failed: %v", err)
		return 1
	}

	finalPath := windowsFinalExePath(target)
	launchPath := target
	if !strings.EqualFold(target, finalPath) {
		if err := renameWindowsExecutable(target, finalPath, logf); err != nil {
			logf("rename to final name failed, launch original path: %v", err)
		} else {
			launchPath = finalPath
		}
	} else {
		logf("target already final name: %s", finalPath)
	}

	env := append(os.Environ(), envCleanupStaged+"="+strings.TrimSpace(cfg.Staged))
	if err := startWindowsUpdatedApplication(launchPath, env, logf); err != nil {
		logf("relaunch failed: %v", err)
		return 1
	}

	logf("update finished")
	return 0
}

func waitForWindowsHostExit(pid int, timeout time.Duration, logf func(string, ...interface{})) error {
	if pid <= 0 {
		return nil
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !windowsProcessExists(pid) {
			return nil
		}
		time.Sleep(time.Second)
	}
	if windowsProcessExists(pid) {
		return fmt.Errorf("host process still running after %s, aborting update", timeout)
	}
	return nil
}

func windowsProcessExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	handle, _, _ := procOpenProcess.Call(
		uintptr(windowsProcessQueryLimitedInformation),
		0,
		uintptr(pid),
	)
	if handle == 0 {
		return false
	}
	_ = syscall.CloseHandle(syscall.Handle(handle))
	return true
}

func replaceWindowsExecutable(source, target string, logf func(string, ...interface{})) error {
	targetOld := target + ".old"
	same, _ := sameWindowsFile(source, target)
	if same {
		logf("source and target are the same file, skip content replace")
		return nil
	}

	for retry := 0; retry < 15; retry++ {
		logf("attempt %d: trying rename-then-copy strategy", retry)
		err := func() error {
			if _, err := os.Stat(target); err == nil {
				_ = os.Remove(targetOld)
				if err := os.Rename(target, targetOld); err != nil {
					return err
				}
			}
			if err := copyFile(source, target); err != nil {
				if _, statErr := os.Stat(targetOld); statErr == nil {
					_ = os.Remove(target)
					_ = os.Rename(targetOld, target)
				}
				return err
			}
			_ = os.Remove(targetOld)
			return nil
		}()
		if err == nil {
			return nil
		}
		logf("rename strategy failed: %v", err)

		logf("rename strategy failed, trying direct copy")
		if err := copyFile(source, target); err == nil {
			return nil
		} else {
			logf("direct copy failed: %v", err)
		}

		wait := time.Second
		if retry >= 3 {
			wait = 2 * time.Second
		}
		if retry >= 6 {
			wait = 3 * time.Second
		}
		if retry >= 9 {
			wait = 5 * time.Second
		}
		logf("waiting %s before retry", wait)
		time.Sleep(wait)
	}
	return fmt.Errorf("replace failed after retries (portable mode, no elevation): check directory write permission or file lock")
}

func renameWindowsExecutable(from, to string, logf func(string, ...interface{})) error {
	if strings.EqualFold(from, to) {
		return nil
	}
	if _, err := os.Stat(from); err != nil {
		return fmt.Errorf("target missing before rename: %w", err)
	}
	if _, err := os.Stat(to); err == nil {
		logf("removing existing final path before rename: %s", to)
		if err := os.Remove(to); err != nil {
			return err
		}
	}
	if err := os.Rename(from, to); err != nil {
		return err
	}
	logf("renamed target: %s -> %s", from, to)
	return nil
}

func startWindowsUpdatedApplication(targetExe string, env []string, logf func(string, ...interface{})) error {
	cmd := exec.Command(targetExe)
	cmd.Dir = filepath.Dir(targetExe)
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windowsCreateBreakawayFromJob,
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if cmd.Process != nil {
		_ = cmd.Process.Release()
	}
	time.Sleep(800 * time.Millisecond)
	logf("started updated application: path=%s", targetExe)
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func sameWindowsFile(a, b string) (bool, error) {
	ai, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	bi, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	return os.SameFile(ai, bi), nil
}

// maybeRelaunchNormalizedWindowsPortable 处理旧版 PS1 留下的版本号文件名：
// 复制为 FlashShell.exe 并拉起，再退出自身（由新进程删除旧文件）。
// 仅匹配发布资源名；自定义名不自动改名。
func maybeRelaunchNormalizedWindowsPortable(args []string) bool {
	if _, isApply := parseApplyUpdateArgs(args); isApply {
		return false
	}
	exe, err := resolveWindowsUpdateTarget()
	if err != nil {
		return false
	}
	base := filepath.Base(exe)
	if !isWindowsReleaseAssetFileName(base) {
		return false
	}
	finalPath := windowsFinalExePath(exe)
	if strings.EqualFold(exe, finalPath) {
		return false
	}

	logPath := filepath.Join(filepath.Dir(exe), "flashdock-normalize.log")
	logf := func(format string, a ...interface{}) {
		line := fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, a...))
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return
		}
		_, _ = f.WriteString(line)
		_ = f.Close()
	}
	logf("normalize portable name: %s -> %s", exe, finalPath)

	if _, err := os.Stat(finalPath); err == nil {
		logf("removing existing %s", finalPath)
		if err := os.Remove(finalPath); err != nil {
			logf("remove existing final failed: %v", err)
			return false
		}
	}
	if err := copyFile(exe, finalPath); err != nil {
		logf("copy to final name failed: %v", err)
		return false
	}

	env := append(os.Environ(), envDeleteOldExe+"="+exe)
	if err := startWindowsUpdatedApplication(finalPath, env, logf); err != nil {
		logf("relaunch final name failed: %v", err)
		return false
	}
	os.Exit(0)
	return true
}
