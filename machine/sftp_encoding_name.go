package machine

import (
	"strings"

	"FlashDock/define"
)

func decodeSftpName(machine *define.Machine, name string) string {
	if machine == nil {
		return name
	}
	enc := strings.ToLower(strings.TrimSpace(machine.SftpEncoding))
	switch enc {
	case "gb18030", "gbk":
		return decodeBytesAsGB18030([]byte(name))
	default:
		return name
	}
}
