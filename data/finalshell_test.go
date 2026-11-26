package data

import (
	"os"
	"path/filepath"
	"testing"
)

// 虚拟的 FinalShell 配置样例（RFC 5737 文档专用地址，不对应任何真实主机）
const sampleFinalShellConfig = `{
  "name": "demo-host",
  "host": "203.0.113.20",
  "port": 22,
  "user_name": "demo"
}`

func TestParseFinalShellFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo-host_connect_config.json")
	if err := os.WriteFile(path, []byte(sampleFinalShellConfig), 0o644); err != nil {
		t.Fatalf("写入样例失败: %v", err)
	}

	session, err := ParseFinalShellFile(path)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if session.Name != "demo-host" {
		t.Fatalf("name 不匹配: %s", session.Name)
	}
	if session.Host != "203.0.113.20" {
		t.Fatalf("host 不匹配: %s", session.Host)
	}
	if session.Port != 22 {
		t.Fatalf("port 不匹配: %d", session.Port)
	}
	if session.UserName != "demo" {
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
