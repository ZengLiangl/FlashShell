package data

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFinalShellFile(t *testing.T) {
	path := filepath.Join("..", "hk-test-118_connect_config.json")
	session, err := ParseFinalShellFile(path)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if session.Name != "hk-test-118" {
		t.Fatalf("name 不匹配: %s", session.Name)
	}
	if session.Host != "172.19.100.118" {
		t.Fatalf("host 不匹配: %s", session.Host)
	}
	if session.Port != 22 {
		t.Fatalf("port 不匹配: %d", session.Port)
	}
	if session.UserName != "root" {
		t.Fatalf("user_name 不匹配: %s", session.UserName)
	}
}

func TestCollectFilesFromPaths(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.xsh"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := CollectFilesFromPaths([]string{dir}, isXshellFile)
	if err != nil {
		t.Fatalf("收集失败: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("期望 1 个 xsh 文件, 实际 %d", len(files))
	}
}
