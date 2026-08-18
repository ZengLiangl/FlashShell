package app

import (
	"os"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"time"
)

const (
	windowsFinalExeName   = ProductName + ".exe"
	envCleanupStaged      = "FLASHDOCK_CLEANUP_STAGED"
	envDeleteOldExe       = "FLASHDOCK_DELETE_OLD"
	applyUpdateFlag       = "--apply-update"
	applyUpdateTargetFlag = "--update-target"
	applyUpdatePIDFlag    = "--update-pid"
	applyUpdateLogFlag    = "--update-log"
	applyUpdateStagedFlag = "--update-staged"
)

// HandleEarlyUpdateArgs 在启动 UI 前处理更新相关参数。
// 返回 true 表示已处理完毕，进程应直接退出（勿再跑 Wails）。
func HandleEarlyUpdateArgs(args []string) bool {
	if goruntime.GOOS != "windows" {
		return false
	}
	if runWindowsApplyUpdateFromArgs(args) {
		return true
	}
	if maybeRelaunchNormalizedWindowsPortable(args) {
		return true
	}
	return false
}

// ConsumeWindowsUpdateCleanupEnv 清理上一轮更新留下的暂存目录 / 旧文件名。
func ConsumeWindowsUpdateCleanupEnv() {
	if goruntime.GOOS != "windows" {
		return
	}
	staged := strings.TrimSpace(os.Getenv(envCleanupStaged))
	oldExe := strings.TrimSpace(os.Getenv(envDeleteOldExe))
	if staged == "" && oldExe == "" {
		return
	}
	go func() {
		time.Sleep(2 * time.Second)
		if staged != "" {
			_ = os.RemoveAll(staged)
		}
		if oldExe != "" {
			_ = os.Remove(oldExe)
		}
	}()
}

type applyUpdateArgs struct {
	Target string
	Log    string
	Staged string
	PID    int
}

func parseApplyUpdateArgs(args []string) (applyUpdateArgs, bool) {
	var out applyUpdateArgs
	found := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		key, val, hasVal := splitArgKeyValue(arg)
		switch key {
		case applyUpdateFlag:
			found = true
		case applyUpdateTargetFlag:
			out.Target = takeArgValue(args, &i, val, hasVal)
		case applyUpdatePIDFlag:
			raw := takeArgValue(args, &i, val, hasVal)
			out.PID, _ = strconv.Atoi(strings.TrimSpace(raw))
		case applyUpdateLogFlag:
			out.Log = takeArgValue(args, &i, val, hasVal)
		case applyUpdateStagedFlag:
			out.Staged = takeArgValue(args, &i, val, hasVal)
		}
	}
	return out, found
}

func splitArgKeyValue(arg string) (key, val string, hasVal bool) {
	if strings.HasPrefix(arg, "--") {
		if i := strings.IndexByte(arg, '='); i > 0 {
			return arg[:i], arg[i+1:], true
		}
	}
	return arg, "", false
}

func takeArgValue(args []string, i *int, inline string, hasInline bool) string {
	if hasInline {
		return inline
	}
	if *i+1 < len(args) {
		*i++
		return args[*i]
	}
	return ""
}

// isWindowsReleaseAssetFileName 匹配发布资源名（如 FlashDock-1.1.13-Windows-Amd64.exe），
// 用于识别旧版 PS1 按 AssetName 改名后的路径，便于启动时归一到 FlashDock.exe。
// 自定义名（如 FlashDock2222.exe）不在此列，避免误改用户故意保留的文件名。
func isWindowsReleaseAssetFileName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || strings.EqualFold(name, windowsFinalExeName) {
		return false
	}
	lower := strings.ToLower(filepath.Base(name))
	if (!strings.HasPrefix(lower, "flashshell-") && !strings.HasPrefix(lower, "flashdock-")) || !strings.HasSuffix(lower, ".exe") {
		return false
	}
	if !strings.Contains(lower, "-windows-") {
		return false
	}
	return strings.Contains(lower, "-amd64") || strings.Contains(lower, "-arm64")
}

func windowsFinalExePath(targetExe string) string {
	dir := filepath.Dir(strings.TrimSpace(targetExe))
	return filepath.Join(dir, windowsFinalExeName)
}
