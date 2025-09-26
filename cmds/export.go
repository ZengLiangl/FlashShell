package cmds

import (
	"quick-cmd/define"
)

var (
	CmdManager = define.CMDManager{SpecialCmd: make(map[string]func(*define.RemoteMachine, []string, chan<- string) error)}
)

func init() {
	RegUploadCmd()
}
