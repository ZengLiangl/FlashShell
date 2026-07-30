package cmds

import (
	"FlashDock/define"
	"FlashDock/utils"
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func RegUploadCmd() {
	CmdManager.RegSpecialCmd("upload", doUpload)
}

func doUpload(rm *define.RemoteMachine, c []string, outputChan chan<- string) error {
	println("doUpload")
	if len(c) != 3 {
		return errors.New("参数错误: upload <本地路径> <远程路径>")
	}

	localPath := c[1]
	remotePath := c[2]

	// 检查本地文件是否存在
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("本地文件不存在: %w", err)
	}

	if fileInfo.IsDir() {
		// 文件夹上传，使用SFTP模式
		return uploadDirectory(rm, localPath, remotePath, fileInfo.Name(), outputChan)
	} else {
		// 单文件上传，使用SFTP模式
		return uploadFile(rm, localPath, remotePath, fileInfo.Name(), outputChan)
	}
}

// uploadFile 上传单个文件
func uploadFile(rm *define.RemoteMachine, localPath, remotePath, targetFileName string, outputChan chan<- string) error {
	// 打开本地文件
	srcFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %w", err)
	}
	defer srcFile.Close()

	// 获取文件信息
	fileInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 处理远程路径
	if strings.HasSuffix(remotePath, "/") {
		remotePath = remotePath + fileInfo.Name()
	}

	transferID := fmt.Sprintf("task-upload-%d", time.Now().UnixNano())
	reportTransfer(&define.SftpTransferRecord{
		ID: transferID, Direction: "upload", Name: targetFileName,
		LocalPath: localPath, RemotePath: remotePath, Status: "running",
		Total: fileInfo.Size(), StartedAt: time.Now().Unix(),
	})

	// 创建远程文件
	dstFile, err := rm.SFTPClient.Create(remotePath)
	if err != nil {
		reportTransfer(&define.SftpTransferRecord{
			ID: transferID, Direction: "upload", Name: targetFileName, Status: "error",
			Error: err.Error(), FinishedAt: time.Now().Unix(),
		})
		return fmt.Errorf("创建远程文件失败: %w", err)
	}
	defer dstFile.Close()

	// 创建进度写入器
	progressWriter := &progressWriter{
		wrapped:    dstFile,
		totalSize:  fileInfo.Size(),
		startTime:  time.Now(),
		fileName:   targetFileName,
		progressID: fmt.Sprintf("upload_%d_%s", time.Now().UnixNano(), targetFileName),
		outputChan: outputChan,
		transferID: transferID,
		localPath:  localPath,
		remotePath: remotePath,
	}

	// 启动进度显示goroutine
	go progressWriter.startProgressDisplay()

	// 复制文件并显示进度
	_, err = utils.CopyBuffer(progressWriter, srcFile)
	if err != nil {
		reportTransfer(&define.SftpTransferRecord{
			ID: transferID, Direction: "upload", Name: targetFileName, Status: "error",
			Error: err.Error(), FinishedAt: time.Now().Unix(),
		})
		return fmt.Errorf("文件传输失败: %w", err)
	}
	reportTransfer(&define.SftpTransferRecord{
		ID: transferID, Direction: "upload", Name: targetFileName,
		LocalPath: localPath, RemotePath: remotePath, Status: "done",
		Total: fileInfo.Size(), Transferred: fileInfo.Size(), Percent: 100,
		FinishedAt: time.Now().Unix(),
	})
	return nil
}

// uploadDirectory 上传文件夹
func uploadDirectory(rm *define.RemoteMachine, localPath, remotePath, targetFileName string, outputChan chan<- string) error {
	fmt.Printf("正在上传文件夹: %s -> %s\n", localPath, remotePath)

	// 使用 zip 管道上传
	err := uploadDirectoryZip(rm, localPath, remotePath, targetFileName, outputChan)
	if err != nil {
		utils.SendOutput(outputChan, fmt.Sprintf("文件夹上传失败: %s", err.Error()))
		return fmt.Errorf("文件夹上传失败: %w", err)
	}

	fmt.Printf("文件夹上传完成: %s -> %s\n", localPath, remotePath)

	return nil
}

func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

