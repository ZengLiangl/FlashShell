package machine

import (
	"fmt"
	"strings"
)

// RemovePathReliable 删除远端路径；SFTP 递归删除失败时回退为 SSH rm -rf。
func (a *ShellAuxManager) RemovePathReliable(remotePath string) error {
	remotePath = strings.TrimSpace(remotePath)
	if remotePath == "" || remotePath == "/" {
		return fmt.Errorf("拒绝删除根路径")
	}
	if err := a.RemovePath(remotePath); err == nil {
		return nil
	} else if _, statErr := a.StatPath(remotePath); statErr != nil {
		return nil
	}
	q := shellSingleQuote(remotePath)
	out, execErr := a.Exec("rm -rf " + q)
	if execErr != nil {
		return fmt.Errorf("删除远端路径失败: %w (%s)", execErr, strings.TrimSpace(out))
	}
	if _, statErr := a.StatPath(remotePath); statErr == nil {
		return fmt.Errorf("删除远端路径后仍存在: %s", remotePath)
	}
	return nil
}
