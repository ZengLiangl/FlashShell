package define

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	OnFailAbort    = "abort"
	OnFailContinue = "continue"
)

// Step 命令步骤，支持条件与失败策略
type Step struct {
	Command string `yaml:"cmd" json:"command"`
	When    string `yaml:"when,omitempty" json:"when,omitempty"`
	OnFail  string `yaml:"on_fail,omitempty" json:"onFail,omitempty"`
	Retry   int    `yaml:"retry,omitempty" json:"retry,omitempty"`
}

// NormalizedOnFail 返回规范化失败策略，默认为 abort
func (s Step) NormalizedOnFail() string {
	switch strings.ToLower(strings.TrimSpace(s.OnFail)) {
	case OnFailContinue:
		return OnFailContinue
	default:
		return OnFailAbort
	}
}

// MaxAttempts 返回最大尝试次数（至少 1）
func (s Step) MaxAttempts() int {
	if s.Retry < 1 {
		return 1
	}
	return s.Retry + 1
}

// StepList 兼容字符串与对象两种 YAML 写法
type StepList []Step

// UnmarshalYAML 解析 steps 字段
func (sl *StepList) UnmarshalYAML(value *yaml.Node) error {
	if value == nil {
		return nil
	}

	var items []yaml.Node
	if err := value.Decode(&items); err != nil {
		return err
	}

	result := make([]Step, 0, len(items))
	for _, item := range items {
		switch item.Kind {
		case yaml.ScalarNode:
			cmd := strings.TrimSpace(item.Value)
			if cmd != "" {
				result = append(result, Step{Command: cmd})
			}
		case yaml.MappingNode:
			var step Step
			if err := item.Decode(&step); err != nil {
				return err
			}
			step.Command = strings.TrimSpace(step.Command)
			if step.Command == "" {
				return fmt.Errorf("步骤对象缺少 cmd 字段")
			}
			result = append(result, step)
		default:
			return fmt.Errorf("不支持的步骤类型")
		}
	}

	*sl = result
	return nil
}

// MarshalYAML 无扩展字段时输出字符串，否则输出对象
func (sl StepList) MarshalYAML() (interface{}, error) {
	if len(sl) == 0 {
		return []string{}, nil
	}

	items := make([]interface{}, 0, len(sl))
	for _, step := range sl {
		if step.When == "" && step.OnFail == "" && step.Retry <= 0 {
			items = append(items, step.Command)
			continue
		}
		obj := map[string]interface{}{
			"cmd": step.Command,
		}
		if step.When != "" {
			obj["when"] = step.When
		}
		if step.OnFail != "" {
			obj["on_fail"] = step.OnFail
		}
		if step.Retry > 0 {
			obj["retry"] = step.Retry
		}
		items = append(items, obj)
	}
	return items, nil
}

// Commands 返回纯命令字符串列表（兼容旧逻辑）
func (sl StepList) Commands() []string {
	cmds := make([]string, len(sl))
	for i, step := range sl {
		cmds[i] = step.Command
	}
	return cmds
}
