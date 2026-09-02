package data

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGlobalConfig_DoesNotOverwriteExistingContent(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "global_config.yaml")

	original := `appId: com.custom
windowsName: 我的运行器
configFile:
  - D:\projects\config.yaml
lastOpenedFile: D:\projects\config.yaml
workPaths:
  ACC-CLOUD: D:\IdeaProjects\acc-cloud
  HOME: ~
machines:
  - id: machine-jz
    name: jz
    key_file: ~/.ssh/id_rsa
    encrypted_data: "test-encrypted"
`
	if err := os.WriteFile(configPath, []byte(original), 0644); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}

	gcm := NewGlobalConfigManager(configPath)
	config, err := gcm.LoadGlobalConfig()
	if err != nil {
		t.Fatalf("加载全局配置失败: %v", err)
	}

	if config.WindowsName != "我的运行器" {
		t.Fatalf("windowsName 期望 '我的运行器'，实际 %q", config.WindowsName)
	}
	if len(config.Machines) != 1 || config.Machines[0].Name != "jz" {
		t.Fatalf("machines 未正确加载: %+v", config.Machines)
	}

	afterLoad, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("读取配置文件失败: %v", err)
	}
	if string(afterLoad) != original {
		t.Fatalf("LoadGlobalConfig 不应修改已有配置文件内容")
	}
}

func TestUpdateLastTaskProject_SkipsSaveWhenUnchanged(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "global_config.yaml")

	original := `appId: com.runner
windowsName: FlashDock
configFile:
  - config.yaml
lastOpenedFile: config.yaml
lastTaskProject: XYJ
workPaths:
  HOME: ~
`
	if err := os.WriteFile(configPath, []byte(original), 0644); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}

	gcm := NewGlobalConfigManager(configPath)
	if _, err := gcm.LoadGlobalConfig(); err != nil {
		t.Fatalf("加载全局配置失败: %v", err)
	}

	if err := gcm.UpdateLastTaskProject("XYJ"); err != nil {
		t.Fatalf("UpdateLastTaskProject 失败: %v", err)
	}

	afterUpdate, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("读取配置文件失败: %v", err)
	}
	if string(afterUpdate) != original {
		t.Fatalf("lastTaskProject 未变化时不应写回配置文件")
	}

	if err := gcm.UpdateLastTaskProject("Looda"); err != nil {
		t.Fatalf("UpdateLastTaskProject 写入失败: %v", err)
	}
	if got := gcm.GetLastTaskProject(); got != "Looda" {
		t.Fatalf("GetLastTaskProject = %q", got)
	}
}

func TestUpdateLastOpenedFile_SkipsSaveWhenUnchanged(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "global_config.yaml")

	original := `appId: com.runner
windowsName: FlashDock
configFile:
  - config.yaml
lastOpenedFile: config.yaml
workPaths:
  HOME: ~
`
	if err := os.WriteFile(configPath, []byte(original), 0644); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}

	gcm := NewGlobalConfigManager(configPath)
	if _, err := gcm.LoadGlobalConfig(); err != nil {
		t.Fatalf("加载全局配置失败: %v", err)
	}

	if err := gcm.UpdateLastOpenedFile("config.yaml"); err != nil {
		t.Fatalf("UpdateLastOpenedFile 失败: %v", err)
	}

	afterUpdate, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("读取配置文件失败: %v", err)
	}
	if string(afterUpdate) != original {
		t.Fatalf("lastOpenedFile 未变化时不应写回配置文件")
	}
}

func TestLoadConfig_DoesNotRewriteGlobalConfigOnRepeatLoad(t *testing.T) {
	tempDir := t.TempDir()

	globalConfigPath := filepath.Join(tempDir, "global_config.yaml")
	businessConfigPath := filepath.Join(tempDir, "config.yaml")

	globalYAML := `appId: com.custom
windowsName: 自定义
configFile:
  - ` + businessConfigPath + `
lastOpenedFile: ` + businessConfigPath + `
workPaths:
  HOME: ~
machines:
  - id: machine-test-server
    name: test-server
    key_file: ~/.ssh/id_rsa
`
	if err := os.WriteFile(globalConfigPath, []byte(globalYAML), 0644); err != nil {
		t.Fatalf("写入全局配置失败: %v", err)
	}

	if err := CreateDefaultConfig(businessConfigPath); err != nil {
		t.Fatalf("创建业务配置失败: %v", err)
	}

	gcm := NewGlobalConfigManager(globalConfigPath)
	cm := &ConfigManager{
		configPath:          businessConfigPath,
		globalConfigManager: gcm,
	}

	if _, err := cm.LoadConfig(); err != nil {
		t.Fatalf("第一次 LoadConfig 失败: %v", err)
	}
	afterFirst, err := os.ReadFile(globalConfigPath)
	if err != nil {
		t.Fatalf("读取全局配置失败: %v", err)
	}

	if _, err := cm.LoadConfig(); err != nil {
		t.Fatalf("第二次 LoadConfig 失败: %v", err)
	}
	afterSecond, err := os.ReadFile(globalConfigPath)
	if err != nil {
		t.Fatalf("读取全局配置失败: %v", err)
	}

	if string(afterFirst) != globalYAML {
		t.Fatalf("第一次 LoadConfig 后全局配置内容被改写")
	}
	if string(afterSecond) != globalYAML {
		t.Fatalf("重复 LoadConfig 不应改写全局配置")
	}

	machine := cm.GetMachine("test-server")
	if machine == nil {
		t.Fatalf("期望从全局配置读取机器 test-server")
	}
	if machine.Name != "test-server" {
		t.Fatalf("机器名称不匹配: %s", machine.Name)
	}
}

func TestNormalizeWindowsName(t *testing.T) {
	if got := NormalizeWindowsName(""); got != "FlashShell" {
		t.Fatalf("empty = %q", got)
	}
	if got := NormalizeWindowsName("FlashDock"); got != "FlashShell" {
		t.Fatalf("legacy default = %q", got)
	}
	if got := NormalizeWindowsName("我的运行器"); got != "我的运行器" {
		t.Fatalf("custom = %q", got)
	}
}
