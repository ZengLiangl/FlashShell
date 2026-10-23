package cmds

import (
	"strings"
	"testing"
)

func TestParseBackupArgs(t *testing.T) {
	spec, err := parseBackupArgs([]string{
		"backup",
		"/root/app/taj-admin/back",
		"taj-admin",
		"/root/app/taj-admin/lib",
		"/root/app/taj-admin/taj-admin.jar",
	})
	if err != nil {
		t.Fatal(err)
	}
	if spec.BackDir != "/root/app/taj-admin/back" || spec.NamePrefix != "taj-admin" || len(spec.Sources) != 2 {
		t.Fatalf("spec: %+v", spec)
	}
}

func TestBuildRemoteBackupShell(t *testing.T) {
	cmd, dirs := buildRemoteBackupShell(backupSpec{
		BackDir:    "/root/app/taj-admin/back",
		NamePrefix: "taj-admin",
		Sources:    []string{"/root/app/taj-admin/lib", "/root/app/taj-admin/taj-admin.jar"},
	})
	if !strings.Contains(cmd, "tar czf") || !strings.Contains(cmd, "date +%Y-%m-%d-%H%M%S") {
		t.Fatalf("unexpected shell: %s", cmd)
	}
	if !strings.Contains(cmd, "-C '/root/app/taj-admin'") {
		t.Fatalf("expected -C parent dir, got: %s", cmd)
	}
	if !strings.Contains(cmd, "'lib'") || !strings.Contains(cmd, "'taj-admin.jar'") {
		t.Fatalf("expected relative members, got: %s", cmd)
	}
	if strings.Contains(cmd, "'/root/app/taj-admin/lib'") {
		t.Fatalf("should not pass absolute paths to tar: %s", cmd)
	}
	if len(dirs) != 1 || dirs[0] != "/root/app/taj-admin" {
		t.Fatalf("extract dirs: %v", dirs)
	}
}

func TestBackupRollbackHint(t *testing.T) {
	h := backupRollbackHint([]string{"/root/app/taj_plus"})
	if !strings.Contains(h, "-C '/root/app/taj_plus'") {
		t.Fatalf("hint: %s", h)
	}
}
