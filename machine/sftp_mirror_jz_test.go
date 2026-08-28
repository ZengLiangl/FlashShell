package machine

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"
)

// 验证目录覆盖上传会清理旧资产并做完整性校验。
// 运行：go test ./machine -run TestJZDirectoryUploadReplaceMirror -count=1 -v
func TestJZDirectoryUploadReplaceMirror(t *testing.T) {
	if testing.Short() {
		t.Skip("skip remote jz mirror test in -short")
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

	remoteDir := path.Join("/tmp", fmt.Sprintf("flashdock-jz-mirror-%d", time.Now().UnixNano()))
	defer func() { _ = aux.RemovePathReliable(remoteDir) }()

	buildV1 := func(root string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(root, "assets"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "index.html"), []byte(`<script src="/assets/index-old.js"></script>`), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "assets", "index-old.js"), []byte("console.log('old')"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	buildV2 := func(root string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(root, "assets"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "index.html"), []byte(`<script src="/assets/index-new.js"></script>`), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "assets", "index-new.js"), []byte("console.log('new')"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	localV1 := t.TempDir()
	buildV1(localV1)
	ctx := context.Background()

	if err := aux.RemovePathReliable(remoteDir); err != nil {
		t.Fatalf("prepare remote dir: %v", err)
	}
	if err := aux.UploadDirectoryRecursive(ctx, localV1, remoteDir, nil); err != nil {
		t.Fatalf("upload v1: %v", err)
	}
	if err := aux.VerifyRemoteDirMirror(localV1, remoteDir); err != nil {
		t.Fatalf("verify v1: %v", err)
	}

	localV2 := t.TempDir()
	buildV2(localV2)
	if err := aux.RemovePathReliable(remoteDir); err != nil {
		t.Fatalf("replace clear remote dir: %v", err)
	}
	if err := aux.UploadDirectoryRecursive(ctx, localV2, remoteDir, nil); err != nil {
		t.Fatalf("upload v2: %v", err)
	}
	if err := aux.PruneRemoteDirToMirror(localV2, remoteDir); err != nil {
		t.Fatalf("prune after v2: %v", err)
	}
	if err := aux.VerifyRemoteDirMirror(localV2, remoteDir); err != nil {
		t.Fatalf("verify v2: %v", err)
	}

	remoteFiles, err := aux.collectRemoteDirFiles(remoteDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := remoteFiles["assets/index-old.js"]; ok {
		t.Fatal("旧版 assets/index-old.js 未被清理，会导致前端白屏")
	}
	if _, ok := remoteFiles["assets/index-new.js"]; !ok {
		t.Fatal("新版 assets/index-new.js 未上传")
	}
	t.Log("directory replace mirror verify ok")
}

// 验证强制覆盖上传会重写同尺寸但内容不同的文件。
// 运行：go test ./machine -run TestJZUploadOverwriteSameSize -count=1 -v
func TestJZUploadOverwriteSameSize(t *testing.T) {
	if testing.Short() {
		t.Skip("skip remote jz overwrite test in -short")
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

	remotePath := path.Join("/tmp", fmt.Sprintf("flashdock-jz-overwrite-%d.txt", time.Now().UnixNano()))
	defer func() { _ = aux.RemovePathReliable(remotePath) }()

	localA, err := os.CreateTemp("", "flashdock-jz-overwrite-a-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	localAPath := localA.Name()
	defer os.Remove(localAPath)
	payloadA := []byte("version-a-same-len!!")
	if _, err := localA.Write(payloadA); err != nil {
		t.Fatal(err)
	}
	_ = localA.Close()

	ctx := context.Background()
	if err := aux.UploadFile(ctx, localAPath, remotePath, nil); err != nil {
		t.Fatalf("upload a: %v", err)
	}

	localB, err := os.CreateTemp("", "flashdock-jz-overwrite-b-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	localBPath := localB.Name()
	defer os.Remove(localBPath)
	payloadB := []byte("version-b-same-len!!")
	if _, err := localB.Write(payloadB); err != nil {
		t.Fatal(err)
	}
	_ = localB.Close()

	if err := aux.UploadFileOverwrite(ctx, localBPath, remotePath, nil); err != nil {
		t.Fatalf("overwrite upload b: %v", err)
	}

	localDst := filepath.Join(t.TempDir(), "verify.txt")
	if err := aux.DownloadFile(ctx, remotePath, localDst, nil); err != nil {
		t.Fatalf("download verify: %v", err)
	}
	got, err := os.ReadFile(localDst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payloadB) {
		t.Fatalf("overwrite failed: got %q want %q", got, payloadB)
	}
	t.Log("same-size overwrite ok")
}
