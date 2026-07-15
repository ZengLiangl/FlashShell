package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"FlashDock/define"
	"FlashDock/machine"

	"github.com/google/uuid"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const shellTransferEvent = "shell:transfer"

type shellTransferStore struct {
	mu      sync.Mutex
	items   map[string]*define.SftpTransferRecord
	cancels map[string]context.CancelFunc
}

func (a *App) ensureTransferStore() *shellTransferStore {
	if a.transfers == nil {
		a.transfers = &shellTransferStore{
			items:   make(map[string]*define.SftpTransferRecord),
			cancels: make(map[string]context.CancelFunc),
		}
	}
	return a.transfers
}

func (a *App) emitTransfer(rec *define.SftpTransferRecord) {
	if a.ctx == nil || rec == nil {
		return
	}
	cp := *rec
	wailsRuntime.EventsEmit(a.ctx, shellTransferEvent, cp)
}

func (a *App) upsertTransfer(rec *define.SftpTransferRecord) {
	store := a.ensureTransferStore()
	store.mu.Lock()
	defer store.mu.Unlock()
	store.items[rec.ID] = rec
	a.emitTransfer(rec)
}

func (a *App) registerTransferCancel(id string, cancel context.CancelFunc) {
	store := a.ensureTransferStore()
	store.mu.Lock()
	defer store.mu.Unlock()
	store.cancels[id] = cancel
}

func (a *App) unregisterTransferCancel(id string) {
	store := a.ensureTransferStore()
	store.mu.Lock()
	defer store.mu.Unlock()
	delete(store.cancels, id)
}

func (a *App) updateTransferProgress(id string, transferred, total int64, speedBPS float64) {
	store := a.ensureTransferStore()
	store.mu.Lock()
	rec, ok := store.items[id]
	if !ok || rec.Status == "paused" {
		store.mu.Unlock()
		return
	}
	rec.Transferred = transferred
	if total > 0 {
		rec.Total = total
		rec.Percent = float64(transferred) / float64(total) * 100
	}
	rec.SpeedBPS = speedBPS
	rec.Status = "running"
	rec.UpdatedAt = time.Now().Unix()
	cp := *rec
	store.mu.Unlock()
	// 异步推送进度，避免 Wails IPC 阻塞传输热路径
	go a.emitTransfer(&cp)
}

func (a *App) finishTransfer(id string, err error) {
	a.unregisterTransferCancel(id)
	store := a.ensureTransferStore()
	store.mu.Lock()
	rec, ok := store.items[id]
	if !ok {
		store.mu.Unlock()
		return
	}
	// 已被暂停则不覆盖为 error
	if rec.Status == "paused" {
		store.mu.Unlock()
		return
	}
	now := time.Now().Unix()
	rec.UpdatedAt = now
	rec.FinishedAt = now
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, define.ErrTransferPaused) {
			rec.Status = "paused"
			rec.Error = ""
			rec.SpeedBPS = 0
		} else {
			rec.Status = "error"
			rec.Error = err.Error()
		}
	} else {
		rec.Status = "done"
		rec.Percent = 100
		if rec.Total > 0 {
			rec.Transferred = rec.Total
		}
	}
	cp := *rec
	store.mu.Unlock()
	a.emitTransfer(&cp)
}

// PickShellUploadPaths 选择要上传的本地文件（可多选）
func (a *App) PickShellUploadPaths() ([]string, error) {
	files, err := wailsRuntime.OpenMultipleFilesDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择要上传的文件",
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// PickShellUploadFolder 选择要上传的本地文件夹
func (a *App) PickShellUploadFolder() (string, error) {
	dir, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择要上传的文件夹",
	})
	if err != nil {
		return "", err
	}
	return dir, nil
}

// GetShellDownloadDir 返回 Downloads/fddownload
func (a *App) GetShellDownloadDir() (string, error) {
	return machine.ResolveFDDownloadDir(resolveDownloadsDir)
}

// OpenShellDownloadDir 打开 fddownload 目录
func (a *App) OpenShellDownloadDir() error {
	dir, err := a.GetShellDownloadDir()
	if err != nil {
		return err
	}
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("打开下载目录失败: %w", err)
	}
	return nil
}

