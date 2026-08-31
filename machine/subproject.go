package machine

import (
	"fmt"
	"strings"
	"sync"

	"FlashDock/define"
	"FlashDock/utils"
)

// ShellClientProvider 任务执行时复用已连接 SSH（Shell 会话或 MCP 持有的连接）
type ShellClientProvider interface {
	SharedClientForConfig(configName string) *SSHClient
}

// RemoteFailureInfo 远程命令失败上下文
type RemoteFailureInfo struct {
	MachineName string
	WorkDir     string
	CommandName string
	Error       string
}

// SubProjectRunner SubProject 执行器实现
type SubProjectRunner struct {
	configManager   ConfigManagerInterface
	shellPool       ShellClientProvider
	runners         map[string]define.Runner
	runnerMutex     sync.RWMutex
	currentStatus   *define.SubProjectStatus
	statusMutex     sync.RWMutex
	runID           uint64
	stopChannel     chan bool
	onStatusChange  func(*define.SubProjectStatus)
	onRemoteFailure func(RemoteFailureInfo)
}

// ConfigManagerInterface 配置管理器接口
type ConfigManagerInterface interface {
	GetRoot() *define.Root
	GetMachine(name string) *define.Machine
	GetWorkPathVars() map[string]string
}

// NewSubProjectRunner 创建 SubProject 执行器
func NewSubProjectRunner(configManager ConfigManagerInterface) *SubProjectRunner {
	return &SubProjectRunner{
		configManager: configManager,
		runners:       make(map[string]define.Runner),
		currentStatus: &define.SubProjectStatus{},
		stopChannel:   make(chan bool, 1),
	}
}

// SetShellClientProvider 注入 Shell 会话池以复用 SSH
func (spr *SubProjectRunner) SetShellClientProvider(provider ShellClientProvider) {
	spr.shellPool = provider
}

// SetStatusChangeHandler 设置状态变更回调（用于事件推送）
func (spr *SubProjectRunner) SetStatusChangeHandler(handler func(*define.SubProjectStatus)) {
	spr.onStatusChange = handler
}

// SetRemoteFailureHandler 远程命令失败回调（用于一键进 Shell）
func (spr *SubProjectRunner) SetRemoteFailureHandler(handler func(RemoteFailureInfo)) {
	spr.onRemoteFailure = handler
}

func (spr *SubProjectRunner) clearStopSignal() {
	for {
		select {
		case <-spr.stopChannel:
		default:
			return
		}
	}
}

// beginRun 开始一次执行，返回本次 runID；defer finishRun(runID)。
func (spr *SubProjectRunner) beginRun(status *define.SubProjectStatus) uint64 {
	spr.statusMutex.Lock()
	spr.runID++
	runID := spr.runID
	spr.currentStatus = status
	handler := spr.onStatusChange
	spr.statusMutex.Unlock()
	if handler != nil {
		handler(spr.GetExecutionStatus())
	}
	return runID
}

func (spr *SubProjectRunner) finishRun(runID uint64) {
	spr.statusMutex.Lock()
	if spr.runID != runID {
		spr.statusMutex.Unlock()
		return
	}
	spr.currentStatus.IsRunning = false
	spr.currentStatus.CurrentCommand = ""
	spr.currentStatus.CurrentStep = ""
	handler := spr.onStatusChange
	spr.statusMutex.Unlock()
	if handler != nil {
		handler(spr.GetExecutionStatus())
	}
}

func (spr *SubProjectRunner) stopCurrentIfRunning() {
	spr.statusMutex.RLock()
	running := spr.currentStatus != nil && spr.currentStatus.IsRunning
	projectName := ""
	subName := ""
	if spr.currentStatus != nil {
		projectName = spr.currentStatus.ProjectName
		subName = spr.currentStatus.SubProjectName
	}
	spr.statusMutex.RUnlock()
	if running {
		_ = spr.StopSubProject(projectName, subName)
	}
}

