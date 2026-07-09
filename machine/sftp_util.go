package machine

import (
	"strings"

	"quick-cmd/define"
)

// CommandNeedsSFTP reports whether a remote command includes file upload steps.
func CommandNeedsSFTP(cmd define.Command) bool {
	for _, step := range cmd.Steps {
		trimmed := strings.TrimSpace(step)
		if strings.HasPrefix(trimmed, "upload ") {
			return true
		}
	}
	return false
}
