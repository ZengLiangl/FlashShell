package cmds

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"FlashDock/define"
	"FlashDock/utils"
)

func RegPurgeCmd() {
	CmdManager.RegSpecialCmd("purge", doPurge)
}

func doPurge(rm *define.RemoteMachine, c []string, outputChan chan<- string) error {
	if rm == nil || !rm.IsConnected() {
		return errors.New("purge 仅能在 type: remote 任务步骤中使用（需已连接 SSH）")
	}
	spec, err := parsePurgeArgs(c)
	if err != nil {
		return err
	}
	cmd := buildRemotePurgeShell(spec)
	if spec.OlderThanDays > 0 {
		utils.SendOutput(outputChan, fmt.Sprintf("清理目录 %s 中早于 %d 天的文件（始终保留最新一份）…", spec.Dir, spec.OlderThanDays))
	} else {
		utils.SendOutput(outputChan, fmt.Sprintf("删除 %d 个远端路径…", len(spec.Files)))
	}
	if err := runRemoteCommand(rm, cmd, outputChan); err != nil {
		return fmt.Errorf("远端清理失败: %w", err)
	}
	return nil
}

type purgeSpec struct {
	Files         []string
	Dir           string
	OlderThanDays int
}

func parsePurgeArgs(c []string) (purgeSpec, error) {
	var spec purgeSpec
	if len(c) < 2 {
		return spec, errors.New("参数错误: purge <远端文件1> [远端文件2 ...] 或 purge <目录> --older-than <天数>")
	}
	var paths []string
	for i := 1; i < len(c); i++ {
		tok := strings.TrimSpace(c[i])
		if tok == "" {
			continue
		}
		lower := strings.ToLower(tok)
		if lower == "--older-than" || lower == "--days" {
			if i+1 >= len(c) {
				return spec, errors.New("参数错误: --older-than 需要天数（正整数）")
			}
			n, err := strconv.Atoi(strings.TrimSpace(c[i+1]))
			if err != nil || n < 1 {
				return spec, errors.New("参数错误: 天数须为正整数")
			}
			spec.OlderThanDays = n
			i++
			continue
		}
		paths = append(paths, posixClean(tok))
	}
	if spec.OlderThanDays > 0 {
		if len(paths) != 1 {
			return spec, errors.New("参数错误: 按天数清理时只能指定一个目录")
		}
		if err := validateRemotePurgePath(paths[0], true); err != nil {
			return spec, err
		}
		spec.Dir = paths[0]
		return spec, nil
	}
	if len(paths) == 0 {
		return spec, errors.New("参数错误: 至少指定一个要删除的远端路径")
	}
	for _, p := range paths {
		if err := validateRemotePurgePath(p, false); err != nil {
			return spec, err
		}
	}
	spec.Files = paths
	return spec, nil
}

func validateRemotePurgePath(p string, dirOnly bool) error {
	if p == "" || p == "/" {
		return errors.New("禁止对根路径 / 执行 purge")
	}
	if !dirOnly {
		return nil
	}
	wide := []string{"/root", "/home", "/var", "/opt", "/bin", "/sbin", "/lib", "/lib64", "/usr", "/etc", "/boot", "/dev", "/proc", "/sys", "/run"}
	for _, b := range wide {
		if p == b {
			return fmt.Errorf("目录过宽，请指定更具体的子目录（如备份目录 back）")
		}
	}
	return nil
}

func buildRemotePurgeShell(spec purgeSpec) string {
	if spec.OlderThanDays > 0 {
		dirQ := shellSingleQuote(spec.Dir)
		days := spec.OlderThanDays
		// 删除早于 N 天的文件，但目录内至少保留 mtime 最新的一份（避免备份目录被清空）
		return fmt.Sprintf(
			`set -e; DIR=%s; DAYS=%d; `+
				`NEWEST=$(find "$DIR" -mindepth 1 -type f -printf '%%T@ %%p\n' 2>/dev/null | sort -nr | head -1 | cut -d' ' -f2-); `+
				`if [ -z "$NEWEST" ]; then echo "目录无文件，跳过: $DIR"; exit 0; fi; `+
				`find "$DIR" -mindepth 1 -type f -mtime +"$DAYS" -print0 | while IFS= read -r -d '' f; do `+
				`if [ "$f" = "$NEWEST" ]; then echo "保留最新: $f"; continue; fi; rm -fv "$f"; done; `+
				`echo "清理完成（早于 %d 天，且始终保留最新: $NEWEST）: $DIR"`,
			dirQ, days, days,
		)
	}
	var quoted []string
	for _, f := range spec.Files {
		quoted = append(quoted, shellSingleQuote(f))
	}
	return fmt.Sprintf(`set -e; rm -fv %s`, strings.Join(quoted, " "))
}
