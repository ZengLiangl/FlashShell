package machine

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"FlashDock/crypto"
	"FlashDock/data"
	"FlashDock/define"
	"FlashDock/utils"
)

func loadJZMachine(t *testing.T) *define.Machine {
	t.Helper()
	// 基础模式也需从钥匙串拉 DEK 才能解密 encrypted_data；App 启动会 InitVault，单测需自行补上。
	if err := crypto.InitVault(); err != nil {
		t.Fatalf("InitVault: %v", err)
	}
	if crypto.IsLocked() {
		if err := crypto.Unlock(""); err != nil {
			t.Fatalf("解锁凭据失败（基础模式应从钥匙串恢复 DEK）: %v", err)
		}
	}
	gcm := data.NewGlobalConfigManager("")
	cfg, err := gcm.LoadGlobalConfig()
	if err != nil {
		t.Fatalf("LoadGlobalConfig: %v", err)
	}
	for i := range cfg.Machines {
		m := &cfg.Machines[i]
		if m.Name == "jz" {
			return m
		}
	}
	t.Fatal("global_config 中未找到名为 jz 的机器")
	return nil
}

// 用 jz 验证并发 SFTP 下载吞吐，并对比旧的 Reader 包装串行路径。
// 运行：go test ./machine -run TestJZDownloadThroughput -count=1 -v
func TestJZDownloadThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("skip remote download bench in -short")
	}

	machine := loadJZMachine(t)
	const size = 16 << 20

	rm := define.NewRemoteMachine()
	if err := rm.Connect(machine, true); err != nil {
		t.Fatalf("Connect jz: %v", err)
	}
	defer rm.Close()

	remotePath := path.Join("/tmp", fmt.Sprintf("flashdock-sftp-dl-%d.bin", time.Now().UnixNano()))
	defer func() { _ = rm.SFTPClient.Remove(remotePath) }()

	// 先上传一份远端测试文件
	{
		srcLocal, err := os.CreateTemp("", "flashdock-sftp-dl-up-*.bin")
		if err != nil {
			t.Fatal(err)
		}
		upPath := srcLocal.Name()
		defer os.Remove(upPath)
		if _, err := srcLocal.Write(make([]byte, size)); err != nil {
			srcLocal.Close()
			t.Fatal(err)
		}
		_ = srcLocal.Close()

		f, err := os.Open(upPath)
		if err != nil {
			t.Fatal(err)
		}
		dst, err := rm.SFTPClient.Create(remotePath)
		if err != nil {
			f.Close()
			t.Fatal(err)
		}
		if _, err := utils.CopySFTPUpload(dst, f); err != nil {
			dst.Close()
			f.Close()
			t.Fatalf("seed upload: %v", err)
		}
		_ = dst.Close()
		_ = f.Close()
	}

	tmpDir := t.TempDir()

	serialPath := filepath.Join(tmpDir, "serial.bin")
	serialMBPS := benchDownload(t, rm, remotePath, serialPath, true)
	t.Logf("jz download (旧串行包装): %.2f MB/s", serialMBPS)

	pipePath := filepath.Join(tmpDir, "piped.bin")
	pipeMBPS := benchDownload(t, rm, remotePath, pipePath, false)
	t.Logf("jz download (并发 WriteTo): %.2f MB/s", pipeMBPS)

	// jz→本机下行常被云主机出口带宽卡住（OpenSSH scp 同样约 0.4MB/s），
	// 因此不设绝对吞吐阈值，只要求并发路径不慢于旧串行包装。
	if pipeMBPS+0.05 < serialMBPS {
		t.Fatalf("pipelined download slower than serial: piped=%.2f serial=%.2f", pipeMBPS, serialMBPS)
	}
}

func benchDownload(t *testing.T, rm *define.RemoteMachine, remotePath, localPath string, serialWrap bool) float64 {
	t.Helper()
	src, err := rm.SFTPClient.Open(remotePath)
	if err != nil {
		t.Fatalf("Open remote: %v", err)
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		t.Fatal(err)
	}

	dst, err := os.Create(localPath)
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()

	start := time.Now()
	var n int64
	if serialWrap {
		// 复现优化前：进度 Reader 包在 *sftp.File 外，挡住 WriteTo
		reader := &countingReader{r: src, total: info.Size()}
		n, err = utils.CopyBuffer(dst, reader)
	} else {
		writer := &countingWriter{w: dst, total: info.Size()}
		n, err = utils.CopySFTPDownload(writer, src)
	}
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	if n != info.Size() {
		t.Fatalf("downloaded %d, want %d", n, info.Size())
	}
	return float64(n) / elapsed.Seconds() / (1024 * 1024)
}
