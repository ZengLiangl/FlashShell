package cmds

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"FlashDock/define"
	"FlashDock/utils"
)

func RegPackCmd() {
	CmdManager.RegSpecialCmd("pack", doPack)
}

func doPack(_ *define.RemoteMachine, c []string, outputChan chan<- string) error {
	outPath, sources, err := parsePackArgs(c)
	if err != nil {
		return err
	}
	for _, src := range sources {
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("源路径不存在: %s", src)
		}
	}
	if err := utils.LocalZipSources(outPath, sources); err != nil {
		return err
	}
	st, _ := os.Stat(outPath)
	size := int64(0)
	if st != nil {
		size = st.Size()
	}
	utils.SendOutput(outputChan, fmt.Sprintf("已打包: %s (%d 字节), 源 %d 项", outPath, size, len(sources)))
	return nil
}

// parsePackArgs:
//   pack <输出.zip> <源1> [源2 ...]
//   pack <输出目录> <压缩包名> <源1> [源2 ...]
func parsePackArgs(c []string) (outPath string, sources []string, err error) {
	if len(c) < 3 {
		return "", nil, errors.New("参数错误: pack <输出.zip> <源1> [源2 ...] 或 pack <输出目录> <压缩包名> <源1> [源2 ...]")
	}
	first := strings.TrimSpace(c[1])
	second := strings.TrimSpace(c[2])
	if strings.HasSuffix(strings.ToLower(first), ".zip") {
		return filepath.Clean(first), c[2:], nil
	}
	if len(c) < 4 {
		return "", nil, errors.New("参数错误: 输出路径不是 .zip 时需指定「输出目录 + 压缩包名」")
	}
	name := second
	if !strings.HasSuffix(strings.ToLower(name), ".zip") {
		name += ".zip"
	}
	out := filepath.Join(filepath.Clean(first), name)
	return out, c[3:], nil
}
