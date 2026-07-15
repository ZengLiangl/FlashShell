package machine

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"FlashDock/utils"

	"github.com/pkg/sftp"
)

// TransferProgressFunc 传输进度回调
type TransferProgressFunc func(transferred, total int64, speedBPS float64)

type countingReader struct {
	ctx         context.Context
	r           io.Reader
	total       int64
	transferred int64
	onProgress  TransferProgressFunc
	lastReport  time.Time
	windowStart time.Time
	windowBytes int64
	speedBPS    float64
}

func (c *countingReader) Read(p []byte) (int, error) {
	if err := ctxErr(c.ctx); err != nil {
		return 0, err
	}
	n, err := c.r.Read(p)
	if n > 0 {
		c.transferred += int64(n)
		c.windowBytes += int64(n)
		now := time.Now()
		if c.windowStart.IsZero() {
			c.windowStart = now
		}
		elapsed := now.Sub(c.windowStart).Seconds()
		if elapsed >= 0.4 {
			c.speedBPS = float64(c.windowBytes) / elapsed
			c.windowStart = now
			c.windowBytes = 0
		}
		if c.onProgress != nil && (now.Sub(c.lastReport) >= 200*time.Millisecond || err == io.EOF || (c.total > 0 && c.transferred >= c.total)) {
			c.lastReport = now
			c.onProgress(c.transferred, c.total, c.speedBPS)
		}
	}
	return n, err
}

type countingWriter struct {
	ctx         context.Context
	w           io.Writer
	total       int64
	transferred int64
	onProgress  TransferProgressFunc
	lastReport  time.Time
	windowStart time.Time
	windowBytes int64
	speedBPS    float64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	if err := ctxErr(c.ctx); err != nil {
		return 0, err
	}
	n, err := c.w.Write(p)
	if n > 0 {
		c.transferred += int64(n)
		c.windowBytes += int64(n)
		now := time.Now()
		if c.windowStart.IsZero() {
			c.windowStart = now
		}
		elapsed := now.Sub(c.windowStart).Seconds()
		if elapsed >= 0.4 {
			c.speedBPS = float64(c.windowBytes) / elapsed
			c.windowStart = now
			c.windowBytes = 0
		}
		if c.onProgress != nil && (now.Sub(c.lastReport) >= 200*time.Millisecond || (c.total > 0 && c.transferred >= c.total)) {
			c.lastReport = now
			c.onProgress(c.transferred, c.total, c.speedBPS)
		}
	}
	return n, err
}

func ctxErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func (a *ShellAuxManager) sftpClient() (*sftp.Client, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.client == nil || a.client.remoteMachine == nil || a.client.remoteMachine.SFTPClient == nil {
		return nil, fmt.Errorf("SFTP 未连接")
	}
	return a.client.remoteMachine.SFTPClient, nil
}

// DownloadFile 下载远端文件到本地路径（支持断点续传：本地已有部分文件时从偏移继续）
func (a *ShellAuxManager) DownloadFile(ctx context.Context, remotePath, localPath string, onProgress TransferProgressFunc) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	sftpClient, err := a.sftpClient()
	if err != nil {
		return err
	}
	src, err := sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("打开远端文件失败: %w", err)
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return fmt.Errorf("获取远端文件信息失败: %w", err)
	}
	total := info.Size()
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return err
	}

	var offset int64
	if st, err := os.Stat(localPath); err == nil && st.Size() > 0 {
		if st.Size() >= total && total > 0 {
			if onProgress != nil {
				onProgress(total, total, 0)
			}
			return nil
		}
		if st.Size() < total {
			offset = st.Size()
		}
	}

	var dst *os.File
	if offset > 0 {
		if _, err := src.Seek(offset, io.SeekStart); err != nil {
			return fmt.Errorf("远端定位失败: %w", err)
		}
		dst, err = os.OpenFile(localPath, os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("打开本地文件失败: %w", err)
		}
		if _, err := dst.Seek(offset, io.SeekStart); err != nil {
			dst.Close()
			return fmt.Errorf("本地定位失败: %w", err)
		}
	} else {
		dst, err = os.Create(localPath)
		if err != nil {
			return fmt.Errorf("创建本地文件失败: %w", err)
		}
	}
	defer dst.Close()

	reader := &countingReader{ctx: ctx, r: src, total: total, transferred: offset, onProgress: onProgress}
	if onProgress != nil && offset > 0 {
		onProgress(offset, total, 0)
	}
	if _, err := io.Copy(dst, reader); err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	if onProgress != nil {
		onProgress(reader.transferred, total, reader.speedBPS)
	}
	return nil
}

