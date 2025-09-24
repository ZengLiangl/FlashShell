package machine

import (
	"testing"
	"time"

	"quick-cmd/define"
)

// MockConfigManager 模拟配置管理器
type MockConfigManager struct {
	root *define.Root
}

func (m *MockConfigManager) GetRoot() *define.Root {
	return m.root
}

func (m *MockConfigManager) GetMachine(name string) *define.Machine {
	if m.root == nil {
		return nil
	}
	for _, machine := range m.root.Machines {
		if machine.Name == name {
			return &machine
		}
	}
	return nil
}

func TestSubProjectRunner_ExecuteSubProject(t *testing.T) {
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
								Name:        "测试命令1",
								Description: "第一个测试命令",
								Type:        "batch",
								Steps:       []string{"echo 'Hello World'"},
							},
							{
								Name:        "测试命令2",
								Description: "第二个测试命令",
								Type:        "batch",
								Steps:       []string{"echo 'Test Complete'"},
							},
						},
					},
				},
			},
		},
		Machines: []define.Machine{},
	}

	// 创建模拟配置管理器
	mockConfig := &MockConfigManager{root: testRoot}

	// 创建 SubProjectRunner
	runner := NewSubProjectRunner(mockConfig)

	// 创建输出通道
	output := make(chan string, 100)

	// 执行 SubProject
	go func() {
		err := runner.ExecuteSubProject("测试项目", "测试子项目", output)
		if err != nil {
			t.Errorf("执行 SubProject 失败: %v", err)
		}
	}()

	// 等待执行完成
	time.Sleep(3 * time.Second)

	// 检查状态
	status := runner.GetExecutionStatus()
	if status.ProjectName != "测试项目" {
		t.Errorf("项目名称不匹配: expected '测试项目', got '%s'", status.ProjectName)
	}
	if status.SubProjectName != "测试子项目" {
		t.Errorf("子项目名称不匹配: expected '测试子项目', got '%s'", status.SubProjectName)
	}
	if status.TotalCommands != 2 {
		t.Errorf("命令总数不匹配: expected 2, got %d", status.TotalCommands)
	}

	// 检查输出
	close(output)
	outputCount := 0
	for range output {
		outputCount++
	}
	if outputCount == 0 {
		t.Error("没有输出")
	}
}

func TestSubProjectRunner_StopSubProject(t *testing.T) {
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
								Name:        "长时间命令",
								Description: "一个长时间运行的命令",
								Type:        "batch",
								Steps:       []string{"sleep 10"},
							},
						},
					},
				},
			},
		},
		Machines: []define.Machine{},
	}

	// 创建模拟配置管理器
	mockConfig := &MockConfigManager{root: testRoot}

	// 创建 SubProjectRunner
	runner := NewSubProjectRunner(mockConfig)

	// 创建输出通道
	output := make(chan string, 100)

	// 开始执行 SubProject
	go func() {
		runner.ExecuteSubProject("测试项目", "测试子项目", output)
	}()

	// 等待一段时间确保开始执行
	time.Sleep(1 * time.Second)

	// 检查是否正在运行
	status := runner.GetExecutionStatus()
	if !status.IsRunning {
		t.Error("SubProject 应该正在运行")
	}

	// 停止执行
	err := runner.StopSubProject("测试项目", "测试子项目")
	if err != nil {
		t.Errorf("停止 SubProject 失败: %v", err)
	}

	// 等待停止完成
	time.Sleep(2 * time.Second)

	// 检查是否已停止
	status = runner.GetExecutionStatus()
	if status.IsRunning {
		t.Error("SubProject 应该已经停止")
	}

	close(output)
}

func TestSubProjectRunner_GetExecutionStatus(t *testing.T) {
	// 创建空配置
	mockConfig := &MockConfigManager{root: &define.Root{}}

	// 创建 SubProjectRunner
	runner := NewSubProjectRunner(mockConfig)

	// 获取初始状态
	status := runner.GetExecutionStatus()
	if status == nil {
		t.Error("状态不应该为空")
	}
	if status.IsRunning {
		t.Error("初始状态应该是未运行")
	}
	if status.CompletedCommands != 0 {
		t.Error("初始完成命令数应该为0")
	}
	if status.TotalCommands != 0 {
		t.Error("初始总命令数应该为0")
	}
}

func TestSubProjectRunner_ExecuteSubProject_NotFound(t *testing.T) {
	// 创建空配置
	mockConfig := &MockConfigManager{root: &define.Root{}}

	// 创建 SubProjectRunner
	runner := NewSubProjectRunner(mockConfig)

	// 创建输出通道
	output := make(chan string, 10)

	// 尝试执行不存在的 SubProject
	err := runner.ExecuteSubProject("不存在的项目", "不存在的子项目", output)
	if err == nil {
		t.Error("应该返回错误")
	}

	expectedError := "未找到 SubProject: 不存在的项目/不存在的子项目"
	if err.Error() != expectedError {
		t.Errorf("错误消息不匹配: expected '%s', got '%s'", expectedError, err.Error())
	}

	close(output)
}
