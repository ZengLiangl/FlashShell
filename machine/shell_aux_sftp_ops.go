package machine

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/pkg/sftp"
)

// MkdirRemotePath 创建远端目录
func (a *ShellAuxManager) MkdirRemotePath(remotePath string) error {
	if err := a.EnsureFileBackend(); err != nil {
		return err
	}
	if a.isSCPBackend() {
		return a.mkdirSCP(remotePath)
	}
	c, err := a.sftpClient()
	if err != nil {
		return err
	}
	return c.Mkdir(remotePath)
}

// RenameRemotePath 重命名远端路径
func (a *ShellAuxManager) RenameRemotePath(oldPath, newPath string) error {
	if err := a.EnsureFileBackend(); err != nil {
		return err
	}
	if a.isSCPBackend() {
		return a.renameSCP(oldPath, newPath)
	}
	c, err := a.sftpClient()
	if err != nil {
		return err
	}
	return c.Rename(oldPath, newPath)
}

// ChmodRemotePath 修改远端权限（mode 为 Unix 权限位，如 0755）
func (a *ShellAuxManager) ChmodRemotePath(remotePath string, mode uint32) error {
	if err := a.EnsureFileBackend(); err != nil {
		return err
	}
	if a.isSCPBackend() {
		return a.chmodSCP(remotePath, mode)
	}
	c, err := a.sftpClient()
	if err != nil {
		return err
	}
	return c.Chmod(remotePath, os.FileMode(mode))
}

// ReadSymlinkTarget 读取符号链接目标
func (a *ShellAuxManager) ReadSymlinkTarget(remotePath string) (string, error) {
	if err := a.EnsureFileBackend(); err != nil {
		return "", err
	}
	if a.isSCPBackend() {
		out, err := a.Exec("readlink -- " + shellQuotePath(remotePath))
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(out), nil
	}
	c, err := a.sftpClient()
	if err != nil {
		return "", err
	}
	target, err := c.ReadLink(remotePath)
	if err != nil {
		return "", err
	}
	return target, nil
}

// CopyRemotePath 同机复制文件或目录（递归）
func (a *ShellAuxManager) CopyRemotePath(srcPath, dstPath string) error {
	if err := a.EnsureFileBackend(); err != nil {
		return err
	}
	if a.isSCPBackend() {
		return a.copyRemoteSCP(srcPath, dstPath)
	}
	c, err := a.sftpClient()
	if err != nil {
		return err
	}
	info, err := c.Stat(srcPath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyRemoteDir(c, srcPath, dstPath)
	}
	return copyRemoteFile(c, srcPath, dstPath)
}

// MoveRemotePath 同机移动：优先 Rename，失败则复制后删除
func (a *ShellAuxManager) MoveRemotePath(srcPath, dstPath string) error {
	srcPath = strings.TrimSpace(srcPath)
	dstPath = strings.TrimSpace(dstPath)
	if srcPath == "" || dstPath == "" {
		return fmt.Errorf("路径为空")
	}
	if srcPath == dstPath {
		return nil
	}
	if err := a.EnsureFileBackend(); err != nil {
		return err
	}
	if a.isSCPBackend() {
		if err := a.renameSCP(srcPath, dstPath); err == nil {
			return nil
		}
		if err := a.copyRemoteSCP(srcPath, dstPath); err != nil {
			return err
		}
		return a.RemovePath(srcPath)
	}
	c, err := a.sftpClient()
	if err != nil {
		return err
	}
	if err := c.Rename(srcPath, dstPath); err == nil {
		return nil
	}
	if err := a.CopyRemotePath(srcPath, dstPath); err != nil {
		return err
	}
	return a.RemovePath(srcPath)
}

func copyRemoteFile(c *sftp.Client, src, dst string) error {
	in, err := c.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := c.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = ioCopy(out, in)
	return err
}

