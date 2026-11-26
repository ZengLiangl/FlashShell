package cmds

import (
	"strings"
	"testing"

	"FlashDock/define"
)

// 诊断目标机器上的应用容器异常：拉日志、校验 jar、查看 compose。
// 运行（需先设置 FLASHDOCK_TEST_MACHINE 与下列可选项）：
//
//	FLASHDOCK_E2E_REMOTE_DIR / FLASHDOCK_E2E_CONTAINER 见 deploy 测试
//	go test ./cmds -run TestTajAdminDiagJZ -count=1 -v -timeout 3m
func TestTajAdminDiagJZ(t *testing.T) {
	if testing.Short() {
		t.Skip("skip diag in -short")
	}
	rm := connectJZ(t)
	defer rm.Close()

	run := func(title, cmd string) {
		t.Log("=== " + title + " ===")
		out, err := runRemoteCommandCapture(rm, cmd)
		if err != nil {
			t.Logf("ERR: %v\n%s", err, out)
			return
		}
		t.Log(out)
	}

	run("docker ps "+tajAdminContainer, "docker ps -a --filter name="+tajAdminContainer+" --format 'table {{.Names}}\t{{.Status}}\t{{.Image}}'")
	run("docker inspect state", "docker inspect -f 'Status={{.State.Status}} ExitCode={{.State.ExitCode}} Error={{.State.Error}} Started={{.State.StartedAt}} Finished={{.State.FinishedAt}}' "+tajAdminContainer+" 2>/dev/null")
	run("docker logs tail", "docker logs --tail 80 "+tajAdminContainer+" 2>&1")
	run("remote jar md5", "md5sum "+shellSingleQuote(tajAdminRemoteMainJar)+" 2>/dev/null; ls -la "+shellSingleQuote(tajAdminRemoteMainJar))
	run("remote lib", "find "+shellSingleQuote(tajAdminRemoteLib)+" -type f -printf '%f %s\\n' 2>/dev/null | head -20; echo '--- count:'; find "+shellSingleQuote(tajAdminRemoteLib)+" -type f | wc -l; echo '--- spring-core:'; find "+shellSingleQuote(tajAdminRemoteLib)+" -name 'spring-core-*.jar' 2>/dev/null")
	run("local lib count", "echo local target/lib should have many jars; target/jar only has the module jar")
	run("jar integrity", "python3 -c "+shellSingleQuote("import zipfile; z=zipfile.ZipFile("+tajAdminRemoteMainJar+"); z.testzip(); print('zip ok', len(z.namelist()), 'entries')")+" 2>&1 || unzip -t "+shellSingleQuote(tajAdminRemoteMainJar)+" 2>&1 | tail -5")
	run("compose service", "grep -A30 "+tajAdminContainer+" "+shellSingleQuote(e2eRemoteDir+"/docker-compose.yaml")+" 2>/dev/null | head -40")
}

func runRemoteCommandCapture(rm *define.RemoteMachine, cmd string) (string, error) {
	session, err := rm.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	out, err := session.CombinedOutput(cmd)
	return strings.TrimSpace(string(out)), err
}
