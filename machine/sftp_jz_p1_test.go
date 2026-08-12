package machine

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// 用 jz 验证 P1：aux 浏览、上传/下载、.part 续传、传输通道池并发。
// 运行：go test ./machine -run TestJZP1SFTPFeatures -count=1 -v
func TestJZP1SFTPFeatures(t *testing.T) {
	if testing.Short() {
		t.Skip("skip remote jz P1 test in -short")
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
	backend := aux.FileBackendName()
	t.Logf("file backend=%s sudo=%v", backend, machine.SftpSudo)
	if backend != fileBackendSFTP {
		t.Fatalf("期望 SFTP 后端，得到 %q", backend)
	}

	entries, err := aux.ListDir("/tmp", false)
	if err != nil {
		t.Fatalf("ListDir /tmp: %v", err)
	}
	t.Logf("ListDir /tmp ok, entries=%d", len(entries))

	const size = 4 << 20 // 4 MiB
	localSrc, err := os.CreateTemp("", "flashdock-jz-p1-src-*.bin")
	if err != nil {
		t.Fatal(err)
	}
	localSrcPath := localSrc.Name()
	defer os.Remove(localSrcPath)

	payload := make([]byte, size)
	if _, err := rand.Read(payload); err != nil {
		t.Fatal(err)
	}
	if _, err := localSrc.Write(payload); err != nil {
		t.Fatal(err)
	}
	_ = localSrc.Close()

	remotePath := path.Join("/tmp", fmt.Sprintf("flashdock-jz-p1-%d.bin", time.Now().UnixNano()))
	defer func() { _ = aux.RemovePath(remotePath) }()

	ctx := context.Background()
	var lastProg int64
	if err := aux.UploadFile(ctx, localSrcPath, remotePath, func(transferred, total int64, _ float64) {
		lastProg = transferred
	}); err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if lastProg < size {
		t.Fatalf("upload progress incomplete: %d/%d", lastProg, size)
	}
	st, err := aux.StatPath(remotePath)
	if err != nil {
		t.Fatalf("Stat uploaded: %v", err)
	}
	if st.Size != size {
		t.Fatalf("remote size %d, want %d", st.Size, size)
	}
	t.Logf("upload ok: %s (%d bytes)", remotePath, st.Size)

	// 下载到临时目录，并验证 .part 续传
	tmpDir := t.TempDir()
	localDst := filepath.Join(tmpDir, "dl.bin")
	partPath := downloadPartPath(localDst)

	// 先制造半成品 .part（前 1MiB）
	partial := payload[:1<<20]
	if err := os.WriteFile(partPath, partial, 0o644); err != nil {
		t.Fatal(err)
	}

	cancelCtx, cancel := context.WithCancel(context.Background())
	var resumedFrom int64
	progressOnce := sync.Once{}
	err = aux.DownloadFile(cancelCtx, remotePath, localDst, func(transferred, total int64, _ float64) {
		progressOnce.Do(func() {
			resumedFrom = transferred
		})
		if transferred > size/2 {
			cancel() // 中途取消，留下 .part
		}
	})
	if err == nil {
		// 可能下载太快没来得及 cancel；仍可接受，继续完整下载验证
		t.Log("download finished before cancel (fast link); continue full verify")
	} else {
		t.Logf("mid-download cancel: %v", err)
	}
	if resumedFrom > 0 && resumedFrom < 1<<20 {
		t.Fatalf("expected resume offset around 1MiB, got first progress=%d", resumedFrom)
	}
	if _, err := os.Stat(partPath); err == nil {
		t.Logf("part file present after cancel: %s", partPath)
	}

	// 完整续传/重下
	if err := aux.DownloadFile(context.Background(), remotePath, localDst, nil); err != nil {
		t.Fatalf("DownloadFile resume/complete: %v", err)
	}
	got, err := os.ReadFile(localDst)
	if err != nil {
		t.Fatalf("read local dst: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("downloaded content mismatch (len=%d want=%d)", len(got), len(payload))
	}
	if _, err := os.Stat(partPath); !os.IsNotExist(err) {
		t.Fatalf(".part should be gone after success, err=%v", err)
	}
	t.Log("download + .part resume ok")

	// 传输池：并发两路下载
	dst1 := filepath.Join(tmpDir, "pool1.bin")
	dst2 := filepath.Join(tmpDir, "pool2.bin")
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		errs <- aux.DownloadFile(context.Background(), remotePath, dst1, nil)
	}()
	go func() {
		defer wg.Done()
		errs <- aux.DownloadFile(context.Background(), remotePath, dst2, nil)
	}()
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatalf("pool concurrent download: %v", e)
		}
	}
	for _, p := range []string{dst1, dst2} {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, payload) {
			t.Fatalf("pool download mismatch: %s", p)
		}
	}
	t.Log("transfer pool concurrent download ok")
}

