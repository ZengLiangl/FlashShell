package machine

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func remoteStatSize(t *testing.T, aux *ShellAuxManager, remotePath string) (int64, bool) {
	t.Helper()
	out, err := aux.Exec("stat -c%s -- " + shellQuotePath(remotePath) + " 2>/dev/null")
	if err != nil {
		return 0, false
	}
	n, err := strconv.ParseInt(strings.TrimSpace(out), 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// SFTP UploadFileOverwrite：覆盖过程中目标不得被截断为 0，且应出现 .part。
// 运行：go test ./machine -run TestJZSFTPUploadNoTruncateDuringOverwrite -count=1 -v
func TestJZSFTPUploadNoTruncateDuringOverwrite(t *testing.T) {
	if testing.Short() {
		t.Skip("skip remote jz sftp atomic overwrite test in -short")
	}

	machine := loadJZMachine(t)
	aux := NewShellAuxManager()
	if err := aux.Connect(machine, nil); err != nil {
		t.Fatalf("Connect jz aux: %v", err)
	}
	defer aux.Close()
	if err := aux.EnsureFileBackend(); err != nil {
		t.Fatalf("EnsureFileBackend: %v", err)
	}
	if aux.FileBackendName() != fileBackendSFTP {
		t.Fatalf("期望 SFTP 后端，得到 %q", aux.FileBackendName())
	}

	remotePath := path.Join("/tmp", fmt.Sprintf("flashdock-jz-sftp-atom-%d.bin", time.Now().UnixNano()))
	partRemote := uploadPartPath(remotePath)
	defer func() {
		_ = aux.RemovePath(remotePath)
		_ = aux.RemovePath(partRemote)
	}()

	const seedSize = 256 << 10
	seedLocal := path.Join(t.TempDir(), "seed.bin")
	if err := os.WriteFile(seedLocal, bytes.Repeat([]byte{0x11}, seedSize), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := aux.UploadFile(ctx, seedLocal, remotePath, nil); err != nil {
		t.Fatalf("seed upload: %v", err)
	}

	const nextSize = 4 << 20
	nextLocal := path.Join(t.TempDir(), "next.bin")
	nextPayload := make([]byte, nextSize)
	if _, err := rand.Read(nextPayload); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nextLocal, nextPayload, 0o644); err != nil {
		t.Fatal(err)
	}

	var (
		mu          sync.Mutex
		sawPart     bool
		minTargetSz int64 = seedSize
		sawBadTrunc bool
		uploadDone  = make(chan error, 1)
	)
	go func() {
		uploadDone <- aux.UploadFileOverwrite(ctx, nextLocal, remotePath, nil)
	}()

	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-uploadDone:
			if err != nil {
				t.Fatalf("overwrite upload: %v", err)
			}
			goto after
		default:
		}
		if sz, ok := remoteStatSize(t, aux, partRemote); ok && sz > 0 {
			mu.Lock()
			sawPart = true
			mu.Unlock()
		}
		if sz, ok := remoteStatSize(t, aux, remotePath); ok {
			mu.Lock()
			if sz < minTargetSz {
				minTargetSz = sz
			}
			if sz == 0 {
				sawBadTrunc = true
			}
			mu.Unlock()
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("upload timed out")

after:
	mu.Lock()
	defer mu.Unlock()
	if !sawPart {
		t.Fatal("overwrite 过程中未观察到 .part，原子上传可能未生效")
	}
	if sawBadTrunc || minTargetSz == 0 {
		t.Fatalf("覆盖上传过程中目标被截断（minSize=%d）", minTargetSz)
	}
	if minTargetSz < seedSize {
		t.Fatalf("覆盖上传过程中目标变小: min=%d seed=%d", minTargetSz, seedSize)
	}
	st, err := aux.StatPath(remotePath)
	if err != nil {
		t.Fatalf("stat final: %v", err)
	}
	if st.Size != nextSize {
		t.Fatalf("final size %d want %d", st.Size, nextSize)
	}
	if _, err := aux.StatPath(partRemote); err == nil {
		t.Fatalf(".part should be gone: %s", partRemote)
	}
	t.Log("sftp atomic overwrite (no truncate) ok")
}