// uploadDirectoryZip 使用 ZIP 压缩上传文件夹；远端无解压工具时回退 SFTP 递归上传
func uploadDirectoryZip(rm *define.RemoteMachine, localPath, remotePath, targetFileName string, outputChan chan<- string) error {
	tempZipFile, err := os.CreateTemp("", "upload_*.zip")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tempZipPath := tempZipFile.Name()
	_ = tempZipFile.Close()
	defer os.Remove(tempZipPath)

	if err := utils.LocalZip(localPath, tempZipPath); err != nil {
		return fmt.Errorf("创建ZIP压缩包失败: %w", err)
	}

	tempRemotePath := fmt.Sprintf("/tmp/upload_%d.zip", time.Now().UnixNano())
	if err := uploadFile(rm, tempZipPath, tempRemotePath, targetFileName, outputChan); err != nil {
		return fmt.Errorf("上传压缩包失败: %w", err)
	}

	if err := extractZipOnRemote(rm, tempRemotePath, remotePath, outputChan); err != nil {
		_ = runRemoteCommand(rm, "rm -f "+shellSingleQuote(tempRemotePath), nil)
		utils.SendOutput(outputChan, "远端无可用解压工具，回退为 SFTP 递归上传…")
		if fallbackErr := uploadDirectoryRecursive(rm, localPath, remotePath, outputChan); fallbackErr != nil {
			return fmt.Errorf("远程解压失败且递归上传失败: %v / %v", err, fallbackErr)
		}
		return nil
	}
	return nil
}

// extractZipOnRemote 尝试 unzip / busybox / python 在远端解压
func extractZipOnRemote(rm *define.RemoteMachine, remoteZipPath, targetPath string, outputChan chan<- string) error {
	tq := shellSingleQuote(targetPath)
	zq := shellSingleQuote(remoteZipPath)
	candidates := []string{
		fmt.Sprintf("mkdir -p %s && unzip -o %s -d %s && rm -f %s", tq, zq, tq, zq),
		fmt.Sprintf("mkdir -p %s && busybox unzip -o %s -d %s && rm -f %s", tq, zq, tq, zq),
		fmt.Sprintf(
			"mkdir -p %s && python3 -c %s && rm -f %s",
			tq,
			shellSingleQuote(fmt.Sprintf("import zipfile; zipfile.ZipFile(%q).extractall(%q)", remoteZipPath, targetPath)),
			zq,
		),
		fmt.Sprintf(
			"mkdir -p %s && python -c %s && rm -f %s",
			tq,
			shellSingleQuote(fmt.Sprintf("import zipfile; zipfile.ZipFile(%q).extractall(%q)", remoteZipPath, targetPath)),
			zq,
		),
	}
	var lastErr error
	for _, cmd := range candidates {
		if err := runRemoteCommand(rm, cmd, outputChan); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("远端无可用解压工具")
	}
	return lastErr
}

func runRemoteCommand(rm *define.RemoteMachine, cmd string, outputChan chan<- string) error {
	session, err := rm.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %w", err)
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("获取stdout管道失败: %w", err)
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("获取stderr管道失败: %w", err)
	}

	if err := session.Start(cmd); err != nil {
		return fmt.Errorf("启动命令失败: %w", err)
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			utils.SendOutput(outputChan, scanner.Text())
		}
	}()
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			utils.SendOutput(outputChan, fmt.Sprintf("[STDERR] %s", scanner.Text()))
		}
	}()

	if err := session.Wait(); err != nil {
		return fmt.Errorf("命令执行失败: %w", err)
	}
	return nil
}

