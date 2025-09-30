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
	configPath          string
	root                *define.Root
	globalConfigManager *GlobalConfigManager
}

// NewConfigManager 创建配置管理器
func NewConfigManager(configPath string) *ConfigManager {
	gcm := NewGlobalConfigManager("")

	// 如果没有指定配置路径，尝试从全局配置获取最后打开的文件
	if configPath == "" {
		if globalConfig, err := gcm.LoadGlobalConfig(); err == nil && globalConfig.LastOpenedFile != "" {
			configPath = globalConfig.LastOpenedFile
		} else {
			configPath = "config.yaml"
		}
	}

	return &ConfigManager{
		configPath:          configPath,
		globalConfigManager: gcm,
	}
}

// LoadConfigForRefresh 专门用于刷新的配置加载，不更新全局配置
func (cm *ConfigManager) LoadConfigForRefresh() (*define.Root, error) {
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

	// 注意：这里不更新全局配置，只是刷新内存中的数据
	cm.root = &root
	return &root, nil
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

	// 更新全局配置中的最后打开文件
	if cm.globalConfigManager != nil {
		cm.globalConfigManager.UpdateLastOpenedFile(cm.configPath)
		// 添加到配置文件列表中（如果不存在）
		cm.globalConfigManager.AddConfigFile(cm.configPath)
	}

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
	// fmt.Println("GetMachine", name)
	if cm.globalConfigManager == nil {
		return nil
	}

	for _, machine := range cm.globalConfigManager.config.Machines {
		if machine.Name == name {
			return &machine
		}
	}
	return nil
}

// AddMachineToGlobal 添加机器配置到全局配置
func (cm *ConfigManager) AddMachineToGlobal(machine *define.Machine) error {
	if cm.globalConfigManager == nil {
		return fmt.Errorf("全局配置管理器未初始化")
	}
	return cm.globalConfigManager.AddMachine(machine)
}

// GetMachineFromGlobal 从全局配置获取机器配置
func (cm *ConfigManager) GetMachineFromGlobal(name string) *define.Machine {
	if cm.globalConfigManager == nil {
		return nil
	}
	return cm.globalConfigManager.GetMachine(name)
}

// GetAllMachinesFromGlobal 从全局配置获取所有机器配置
func (cm *ConfigManager) GetAllMachinesFromGlobal() []define.Machine {
	if cm.globalConfigManager == nil {
		return []define.Machine{}
	}
	return cm.globalConfigManager.GetAllMachines()
}

// RemoveMachineFromGlobal 从全局配置移除机器配置
func (cm *ConfigManager) RemoveMachineFromGlobal(name string) error {
	if cm.globalConfigManager == nil {
		return fmt.Errorf("全局配置管理器未初始化")
	}
	return cm.globalConfigManager.RemoveMachine(name)
}

// processPathVariables 处理路径变量替换
func (cm *ConfigManager) processPathVariables(root *define.Root) {
	for i := range root.Projects {
		// 先进行工作路径变量替换，再展开路径
		root.Projects[i].WorkDir = expandPath(cm.replaceWorkPaths(root.Projects[i].WorkDir))

		for j := range root.Projects[i].SubProjects {
			// 处理 SubProject 的 WorkDir
			root.Projects[i].SubProjects[j].WorkDir = expandPath(cm.replaceWorkPaths(root.Projects[i].SubProjects[j].WorkDir))

			for k := range root.Projects[i].SubProjects[j].Commands {
				cmd := &root.Projects[i].SubProjects[j].Commands[k]
				cmd.WorkDir = expandPath(cm.replaceWorkPaths(cmd.WorkDir))

				// 处理命令步骤中的工作路径变量
				for l := range cmd.Steps {
					cmd.Steps[l] = cm.replaceWorkPaths(cmd.Steps[l])
				}
			}
		}
	}

	for i := range root.Machines {
		root.Machines[i].KeyFile = expandPath(cm.replaceWorkPaths(root.Machines[i].KeyFile))
	}
}