// ListShellTransfers 返回传输记录（新到旧）
func (a *App) ListShellTransfers() []define.SftpTransferRecord {
	store := a.ensureTransferStore()
	store.mu.Lock()
	defer store.mu.Unlock()
	out := make([]define.SftpTransferRecord, 0, len(store.items))
	for _, rec := range store.items {
		out = append(out, *rec)
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].StartedAt > out[i].StartedAt {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// ClearFinishedShellTransfers 清除已完成/失败/已暂停的记录
func (a *App) ClearFinishedShellTransfers() {
	store := a.ensureTransferStore()
	store.mu.Lock()
	defer store.mu.Unlock()
	for id, rec := range store.items {
		if rec.Status == "done" || rec.Status == "error" || rec.Status == "paused" {
			delete(store.items, id)
			delete(store.cancels, id)
		}
	}
}

// PauseShellTransfer 暂停进行中的传输
func (a *App) PauseShellTransfer(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("参数无效")
	}
	store := a.ensureTransferStore()
	store.mu.Lock()
	rec, ok := store.items[id]
	if !ok {
		store.mu.Unlock()
		return fmt.Errorf("记录不存在")
	}
	if rec.Status != "running" && rec.Status != "pending" {
		store.mu.Unlock()
		return fmt.Errorf("当前状态无法暂停")
	}
	cancel := store.cancels[id]
	rec.Status = "paused"
	rec.SpeedBPS = 0
	rec.UpdatedAt = time.Now().Unix()
	rec.FinishedAt = rec.UpdatedAt
	cp := *rec
	store.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	a.unregisterTransferCancel(id)
	a.emitTransfer(&cp)
	return nil
}

// ResumeShellTransfer 继续已暂停的传输（单文件断点续传；目录跳过已完成文件）
func (a *App) ResumeShellTransfer(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("参数无效")
	}
	store := a.ensureTransferStore()
	store.mu.Lock()
	rec, ok := store.items[id]
	if !ok {
		store.mu.Unlock()
		return fmt.Errorf("记录不存在")
	}
	if rec.Status != "paused" && rec.Status != "error" {
		store.mu.Unlock()
		return fmt.Errorf("仅已暂停/失败的任务可继续")
	}
	cp := *rec
	store.mu.Unlock()

	a.launchTransfer(&cp)
	return nil
}

// RemoveShellTransfer 删除传输记录；若进行中则先暂停
func (a *App) RemoveShellTransfer(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("参数无效")
	}
	store := a.ensureTransferStore()
	store.mu.Lock()
	rec, ok := store.items[id]
	if !ok {
		store.mu.Unlock()
		return nil
	}
	cancel := store.cancels[id]
	active := rec.Status == "running" || rec.Status == "pending"
	delete(store.items, id)
	delete(store.cancels, id)
	store.mu.Unlock()

	if active && cancel != nil {
		cancel()
	}
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, shellTransferEvent, map[string]interface{}{
			"id":     id,
			"status": "removed",
		})
	}
	return nil
}

