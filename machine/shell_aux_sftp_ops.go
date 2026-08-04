package machine

import (
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
	RemotePath string `json:"remotePath"`
	LocalSize  int64  `json:"localSize"`
	RemoteSize int64  `json:"remoteSize"`
	RemoteMtime int64 `json:"remoteMtime"`
	IsDir      bool   `json:"isDir"`
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
	conflict := &SftpUploadConflict{
		RemotePath: remotePath,
		LocalSize:  localInfo.Size(),
		RemoteSize: remoteInfo.Size(),
		RemoteMtime: remoteInfo.ModTime().Unix(),
		IsDir:      remoteInfo.IsDir(),
	}
	if localInfo.IsDir() != remoteInfo.IsDir() {
		return conflict, nil
	}
	if !localInfo.IsDir() && localInfo.Size() == remoteInfo.Size() {
		return nil, nil
	}
	return conflict, nil
}