// replaceWorkPaths 替换工作路径变量
func (cm *ConfigManager) replaceWorkPaths(input string) string {
	if cm.globalConfigManager == nil {
		return input
	}

	// 确保全局配置已加载
	if cm.globalConfigManager.GetConfig() == nil {
		cm.globalConfigManager.LoadGlobalConfig()
	}

	return cm.globalConfigManager.ReplaceWorkPaths(input)
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

// GetGlobalConfig 获取全局配置
func (cm *ConfigManager) GetGlobalConfig() (*GlobalConfig, error) {
	if cm.globalConfigManager == nil {
		return nil, fmt.Errorf("全局配置管理器未初始化")
	}
	return cm.globalConfigManager.LoadGlobalConfig()
}

// GetGlobalConfigForRefresh 获取全局配置（用于刷新，从文件重新读取）
func (cm *ConfigManager) GetGlobalConfigForRefresh() (*GlobalConfig, error) {
	if cm.globalConfigManager == nil {
		return nil, fmt.Errorf("全局配置管理器未初始化")
	}
	return cm.globalConfigManager.LoadGlobalConfig()
}

// SaveGlobalConfig 保存全局配置
func (cm *ConfigManager) SaveGlobalConfig(config *GlobalConfig) error {
	if cm.globalConfigManager == nil {
		return fmt.Errorf("全局配置管理器未初始化")
	}
	return cm.globalConfigManager.SaveGlobalConfig(config)
}

// SwitchConfigFile 切换配置文件
func (cm *ConfigManager) SwitchConfigFile(configPath string) error {
	cm.configPath = configPath

	// 更新全局配置中的最后打开文件
	if cm.globalConfigManager != nil {
		if err := cm.globalConfigManager.UpdateLastOpenedFile(configPath); err != nil {
			return fmt.Errorf("更新最后打开文件失败: %w", err)
		}
	}

	// 重新加载配置
	_, err := cm.LoadConfig()
	return err
}

// GetConfigFiles 获取所有配置文件列表
func (cm *ConfigManager) GetConfigFiles() ([]string, error) {
	globalConfig, err := cm.GetGlobalConfig()
	if err != nil {
		return nil, err
	}
	return globalConfig.ConfigFiles, nil
}

// CreateDefaultConfig 创建默认配置文件
func CreateDefaultConfig(path string) error {
	defaultConfig := &define.Root{
		Projects: []define.Project{
			{
				Name:        "示例项目",
				Description: "这是一个示例项目",
				WorkDir:     "${HOME}/workspace/example",
				SubProjects: []define.SubProject{
					{
						Name:        "构建",
						Description: "构建相关命令",
						Commands: []define.Command{
							{
								Name:        "编译",
								Description: "编译项目",
								Type:        "batch",
								Steps:       []string{"${MVM} clean compile"},
							},
							{
								Name:        "测试",
								Description: "运行测试",
								Type:        "batch",
								Steps:       []string{"${MVM} test"},
							},
						},
					},
				},
			},
		},
		Machines: []define.Machine{
			{
				Name:    "示例服务器",
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

// GetAllWorkPathsFromGlobal 获取所有工作路径
func (cm *ConfigManager) GetAllWorkPathsFromGlobal() map[string]string {
	return cm.globalConfigManager.GetAllWorkPaths()
}

// AddWorkPathToGlobal 添加工作路径
func (cm *ConfigManager) AddWorkPathToGlobal(key, value string) error {
	return cm.globalConfigManager.AddWorkPath(key, value)
}

// UpdateWorkPathInGlobal 更新工作路径
func (cm *ConfigManager) UpdateWorkPathInGlobal(key, value string) error {
	return cm.globalConfigManager.UpdateWorkPath(key, value)
}

// RemoveWorkPathFromGlobal 移除工作路径
func (cm *ConfigManager) RemoveWorkPathFromGlobal(key string) error {
	return cm.globalConfigManager.RemoveWorkPath(key)
}
