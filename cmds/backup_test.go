package cmds

import (
	"strings"
	"testing"
)

func TestParseBackupArgs(t *testing.T) {
	spec, err := parseBackupArgs([]string{
		"backup",
		"/opt/example-app/back",
		"example-app",
		"/opt/example-app/lib",
		"/opt/example-app/example-app.jar",
	})
	if err != nil {
		t.Fatal(err)
	}
	if spec.BackDir != "/opt/example-app/back" || spec.NamePrefix != "example-app" || len(spec.Sources) != 2 {
		t.Fatalf("spec: %+v", spec)
	}
}

func TestBuildRemoteBackupShell(t *testing.T) {
	cmd, dirs := buildRemoteBackupShell(backupSpec{
		BackDir:    "/opt/example-app/back",
		NamePrefix: "example-app",
		Sources:    []string{"/opt/example-app/lib", "/opt/example-app/example-app.jar"},
	})
	if !strings.Contains(cmd, "tar czf") || !strings.Contains(cmd, "date +%Y-%m-%d-%H%M%S") {
		t.Fatalf("unexpected shell: %s", cmd)
	}
	if !strings.Contains(cmd, "-C '/opt/example-app'") {
		t.Fatalf("expected -C parent dir, got: %s", cmd)
	}
	if !strings.Contains(cmd, "'lib'") || !strings.Contains(cmd, "'example-app.jar'") {
		t.Fatalf("expected relative members, got: %s", cmd)
	}
	if strings.Contains(cmd, "'/opt/example-app/lib'") {
		t.Fatalf("should not pass absolute paths to tar: %s", cmd)
	}
	if len(dirs) != 1 || dirs[0] != "/opt/example-app" {
		t.Fatalf("extract dirs: %v", dirs)
	}
}

func TestBackupRollbackHint(t *testing.T) {
	h := backupRollbackHint([]string{"/opt/example-app"})
	if !strings.Contains(h, "-C '/opt/example-app'") {
		t.Fatalf("hint: %s", h)
	}
}