// UploadFile 上传本地文件到远端路径（支持断点续传）
func (a *ShellAuxManager) UploadFile(ctx context.Context, localPath, remotePath string, onProgress TransferProgressFunc) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	sftpClient, err := a.sftpClient()
	if err != nil {
		return err
	}
	src, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %w", err)
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return fmt.Errorf("获取本地文件信息失败: %w", err)
	}
	total := info.Size()

	remoteDir := path.Dir(remotePath)
	if remoteDir != "" && remoteDir != "." && remoteDir != "/" {
		_ = sftpClient.MkdirAll(remoteDir)
	}

	var offset int64
	if rst, err := sftpClient.Stat(remotePath); err == nil && rst.Size() > 0 {
		if rst.Size() >= total && total > 0 {
			if onProgress != nil {
				onProgress(total, total, 0)
			}
			return nil
		}
		if rst.Size() < total {
			offset = rst.Size()
		}
	}

	var dst *sftp.File
	if offset > 0 {
		if _, err := src.Seek(offset, io.SeekStart); err != nil {
			return fmt.Errorf("本地定位失败: %w", err)
		}
		dst, err = sftpClient.OpenFile(remotePath, os.O_WRONLY)
		if err != nil {
			return fmt.Errorf("打开远端文件失败: %w", err)
		}
		if _, err := dst.Seek(offset, io.SeekStart); err != nil {
			dst.Close()
			return fmt.Errorf("远端定位失败: %w", err)
		}
	} else {
		dst, err = sftpClient.Create(remotePath)
		if err != nil {
			return fmt.Errorf("创建远端文件失败: %w", err)
		}
	}
	defer dst.Close()

	writer := &countingWriter{ctx: ctx, w: dst, total: total, transferred: offset, onProgress: onProgress}
	if onProgress != nil && offset > 0 {
		onProgress(offset, total, 0)
	}
	if _, err := io.Copy(writer, src); err != nil {
		return fmt.Errorf("上传失败: %w", err)
	}
	if onProgress != nil {
		onProgress(writer.transferred, total, writer.speedBPS)
	}
	return nil
}

// UploadDirectoryZip 将本地目录打成 zip 上传后在远端解压，保留本地原目录，删除远端压缩包
func (a *ShellAuxManager) UploadDirectoryZip(ctx context.Context, localDir, remoteParent string, onProgress TransferProgressFunc) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	info, err := os.Stat(localDir)
	if err != nil {
		return fmt.Errorf("本地目录不存在: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("不是目录: %s", localDir)
	}
	folderName := info.Name()
	remoteParent = strings.TrimSpace(remoteParent)
	if remoteParent == "" {
		remoteParent = "."
	}

	tempZip, err := os.CreateTemp("", "shell_upload_*.zip")
	if err != nil {
		return fmt.Errorf("创建临时压缩包失败: %w", err)
	}
	tempZipPath := tempZip.Name()
	_ = tempZip.Close()
	defer os.Remove(tempZipPath)

	if err := utils.LocalZip(localDir, tempZipPath); err != nil {
		return fmt.Errorf("压缩目录失败: %w", err)
	}
	if err := ctxErr(ctx); err != nil {
		return err
	}

	remoteZip := fmt.Sprintf("/tmp/flashdock_upload_%d_%s.zip", time.Now().UnixNano(), sanitizeName(folderName))
	if err := a.UploadFile(ctx, tempZipPath, remoteZip, onProgress); err != nil {
		return err
	}

	targetDir := path.Join(remoteParent, folderName)
	cmd := fmt.Sprintf("mkdir -p %s && unzip -o %s -d %s && rm -f %s",
		shellSingleQuote(targetDir),
		shellSingleQuote(remoteZip),
		shellSingleQuote(targetDir),
		shellSingleQuote(remoteZip),
	)
	out, err := a.Exec(cmd)
	if err != nil {
		_, _ = a.Exec("rm -f " + shellSingleQuote(remoteZip))
		return fmt.Errorf("远端解压失败: %w (%s)", err, strings.TrimSpace(out))
	}
	return nil
}

