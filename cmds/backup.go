package cmds

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"FlashDock/define"
	"FlashDock/utils"
)

func RegBackupCmd() {
	CmdManager.RegSpecialCmd("backup", doBackup)
}

func doBackup(rm *define.RemoteMachine, c []string, outputChan chan<- string) error {
	if rm == nil || !rm.IsConnected() {
		return errors.New("backup 仅能在 type: remote 任务步骤中使用（需已连接 SSH）")
	}
	spec, err := parseBackupArgs(c)
	if err != nil {
		return err
	}
	cmd, extractDirs := buildRemoteBackupShell(spec)
	utils.SendOutput(outputChan, fmt.Sprintf("开始远端备份 → %s/%s-<时间戳>.tar.gz", spec.BackDir, spec.NamePrefix))
	if err := runRemoteCommand(rm, cmd, outputChan); err != nil {
		return fmt.Errorf("远端备份失败: %w", err)
	}
	utils.SendOutput(outputChan, backupRollbackHint(extractDirs))
	return nil
}

type backupSpec struct {
	BackDir    string
	NamePrefix string
	Sources    []string
}

func parseBackupArgs(c []string) (backupSpec, error) {
	var spec backupSpec
	if len(c) < 4 {
		return spec, errors.New("参数错误: backup <备份目录> <名称前缀> <远端路径1> [远端路径2 ...]")
	}
	spec.BackDir = strings.TrimSpace(c[1])
	spec.NamePrefix = strings.TrimSpace(c[2])
	if spec.BackDir == "" || spec.NamePrefix == "" {
		return spec, errors.New("参数错误: 备份目录与名称前缀不能为空")
	}
	for i := 3; i < len(c); i++ {
		p := strings.TrimSpace(c[i])
		if p == "" {
			continue
		}
		spec.Sources = append(spec.Sources, posixClean(p))
	}
	if len(spec.Sources) == 0 {
		return spec, errors.New("参数错误: 至少指定一个远端路径")
	}
	return spec, nil
}

func posixClean(p string) string {
	p = strings.TrimSpace(p)
	p = strings.ReplaceAll(p, "\\", "/")
	return path.Clean(p)
}

// groupBackupTarSources 按父目录分组，供 tar -C 使用（避免绝对路径触发 Removing leading '/' 提示）。
func groupBackupTarSources(sources []string) map[string][]string {
	grouped := make(map[string][]string)
	for _, src := range sources {
		src = posixClean(src)
		if src == "/" || src == "." {
			continue
		}
		dir := path.Dir(src)
		member := path.Base(src)
		if dir == "." || member == "." {
			continue
		}
		members := grouped[dir]
		if !containsString(members, member) {
			grouped[dir] = append(members, member)
		}
	}
	return grouped
}

func containsString(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func buildRemoteBackupShell(spec backupSpec) (shell string, extractDirs []string) {
	backQ := shellSingleQuote(spec.BackDir)
	prefixQ := shellSingleQuote(spec.NamePrefix)
	grouped := groupBackupTarSources(spec.Sources)
	dirs := make([]string, 0, len(grouped))
	for d := range grouped {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)

	var tarParts []string
	for _, dir := range dirs {
		members := grouped[dir]
		sort.Strings(members)
		tarParts = append(tarParts, "-C", shellSingleQuote(dir))
		for _, m := range members {
			tarParts = append(tarParts, shellSingleQuote(m))
		}
	}

	body := fmt.Sprintf(
		`set -e; mkdir -p %s; STAMP=$(date +%%Y-%%m-%%d-%%H%%M%%S); ARCHIVE=$(printf '%%s/%%s-%%s.tar.gz' %s %s "$STAMP"); tar czf "$ARCHIVE" %s; ls -lh "$ARCHIVE"; echo "备份完成: $ARCHIVE"`,
		backQ,
		backQ,
		prefixQ,
		strings.Join(tarParts, " "),
	)
	return body, dirs
}

func backupRollbackHint(extractDirs []string) string {
	switch len(extractDirs) {
	case 0:
		return "提示: 回滚可在服务器执行 tar xzf <备份包> -C <应用目录>"
	case 1:
		return fmt.Sprintf("提示: 回滚可在服务器执行 tar xzf <备份包> -C %s", shellSingleQuote(extractDirs[0]))
	default:
		var parts []string
		for _, d := range extractDirs {
			parts = append(parts, shellSingleQuote(d))
		}
		return fmt.Sprintf("提示: 回滚按目录解压，例如 tar xzf <备份包> -C %s（其余目录: %s）", parts[0], strings.Join(parts[1:], ", "))
	}
}
