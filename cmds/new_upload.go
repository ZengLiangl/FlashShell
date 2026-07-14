package cmds

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"FlashDock/define"
	"FlashDock/utils"
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

	// 创建远程文件
	dstFile, err := rm.SFTPClient.Create(remotePath)
	if err != nil {
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
	}

	// 启动进度显示goroutine
	go progressWriter.startProgressDisplay()

	// 复制文件并显示进度
	_, err = io.Copy(progressWriter, srcFile)
	if err != nil {
		return fmt.Errorf("文件传输失败: %w", err)
	}
	time.Sleep(1 * time.Second)
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

// uploadDirectoryZip 使用 ZIP 压缩上传文件夹
func uploadDirectoryZip(rm *define.RemoteMachine, localPath, remotePath, targetFileName string, outputChan chan<- string) error {
	// 创建临时 ZIP 文件
	tempZipFile, err := os.CreateTemp("", "upload_*.zip")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer os.Remove(tempZipFile.Name()) // 确保删除临时文件
	defer tempZipFile.Close()

	// 使用 LocalZip 函数创建压缩包
	err = utils.LocalZip(localPath, tempZipFile.Name())
	if err != nil {
		return fmt.Errorf("创建ZIP压缩包失败: %w", err)
	}

	// 上传压缩包到远程临时位置
	tempRemotePath := fmt.Sprintf("/tmp/upload_%d.zip", time.Now().Unix())
	err = uploadFile(rm, tempZipFile.Name(), tempRemotePath, targetFileName, outputChan)
	if err != nil {
		return fmt.Errorf("上传压缩包失败: %w", err)
	}

	// 在远程解压并删除压缩包
	err = extractTarOnRemote(rm, tempRemotePath, remotePath, outputChan)
	if err != nil {
		return fmt.Errorf("远程解压失败: %w", err)
	}
	return nil
}

// extractTarOnRemote 在远程解压 tar 文件
func extractTarOnRemote(rm *define.RemoteMachine, remoteTarPath, targetPath string, outputChan chan<- string) error {
	// 创建 SSH session
	session, err := rm.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %w", err)
	}
	defer session.Close()

	// 构建解压命令
	extractCmd := fmt.Sprintf("mkdir -p %s && cd %s && unzip -o %s && rm -f %s",
		targetPath, targetPath, remoteTarPath, remoteTarPath)

	// 设置输出管道
	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("获取stdout管道失败: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("获取stderr管道失败: %w", err)
	}

	// 启动命令
	if err := session.Start(extractCmd); err != nil {
		return fmt.Errorf("启动解压命令失败: %w", err)
	}

	// 读取输出
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

	// 等待命令完成
	if err := session.Wait(); err != nil {
		return fmt.Errorf("解压命令执行失败: %w", err)
	}

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
}

// Write 实现io.Writer接口，记录写入的数据
func (pw *progressWriter) Write(p []byte) (n int, err error) {
	n, err = pw.wrapped.Write(p)
	if err != nil {
		return n, err
	}
	pw.written += int64(n)
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
