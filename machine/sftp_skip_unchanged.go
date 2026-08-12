package machine

import (
	"os"
	"time"
)

// SameSizeAndMtime 本地与远端文件大小、修改时间（秒）是否一致
func SameSizeAndMtime(localSize int64, localMod time.Time, remoteSize int64, remoteMod time.Time) bool {
	return localSize == remoteSize && localMod.Unix() == remoteMod.Unix()
}

// ShouldSkipUnchangedUpload 本地文件与远端同路径是否可跳过上传
func (a *ShellAuxManager) ShouldSkipUnchangedUpload(localPath, remotePath string) (bool, error) {
	localInfo, err := os.Stat(localPath)
	if err != nil {
		return false, err
	}
	if localInfo.IsDir() {
		return false, nil
	}
	if err := a.EnsureFileBackend(); err != nil {
		return false, err
	}
	if a.isSCPBackend() {
		return false, nil
	}
	c, err := a.sftpClient()
	if err != nil {
		return false, err
	}
	remoteInfo, err := c.Stat(remotePath)
	if err != nil {
		return false, nil
	}
	if remoteInfo.IsDir() {
		return false, nil
	}
	return SameSizeAndMtime(localInfo.Size(), localInfo.ModTime(), remoteInfo.Size(), remoteInfo.ModTime()), nil
}

// ShouldSkipUnchangedDownload 远端文件与本地路径是否可跳过下载
func (a *ShellAuxManager) ShouldSkipUnchangedDownload(remotePath, localPath string) (bool, error) {
	localInfo, err := os.Stat(localPath)
	if err != nil {
		return false, nil
	}
	if localInfo.IsDir() {
		return false, nil
	}
	if err := a.EnsureFileBackend(); err != nil {
		return false, err
	}
	if a.isSCPBackend() {
		return false, nil
	}
	c, err := a.sftpClient()
	if err != nil {
		return false, err
	}
	remoteInfo, err := c.Stat(remotePath)
	if err != nil {
		return false, err
	}
	if remoteInfo.IsDir() {
		return false, nil
	}
	return SameSizeAndMtime(localInfo.Size(), localInfo.ModTime(), remoteInfo.Size(), remoteInfo.ModTime()), nil
}

// PreserveRemoteMtime 将远端文件 mtime 设为与本地一致
func (a *ShellAuxManager) PreserveRemoteMtime(remotePath string, modTime time.Time) {
	if err := a.EnsureFileBackend(); err != nil || a.isSCPBackend() {
		return
	}
	c, err := a.sftpClient()
	if err != nil {
		return
	}
	_ = c.Chtimes(remotePath, modTime, modTime)
}
