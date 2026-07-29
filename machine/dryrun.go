package machine

import (
	"fmt"
	"strings"

	"FlashDock/define"
	"FlashDock/utils"
)

// DryRunLine 干跑展开的一行
type DryRunLine struct {
	CommandName string `json:"commandName"`
	CommandType string `json:"commandType"`
	Machine     string `json:"machine,omitempty"`
	WorkDir     string `json:"workdir,omitempty"`
	StepIndex   int    `json:"stepIndex"`
	StepCommand string `json:"stepCommand"`
	WhenExpr    string `json:"whenExpr,omitempty"`
	WhenResult  string `json:"whenResult,omitempty"`
	Skipped     bool   `json:"skipped"`
	Parallel    bool   `json:"parallel,omitempty"`
}

func resolveCommandWorkDir(command define.Command, ctx *define.ExecutionContext) string {
	workDir := ctx.ProjectWorkDir
	if ctx.SubProjectWorkDir != "" {
		workDir = ctx.SubProjectWorkDir
	}
	if command.WorkDir != "" {
		workDir = command.WorkDir
	}
	return workDir
}

func dryRunSteps(command define.Command, ctx *define.ExecutionContext, lines *[]DryRunLine) error {
	workDir := resolveCommandWorkDir(command, ctx)
	for i, step := range command.Steps {
		ok, whenLabel, err := FormatWhenResult(step.When, ctx.WorkPathVars)
		if err != nil {
			return fmt.Errorf("命令 '%s' 步骤 %d when 表达式无效: %w", command.Name, i+1, err)
		}
		line := DryRunLine{
			CommandName: command.Name,
			CommandType: command.Type,
			Machine:     command.Machine,
			WorkDir:     workDir,
			StepIndex:   i + 1,
			StepCommand: step.Command,
			WhenExpr:    strings.TrimSpace(step.When),
			WhenResult:  whenLabel,
			Skipped:     !ok,
			Parallel:    command.Parallel,
		}
		*lines = append(*lines, line)
	}
	return nil
}

func formatDryRunLine(line DryRunLine) string {
	parts := []string{
		fmt.Sprintf("[%s/%s]", line.CommandType, line.CommandName),
	}
	if line.Parallel {
		parts = append(parts, "(并行组)")
	}
	if line.Machine != "" {
		parts = append(parts, fmt.Sprintf("机器=%s", line.Machine))
	}
	if line.WorkDir != "" {
		parts = append(parts, fmt.Sprintf("目录=%s", line.WorkDir))
	}
	parts = append(parts, fmt.Sprintf("步骤%d: %s", line.StepIndex, line.StepCommand))
	if line.WhenResult != "" {
		parts = append(parts, line.WhenResult)
	}
	if line.Skipped {
		parts = append(parts, "→ 跳过")
	} else {
		parts = append(parts, "→ 将执行")
	}
	return strings.Join(parts, " ")
}

// DryRunSubProject 展开 SubProject 将执行的步骤（不真正执行）
func (spr *SubProjectRunner) DryRunSubProject(projectName, subProjectName string, output chan<- string) ([]DryRunLine, error) {
	root := spr.configManager.GetRoot()
	if root == nil {
		return nil, fmt.Errorf("配置未加载")
	}
	var subProject *define.SubProject
	var project *define.Project
	for _, p := range root.Projects {
		if p.Name == projectName {
			project = &p
			for _, sp := range p.SubProjects {
				if sp.Name == subProjectName {
					subProject = &sp
					break
				}
			}
			break
		}
	}
	if subProject == nil {
		return nil, fmt.Errorf("未找到 SubProject: %s/%s", projectName, subProjectName)
	}
	ctx := &define.ExecutionContext{
		ProjectName:       projectName,
		SubProjectName:    subProjectName,
		Commands:          subProject.Commands,
		ProjectWorkDir:    project.WorkDir,
		SubProjectWorkDir: subProject.WorkDir,
		WorkPathVars:      spr.configManager.GetWorkPathVars(),
	}
	utils.SendOutput(output, fmt.Sprintf("干跑 SubProject: %s/%s（共 %d 个命令）", projectName, subProjectName, len(subProject.Commands)))
	var lines []DryRunLine
	for _, group := range GroupParallelCommands(subProject.Commands) {
		if len(group) > 1 {
			names := make([]string, len(group))
			for i, cmd := range group {
				names[i] = cmd.Name
			}
			utils.SendOutput(output, fmt.Sprintf("并行组: %s", strings.Join(names, ", ")))
		}
		for _, command := range group {
			start := len(lines)
			if err := dryRunSteps(command, ctx, &lines); err != nil {
				return lines, err
			}
			for _, line := range lines[start:] {
				utils.SendOutput(output, formatDryRunLine(line))
			}
		}
	}
	utils.SendOutput(output, "干跑完成（未实际执行任何命令）")
	return lines, nil
}
