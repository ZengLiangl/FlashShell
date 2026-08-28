package machine

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type dirFileMeta struct {
	size int64
}

// collectLocalDirFiles 收集本地目录下所有常规文件的相对路径与大小。
func collectLocalDirFiles(root string) (map[string]dirFileMeta, error) {
	root = filepath.Clean(root)
	out := make(map[string]dirFileMeta)
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = dirFileMeta{size: info.Size()}
		return nil
	})
	return out, err
}

// collectRemoteDirFiles 收集远端目录下所有文件的相对路径与大小。
func (a *ShellAuxManager) collectRemoteDirFiles(root string) (map[string]dirFileMeta, error) {
	root = path.Clean(strings.TrimSpace(root))
	out := make(map[string]dirFileMeta)
	var walk func(string, string) error
	walk = func(base, rel string) error {
		dir := base
		if rel != "" {
			dir = path.Join(base, rel)
		}
		entries, err := a.ListDir(dir, false)
		if err != nil {
			return err
		}
		for _, e := range entries {
			childRel := e.Name
			if rel != "" {
				childRel = path.Join(rel, e.Name)
			}
			if e.IsDir {
				if err := walk(base, childRel); err != nil {
					return err
				}
				continue
			}
			out[childRel] = dirFileMeta{size: e.Size}
		}
		return nil
	}
	if err := walk(root, ""); err != nil {
		return nil, err
	}
	return out, nil
}

// PruneRemoteDirToMirror 删除远端目录中本地不存在的文件，使远端与本地目录结构一致。
func (a *ShellAuxManager) PruneRemoteDirToMirror(localDir, remoteDir string) error {
	localFiles, err := collectLocalDirFiles(localDir)
	if err != nil {
		return fmt.Errorf("读取本地目录失败: %w", err)
	}
	remoteFiles, err := a.collectRemoteDirFiles(remoteDir)
	if err != nil {
		return fmt.Errorf("读取远端目录失败: %w", err)
	}
	localSet := make(map[string]struct{}, len(localFiles))
	for rel := range localFiles {
		localSet[rel] = struct{}{}
	}
	for rel := range remoteFiles {
		if _, ok := localSet[rel]; ok {
			continue
		}
		target := path.Join(remoteDir, rel)
		if err := a.RemovePathReliable(target); err != nil {
			return fmt.Errorf("清理多余远端文件 %s 失败: %w", target, err)
		}
	}
	return a.pruneEmptyRemoteDirs(remoteDir, localSet)
}

func (a *ShellAuxManager) pruneEmptyRemoteDirs(remoteDir string, localFiles map[string]struct{}) error {
	remoteFiles, err := a.collectRemoteDirFiles(remoteDir)
	if err != nil {
		return err
	}
	dirSet := make(map[string]struct{})
	for rel := range remoteFiles {
		if _, ok := localFiles[rel]; ok {
			continue
		}
		dir := path.Dir(rel)
		for dir != "." && dir != "/" && dir != "" {
			dirSet[dir] = struct{}{}
			dir = path.Dir(dir)
		}
	}
	for rel := range localFiles {
		dir := path.Dir(rel)
		for dir != "." && dir != "/" && dir != "" {
			delete(dirSet, dir)
			dir = path.Dir(dir)
		}
	}
	if len(dirSet) == 0 {
		return nil
	}
	dirs := make([]string, 0, len(dirSet))
	for d := range dirSet {
		dirs = append(dirs, d)
	}
	for i := 0; i < len(dirs); i++ {
		for j := i + 1; j < len(dirs); j++ {
			if len(dirs[j]) > len(dirs[i]) {
				dirs[i], dirs[j] = dirs[j], dirs[i]
			}
		}
	}
	for _, rel := range dirs {
		target := path.Join(remoteDir, rel)
		entries, err := a.ListDir(target, false)
		if err != nil {
			continue
		}
		if len(entries) == 0 {
			_ = a.RemovePath(target)
		}
	}
	return nil
}

// VerifyRemoteDirMirror 校验远端目录与本地目录的文件清单与大小是否一致。
func (a *ShellAuxManager) VerifyRemoteDirMirror(localDir, remoteDir string) error {
	localFiles, err := collectLocalDirFiles(localDir)
	if err != nil {
		return fmt.Errorf("读取本地目录失败: %w", err)
	}
	remoteFiles, err := a.collectRemoteDirFiles(remoteDir)
	if err != nil {
		return fmt.Errorf("读取远端目录失败: %w", err)
	}
	for rel, lm := range localFiles {
		rm, ok := remoteFiles[rel]
		if !ok {
			return fmt.Errorf("远端缺少文件: %s", rel)
		}
		if rm.size != lm.size {
			return fmt.Errorf("远端文件大小不一致: %s (本地 %d, 远端 %d)", rel, lm.size, rm.size)
		}
	}
	for rel := range remoteFiles {
		if _, ok := localFiles[rel]; !ok {
			return fmt.Errorf("远端存在多余文件: %s", rel)
		}
	}
	return nil
}