// ExecuteSubProject 执行 SubProject
func (spr *SubProjectRunner) ExecuteSubProject(projectName, subProjectName string, output chan<- string) error {
	spr.stopCurrentIfRunning()
	spr.clearStopSignal()

	root := spr.configManager.GetRoot()
	if root == nil {
		return fmt.Errorf("配置未加载")
	}

	var subProject *define.SubProject
	var project *define.Project
	for i := range root.Projects {
		if root.Projects[i].Name != projectName {
			continue
		}
		project = &root.Projects[i]
		for j := range project.SubProjects {
			if project.SubProjects[j].Name == subProjectName {
				subProject = &project.SubProjects[j]
				break
			}
		}
		break
	}
	if subProject == nil {
		return fmt.Errorf("未找到 SubProject: %s/%s", projectName, subProjectName)
	}

	totalSteps := 0
	for _, cmd := range subProject.Commands {
		totalSteps += len(cmd.Steps)
	}

	runID := spr.beginRun(&define.SubProjectStatus{
		ProjectName:    projectName,
		SubProjectName: subProjectName,
		IsRunning:      true,
		TotalCommands:  len(subProject.Commands),
		TotalSteps:     totalSteps,
	})
	defer spr.finishRun(runID)

	utils.SendOutput(output, fmt.Sprintf("开始执行 SubProject: %s/%s", projectName, subProjectName))
	utils.SendOutput(output, fmt.Sprintf("总共 %d 个命令需要执行", len(subProject.Commands)))

	ctx := &define.ExecutionContext{
		ProjectName:       projectName,
		SubProjectName:    subProjectName,
		Commands:          subProject.Commands,
		ProjectWorkDir:    project.WorkDir,
		SubProjectWorkDir: subProject.WorkDir,
		WorkPathVars:      spr.configManager.GetWorkPathVars(),
	}

	completedCommands := 0
	for _, group := range GroupParallelCommands(subProject.Commands) {
		select {
		case <-spr.stopChannel:
			utils.SendOutput(output, "执行已被用户停止")
			spr.updateStatusForRun(runID, "", "", completedCommands, spr.completedSteps(), false)
			return fmt.Errorf("执行被用户停止")
		default:
		}
		if len(group) == 1 {
			command := group[0]
			spr.updateStatusForRun(runID, command.Name, "", completedCommands, spr.completedSteps(), true)
			utils.SendOutput(output, fmt.Sprintf("执行命令 %d/%d: %s", completedCommands+1, len(subProject.Commands), command.Name))
			utils.SendOutput(output, fmt.Sprintf("命令类型: %s", command.Type))
			if err := spr.executeCommand(runID, command, ctx, output); err != nil {
				spr.updateStatusForRun(runID, command.Name, "", completedCommands, spr.completedSteps(), false)
				return fmt.Errorf("命令 '%s' 执行失败: %w", command.Name, err)
			}
			utils.SendOutput(output, fmt.Sprintf("命令 '%s' 执行完成", command.Name))
			completedCommands++
			spr.updateStatusForRun(runID, command.Name, "", completedCommands, spr.completedSteps(), true)
			continue
		}

		names := make([]string, len(group))
		for i, cmd := range group {
			names[i] = cmd.Name
		}
		utils.SendOutput(output, fmt.Sprintf("并行执行命令组: %s", strings.Join(names, ", ")))
		if err := spr.executeParallelGroup(runID, group, ctx, output); err != nil {
			spr.updateStatusForRun(runID, "", "", completedCommands, spr.completedSteps(), false)
			return err
		}
		completedCommands += len(group)
		spr.updateStatusForRun(runID, "", "", completedCommands, spr.completedSteps(), true)
	}

	utils.SendOutput(output, fmt.Sprintf("'%s' 执行完成！", subProjectName))
	spr.updateStatusForRun(runID, "", "", len(subProject.Commands), totalSteps, false)
	return nil
}

