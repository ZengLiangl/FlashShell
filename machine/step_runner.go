package machine

import (
	"fmt"

	"FlashDock/define"
	"FlashDock/utils"
)

type stepExecutor func(command string, output chan<- string) error

// executeSteps 按顺序执行步骤，支持重试与失败策略
func executeSteps(steps define.StepList, output chan<- string, onStepStart func(string), onStepComplete func(), exec stepExecutor) error {
	for i, step := range steps {
		if onStepStart != nil {
			onStepStart(step.Command)
		}

		var lastErr error
		attempts := step.MaxAttempts()
		for attempt := 1; attempt <= attempts; attempt++ {
			if attempts > 1 {
				utils.SendOutput(output, fmt.Sprintf("执行步骤 %d（第 %d/%d 次）: %s", i+1, attempt, attempts, step.Command))
			} else {
				utils.SendOutput(output, fmt.Sprintf("执行步骤 %d: %s", i+1, step.Command))
			}

			lastErr = exec(step.Command, output)
			if lastErr == nil {
				break
			}

			if attempt < attempts {
				utils.SendOutput(output, fmt.Sprintf("步骤 %d 失败，准备重试: %s", i+1, lastErr.Error()))
			}
		}

		if lastErr != nil {
			if step.NormalizedOnFail() == define.OnFailContinue {
				utils.SendOutput(output, fmt.Sprintf("步骤 %d 失败但继续执行: %s", i+1, lastErr.Error()))
				if onStepComplete != nil {
					onStepComplete()
				}
				continue
			}
			return fmt.Errorf("步骤 %d 执行失败: %w", i+1, lastErr)
		}

		if onStepComplete != nil {
			onStepComplete()
		}
	}
	return nil
}

// CommandNeedsSFTP 判断命令步骤是否需要 SFTP（见 sftp_util.go）