// uploadDirectoryRecursive 通过 SFTP 递归上传本地目录内容到远端目标路径
func uploadDirectoryRecursive(rm *define.RemoteMachine, localPath, remotePath string, outputChan chan<- string) error {
	if rm.SFTPClient == nil {
		return fmt.Errorf("SFTP 未连接")
	}
	if err := rm.SFTPClient.MkdirAll(remotePath); err != nil {
		return fmt.Errorf("创建远程目录失败: %w", err)
	}

	base, err := filepath.Abs(filepath.Clean(localPath))
	if err != nil {
		return fmt.Errorf("解析本地目录失败: %w", err)
	}

	var fileCount int
	err = filepath.Walk(base, func(p string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(base, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		remoteFile := path.Join(remotePath, filepath.ToSlash(rel))
		if info.IsDir() {
			return rm.SFTPClient.MkdirAll(remoteFile)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		fileCount++
		utils.SendOutput(outputChan, fmt.Sprintf("递归上传: %s", filepath.ToSlash(rel)))
		return uploadFile(rm, p, remoteFile, info.Name(), outputChan)
	})
	if err != nil {
		return err
	}
	utils.SendOutput(outputChan, fmt.Sprintf("SFTP 递归上传完成，共 %d 个文件", fileCount))
	return nil
}

// bytesToHumanReadable 将字节数转换为人类可读的格式
func bytesToHumanReadable(bytes int64) string {
	if bytes == 0 {
		return "0 KB"
	}

	unit := int64(1024)
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	unitLabels := []string{"KB", "MB", "GB", "TB", "PB"}
	for _, label := range unitLabels {
		if bytes < unit*1024 {
			return fmt.Sprintf("%.2f %s", float64(bytes)/float64(unit), label)
		}
		unit *= 1024
	}
	return fmt.Sprintf("%.2f PB", float64(bytes)/float64(unit))
}

// progressWriter 结构体用于追踪写入进度和速度
type progressWriter struct {
	wrapped    io.WriteCloser // 原始写入目标
	totalSize  int64          // 文件总大小
	written    int64          // 已写入字节数
	startTime  time.Time      // 开始时间
	fileName   string         // 文件名
	progressID string         // 进度唯一标识
	outputChan chan<- string  // 输出通道
	transferID string
	localPath  string
	remotePath string
}

// Write 实现io.Writer接口，记录写入的数据
func (pw *progressWriter) Write(p []byte) (n int, err error) {
	n, err = pw.wrapped.Write(p)
	if err != nil {
		return n, err
	}
	pw.written += int64(n)
	if pw.transferID != "" && pw.totalSize > 0 {
		reportTransfer(&define.SftpTransferRecord{
			ID: pw.transferID, Direction: "upload", Name: pw.fileName,
			LocalPath: pw.localPath, RemotePath: pw.remotePath, Status: "running",
			Total: pw.totalSize, Transferred: pw.written,
			Percent:   float64(pw.written) / float64(pw.totalSize) * 100,
			UpdatedAt: time.Now().Unix(),
		})
	}
	return n, err
}

// startProgressDisplay 启动进度显示
func (pw *progressWriter) startProgressDisplay() {
	ticker := time.NewTicker(500 * time.Millisecond) // 每500ms更新一次
	defer ticker.Stop()
	for range ticker.C {
		elapsed := time.Since(pw.startTime).Seconds()
		if elapsed > 0 && pw.written < pw.totalSize {
			speed := float64(pw.written) / elapsed / 1024 // KB/s
			progress := float64(pw.written) / float64(pw.totalSize) * 100
			// 构建当前进度行
			currentProgressLine := fmt.Sprintf("[%s] 进度: %.2f%%, 速度: %.2f KB/s", pw.fileName, progress, speed)
			// 使用特殊标记表示这是进度更新，包含进度ID
			utils.SendOutput(pw.outputChan, fmt.Sprintf("PROGRESS_UPDATE:%s:%s", pw.progressID, currentProgressLine))
		} else if pw.written >= pw.totalSize {
			// 传输完成
			elapsed := time.Since(pw.startTime).Seconds()
			if elapsed > 0 {
				speed := float64(pw.written) / elapsed / 1024 // KB/s
				// 传输完成，输出最终结果
				completionLine := fmt.Sprintf("[%s] 传输完成: 100%%, 文件大小: %s, 平均速度: %.2f KB/s, 耗时: %.2f秒", pw.fileName, bytesToHumanReadable(pw.totalSize), speed, elapsed)
				utils.SendOutput(pw.outputChan, fmt.Sprintf("PROGRESS_UPDATE:%s:%s", pw.progressID, completionLine))
			}
			return
		}
	}
}

// Close 关闭底层的io.WriteCloser
func (pw *progressWriter) Close() error {
	return pw.wrapped.Close()
}