func copyRemoteDir(c *sftp.Client, src, dst string) error {
	if err := c.Mkdir(dst); err != nil {
		return err
	}
	entries, err := c.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		childSrc := path.Join(src, name)
		childDst := path.Join(dst, name)
		if e.IsDir() {
			if err := copyRemoteDir(c, childSrc, childDst); err != nil {
				return err
			}
		} else {
			if err := copyRemoteFile(c, childSrc, childDst); err != nil {
				return err
			}
		}
	}
	return nil
}

func ioCopy(dst io.Writer, src io.Reader) (int64, error) {
	buf := make([]byte, 32*1024)
	var written int64
	for {
		n, err := src.Read(buf)
		if n > 0 {
			m, werr := dst.Write(buf[:n])
			written += int64(m)
			if werr != nil {
				return written, werr
			}
		}
		if err != nil {
			if err == io.EOF {
				return written, nil
			}
			return written, err
		}
	}
}

// SftpUploadConflict 上传冲突信息
type SftpUploadConflict struct {
	RemotePath   string `json:"remotePath"`
	LocalSize    int64  `json:"localSize"`
	RemoteSize   int64  `json:"remoteSize"`
	RemoteMtime  int64  `json:"remoteMtime"`
	IsDir        bool   `json:"isDir"`        // 远端是否为目录
	LocalIsDir   bool   `json:"localIsDir"`   // 本地是否为目录
	ExistingType string `json:"existingType"` // file | directory
}

// CheckUploadConflict 检测上传目标是否已存在且与本地不同
func (a *ShellAuxManager) CheckUploadConflict(localPath, remotePath string) (*SftpUploadConflict, error) {
	localInfo, err := os.Stat(localPath)
	if err != nil {
		return nil, err
	}
	c, err := a.sftpClient()
	if err != nil {
		return nil, err
	}
	remoteInfo, err := c.Stat(remotePath)
	if err != nil {
		return nil, nil // 不存在则无冲突
	}
	existingType := "file"
	if remoteInfo.IsDir() {
		existingType = "directory"
	}
	conflict := &SftpUploadConflict{
		RemotePath:   remotePath,
		LocalSize:    localInfo.Size(),
		RemoteSize:   remoteInfo.Size(),
		RemoteMtime:  remoteInfo.ModTime().Unix(),
		IsDir:        remoteInfo.IsDir(),
		LocalIsDir:   localInfo.IsDir(),
		ExistingType: existingType,
	}
	if localInfo.IsDir() != remoteInfo.IsDir() {
		return conflict, nil
	}
	// 大小+修改时间一致则视为未变更，不当作冲突
	if !localInfo.IsDir() &&
		localInfo.Size() == remoteInfo.Size() &&
		localInfo.ModTime().Unix() == remoteInfo.ModTime().Unix() {
		return nil, nil
	}
	if !localInfo.IsDir() && localInfo.Size() == remoteInfo.Size() {
		// 仅大小相同仍提示冲突（mtime 不同可能内容不同）
		return conflict, nil
	}
	return conflict, nil
}

// AllocateUniqueRemotePath 在同目录下分配不冲突的远端路径（name、name (1)、name (2)…）
func (a *ShellAuxManager) AllocateUniqueRemotePath(remotePath string) (string, error) {
	c, err := a.sftpClient()
	if err != nil {
		return "", err
	}
	remotePath = path.Clean(strings.TrimSpace(remotePath))
	if remotePath == "" || remotePath == "." {
		return "", fmt.Errorf("远端路径无效")
	}
	if _, err := c.Stat(remotePath); err != nil {
		return remotePath, nil
	}
	dir := path.Dir(remotePath)
	base := path.Base(remotePath)
	ext := path.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	for i := 1; i < 10000; i++ {
		candidate := path.Join(dir, fmt.Sprintf("%s (%d)%s", stem, i, ext))
		if _, err := c.Stat(candidate); err != nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("无法为 %s 分配唯一文件名", base)
}
