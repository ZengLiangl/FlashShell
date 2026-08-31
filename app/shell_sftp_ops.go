package app

import (
	"fmt"
	"strings"

	"FlashDock/machine"
)

// MkdirShellRemotePath 创建远端目录
func (a *App) MkdirShellRemotePath(sessionID, remotePath string) error {
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	aux, err := a.getShellAux(sessionID)
	if err != nil {
		return err
	}
	return aux.MkdirRemotePath(strings.TrimSpace(remotePath))
}

// RenameShellRemotePath 重命名远端路径
func (a *App) RenameShellRemotePath(sessionID, oldPath, newPath string) error {
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	aux, err := a.getShellAux(sessionID)
	if err != nil {
		return err
	}
	return aux.RenameRemotePath(strings.TrimSpace(oldPath), strings.TrimSpace(newPath))
}

// ChmodShellRemotePath 修改远端权限（Unix mode，如 0755）
func (a *App) ChmodShellRemotePath(sessionID, remotePath string, mode uint32) error {
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	aux, err := a.getShellAux(sessionID)
	if err != nil {
		return err
	}
	return aux.ChmodRemotePath(strings.TrimSpace(remotePath), mode)
}

// CopyShellRemotePath 同机复制远端文件或目录
func (a *App) CopyShellRemotePath(sessionID, srcPath, dstPath string) error {
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	aux, err := a.getShellAux(sessionID)
	if err != nil {
		return err
	}
	return aux.CopyRemotePath(strings.TrimSpace(srcPath), strings.TrimSpace(dstPath))
}

// MoveShellRemotePath 同机移动远端文件或目录
func (a *App) MoveShellRemotePath(sessionID, srcPath, dstPath string) error {
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	aux, err := a.getShellAux(sessionID)
	if err != nil {
		return err
	}
	return aux.MoveRemotePath(strings.TrimSpace(srcPath), strings.TrimSpace(dstPath))
}

// CheckShellUploadConflict 检测上传冲突
func (a *App) CheckShellUploadConflict(sessionID, localPath, remotePath string) (*machine.SftpUploadConflict, error) {
	if err := a.requireUnlocked(); err != nil {
		return nil, err
	}
	aux, err := a.getShellAux(sessionID)
	if err != nil {
		return nil, err
	}
	return aux.CheckUploadConflict(localPath, remotePath)
}

// SendShellCd 向终端发送 cd 到指定路径（路径自动加引号）
func (a *App) SendShellCd(sessionID, remotePath string) error {
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	remotePath = strings.TrimSpace(remotePath)
	if remotePath == "" {
		return fmt.Errorf("路径为空")
	}
	quoted := machine.ShellQuotePath(remotePath)
	return a.SendShellInput(sessionID, "cd "+quoted+"\r")
}
