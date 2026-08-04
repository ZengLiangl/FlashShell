package app

import (
	"strings"

	"FlashDock/machine"
)

// MkdirShellRemotePath 创建远端目录
func (a *App) MkdirShellRemotePath(sessionID, remotePath string) error {
	aux, err := a.getShellAux(sessionID)
	if err != nil {
		return err
	}
	return aux.MkdirRemotePath(strings.TrimSpace(remotePath))
}

// RenameShellRemotePath 重命名远端路径
func (a *App) RenameShellRemotePath(sessionID, oldPath, newPath string) error {
	aux, err := a.getShellAux(sessionID)
	if err != nil {
		return err
	}
	return aux.RenameRemotePath(strings.TrimSpace(oldPath), strings.TrimSpace(newPath))
}

// ChmodShellRemotePath 修改远端权限（Unix mode，如 0755）
func (a *App) ChmodShellRemotePath(sessionID, remotePath string, mode uint32) error {
	aux, err := a.getShellAux(sessionID)
	if err != nil {
		return err
	}
	return aux.ChmodRemotePath(strings.TrimSpace(remotePath), mode)
}

// CopyShellRemotePath 同机复制远端文件或目录
func (a *App) CopyShellRemotePath(sessionID, srcPath, dstPath string) error {
	aux, err := a.getShellAux(sessionID)
	if err != nil {
		return err
	}
	return aux.CopyRemotePath(strings.TrimSpace(srcPath), strings.TrimSpace(dstPath))
}

// CheckShellUploadConflict 检测上传冲突
func (a *App) CheckShellUploadConflict(sessionID, localPath, remotePath string) (*machine.SftpUploadConflict, error) {
	aux, err := a.getShellAux(sessionID)
	if err != nil {
		return nil, err
	}
	return aux.CheckUploadConflict(localPath, remotePath)
}
