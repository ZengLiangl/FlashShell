package data

import (
	"os"
	"path/filepath"
	"testing"

	"quick-cmd/define"
)

func TestParseXshellContent(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "172.19.100.18.xsh"))
	if err != nil {
		t.Fatalf("读取示例文件失败: %v", err)
	}

	session, err := ParseXshellContent(data)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if session.Host != "172.19.100.18" {
		t.Fatalf("Host 不匹配: %s", session.Host)
	}
	if session.Port != 22 {
		t.Fatalf("Port 不匹配: %d", session.Port)
	}
	if session.User != "app" {
		t.Fatalf("User 不匹配: %s", session.User)
	}
}

func TestParseXshellFile(t *testing.T) {
	path := filepath.Join("..", "172.19.100.18.xsh")
	session, err := ParseXshellFile(path)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if session.Name != "172.19.100.18" {
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
