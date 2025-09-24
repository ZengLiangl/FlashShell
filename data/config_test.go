package data

import (
	"os"
	"path/filepath"
	"testing"

	"quick-cmd/define"
)

func TestConfigManager_LoadConfig(t *testing.T) {
	// 创建临时配置文件
	tempFile := "test_config.yaml"
	defer os.Remove(tempFile)

	// 创建默认配置
	if err := CreateDefaultConfig(tempFile); err != nil {
		t.Fatalf("创建默认配置失败: %v", err)
	}

	// 创建配置管理器
	cm := NewConfigManager(tempFile)

	// 加载配置
	root, err := cm.LoadConfig()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证配置
	if root == nil {
		t.Fatal("配置为空")
	}

	if len(root.Projects) == 0 {
		t.Fatal("项目列表为空")
	}

	if len(root.Machines) == 0 {
		t.Fatal("机器列表为空")
	}

	// 验证项目结构
	project := root.Projects[0]
	if project.Name == "" {
		t.Fatal("项目名称为空")
	}

	if len(project.SubProjects) == 0 {
		t.Fatal("子项目列表为空")
	}

	subProject := project.SubProjects[0]
	if len(subProject.Commands) == 0 {
		t.Fatal("命令列表为空")
	}
}

func TestConfigManager_SaveConfig(t *testing.T) {
	// 创建临时配置文件
	tempFile := "test_save_config.yaml"
	defer os.Remove(tempFile)

	// 创建配置管理器
	cm := NewConfigManager(tempFile)

	// 创建测试配置
	testRoot := &define.Root{
		Projects: []define.Project{
			{
				Name:        "测试项目",
				Description: "测试项目描述",
				WorkDir:     "~/test",
				SubProjects: []define.SubProject{
					{
						Name:        "测试子项目",
						Description: "测试子项目描述",
						Commands: []define.Command{
							{
								Name:        "测试命令",
								Description: "测试命令描述",
								Type:        "batch",
								Steps:       []string{"echo 'test'"},
							},
						},
					},
				},
			},
		},
		Machines: []define.Machine{
			{
				Name:    "测试机器",
				KeyFile: "~/.ssh/id_rsa",
			},
		},
	}

	// 保存配置
	if err := cm.SaveConfig(testRoot); err != nil {
		t.Fatalf("保存配置失败: %v", err)
	}

	// 重新加载配置验证
	loadedRoot, err := cm.LoadConfig()
	if err != nil {
		t.Fatalf("重新加载配置失败: %v", err)
	}

	// 验证保存的配置
	if loadedRoot.Projects[0].Name != "测试项目" {
		t.Fatal("项目名称不匹配")
	}

	if loadedRoot.Machines[0].Name != "测试机器" {
		t.Fatal("机器名称不匹配")
	}
}

func TestConfigManager_GetMachine(t *testing.T) {
	// 创建临时配置文件
	tempFile := "test_get_machine.yaml"
	defer os.Remove(tempFile)

	// 创建默认配置
	if err := CreateDefaultConfig(tempFile); err != nil {
		t.Fatalf("创建默认配置失败: %v", err)
	}

	// 创建配置管理器并加载配置
	cm := NewConfigManager(tempFile)
	if _, err := cm.LoadConfig(); err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 测试获取存在的机器
	machine := cm.GetMachine("示例服务器")
	if machine == nil {
		t.Fatal("未找到示例服务器")
	}

	if machine.Name != "示例服务器" {
		t.Fatal("机器名称不匹配")
	}

	// 测试获取不存在的机器
	nonExistentMachine := cm.GetMachine("不存在的机器")
	if nonExistentMachine != nil {
		t.Fatal("不应该找到不存在的机器")
	}
}

func TestExpandPath(t *testing.T) {
	// 测试 ~ 符号展开
	homeDir, _ := os.UserHomeDir()

	testCases := []struct {
		input    string
		expected string
	}{
		{"~/test", filepath.Join(homeDir, "test")},
		{"./test", "./test"},
		{"/absolute/path", "/absolute/path"},
		{"", ""},
	}

	for _, tc := range testCases {
		result := expandPath(tc.input)
		if result != tc.expected {
			t.Errorf("expandPath(%s) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

func TestCreateDefaultConfig(t *testing.T) {
	// 创建临时配置文件
	tempFile := "test_default_config.yaml"
	defer os.Remove(tempFile)

	// 创建默认配置
	if err := CreateDefaultConfig(tempFile); err != nil {
		t.Fatalf("创建默认配置失败: %v", err)
	}

	// 验证文件是否存在
	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Fatal("默认配置文件未创建")
	}

	// 验证文件内容
	cm := NewConfigManager(tempFile)
	root, err := cm.LoadConfig()
	if err != nil {
		t.Fatalf("加载默认配置失败: %v", err)
	}

	if len(root.Projects) == 0 {
		t.Fatal("默认配置项目列表为空")
	}

	if len(root.Machines) == 0 {
		t.Fatal("默认配置机器列表为空")
	}
}