// ExecuteCommand 仅执行指定 Command
func (spr *SubProjectRunner) ExecuteCommand(projectName, subProjectName, commandName string, output chan<- string) error {
	spr.stopCurrentIfRunning()
	spr.clearStopSignal()

	root := spr.configManager.GetRoot()
	if root == nil {
		return fmt.Errorf("配置未加载")
	}

	var subProject *define.SubProject
	var project *define.Project
	var command *define.Command
	for i := range root.Projects {
		if root.Projects[i].Name != projectName {
			continue
		}
		project = &root.Projects[i]
		for j := range project.SubProjects {
			if project.SubProjects[j].Name != subProjectName {
				continue
			}
			subProject = &project.SubProjects[j]
			for k := range subProject.Commands {
				if subProject.Commands[k].Name == commandName {
					command = &subProject.Commands[k]
					break
				}
			}
			break
		}
		break
	}
	if subProject == nil {
		return fmt.Errorf("未找到 SubProject: %s/%s", projectName, subProjectName)
	}
	if command == nil {
		return fmt.Errorf("未找到命令: %s/%s/%s", projectName, subProjectName, commandName)
	}

	totalSteps := len(command.Steps)
	runID := spr.beginRun(&define.SubProjectStatus{
		ProjectName:    projectName,
		SubProjectName: subProjectName,
		IsRunning:      true,
		TotalCommands:  1,
		TotalSteps:     totalSteps,
	})
	defer spr.finishRun(runID)

	utils.SendOutput(output, fmt.Sprintf("开始执行命令: %s/%s/%s", projectName, subProjectName, commandName))
	utils.SendOutput(output, fmt.Sprintf("命令类型: %s", command.Type))

	ctx := &define.ExecutionContext{
		ProjectName:       projectName,
		SubProjectName:    subProjectName,
		Commands:          []define.Command{*command},
		ProjectWorkDir:    project.WorkDir,
		SubProjectWorkDir: subProject.WorkDir,
		WorkPathVars:      spr.configManager.GetWorkPathVars(),
	}

	spr.updateStatusForRun(runID, command.Name, "", 0, 0, true)
	if err := spr.executeCommand(runID, *command, ctx, output); err != nil {
		spr.updateStatusForRun(runID, command.Name, "", 0, spr.completedSteps(), false)
		return fmt.Errorf("命令 '%s' 执行失败: %w", command.Name, err)
	}

	utils.SendOutput(output, fmt.Sprintf("命令 '%s' 执行完成", command.Name))
	spr.updateStatusForRun(runID, "", "", 1, totalSteps, false)
	return nil
}

func (spr *SubProjectRunner) completedSteps() int {
	spr.statusMutex.RLock()
	defer spr.statusMutex.RUnlock()
	if spr.currentStatus == nil {
		return 0
	}
	return spr.currentStatus.CompletedSteps
}

func (spr *SubProjectRunner) completedCommands() int {
	spr.statusMutex.RLock()
	defer spr.statusMutex.RUnlock()
	if spr.currentStatus == nil {
		return 0
	}
	return spr.currentStatus.CompletedCommands
}

func (spr *SubProjectRunner) executeParallelGroup(runID uint64, group []define.Command, ctx *define.ExecutionContext, output chan<- string) error {
	type result struct {
		command define.Command
		err     error
	}
	results := make(chan result, len(group))
	var wg sync.WaitGroup

	for _, command := range group {
		cmd := command
		wg.Add(1)
		go func() {
			defer wg.Done()
			utils.SendOutput(output, fmt.Sprintf("并行命令开始: %s (%s)", cmd.Name, cmd.Type))
			err := spr.executeCommand(runID, cmd, ctx, output)
			if err == nil {
				utils.SendOutput(output, fmt.Sprintf("并行命令完成: %s", cmd.Name))
			}
			results <- result{command: cmd, err: err}
		}()
	}

	wg.Wait()
	close(results)

	var firstErr error
	var failedCmd define.Command
	for res := range results {
		if res.err != nil && firstErr == nil {
			firstErr = res.err
			failedCmd = res.command
		}
	}
	if firstErr != nil {
		return fmt.Errorf("并行命令 '%s' 执行失败: %w", failedCmd.Name, firstErr)
	}
	return nil
}

