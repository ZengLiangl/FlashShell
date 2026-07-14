package machine

import (
	"strings"

	"FlashDock/define"
)

// CommandNeedsSFTP reports whether a remote command includes file upload steps.
func CommandNeedsSFTP(cmd define.Command) bool {
	for _, step := range cmd.Steps {
		trimmed := strings.TrimSpace(step.Command)
		if strings.HasPrefix(trimmed, "upload ") {
			return true
		}
	}
	return false
}
