package data

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"quick-cmd/define"

	"gopkg.in/yaml.v3"
)

// ConfigManager 配置管理器
type ConfigManager struct {
	configPath string
	root       *define.Root
}

// NewConfigManager 创建配置管理器
func NewConfigManager(configPath string) *ConfigManager {
	return &ConfigManager{
		configPath: configPath,
	}
}

// LoadConfig 加载配置文件
func (cm *ConfigManager) LoadConfig() (*define.Root, error) {
	if cm.configPath == "" {
		cm.configPath = "config.yaml"
	}

	// 展开路径中的 ~ 符号
	expandedPath := expandPath(cm.configPath)

	data, err := os.ReadFile(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var root define.Root
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 处理路径变量替换
	cm.processPathVariables(&root)

	cm.root = &root
	return &root, nil
}

// SaveConfig 保存配置文件
func (cm *ConfigManager) SaveConfig(root *define.Root) error {
	data, err := yaml.Marshal(root)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	expandedPath := expandPath(cm.configPath)
	if err := os.WriteFile(expandedPath, data, 0644); err != nil {
		return fmt.Errorf("保存配置文件失败: %w", err)
	}

	cm.root = root
	return nil
}

// GetRoot 获取根配置
func (cm *ConfigManager) GetRoot() *define.Root {
	return cm.root
}

// GetMachine 根据名称获取机器配置
func (cm *ConfigManager) GetMachine(name string) *define.Machine {
	if cm.root == nil {
		return nil
	}

	for _, machine := range cm.root.Machines {
		if machine.Name == name {
			return &machine
		}
	}
	return nil
}

// processPathVariables 处理路径变量替换
func (cm *ConfigManager) processPathVariables(root *define.Root) {
	for i := range root.Projects {
		root.Projects[i].WorkDir = expandPath(root.Projects[i].WorkDir)

		for j := range root.Projects[i].SubProjects {
			for k := range root.Projects[i].SubProjects[j].Commands {
				cmd := &root.Projects[i].SubProjects[j].Commands[k]
				cmd.WorkDir = expandPath(cmd.WorkDir)
			}
		}
	}

	for i := range root.Machines {
		root.Machines[i].KeyFile = expandPath(root.Machines[i].KeyFile)
	}
}

// expandPath 展开路径中的环境变量和 ~ 符号
func expandPath(path string) string {
	if path == "" {
		return path
	}

	// 展开 ~ 符号
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(homeDir, path[2:])
		}
	}

	// 展开环境变量
	path = os.ExpandEnv(path)

	return path
}

// CreateDefaultConfig 创建默认配置文件
func CreateDefaultConfig(path string) error {
	defaultConfig := &define.Root{
		Projects: []define.Project{
			{
				Name:        "示例项目",
				Description: "这是一个示例项目",
				WorkDir:     "~/workspace/example",
				SubProjects: []define.SubProject{
					{
						Name:        "构建",
						Description: "构建相关命令",
						Commands: []define.Command{
							{
								Name:        "编译",
								Description: "编译项目",
								Type:        "batch",
								Steps:       []string{"go build ."},
							},
							{
								Name:        "测试",
								Description: "运行测试",
								Type:        "batch",
								Steps:       []string{"go test ./..."},
							},
						},
					},
				},
			},
		},
		Machines: []define.Machine{
			{
				Name:    "示例服务器",
				Host:    "example.com",
				Port:    22,
				User:    "deploy",
				KeyFile: "~/.ssh/id_rsa",
			},
		},
	}

	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("序列化默认配置失败: %w", err)
	}

	expandedPath := expandPath(path)
	if err := os.WriteFile(expandedPath, data, 0644); err != nil {
		return fmt.Errorf("创建默认配置文件失败: %w", err)
	}

	return nil
}
