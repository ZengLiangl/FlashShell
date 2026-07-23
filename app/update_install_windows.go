//go:build windows

package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

var (
	modKernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procGetModuleFileNameW       = modKernel32.NewProc("GetModuleFileNameW")
	procGetLongPathNameW         = modKernel32.NewProc("GetLongPathNameW")
	procOpenProcess              = modKernel32.NewProc("OpenProcess")
	procQueryFullProcessImageNameW = modKernel32.NewProc("QueryFullProcessImageNameW")
)

const windowsProcessQueryLimitedInformation = 0x1000

func resolveWindowsUpdateTarget() (string, error) {
	candidates := make([]string, 0, 4)
	if path, err := getWindowsModuleFileName(); err == nil {
		candidates = append(candidates, path)
	}
	if path, err := os.Executable(); err == nil {
		candidates = append(candidates, path)
	}
	if path, err := queryWindowsProcessImagePath(os.Getpid()); err == nil {
		candidates = append(candidates, path)
	}
	if len(os.Args) > 0 {
		if arg0 := strings.TrimSpace(os.Args[0]); arg0 != "" {
			candidates = append(candidates, arg0)
		}
	}

	seen := make(map[string]struct{}, len(candidates))
	for _, raw := range candidates {
		key := strings.ToLower(strings.TrimSpace(raw))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if resolved, ok := normalizeWindowsUpdateTargetCandidate(raw); ok {
			return resolved, nil
		}
	}
	return "", fmt.Errorf("无法解析当前可执行文件路径")
}

func normalizeWindowsUpdateTargetCandidate(raw string) (string, bool) {
	path := strings.TrimSpace(raw)
	if path == "" {
		return "", false
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	path = getWindowsLongPath(path)
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return "", false
	}
	if !strings.EqualFold(filepath.Ext(path), ".exe") {
		return "", false
	}
	return path, true
}

func getWindowsModuleFileName() (string, error) {
	buf := make([]uint16, syscall.MAX_PATH)
	for {
		r, _, err := procGetModuleFileNameW.Call(0, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if r == 0 {
			return "", err
		}
		if r < uintptr(len(buf)) {
			return syscall.UTF16ToString(buf[:r]), nil
		}
		buf = make([]uint16, len(buf)*2)
	}
}

func getWindowsLongPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	pathUTF16, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return path
	}
	buf := make([]uint16, syscall.MAX_PATH)
	for {
		n, _, _ := procGetLongPathNameW.Call(
			uintptr(unsafe.Pointer(pathUTF16)),
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(len(buf)),
		)
		if n == 0 {
			return path
		}
		if n < uintptr(len(buf)) {
			return syscall.UTF16ToString(buf[:n])
		}
		buf = make([]uint16, len(buf)*2)
	}
}

func queryWindowsProcessImagePath(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("invalid pid: %d", pid)
	}
	handle, _, err := procOpenProcess.Call(
		uintptr(windowsProcessQueryLimitedInformation),
		0,
		uintptr(pid),
	)
	if handle == 0 {
		return "", err
	}
	defer syscall.CloseHandle(syscall.Handle(handle))

	buf := make([]uint16, syscall.MAX_PATH)
	size := uint32(len(buf))
	r, _, err := procQueryFullProcessImageNameW.Call(
		handle,
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if r == 0 {
		return "", err
	}
	path := strings.TrimSpace(syscall.UTF16ToString(buf[:size]))
	if path == "" {
		return "", fmt.Errorf("process image path is empty for pid %d", pid)
	}
	return path, nil
}