func (a *App) launchTransfer(seed *define.SftpTransferRecord) {
	if seed == nil || seed.ID == "" {
		return
	}
	store := a.ensureTransferStore()
	store.mu.Lock()
	rec, ok := store.items[seed.ID]
	if !ok {
		rec = seed
		store.items[seed.ID] = rec
	}
	rec.Status = "pending"
	rec.Error = ""
	rec.SpeedBPS = 0
	rec.FinishedAt = 0
	rec.UpdatedAt = time.Now().Unix()
	cp := *rec
	store.mu.Unlock()
	a.emitTransfer(&cp)

	aux, err := a.shellAuxPool.Get(cp.MachineName)
	if err != nil {
		a.finishTransfer(cp.ID, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.registerTransferCancel(cp.ID, cancel)

	go func() {
		defer a.unregisterTransferCancel(cp.ID)
		a.updateTransferProgress(cp.ID, cp.Transferred, cp.Total, 0)
		progress := func(transferred, total int64, speedBPS float64) {
			a.updateTransferProgress(cp.ID, transferred, total, speedBPS)
		}
		var transferErr error
		if cp.Direction == "download" {
			if cp.IsDir {
				actual, err := aux.DownloadDirectory(ctx, cp.RemotePath, cp.LocalPath, progress)
				transferErr = err
				if err == nil && actual != "" && actual != cp.LocalPath {
					store := a.ensureTransferStore()
					store.mu.Lock()
					if r, ok := store.items[cp.ID]; ok {
						r.LocalPath = actual
						r.Name = filepath.Base(actual)
					}
					store.mu.Unlock()
				}
			} else {
				transferErr = aux.DownloadFile(ctx, cp.RemotePath, cp.LocalPath, progress)
			}
		} else {
			if cp.IsDir {
				remoteParent := path.Dir(cp.RemotePath)
				transferErr = aux.UploadDirectoryZip(ctx, cp.LocalPath, remoteParent, progress)
			} else {
				transferErr = aux.UploadFile(ctx, cp.LocalPath, cp.RemotePath, progress)
			}
		}
		a.finishTransfer(cp.ID, transferErr)
	}()
}

// StartShellDownload 下载远端路径到 Downloads/fddownload（异步）
func (a *App) StartShellDownload(machineName, remotePath string) (string, error) {
	remotePath = strings.TrimSpace(remotePath)
	if machineName == "" || remotePath == "" || remotePath == "/" {
		return "", fmt.Errorf("参数无效")
	}
	aux, err := a.shellAuxPool.Get(machineName)
	if err != nil {
		return "", err
	}

	downloadDir, err := a.GetShellDownloadDir()
	if err != nil {
		return "", err
	}

	isDir, err := aux.DirExists(remotePath)
	if err != nil {
		return "", err
	}
	if !isDir {
		parent := path.Dir(remotePath)
		base := path.Base(remotePath)
		entries, listErr := aux.ListDir(parent, true)
		if listErr != nil {
			return "", listErr
		}
		found := false
		for _, e := range entries {
			if e.Name == base {
				found = true
				isDir = e.IsDir
				break
			}
		}
		if !found {
			return "", fmt.Errorf("远端路径不存在: %s", remotePath)
		}
	}

	name := path.Base(remotePath)
	localPath := machine.UniqueLocalPath(filepath.Join(downloadDir, name))

	id := uuid.NewString()
	now := time.Now().Unix()
	rec := &define.SftpTransferRecord{
		ID:          id,
		MachineName: machineName,
		Direction:   "download",
		Name:        filepath.Base(localPath),
		LocalPath:   localPath,
		RemotePath:  remotePath,
		IsDir:       isDir,
		Status:      "pending",
		StartedAt:   now,
		UpdatedAt:   now,
	}
	a.upsertTransfer(rec)
	a.launchTransfer(rec)
	return id, nil
}

// StartShellUpload 上传本地路径到远端目录（异步）；目录以 zip 上传后解压
func (a *App) StartShellUpload(machineName, localPath, remoteDir string) (string, error) {
	localPath = strings.TrimSpace(localPath)
	remoteDir = strings.TrimSpace(remoteDir)
	if machineName == "" || localPath == "" {
		return "", fmt.Errorf("参数无效")
	}
	if remoteDir == "" {
		remoteDir = "."
	}
	info, err := os.Stat(localPath)
	if err != nil {
		return "", fmt.Errorf("本地路径不存在: %w", err)
	}
	if _, err := a.shellAuxPool.Get(machineName); err != nil {
		return "", err
	}

	name := info.Name()
	remotePath := path.Join(remoteDir, name)
	id := uuid.NewString()
	now := time.Now().Unix()
	rec := &define.SftpTransferRecord{
		ID:          id,
		MachineName: machineName,
		Direction:   "upload",
		Name:        name,
		LocalPath:   localPath,
		RemotePath:  remotePath,
		IsDir:       info.IsDir(),
		Status:      "pending",
		Total:       0,
		StartedAt:   now,
		UpdatedAt:   now,
	}
	if !info.IsDir() {
		rec.Total = info.Size()
	}
	a.upsertTransfer(rec)
	a.launchTransfer(rec)
	return id, nil
}