// 用 jz 验证上传隐藏 .part 断点续传。
// 运行：go test ./machine -run TestJZP1UploadPartResume -count=1 -v
func TestJZP1UploadPartResume(t *testing.T) {
	if testing.Short() {
		t.Skip("skip remote jz upload .part test in -short")
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

	const size = 4 << 20
	const partialSize = 1 << 20

	localSrc, err := os.CreateTemp("", "flashdock-jz-up-part-*.bin")
	if err != nil {
		t.Fatal(err)
	}
	localSrcPath := localSrc.Name()
	defer os.Remove(localSrcPath)

	payload := make([]byte, size)
	if _, err := rand.Read(payload); err != nil {
		t.Fatal(err)
	}
	if _, err := localSrc.Write(payload); err != nil {
		t.Fatal(err)
	}
	_ = localSrc.Close()

	remotePath := path.Join("/tmp", fmt.Sprintf("flashdock-jz-up-part-%d.bin", time.Now().UnixNano()))
	partRemote := uploadPartPath(remotePath)
	defer func() {
		_ = aux.RemovePath(remotePath)
		_ = aux.RemovePath(partRemote)
	}()

	sftpClient, err := aux.sftpClient()
	if err != nil {
		t.Fatalf("sftpClient: %v", err)
	}
	partFile, err := sftpClient.Create(partRemote)
	if err != nil {
		t.Fatalf("create remote part: %v", err)
	}
	if _, err := partFile.Write(payload[:partialSize]); err != nil {
		_ = partFile.Close()
		t.Fatalf("seed remote part: %v", err)
	}
	_ = partFile.Close()
	t.Logf("seeded remote part %s (%d bytes)", partRemote, partialSize)

	var firstProg int64
	var sawFinal bool
	progressOnce := sync.Once{}
	if err := aux.UploadFile(context.Background(), localSrcPath, remotePath, func(transferred, total int64, _ float64) {
		progressOnce.Do(func() { firstProg = transferred })
		if transferred >= total && total > 0 {
			sawFinal = true
		}
	}); err != nil {
		t.Fatalf("UploadFile resume: %v", err)
	}
	if firstProg < partialSize {
		t.Fatalf("expected resume from >= %d, first progress=%d", partialSize, firstProg)
	}
	t.Logf("upload resumed from first progress=%d", firstProg)

	st, err := aux.StatPath(remotePath)
	if err != nil {
		t.Fatalf("Stat final remote: %v", err)
	}
	if st.Size != size {
		t.Fatalf("remote size %d, want %d", st.Size, size)
	}
	if _, err := aux.StatPath(partRemote); err == nil {
		t.Fatalf("remote .part should be gone after success: %s", partRemote)
	}
	if !sawFinal {
		t.Fatal("did not observe final progress callback")
	}

	// 拉回校验内容一致
	localDst := filepath.Join(t.TempDir(), "verify.bin")
	if err := aux.DownloadFile(context.Background(), remotePath, localDst, nil); err != nil {
		t.Fatalf("DownloadFile verify: %v", err)
	}
	got, err := os.ReadFile(localDst)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("content mismatch after upload resume (len=%d want=%d)", len(got), len(payload))
	}
	t.Log("upload .part resume ok")
}

