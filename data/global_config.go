package data

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// GlobalConfig 全局配置结构
type GlobalConfig struct {
	AppId          string            `yaml:"appId" json:"appId"`
	WindowsName    string            `yaml:"windowsName" json:"windowsName"`
	ConfigFiles    []string          `yaml:"configFile" json:"configFile"`
	LastOpenedFile string            `yaml:"lastOpenedFile" json:"lastOpenedFile"`
	WorkPaths      map[string]string `yaml:"workPaths" json:"workPaths"`
}

// GlobalConfigManager 全局配置管理器
type GlobalConfigManager struct {
	configPath string
	config     *GlobalConfig
}

// NewGlobalConfigManager 创建全局配置管理器
func NewGlobalConfigManager(configPath string) *GlobalConfigManager {
	if configPath == "" {
		configPath = "global_config.yaml"
	}
	return &GlobalConfigManager{
		configPath: configPath,
	}
}

// LoadGlobalConfig 加载全局配置文件
func (gcm *GlobalConfigManager) LoadGlobalConfig() (*GlobalConfig, error) {
	expandedPath := expandPath(gcm.configPath)

	// 如果文件不存在，创建默认配置
	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		if err := gcm.createDefaultGlobalConfig(); err != nil {
			return nil, fmt.Errorf("创建默认全局配置失败: %w", err)
		}
	}

	data, err := os.ReadFile(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("读取全局配置文件失败: %w", err)
	}

	var config GlobalConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析全局配置文件失败: %w", err)
	}

	gcm.config = &config
	return &config, nil
}

// SaveGlobalConfig 保存全局配置文件
func (gcm *GlobalConfigManager) SaveGlobalConfig(config *GlobalConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化全局配置失败: %w", err)
	}

	expandedPath := expandPath(gcm.configPath)
	if err := os.WriteFile(expandedPath, data, 0644); err != nil {
		return fmt.Errorf("保存全局配置文件失败: %w", err)
	}

	gcm.config = config
	return nil
}

// GetConfig 获取全局配置
func (gcm *GlobalConfigManager) GetConfig() *GlobalConfig {
	return gcm.config
}

// UpdateLastOpenedFile 更新最后打开的配置文件
func (gcm *GlobalConfigManager) UpdateLastOpenedFile(filePath string) error {
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return err
		}
	}

	gcm.config.LastOpenedFile = filePath
	return gcm.SaveGlobalConfig(gcm.config)
}

// GetLastOpenedFile 获取最后打开的配置文件路径
func (gcm *GlobalConfigManager) GetLastOpenedFile() string {
	if gcm.config == nil {
		return ""
	}
	return gcm.config.LastOpenedFile
}

// AddConfigFile 添加配置文件路径
func (gcm *GlobalConfigManager) AddConfigFile(filePath string) error {
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return err
		}
	}

	// 检查是否已存在
	for _, existing := range gcm.config.ConfigFiles {
		if existing == filePath {
			return nil // 已存在，不需要添加
		}
	}

	gcm.config.ConfigFiles = append(gcm.config.ConfigFiles, filePath)
	return gcm.SaveGlobalConfig(gcm.config)
}

// RemoveConfigFile 移除配置文件路径
func (gcm *GlobalConfigManager) RemoveConfigFile(filePath string) error {
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return err
		}
	}

	for i, existing := range gcm.config.ConfigFiles {
		if existing == filePath {
			gcm.config.ConfigFiles = append(gcm.config.ConfigFiles[:i], gcm.config.ConfigFiles[i+1:]...)
			break
		}
	}

	return gcm.SaveGlobalConfig(gcm.config)
}

// ReplaceWorkPaths 替换字符串中的工作路径变量
func (gcm *GlobalConfigManager) ReplaceWorkPaths(input string) string {
	if gcm.config == nil || gcm.config.WorkPaths == nil {
		return input
	}

	result := input
	for key, value := range gcm.config.WorkPaths {
		placeholder := fmt.Sprintf("${%s}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

// createDefaultGlobalConfig 创建默认全局配置
func (gcm *GlobalConfigManager) createDefaultGlobalConfig() error {
	defaultConfig := &GlobalConfig{
		AppId:       "com.runner",
		WindowsName: "运行器",
		ConfigFiles: []string{
			"config.yaml",
		},
		LastOpenedFile: "config.yaml",
		WorkPaths: map[string]string{
			"HOME": "~",
		},
	}

	return gcm.SaveGlobalConfig(defaultConfig)
}
