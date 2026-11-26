package cmds

import (
	"strings"
	"testing"
)

func TestParsePurgeArgsFiles(t *testing.T) {
	spec, err := parsePurgeArgs([]string{"purge", "/opt/example-app/back/a.tar.gz", "/opt/example-app/back/b.tar.gz"})
	if err != nil {
		t.Fatal(err)
	}
	if spec.OlderThanDays != 0 || len(spec.Files) != 2 {
		t.Fatalf("%+v", spec)
	}
}

func TestParsePurgeArgsOlderThan(t *testing.T) {
	spec, err := parsePurgeArgs([]string{"purge", "/opt/example-app/back", "--older-than", "30"})
	if err != nil {
		t.Fatal(err)
	}
	if spec.Dir != "/opt/example-app/back" || spec.OlderThanDays != 30 {
		t.Fatalf("%+v", spec)
	}
}

func TestBuildRemotePurgeShell(t *testing.T) {
	rm := buildRemotePurgeShell(purgeSpec{Dir: "/opt/example-app/back", OlderThanDays: 7})
	if !strings.Contains(rm, "DIR='/opt/example-app/back'") || !strings.Contains(rm, `-mtime +"$DAYS"`) || !strings.Contains(rm, "NEWEST") {
		t.Fatalf("%s", rm)
	}
	f := buildRemotePurgeShell(purgeSpec{Files: []string{"/tmp/a.jar"}})
	if !strings.Contains(f, "rm -fv '/tmp/a.jar'") {
		t.Fatalf("%s", f)
	}
}

func TestValidateRemotePurgePathBlocksWideDir(t *testing.T) {
	if err := validateRemotePurgePath("/root", true); err == nil {
		t.Fatal("expected error for /root age purge")
	}
}
