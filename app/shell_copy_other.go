package app

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"FlashDock/define"
	"FlashDock/machine"

	"github.com/google/uuid"
)

// ShellCopyToOtherResult 跨会话复制结果
type ShellCopyToOtherResult struct {
	Mode       string `json:"mode"`                 // instant | transfer
	TransferID string `json:"transferId,omitempty"` // transfer 模式时有值
	DestPath   string `json:"destPath"`
	SameHost   bool   `json:"sameHost"`
}

// SameShellHost 判断两个已归一化的机器配置名是否同机
func SameShellHost(srcConfig, dstConfig string) bool {
	srcConfig = strings.TrimSpace(srcConfig)
	dstConfig = strings.TrimSpace(dstConfig)
	return srcConfig != "" && srcConfig == dstConfig
}

// JoinCopyDestPath 拼出目标侧完整远端路径（取源 basename）
func JoinCopyDestPath(dstDir, srcPath string) string {
	name := path.Base(strings.TrimSpace(srcPath))
	if name == "" || name == "." || name == "/" {
		name = "copy"
	}
	return PathJoinRemote(dstDir, name)
}

// CopyToOtherMode 同机即时复制 / 异机走传输队列
func CopyToOtherMode(sameHost bool) string {
	if sameHost {
		return "instant"
	}
	return "transfer"
}

// StartShellCopyToOther 将源会话远端路径复制到目标会话目录。
// 同机：远端直接复制（冲突则分配唯一名）；异机：本机中转下载再上传。
func (a *App) StartShellCopyToOther(srcSession, srcPath, dstSession, dstDir string) (*ShellCopyToOtherResult, error) {
	srcSession = strings.TrimSpace(srcSession)
	dstSession = strings.TrimSpace(dstSession)
	srcPath = strings.TrimSpace(srcPath)
	dstDir = strings.TrimSpace(dstDir)
	if srcSession == "" || dstSession == "" || srcPath == "" || srcPath == "/" {
		return nil, fmt.Errorf("参数无效")
	}
	if dstDir == "" {
		return nil, fmt.Errorf("请先在目标终端进入有效目录")
	}
	if srcSession == dstSession {
		return nil, fmt.Errorf("目标不能是当前会话")
	}
	if machine.IsLocalShellID(srcSession) || machine.IsLocalShellID(dstSession) {
		return nil, fmt.Errorf("本地终端不支持跨会话复制")
	}

	srcCfg := a.remoteConfigName(srcSession)
	dstCfg := a.remoteConfigName(dstSession)
	sameHost := SameShellHost(srcCfg, dstCfg)
	mode := CopyToOtherMode(sameHost)

	srcAux, err := a.getShellAux(srcSession)
	if err != nil {
		return nil, fmt.Errorf("源会话未就绪: %w", err)
	}
	dstAux, err := a.getShellAux(dstSession)
	if err != nil {
		return nil, fmt.Errorf("目标会话未就绪: %w", err)
	}

	isDir, err := remotePathIsDir(srcAux, srcPath)
	if err != nil {
		return nil, err
	}

	destPath := JoinCopyDestPath(dstDir, srcPath)
	unique, err := dstAux.AllocateUniqueRemotePath(destPath)
	if err != nil {
		return nil, err
	}
	destPath = unique

	if sameHost {
		if err := srcAux.CopyRemotePath(srcPath, destPath); err != nil {
			return nil, err
		}
		a.recordInstantCopyTransfer(srcSession, dstSession, srcPath, destPath, isDir)
		return &ShellCopyToOtherResult{
			Mode:     mode,
			DestPath: destPath,
			SameHost: true,
		}, nil
	}

	id, err := a.launchCrossHostCopy(srcSession, dstSession, srcPath, destPath, isDir)
	if err != nil {
		return nil, err
	}
	return &ShellCopyToOtherResult{
		Mode:       mode,
		TransferID: id,
		DestPath:   destPath,
		SameHost:   false,
	}, nil
}

