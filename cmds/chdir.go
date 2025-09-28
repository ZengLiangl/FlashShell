package cmds

import (
	"errors"
	"os"
	"quick-cmd/define"
	"strings"
)

func RegChdir() {
	CmdManager.RegSpecialCmd("chdir", doChdir)
}
func doChdir(rm *define.RemoteMachine, c []string, outputChan chan<- string) error {
	if len(c) != 2 {
		return errors.New("参数错误" + strings.Join(c, ","))
	}
	return os.Chdir(c[1])
}
