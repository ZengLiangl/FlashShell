package app

import (
	"os"
	"testing"
	"time"

	"quick-cmd/data"
	"quick-cmd/define"
)

func TestApp_GetConfig(t *testing.T) {
	// 创建临时配置文件
	tempConfig := "config.yaml"
	defer os.Remove(tempConfig)

	// 创建测试配置
	if err := data.CreateDefaultConfig(tempConfig); err != nil {
		t.Fatalf("创建测试配置失败: %v", err)
	}

	// 创建应用实例
	app := NewApp()
	app.configManager = data.NewConfigManager(tempConfig)

	// 测试获取配置
	config, err := app.GetConfig()
	if err != nil {
		t.Fatalf("获取配置失败: %v", err)
	}

	if config == nil {
		t.Fatal("配置为空")
	}

	if len(config.Projects) == 0 {
		t.Fatal("项目列表为空")
	}

	if len(config.Machines) == 0 {
		t.Fatal("机器列表为空")
	}
}

func TestApp_ExecuteLocalCommand(t *testing.T) {
	// 创建临时配置文件
	tempConfig := "test_config.yaml"
	defer os.Remove(tempConfig)

	// 创建测试配置
	testRoot := &define.Root{
		Projects: []define.Project{
			{
				Name:        "测试项目",
				Description: "测试项目描述",
				WorkDir:     ".",
				SubProjects: []define.SubProject{
					{
						Name:        "测试子项目",
						Description: "测试子项目描述",
						Commands: []define.Command{
							{
								Name:        "测试命令",
								Description: "测试命令描述",
								Type:        "batch",
								Steps:       []string{"echo 'Hello World'"},
							},
						},
					},
				},
			},
		},
		Machines: []define.Machine{},
	}

	// 保存测试配置
	configManager := data.NewConfigManager(tempConfig)
	if err := configManager.SaveConfig(testRoot); err != nil {
		t.Fatalf("保存测试配置失败: %v", err)
	}

	// 创建应用实例
	app := NewApp()
	app.configManager = configManager

	// 加载配置
	if _, err := app.GetConfig(); err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 执行命令
	err := app.ExecuteCommand("测试项目", "测试子项目", "测试命令")
	if err != nil {
		t.Fatalf("执行命令失败: %v", err)
	}

	// 等待命令执行完成
	time.Sleep(2 * time.Second)

	// 检查输出
	output := app.GetOutput()
	if len(output) == 0 {
		t.Fatal("没有输出")
	}

	// 检查状态
	status := app.GetStatus()
	if status == nil {
		t.Fatal("状态为空")
	}
}

func TestApp_MachineManagement(t *testing.T) {
	// 创建临时配置文件
	tempConfig := "test_config.yaml"
	defer os.Remove(tempConfig)

	// 创建测试配置
	if err := data.CreateDefaultConfig(tempConfig); err != nil {
		t.Fatalf("创建测试配置失败: %v", err)
	}

	// 创建应用实例
	app := NewApp()
	app.configManager = data.NewConfigManager(tempConfig)

	// 加载配置
	if _, err := app.GetConfig(); err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 测试添加机器
	newMachine := define.Machine{
		Name:    "测试机器",
		Host:    "test.example.com",
		Port:    22,
		User:    "testuser",
		KeyFile: "~/.ssh/id_rsa",
	}

	if err := app.AddMachine(newMachine); err != nil {
		t.Fatalf("添加机器失败: %v", err)
	}

	// 测试获取机器列表
	machines := app.GetMachines()
	found := false
	for _, machine := range machines {
		if machine.Name == "测试机器" {
			found = true
			break
		}
	}

	if !found {
		t.Fatal("未找到添加的机器")
	}

	// 测试更新机器
	newMachine.Host = "updated.example.com"
	if err := app.UpdateMachine("测试机器", newMachine); err != nil {
		t.Fatalf("更新机器失败: %v", err)
	}

	// 测试删除机器
	if err := app.DeleteMachine("测试机器"); err != nil {
		t.Fatalf("删除机器失败: %v", err)
	}

	// 验证机器已删除
	machines = app.GetMachines()
	for _, machine := range machines {
		if machine.Name == "测试机器" {
			t.Fatal("机器未被删除")
		}
	}
}
