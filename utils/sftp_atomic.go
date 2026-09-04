package utils

import (
	"fmt"
	"path"
	"strings"
)

const remoteUploadPartSuffix = ".flashdock.part"

// RemoteUploadPartPath 返回远端原子上传的隐藏暂存路径（与目标同目录）。
func RemoteUploadPartPath(remotePath string) string {
	dir := path.Dir(remotePath)
	base := path.Base(remotePath)
	if dir == "" || dir == "." {
		return "." + base + remoteUploadPartSuffix
	}
	return path.Join(dir, "."+base+remoteUploadPartSuffix)
}

func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

// RemoteAtomicUnzipCandidates 远端解压命令候选：先解到 staging，再 mv -f 覆盖目标文件，
// 避免 unzip/extractall 原地截断正在被 Docker/JVM 打开的 jar。
func RemoteAtomicUnzipCandidates(remoteZip, targetDir string) []string {
	tq := shellSingleQuote(targetDir)
	zq := shellSingleQuote(remoteZip)
	staging := strings.TrimRight(targetDir, "/") + ".flashdock.extract"
	sq := shellSingleQuote(staging)
	promote := fmt.Sprintf(
		`find %s -type f -print0 | while IFS= read -r -d '' f; do rel="${f#%s/}"; mkdir -p %s/"$(dirname "$rel")"; mv -f "$f" %s/"$rel"; done && rm -rf %s && rm -f %s`,
		sq, staging, tq, tq, sq, zq,
	)
	return []string{
		fmt.Sprintf("rm -rf %s && mkdir -p %s %s && unzip -o %s -d %s && %s", sq, sq, tq, zq, sq, promote),
		fmt.Sprintf("rm -rf %s && mkdir -p %s %s && busybox unzip -o %s -d %s && %s", sq, sq, tq, zq, sq, promote),
		fmt.Sprintf(
			"rm -rf %s && mkdir -p %s %s && python3 -c %s && %s",
			sq, sq, tq,
			shellSingleQuote(fmt.Sprintf("import zipfile; zipfile.ZipFile(%q).extractall(%q)", remoteZip, staging)),
			promote,
		),
		fmt.Sprintf(
			"rm -rf %s && mkdir -p %s %s && python -c %s && %s",
			sq, sq, tq,
			shellSingleQuote(fmt.Sprintf("import zipfile; zipfile.ZipFile(%q).extractall(%q)", remoteZip, staging)),
			promote,
		),
	}
}