// DownloadDirectory 下载远端目录到本地目录。
// 优先远端 zip 打包后本机解压；若无 zip 则 SFTP 递归下载。
func (a *ShellAuxManager) DownloadDirectory(ctx context.Context, remoteDir, localDir string, onProgress TransferProgressFunc) (string, error) {
	if err := ctxErr(ctx); err != nil {
		return "", err
	}
	sftpClient, err := a.sftpClient()
	if err != nil {
		return "", err
	}
	info, err := sftpClient.Stat(remoteDir)
	if err != nil {
		return "", fmt.Errorf("远端目录不存在: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("不是目录: %s", remoteDir)
	}

	parent := path.Dir(remoteDir)
	base := path.Base(remoteDir)
	if parent == "" || parent == "." {
		parent = "/"
	}

	finalDir := localDir
	if err := os.MkdirAll(filepath.Dir(finalDir), 0o755); err != nil {
		return "", err
	}

	if err := a.downloadDirectoryViaZip(ctx, parent, base, finalDir, onProgress); err == nil {
		return finalDir, nil
	}

	if err := a.downloadDirectoryRecursive(ctx, remoteDir, finalDir, onProgress); err != nil {
		_ = os.RemoveAll(finalDir)
		return "", err
	}
	return finalDir, nil
}

func (a *ShellAuxManager) downloadDirectoryViaZip(ctx context.Context, parent, base, finalDir string, onProgress TransferProgressFunc) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	remoteZip := fmt.Sprintf("/tmp/flashdock_dl_%d_%s.zip", time.Now().UnixNano(), sanitizeName(base))
	cmd := fmt.Sprintf("cd %s && zip -rq %s %s",
		shellSingleQuote(parent),
		shellSingleQuote(remoteZip),
		shellSingleQuote(base),
	)
	out, err := a.Exec(cmd)
	if err != nil {
		_, _ = a.Exec("rm -f " + shellSingleQuote(remoteZip))
		return fmt.Errorf("远端 zip 失败: %w (%s)", err, strings.TrimSpace(out))
	}
	defer func() { _, _ = a.Exec("rm -f " + shellSingleQuote(remoteZip)) }()

	tempZip, err := os.CreateTemp("", "flashdock_dl_*.zip")
	if err != nil {
		return err
	}
	tempPath := tempZip.Name()
	_ = tempZip.Close()
	defer os.Remove(tempPath)

	if err := a.DownloadFile(ctx, remoteZip, tempPath, onProgress); err != nil {
		return err
	}

	staging := finalDir + ".extracting"
	_ = os.RemoveAll(staging)
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(staging)

	if err := utils.LocalUnzip(tempPath, staging); err != nil {
		return err
	}
	return promoteExtractedDir(staging, base, finalDir)
}

func (a *ShellAuxManager) downloadDirectoryRecursive(ctx context.Context, remoteDir, localDir string, onProgress TransferProgressFunc) error {
	sftpClient, err := a.sftpClient()
	if err != nil {
		return err
	}
	total, err := a.estimateRemoteDirSize(sftpClient, remoteDir)
	if err != nil {
		total = 0
	}
	var transferred int64
	progress := func(n int64) {
		transferred += n
		if onProgress != nil {
			onProgress(transferred, total, 0)
		}
	}
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return err
	}
	return a.copyRemoteDir(ctx, sftpClient, remoteDir, localDir, progress)
}