func (spr *SubProjectRunner) executeCommand(runID uint64, command define.Command, ctx *define.ExecutionContext, output chan<- string) error {
	var runner define.Runner
	workDir := resolveCommandWorkDir(command, ctx)

	switch command.Type {
	case "batch":
		runner = NewLocalRunner(workDir, ctx.WorkPathVars)
	case "remote":
		if command.Machine == "" {
			return fmt.Errorf("远程命令未指定机器")
		}
		machineConfig := spr.configManager.GetMachine(command.Machine)
		if machineConfig == nil {
			return fmt.Errorf("未找到机器配置: %s", command.Machine)
		}

		var sshClient *SSHClient
		var ownsClient bool
		if spr.shellPool != nil {
			if shared := spr.shellPool.SharedClientForConfig(command.Machine); shared != nil && shared.IsConnected() {
				sshClient = NewSSHClient(machineConfig, ctx.WorkPathVars)
				sshClient.AttachRemote(shared.SharedRemoteMachine(), machineConfig, ctx.WorkPathVars)
				utils.SendOutput(output, fmt.Sprintf("复用已连接 SSH: %s", command.Machine))
			}
		}
		if sshClient == nil {
			sshClient = NewSSHClient(machineConfig, ctx.WorkPathVars)
			if err := sshClient.ConnectAutoTrustOnce(machineConfig, CommandNeedsSFTP(command)); err != nil {
				spr.notifyRemoteFailure(command, workDir, err)
				return fmt.Errorf("连接远程机器失败: %w", err)
			}
			ownsClient = true
		}
		if ownsClient {
			defer sshClient.Close()
		} else {
			defer func() {
				_ = sshClient.Stop()
			}()
		}
		runner = sshClient
	default:
		return fmt.Errorf("不支持的命令类型: %s", command.Type)
	}

	runnerKey := fmt.Sprintf("%s/%s/%s", ctx.ProjectName, ctx.SubProjectName, command.Name)
	spr.runnerMutex.Lock()
	spr.runners[runnerKey] = runner
	spr.runnerMutex.Unlock()

	shouldStop := func() bool {
		select {
		case <-spr.stopChannel:
			return true
		default:
			return false
		}
	}
	err := runner.Execute(command, output,
		func(step string) {
			spr.updateStatusForRun(runID, command.Name, step, spr.completedCommands(), spr.completedSteps(), true)
		},
		func() {
			spr.updateStatusForRun(runID, command.Name, "", spr.completedCommands(), spr.completedSteps()+1, true)
		},
		shouldStop)

	spr.runnerMutex.Lock()
	delete(spr.runners, runnerKey)
	spr.runnerMutex.Unlock()

	if err != nil && command.Type == "remote" {
		spr.notifyRemoteFailure(command, workDir, err)
	}
	return err
}

func (spr *SubProjectRunner) notifyRemoteFailure(command define.Command, workDir string, err error) {
	if spr.onRemoteFailure == nil || command.Type != "remote" {
		return
	}
	spr.onRemoteFailure(RemoteFailureInfo{
		MachineName: command.Machine,
		WorkDir:     workDir,
		CommandName: command.Name,
		Error:       err.Error(),
	})
}

// StopSubProject 停止 SubProject 执行
func (spr *SubProjectRunner) StopSubProject(projectName, subProjectName string) error {
	select {
	case spr.stopChannel <- true:
	default:
	}

	prefix := fmt.Sprintf("%s/%s", projectName, subProjectName)
	spr.runnerMutex.RLock()
	runners := make([]define.Runner, 0, len(spr.runners))
	for key, runner := range spr.runners {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			runners = append(runners, runner)
		}
	}
	spr.runnerMutex.RUnlock()

	for _, runner := range runners {
		if err := runner.Stop(); err != nil {
			fmt.Printf("停止执行器时出错: %v\n", err)
		}
	}

	spr.statusMutex.Lock()
	spr.runID++ // 使进行中的 run 状态回调失效
	if spr.currentStatus != nil {
		spr.currentStatus.IsRunning = false
		spr.currentStatus.CurrentCommand = ""
		spr.currentStatus.CurrentStep = ""
	}
	handler := spr.onStatusChange
	spr.statusMutex.Unlock()
	if handler != nil {
		handler(spr.GetExecutionStatus())
	}

	return nil
}

// GetExecutionStatus 获取执行状态
func (spr *SubProjectRunner) GetExecutionStatus() *define.SubProjectStatus {
	spr.statusMutex.RLock()
	defer spr.statusMutex.RUnlock()

	status := *spr.currentStatus
	return &status
}

func (spr *SubProjectRunner) updateStatusForRun(runID uint64, currentCommand string, currentStep string, completedCommands int, completedSteps int, isRunning bool) {
	spr.statusMutex.Lock()
	if spr.runID != runID {
		spr.statusMutex.Unlock()
		return
	}
	spr.currentStatus.CurrentCommand = currentCommand
	spr.currentStatus.CurrentStep = currentStep
	spr.currentStatus.CompletedCommands = completedCommands
	spr.currentStatus.CompletedSteps = completedSteps
	spr.currentStatus.IsRunning = isRunning
	handler := spr.onStatusChange
	spr.statusMutex.Unlock()

	if handler != nil {
		handler(spr.GetExecutionStatus())
	}
}