func buildWindowsPowerShellUpdateScript(pid int) string {
	script := `$ErrorActionPreference = 'Stop'
$Source = $env:FLASHDOCK_UPDATE_SOURCE
$Target = $env:FLASHDOCK_UPDATE_TARGET
$Staged = $env:FLASHDOCK_UPDATE_STAGED
$LogFile = $env:FLASHDOCK_UPDATE_LOG
$HostPid = [int]$env:FLASHDOCK_UPDATE_PID
$FinalName = ($env:FLASHDOCK_UPDATE_FINAL_NAME + '').Trim()

function Write-UpdateLog([string]$Message) {
  $line = '[{0}] {1}' -f (Get-Date -Format 'yyyy-MM-dd HH:mm:ss'), $Message
  Add-Content -Path $LogFile -Value $line
}

function Wait-ForHostExit {
  $deadline = (Get-Date).AddSeconds(90)
  while ((Get-Process -Id $HostPid -ErrorAction SilentlyContinue) -and (Get-Date) -lt $deadline) {
    Start-Sleep -Seconds 1
  }
  if (Get-Process -Id $HostPid -ErrorAction SilentlyContinue) {
    Write-UpdateLog "host process still running after 90 seconds, aborting update"
    exit 1
  }
}

function Resolve-SourceExecutable([string]$SourcePath, [string]$TargetPath, [string]$StagedDir) {
  $targetName = [System.IO.Path]::GetFileName($TargetPath)
  $sourceExt = [System.IO.Path]::GetExtension($SourcePath)
  if ($sourceExt -ieq '.zip') {
    $extractDir = Join-Path $StagedDir '_extract'
    if (Test-Path -Path $extractDir) {
      Remove-Item -Path $extractDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $extractDir -Force | Out-Null
    Expand-Archive -Path $SourcePath -DestinationPath $extractDir -Force
    $candidate = Join-Path $extractDir $targetName
    if (Test-Path -Path $candidate) {
      return $candidate
    }
    $found = Get-ChildItem -Path $extractDir -Filter '*.exe' -Recurse -File |
      Select-Object -First 1 -ExpandProperty FullName
    if ($found) {
      return $found
    }
    throw "no executable found in portable zip: $SourcePath"
  }
  return $SourcePath
}

function Replace-TargetExecutable([string]$SourceExe, [string]$TargetExe) {
  $targetOld = "$TargetExe.old"
  for ($retry = 0; $retry -lt 15; $retry++) {
    Write-UpdateLog "attempt ${retry}: trying rename-then-copy strategy"
    try {
      if (Test-Path -Path $TargetExe) {
        if (Test-Path -Path $targetOld) {
          Remove-Item -Path $targetOld -Force
        }
        Move-Item -Path $TargetExe -Destination $targetOld -Force
      }
      Copy-Item -Path $SourceExe -Destination $TargetExe -Force
      if (Test-Path -Path $targetOld) {
        Remove-Item -Path $targetOld -Force
      }
      return
    } catch {
      Write-UpdateLog "rename strategy failed: $($_.Exception.Message)"
      if (Test-Path -Path $targetOld) {
        try {
          if (Test-Path -Path $TargetExe) {
            Remove-Item -Path $TargetExe -Force
          }
          Move-Item -Path $targetOld -Destination $TargetExe -Force
        } catch {
          Write-UpdateLog "restore old executable failed: $($_.Exception.Message)"
        }
      }
    }

    Write-UpdateLog 'rename strategy failed, trying direct move'
    try {
      Move-Item -Path $SourceExe -Destination $TargetExe -Force
      return
    } catch {
      Write-UpdateLog "direct move failed: $($_.Exception.Message)"
    }
    try {
      Copy-Item -Path $SourceExe -Destination $TargetExe -Force
      return
    } catch {
      Write-UpdateLog "direct copy failed: $($_.Exception.Message)"
    }

    $wait = 1
    if ($retry -ge 3) { $wait = 2 }
    if ($retry -ge 6) { $wait = 3 }
    if ($retry -ge 9) { $wait = 5 }
    Write-UpdateLog "waiting $wait seconds before retry"
    Start-Sleep -Seconds $wait
  }
  throw 'replace failed after retries (portable mode, no elevation): check directory write permission or file lock'
}

function Resolve-FinalTargetPath([string]$TargetExe, [string]$DesiredName) {
  $DesiredName = ($DesiredName + '').Trim()
  if (-not $DesiredName) {
    return $TargetExe
  }
  if ($DesiredName -notmatch '(?i)\.exe$') {
    Write-UpdateLog "final name is not exe, skip rename: $DesiredName"
    return $TargetExe
  }
  if ($DesiredName -match '[\\/]') {
    Write-UpdateLog "final name contains path separator, skip rename: $DesiredName"
    return $TargetExe
  }
  $dir = [System.IO.Path]::GetDirectoryName($TargetExe)
  return (Join-Path $dir $DesiredName)
}

function Rename-TargetToFinalName([string]$TargetExe, [string]$FinalPath) {
  if ($TargetExe -ieq $FinalPath) {
    return $TargetExe
  }
  if (-not (Test-Path -Path $TargetExe)) {
    throw "target missing before rename: $TargetExe"
  }
  if (Test-Path -Path $FinalPath) {
    Write-UpdateLog "removing existing final path before rename: $FinalPath"
    Remove-Item -Path $FinalPath -Force
  }
  Move-Item -Path $TargetExe -Destination $FinalPath -Force
  Write-UpdateLog "renamed target: $TargetExe -> $FinalPath"
  return $FinalPath
}

function Start-UpdatedApplication([string]$TargetExe) {
  $targetDir = [System.IO.Path]::GetDirectoryName($TargetExe)
  # 必须用 Normal：Hidden 会把 Wails GUI 主窗口藏掉，用户感觉「没有自动打开」
  $proc = Start-Process -FilePath $TargetExe -WorkingDirectory $targetDir -WindowStyle Normal -PassThru -ErrorAction Stop
  if (-not $proc -or $proc.HasExited) {
    throw "relaunch failed for target: $TargetExe"
  }
  Start-Sleep -Milliseconds 800
  if ($proc.HasExited) {
    throw ("relaunch exited immediately: pid={0} code={1} path={2}" -f $proc.Id, $proc.ExitCode, $TargetExe)
  }
  Write-UpdateLog ("started updated application: pid={0} path={1}" -f $proc.Id, $TargetExe)
}

try {
  Write-UpdateLog 'updater started'
  Write-UpdateLog "source=$Source"
  Write-UpdateLog "target=$Target"
  Write-UpdateLog "finalName=$FinalName"

  if (-not (Test-Path -Path $Source)) {
    throw "source file not found: $Source"
  }
  if (-not (Test-Path -Path $Target)) {
    throw "target executable not found: $Target"
  }

  $sourceExe = Resolve-SourceExecutable -SourcePath $Source -TargetPath $Target -StagedDir $Staged
  Write-UpdateLog "resolved source executable: $sourceExe"

  Wait-ForHostExit
  Write-UpdateLog 'host process exited'
  Start-Sleep -Seconds 3
  Write-UpdateLog 'cooldown finished, starting file replace'

  Replace-TargetExecutable -SourceExe $sourceExe -TargetExe $Target
  # 统一重命名为 FlashDock.exe（不含版本号 / 平台标识）
  $finalPath = Resolve-FinalTargetPath -TargetExe $Target -DesiredName $FinalName
  $launchPath = $Target
  try {
    $launchPath = Rename-TargetToFinalName -TargetExe $Target -FinalPath $finalPath
  } catch {
    Write-UpdateLog "rename to final name failed, launch original path: $($_.Exception.Message)"
    $launchPath = $Target
  }
  Start-UpdatedApplication -TargetExe $launchPath
  if (Test-Path -Path $Staged) {
    Remove-Item -Path $Staged -Recurse -Force
  }
  Write-UpdateLog 'update finished'
  exit 0
} catch {
  Write-UpdateLog ("update failed: " + $_.Exception.Message)
  exit 1
}
`
	_ = pid
	return strings.ReplaceAll(script, "\n", "\r\n")
}

func windowsUpdateScriptEnv(source, target, stagedDir, logPath string, pid int, finalName string) []string {
	return []string{
		"FLASHDOCK_UPDATE_SOURCE=" + source,
		"FLASHDOCK_UPDATE_TARGET=" + target,
		"FLASHDOCK_UPDATE_STAGED=" + stagedDir,
		"FLASHDOCK_UPDATE_LOG=" + logPath,
		"FLASHDOCK_UPDATE_PID=" + strconv.Itoa(pid),
		"FLASHDOCK_UPDATE_FINAL_NAME=" + strings.TrimSpace(finalName),
	}
}