func (a *ShellAuxManager) estimateRemoteDirSize(c *sftp.Client, dir string) (int64, error) {
	var total int64
	var walk func(string) error
	walk = func(p string) error {
		entries, err := c.ReadDir(p)
		if err != nil {
			return err
		}
		for _, e := range entries {
			full := path.Join(p, e.Name())
			if e.IsDir() {
				if err := walk(full); err != nil {
					return err
				}
				continue
			}
			total += e.Size()
		}
		return nil
	}
	if err := walk(dir); err != nil {
		return 0, err
	}
	return total, nil
}

func (a *ShellAuxManager) copyRemoteDir(ctx context.Context, c *sftp.Client, remoteDir, localDir string, onChunk func(int64)) error {
	if err := ctxErr(ctx); err != nil {
		return err
	}
	entries, err := c.ReadDir(remoteDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := ctxErr(ctx); err != nil {
			return err
		}
		rpath := path.Join(remoteDir, e.Name())
		lpath := filepath.Join(localDir, e.Name())
		if e.IsDir() {
			if err := os.MkdirAll(lpath, 0o755); err != nil {
				return err
			}
			if err := a.copyRemoteDir(ctx, c, rpath, lpath, onChunk); err != nil {
				return err
			}
			continue
		}
		if err := a.downloadFileWithChunk(ctx, c, rpath, lpath, onChunk); err != nil {
			return fmt.Errorf("下载 %s 失败: %w", rpath, err)
		}
	}
	return nil
}

func (a *ShellAuxManager) downloadFileWithChunk(ctx context.Context, c *sftp.Client, remotePath, localPath string, onChunk func(int64)) error {
	ri, err := c.Stat(remotePath)
	if err != nil {
		return err
	}
	total := ri.Size()
	if st, err := os.Stat(localPath); err == nil {
		if st.Size() == total && total >= 0 {
			if onChunk != nil {
				onChunk(total)
			}
			return nil
		}
		// 不完整文件：截断后重下该文件（目录任务按文件粒度续传）
		if st.Size() > total {
			_ = os.Remove(localPath)
		}
	}

	src, err := c.Open(remotePath)
	if err != nil {
		return err
	}
	defer src.Close()
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return err
	}
	dst, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	buf := make([]byte, 32*1024)
	for {
		if err := ctxErr(ctx); err != nil {
			return err
		}
		n, readErr := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return werr
			}
			if onChunk != nil {
				onChunk(int64(n))
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	return nil
}

func promoteExtractedDir(staging, expectName, finalDir string) error {
	_ = expectName
	entries, err := os.ReadDir(staging)
	if err != nil {
		return err
	}
	var meaningful []os.DirEntry
	for _, e := range entries {
		name := e.Name()
		if name == "." || name == ".." {
			continue
		}
		meaningful = append(meaningful, e)
	}
	if len(meaningful) == 1 && meaningful[0].IsDir() {
		src := filepath.Join(staging, meaningful[0].Name())
		_ = os.RemoveAll(finalDir)
		if err := os.Rename(src, finalDir); err != nil {
			if copyErr := copyDir(src, finalDir); copyErr != nil {
				return fmt.Errorf("移动解压目录失败: %v / %v", err, copyErr)
			}
		}
		return nil
	}
	_ = os.RemoveAll(finalDir)
	if err := os.Rename(staging, finalDir); err != nil {
		if copyErr := copyDir(staging, finalDir); copyErr != nil {
			return fmt.Errorf("整理解压目录失败: %v / %v", err, copyErr)
		}
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "item"
	}
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_", "?", "_",
		"\"", "_", "<", "_", ">", "_", "|", "_", " ", "_",
	)
	return replacer.Replace(name)
}

// UniqueLocalPath 若目标已存在则追加 (1)(2)...
func UniqueLocalPath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		base = path
		ext = ""
	}
	for i := 1; i < 10000; i++ {
		candidate := fmt.Sprintf("%s (%d)%s", base, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return fmt.Sprintf("%s_%d%s", base, time.Now().UnixNano(), ext)
}

// ResolveFDDownloadDir 返回 Downloads/fddownload，必要时创建
func ResolveFDDownloadDir(resolveDownloads func() (string, error)) (string, error) {
	downloads, err := resolveDownloads()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(downloads, "fddownload")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}
