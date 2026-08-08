package machine

import (
	"crypto/rand"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"FlashDock/data"
	"FlashDock/define"
	"FlashDock/utils"
)

// 用全局配置中的 jz 机器验证并发 SFTP 上传吞吐。
// 运行：go test ./machine -run TestJZUploadThroughput -count=1 -v
func TestJZUploadThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("skip remote upload bench in -short")
	}

	gcm := data.NewGlobalConfigManager("")
	cfg, err := gcm.LoadGlobalConfig()
	if err != nil {
		t.Fatalf("LoadGlobalConfig: %v", err)
	}

	var machine *define.Machine
	for i := range cfg.Machines {
		m := &cfg.Machines[i]
		if m.Name == "jz" {
			machine = m
			break
		}
	}
	if machine == nil {
		t.Fatal("global_config 中未找到名为 jz 的机器")
	}

	const size = 16 << 20 // 16 MiB：足够体现流水线，又不会太久
	localFile, err := os.CreateTemp("", "flashdock-sftp-bench-*.bin")
	if err != nil {
		t.Fatal(err)
	}
	localPath := localFile.Name()
	defer os.Remove(localPath)

	buf := make([]byte, 1<<20)
	var written int64
	for written < size {
		n := len(buf)
		if int64(n) > size-written {
			n = int(size - written)
		}
		if _, err := rand.Read(buf[:n]); err != nil {
			localFile.Close()
			t.Fatal(err)
		}
		if _, err := localFile.Write(buf[:n]); err != nil {
			localFile.Close()
			t.Fatal(err)
		}
		written += int64(n)
	}
	if err := localFile.Close(); err != nil {
		t.Fatal(err)
	}

	rm := define.NewRemoteMachine()
	if err := rm.Connect(machine, true); err != nil {
		t.Fatalf("Connect jz: %v", err)
	}
	defer rm.Close()

	remotePath := path.Join("/tmp", fmt.Sprintf("flashdock-sftp-bench-%d.bin", time.Now().UnixNano()))
	defer func() { _ = rm.SFTPClient.Remove(remotePath) }()

	src, err := os.Open(localPath)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()

	dst, err := rm.SFTPClient.Create(remotePath)
	if err != nil {
		t.Fatalf("Create remote: %v", err)
	}

	start := time.Now()
	n, err := utils.CopySFTPUpload(dst, src)
	_ = dst.Close()
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("CopySFTPUpload: %v", err)
	}
	if n != size {
		t.Fatalf("uploaded %d bytes, want %d", n, size)
	}

	st, err := rm.SFTPClient.Stat(remotePath)
	if err != nil {
		t.Fatalf("Stat remote: %v", err)
	}
	if st.Size() != size {
		t.Fatalf("remote size %d, want %d", st.Size(), size)
	}

	mbps := float64(n) / elapsed.Seconds() / (1024 * 1024)
	kbps := float64(n) / elapsed.Seconds() / 1024
	t.Logf("jz upload: %s in %s → %.2f MB/s (%.0f KB/s)",
		bytesHuman(n), elapsed.Round(time.Millisecond), mbps, kbps)

	// 串行 SFTP 在高延迟下常见 <1MB/s；优化后应明显高于该水位。
	// 阈值保守，避免偶发抖动误杀；真正目标是接近测速上行。
	if mbps < 1.5 {
		t.Fatalf("upload too slow: %.2f MB/s (expect pipelined SFTP ≫ 0.5 MB/s)", mbps)
	}
}

func bytesHuman(n int64) string {
	const mb = 1024 * 1024
	if n >= mb {
		return fmt.Sprintf("%.2f MB", float64(n)/mb)
	}
	return fmt.Sprintf("%d B", n)
}
