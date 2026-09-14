package cmds

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"FlashDock/define"
	"FlashDock/utils"
)

func RegCheckJarCmd() {
	CmdManager.RegSpecialCmd("check-jar", doCheckJar)
}

func doCheckJar(_ *define.RemoteMachine, c []string, outputChan chan<- string) error {
	spec, err := parseCheckJarArgs(c)
	if err != nil {
		return err
	}
	result, err := CheckJar(spec)
	if err != nil {
		return err
	}
	utils.SendOutput(outputChan, result)
	return nil
}

type checkJarSpec struct {
	JarPath           string
	Classes           []string
	MinBytes          int64
	MinClasses        int
	AllowPlaceholder  bool
}

func parseCheckJarArgs(c []string) (checkJarSpec, error) {
	var spec checkJarSpec
	if len(c) < 2 {
		return spec, errors.New("参数错误: check-jar <jar路径> [--class 全限定名] [--min-bytes N] [--min-classes N] [--allow-placeholder]")
	}
	spec.JarPath = c[1]
	for i := 2; i < len(c); i++ {
		switch c[i] {
		case "--class":
			if i+1 >= len(c) {
				return spec, errors.New("参数错误: --class 需要全限定类名")
			}
			i++
			name := strings.TrimSpace(c[i])
			if name == "" {
				return spec, errors.New("参数错误: --class 不能为空")
			}
			spec.Classes = append(spec.Classes, name)
		case "--min-bytes":
			if i+1 >= len(c) {
				return spec, errors.New("参数错误: --min-bytes 需要数字")
			}
			i++
			n, err := strconv.ParseInt(c[i], 10, 64)
			if err != nil || n < 0 {
				return spec, fmt.Errorf("参数错误: --min-bytes 无效: %s", c[i])
			}
			spec.MinBytes = n
		case "--min-classes":
			if i+1 >= len(c) {
				return spec, errors.New("参数错误: --min-classes 需要数字")
			}
			i++
			n, err := strconv.Atoi(c[i])
			if err != nil || n < 0 {
				return spec, fmt.Errorf("参数错误: --min-classes 无效: %s", c[i])
			}
			spec.MinClasses = n
		case "--allow-placeholder":
			spec.AllowPlaceholder = true
		default:
			return spec, fmt.Errorf("参数错误: 未知选项 %s", c[i])
		}
	}
	return spec, nil
}

func classEntryCandidates(fqcn string) []string {
	rel := strings.ReplaceAll(strings.TrimSpace(fqcn), ".", "/") + ".class"
	rel = strings.TrimPrefix(rel, "/")
	return []string{
		"BOOT-INF/classes/" + rel,
		"WEB-INF/classes/" + rel,
		rel,
	}
}

func isConfigEntry(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".properties")
}

func looksLikeMavenPlaceholder(s string) bool {
	for {
		start := strings.IndexByte(s, '@')
		if start < 0 {
			return false
		}
		rest := s[start+1:]
		end := strings.IndexByte(rest, '@')
		if end <= 0 {
			return false
		}
		token := rest[:end]
		if isMavenPlaceholderToken(token) {
			return true
		}
		s = rest[end:]
	}
}

func isMavenPlaceholderToken(token string) bool {
	if len(token) == 0 || len(token) > 80 {
		return false
	}
	for i, r := range token {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		case i > 0 && ((r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-'):
		default:
			return false
		}
	}
	return true
}

// CheckJar 校验本地 Spring 瘦包/普通 jar：主类、体积、class 数量、未过滤的 Maven 占位符。
func CheckJar(spec checkJarSpec) (string, error) {
	st, err := os.Stat(spec.JarPath)
	if err != nil {
		return "", fmt.Errorf("找不到 jar: %w", err)
	}
	if st.IsDir() {
		return "", fmt.Errorf("不是文件: %s", spec.JarPath)
	}
	if spec.MinBytes > 0 && st.Size() < spec.MinBytes {
		return "", fmt.Errorf("jar 过小: %d 字节（阈值 %d），多半是缺 class 的坏包", st.Size(), spec.MinBytes)
	}

	zr, err := zip.OpenReader(spec.JarPath)
	if err != nil {
		return "", fmt.Errorf("不是有效 jar/zip: %w", err)
	}
	defer zr.Close()

	classCount := 0
	found := make([]bool, len(spec.Classes))
	candidates := make([][]string, len(spec.Classes))
	for i, fqcn := range spec.Classes {
		candidates[i] = classEntryCandidates(fqcn)
	}

	var placeholderHits []string
	for _, f := range zr.File {
		name := strings.TrimPrefix(f.Name, "/")
		if strings.HasSuffix(strings.ToLower(name), ".class") && !strings.HasSuffix(name, "/") {
			classCount++
			for i, cands := range candidates {
				if found[i] {
					continue
				}
				for _, cand := range cands {
					if name == cand {
						if f.UncompressedSize64 == 0 {
							return "", fmt.Errorf("主类文件为空: %s", name)
						}
						found[i] = true
					}
				}
			}
		}
		if spec.AllowPlaceholder || !isConfigEntry(name) || f.UncompressedSize64 == 0 {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("读取 %s 失败: %w", name, err)
		}
		data, err := io.ReadAll(io.LimitReader(rc, 1<<20))
		_ = rc.Close()
		if err != nil {
			return "", fmt.Errorf("读取 %s 失败: %w", name, err)
		}
		text := string(data)
		if !utf8.ValidString(text) {
			continue
		}
		if looksLikeMavenPlaceholder(text) {
			placeholderHits = append(placeholderHits, name)
		}
	}

	if spec.MinClasses > 0 && classCount < spec.MinClasses {
		return "", fmt.Errorf("class 数量过少: %d（阈值 %d），主类或业务类可能没打进包", classCount, spec.MinClasses)
	}
	for i, fqcn := range spec.Classes {
		if !found[i] {
			return "", fmt.Errorf("jar 缺少主类 %s（已扫 %d 个 class）", fqcn, classCount)
		}
	}
	if len(placeholderHits) > 0 {
		return "", fmt.Errorf("配置未过滤 Maven 占位符: %s", strings.Join(placeholderHits, ", "))
	}

	msg := fmt.Sprintf("jar 校验通过: %s（%d 字节, %d 个 class）", spec.JarPath, st.Size(), classCount)
	return msg, nil
}