func remotePathIsDir(aux *machine.ShellAuxManager, remotePath string) (bool, error) {
	isDir, err := aux.DirExists(remotePath)
	if err != nil {
		return false, err
	}
	if isDir {
		return true, nil
	}
	parent := path.Dir(remotePath)
	base := path.Base(remotePath)
	entries, listErr := aux.ListDir(parent, true)
	if listErr != nil {
		return false, listErr
	}
	for _, e := range entries {
		if e.Name == base {
			return e.IsDir, nil
		}
	}
	return false, fmt.Errorf("远端路径不存在: %s", remotePath)
}

func (a *App) recordInstantCopyTransfer(srcSession, dstSession, srcPath, destPath string, isDir bool) {
	now := time.Now().Unix()
	rec := &define.SftpTransferRecord{
		ID:                uuid.NewString(),
		MachineName:       dstSession,
		SourceMachineName: srcSession,
		Direction:         "copy",
		Name:              path.Base(destPath),
		LocalPath:         "",
		RemotePath:        destPath,
		SourceRemotePath:  srcPath,
		IsDir:             isDir,
		Status:            "done",
		Percent:           100,
		StartedAt:         now,
		UpdatedAt:         now,
		FinishedAt:        now,
	}
	a.upsertTransfer(rec)
}

func (a *App) launchCrossHostCopy(srcSession, dstSession, srcPath, destPath string, isDir bool) (string, error) {
	id := uuid.NewString()
	now := time.Now().Unix()
	rec := &define.SftpTransferRecord{
		ID:                id,
		MachineName:       dstSession,
		SourceMachineName: srcSession,
		Direction:         "copy",
		Name:              path.Base(destPath),
		RemotePath:        destPath,
		SourceRemotePath:  srcPath,
		IsDir:             isDir,
		ConflictAction:    "replace",
		Status:            "pending",
		StartedAt:         now,
		UpdatedAt:         now,
	}
	a.upsertTransfer(rec)
	a.launchTransfer(rec)
	return id, nil
}

func (a *App) runCrossHostCopyTransfer(cp *define.SftpTransferRecord) error {
	srcSession := strings.TrimSpace(cp.SourceMachineName)
	dstSession := strings.TrimSpace(cp.MachineName)
	srcPath := strings.TrimSpace(cp.SourceRemotePath)
	destPath := strings.TrimSpace(cp.RemotePath)
	if srcSession == "" || dstSession == "" || srcPath == "" || destPath == "" {
		return fmt.Errorf("跨机复制参数不完整")
	}
	srcAux, err := a.shellAuxPool.Get(srcSession)
	if err != nil {
		return fmt.Errorf("源会话未就绪: %w", err)
	}
	dstAux, err := a.shellAuxPool.Get(dstSession)
	if err != nil {
		return fmt.Errorf("目标会话未就绪: %w", err)
	}

	tmpRoot, err := os.MkdirTemp("", "flashdock-copy-*")
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpRoot) }()

	localPath := filepath.Join(tmpRoot, path.Base(srcPath))
	ctx, cancel := context.WithCancel(context.Background())
	a.registerTransferCancel(cp.ID, cancel)
	defer a.unregisterTransferCancel(cp.ID)

	progress := func(transferred, total int64, speedBPS float64) {
		a.updateTransferProgress(cp.ID, transferred, total, speedBPS)
	}

	a.setTransferPhase(cp.ID, "downloading")
	if cp.IsDir {
		actual, err := srcAux.DownloadDirectory(ctx, srcPath, localPath, progress)
		if err != nil {
			return err
		}
		if actual != "" {
			localPath = actual
		}
	} else {
		if err := srcAux.DownloadFile(ctx, srcPath, localPath, progress); err != nil {
			return err
		}
	}

	store := a.ensureTransferStore()
	store.mu.Lock()
	if r, ok := store.items[cp.ID]; ok {
		r.LocalPath = localPath
	}
	store.mu.Unlock()

	a.setTransferPhase(cp.ID, "uploading")
	if cp.IsDir {
		_ = dstAux.RemovePath(destPath)
		// 跨机复制用递归上传，避免依赖目标机 unzip
		return dstAux.UploadDirectoryRecursive(ctx, localPath, destPath, progress)
	}
	if err := dstAux.UploadFile(ctx, localPath, destPath, progress); err != nil {
		return err
	}
	if info, err := os.Stat(localPath); err == nil {
		dstAux.PreserveRemoteMtime(destPath, info.ModTime())
	}
	return nil
}
