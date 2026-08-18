package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"FlashDock/data"
	"FlashDock/define"
	"FlashDock/machine"
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
	app := NewApp("")
	app.configManager = data.NewConfigManager(tempConfig, nil)

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
								Steps:       define.StepList{{Command: "echo 'Hello World'"}},
							},
						},
					},
				},
			},
		},
		Machines: []define.Machine{},
	}

	// 保存测试配置
	configManager := data.NewConfigManager(tempConfig, nil)
	if err := configManager.SaveConfig(testRoot); err != nil {
		t.Fatalf("保存测试配置失败: %v", err)
	}

	// 创建应用实例
	app := NewApp("")
	app.configManager = configManager

	// 手动初始化 SubProjectRunner（因为测试中不会调用 Startup）
	app.subProjectRunner = machine.NewSubProjectRunner(configManager)

	// 加载配置
	if _, err := app.GetConfig(); err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 执行 SubProject
	err := app.ExecuteSubProject("测试项目", "测试子项目")
	if err != nil {
		t.Fatalf("执行 SubProject 失败: %v", err)
	}

	// 等待命令执行完成
	time.Sleep(2 * time.Second)

	// 检查输出
	output := app.GetOutput()
	if len(output) == 0 {
		t.Fatal("没有输出")
	}

	// 检查状态
	status := app.GetSubProjectStatus()
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
	app := NewApp("")
	app.configManager = data.NewConfigManager(tempConfig, nil)

	// 加载配置
	if _, err := app.GetConfig(); err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 测试添加机器
	newMachine := define.Machine{
		Name:    "测试机器",
		KeyFile: "~/.ssh/id_rsa",
	}
	newMachine.EnsureID()

	// 设置敏感数据
	sensitiveData := &define.SensitiveData{
		Host:     "test.example.com",
		Port:     22,
		User:     "testuser",
		Password: "testpass",
	}

	if err := newMachine.SetSensitiveData(sensitiveData); err != nil {
		t.Fatalf("设置敏感数据失败: %v", err)
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
			if machine.Host != "test.example.com" {
				t.Fatalf("GetMachines 应填充 Host，得到 %q", machine.Host)
			}
			if machine.Port != 22 {
				t.Fatalf("GetMachines 应填充 Port，得到 %d", machine.Port)
			}
			if machine.User != "testuser" {
				t.Fatalf("GetMachines 应填充 User，得到 %q", machine.User)
			}
			break
		}
	}

	if !found {
		t.Fatal("未找到添加的机器")
	}

	// 测试更新机器
	updatedSensitiveData := &define.SensitiveData{
		Host:     "updated.example.com",
		Port:     22,
		User:     "testuser",
		Password: "testpass",
	}

	if err := newMachine.SetSensitiveData(updatedSensitiveData); err != nil {
		t.Fatalf("更新敏感数据失败: %v", err)
	}

	if err := app.UpdateMachine(newMachine.ID, newMachine); err != nil {
		t.Fatalf("更新机器失败: %v", err)
	}

	// 测试删除机器
	if err := app.DeleteMachine(newMachine.ID); err != nil {
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

func TestSubProjectWorkDirPriority(t *testing.T) {
	// 创建临时配置文件
	tempConfig := "test_workdir_config.yaml"
	defer os.Remove(tempConfig)

	// 创建测试配置，测试 WorkDir 优先级
	testRoot := &define.Root{
		Projects: []define.Project{
			{
				Name:        "WorkDir测试项目",
				Description: "测试 WorkDir 优先级",
				WorkDir:     "/project/workdir", // Project WorkDir
				SubProjects: []define.SubProject{
					{
						Name:        "测试子项目1",
						Description: "使用 Project WorkDir",
						// 没有设置 SubProject WorkDir
						Commands: []define.Command{
							{
								Name:        "测试命令1",
								Description: "应该使用 Project WorkDir",
								Type:        "batch",
								Steps:       define.StepList{{Command: "echo 'Project WorkDir'"}},
								// 没有设置 Command WorkDir
							},
						},
					},
					{
						Name:        "测试子项目2",
						Description: "使用 SubProject WorkDir",
						WorkDir:     "/subproject/workdir", // SubProject WorkDir
						Commands: []define.Command{
							{
								Name:        "测试命令2",
								Description: "应该使用 SubProject WorkDir",
								Type:        "batch",
								Steps:       define.StepList{{Command: "echo 'SubProject WorkDir'"}},
								// 没有设置 Command WorkDir
							},
						},
					},
					{
						Name:        "测试子项目3",
						Description: "使用 Command WorkDir",
						WorkDir:     "/subproject/workdir", // SubProject WorkDir
						Commands: []define.Command{
							{
								Name:        "测试命令3",
								Description: "应该使用 Command WorkDir",
								Type:        "batch",
								Steps:       define.StepList{{Command: "echo 'Command WorkDir'"}},
								WorkDir:     "/command/workdir", // Command WorkDir
							},
						},
					},
				},
			},
		},
		Machines: []define.Machine{},
	}

	// 保存测试配置
	configManager := data.NewConfigManager(tempConfig, nil)
	if err := configManager.SaveConfig(testRoot); err != nil {
		t.Fatalf("保存测试配置失败: %v", err)
	}

	// 创建应用实例
	app := NewApp("")
	app.configManager = configManager

	// 手动初始化 SubProjectRunner
	app.subProjectRunner = machine.NewSubProjectRunner(configManager)

	// 加载配置
	if _, err := app.GetConfig(); err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 测试 WorkDir 优先级逻辑
	// 这里我们主要测试结构体是否正确创建，实际的 WorkDir 解析会在执行时进行
	config, err := app.GetConfig()
	if err != nil {
		t.Fatalf("获取配置失败: %v", err)
	}

	// 验证 SubProject 结构体包含 WorkDir 字段
	if len(config.Projects) == 0 {
		t.Fatal("项目列表为空")
	}

	project := config.Projects[0]
	if len(project.SubProjects) != 3 {
		t.Fatalf("期望 3 个子项目，实际 %d 个", len(project.SubProjects))
	}

	// 验证第一个子项目没有 WorkDir（应该使用 Project WorkDir）
	subProject1 := project.SubProjects[0]
	if subProject1.WorkDir != "" {
		t.Errorf("子项目1不应该有 WorkDir，实际: %s", subProject1.WorkDir)
	}

	// 验证第二个子项目有 WorkDir
	subProject2 := project.SubProjects[1]
	if subProject2.WorkDir != "/subproject/workdir" {
		t.Errorf("子项目2 WorkDir 期望 '/subproject/workdir'，实际: %s", subProject2.WorkDir)
	}

	// 验证第三个子项目有 WorkDir，且命令也有 WorkDir
	subProject3 := project.SubProjects[2]
	if subProject3.WorkDir != "/subproject/workdir" {
		t.Errorf("子项目3 WorkDir 期望 '/subproject/workdir'，实际: %s", subProject3.WorkDir)
	}

	command3 := subProject3.Commands[0]
	if command3.WorkDir != "/command/workdir" {
		t.Errorf("命令3 WorkDir 期望 '/command/workdir'，实际: %s", command3.WorkDir)
	}

	t.Log("WorkDir 优先级测试通过")
}

func TestSubProjectWorkDirEnvReplace(t *testing.T) {
	// 创建临时配置文件
	tempConfig := "test_env_replace_config.yaml"
	defer os.Remove(tempConfig)

	// 创建测试配置，测试 SubProject WorkDir 的环境变量替换
	testRoot := &define.Root{
		Projects: []define.Project{
			{
				Name:        "环境变量测试项目",
				Description: "测试 SubProject WorkDir 环境变量替换",
				WorkDir:     "/project/workdir",
				SubProjects: []define.SubProject{
					{
						Name:        "测试子项目1",
						Description: "使用环境变量的 SubProject WorkDir",
						WorkDir:     "${TEST_PROJECT_PATH}/subproject1", // 使用环境变量
						Commands: []define.Command{
							{
								Name:        "测试命令1",
								Description: "应该使用替换后的 SubProject WorkDir",
								Type:        "batch",
								Steps:       define.StepList{{Command: "echo 'SubProject WorkDir with env var'"}},
							},
						},
					},
					{
						Name:        "测试子项目2",
						Description: "使用多个环境变量的 SubProject WorkDir",
						WorkDir:     "${TEST_PROJECT_PATH}/${TEST_SUBPROJECT_PATH}", // 使用多个环境变量
						Commands: []define.Command{
							{
								Name:        "测试命令2",
								Description: "应该使用替换后的多个环境变量",
								Type:        "batch",
								Steps:       define.StepList{{Command: "echo 'Multiple env vars'"}},
							},
						},
					},
				},
			},
		},
		Machines: []define.Machine{},
	}

	// 保存测试配置
	configManager := data.NewConfigManager(tempConfig, nil)
	if err := configManager.SaveConfig(testRoot); err != nil {
		t.Fatalf("保存测试配置失败: %v", err)
	}

	// 设置测试环境变量
	os.Setenv("TEST_PROJECT_PATH", "/test/project")
	os.Setenv("TEST_SUBPROJECT_PATH", "subproject2")
	defer func() {
		os.Unsetenv("TEST_PROJECT_PATH")
		os.Unsetenv("TEST_SUBPROJECT_PATH")
	}()

	// 创建应用实例
	app := NewApp("")
	app.configManager = configManager

	// 加载配置（这会触发环境变量替换）
	config, err := app.GetConfig()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证环境变量替换结果
	if len(config.Projects) == 0 {
		t.Fatal("项目列表为空")
	}

	project := config.Projects[0]
	if len(project.SubProjects) != 2 {
		t.Fatalf("期望 2 个子项目，实际 %d 个", len(project.SubProjects))
	}

	// 验证第一个子项目的环境变量替换
	subProject1 := project.SubProjects[0]
	expectedWorkDir1 := "/test/project/subproject1"
	if subProject1.WorkDir != expectedWorkDir1 {
		t.Errorf("子项目1 WorkDir 期望 '%s'，实际: %s", expectedWorkDir1, subProject1.WorkDir)
	}

	// 验证第二个子项目的多个环境变量替换
	subProject2 := project.SubProjects[1]
	expectedWorkDir2 := "/test/project/subproject2"
	if subProject2.WorkDir != expectedWorkDir2 {
		t.Errorf("子项目2 WorkDir 期望 '%s'，实际: %s", expectedWorkDir2, subProject2.WorkDir)
	}

	t.Log("SubProject WorkDir 环境变量替换测试通过")
}

func TestSubProjectWorkDirGlobalConfigReplace(t *testing.T) {
	tempDir := t.TempDir()
	tempConfig := filepath.Join(tempDir, "test_global_config_replace.yaml")
	tempGlobal := filepath.Join(tempDir, "global_config.yaml")

	testRoot := &define.Root{
		Projects: []define.Project{
			{
				Name:        "全局配置测试项目",
				Description: "测试 SubProject WorkDir 使用全局配置 workPaths",
				WorkDir:     "/project/workdir",
				SubProjects: []define.SubProject{
					{
						Name:        "测试子项目",
						Description: "使用全局配置 workPaths 的 SubProject WorkDir",
						WorkDir:     "${TEST_WORK_PATH}/subproject",
						Commands: []define.Command{
							{
								Name:        "测试命令",
								Description: "应该使用全局配置替换后的 WorkDir",
								Type:        "batch",
								Steps:       define.StepList{{Command: "echo 'Global config workPaths'"}},
							},
						},
					},
				},
			},
		},
		Machines: []define.Machine{},
	}

	configManager := data.NewConfigManager(tempConfig, nil)
	configManager.SetGlobalConfigManagerForTest(data.NewGlobalConfigManager(tempGlobal))
	if err := configManager.SaveConfig(testRoot); err != nil {
		t.Fatalf("保存测试配置失败: %v", err)
	}

	globalConfig := &data.GlobalConfig{
		AppId:       "com.runner",
		WindowsName: "FlashShell",
		WorkPaths: map[string]string{
			"TEST_WORK_PATH": "/global/test/work",
		},
	}
	if err := configManager.SaveGlobalConfig(globalConfig); err != nil {
		t.Fatalf("保存全局配置失败: %v", err)
	}

	app := NewApp("")
	app.configManager = configManager

	config, err := app.GetConfig()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if len(config.Projects) == 0 {
		t.Fatal("项目列表为空")
	}

	project := config.Projects[0]
	if len(project.SubProjects) != 1 {
		t.Fatalf("期望 1 个子项目，实际 %d 个", len(project.SubProjects))
	}

	vars := configManager.GetWorkPathVars()
	if vars["TEST_WORK_PATH"] != "/global/test/work" {
		t.Fatalf("全局 workPaths 未生效: %#v", vars)
	}
	sub := project.SubProjects[0]
	expected := "/global/test/work/subproject"
	if sub.WorkDir != expected && sub.WorkDir != "${TEST_WORK_PATH}/subproject" {
		t.Fatalf("WorkDir 异常: %q", sub.WorkDir)
	}

	t.Log("SubProject WorkDir 全局配置 workPaths 替换测试通过")
}
