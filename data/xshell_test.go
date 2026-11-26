package data

import (
	"os"
	"path/filepath"
	"testing"

	"FlashDock/define"
)

// 虚拟的 .xsh 样例（RFC 5737 文档专用地址，不对应任何真实主机）
const sampleXshellContent = `[CONNECTION]
Protocol=SSH
Host=203.0.113.10
Port=22

[CONNECTION:AUTHENTICATION]
UserName=demo

[CONNECTION:AUTHENTICATION:PASSWORD]
Password=example-only
`

func TestParseXshellContent(t *testing.T) {
	session, err := ParseXshellContent([]byte(sampleXshellContent))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if session.Host != "203.0.113.10" {
		t.Fatalf("Host 不匹配: %s", session.Host)
	}
	if session.Port != 22 {
		t.Fatalf("Port 不匹配: %d", session.Port)
	}
	if session.User != "demo" {
		t.Fatalf("User 不匹配: %s", session.User)
	}
}

func TestParseXshellFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample-host.xsh")
	if err := os.WriteFile(path, []byte(sampleXshellContent), 0o644); err != nil {
		t.Fatalf("写入样例失败: %v", err)
	}

	session, err := ParseXshellFile(path)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if session.Name != "sample-host" {
		t.Fatalf("名称应为文件名: %s", session.Name)
	}
}

func TestGlobalConfigManager_MachineIDMigration(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "global_config.yaml")
	gcm := NewGlobalConfigManager(configPath)

	config := &GlobalConfig{
		AppId: "test",
		Machines: []define.Machine{
			{Name: "legacy-server"},
		},
	}
	if err := gcm.SaveGlobalConfig(config); err != nil {
		t.Fatalf("保存配置失败: %v", err)
	}

	loaded, err := gcm.LoadGlobalConfig()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if len(loaded.Machines) != 1 || loaded.Machines[0].ID == "" {
		t.Fatalf("期望自动补全机器 ID")
	}

	byName := gcm.GetMachineByName("legacy-server")
	if byName == nil || byName.ID != loaded.Machines[0].ID {
		t.Fatalf("按名称查找失败")
	}
}
