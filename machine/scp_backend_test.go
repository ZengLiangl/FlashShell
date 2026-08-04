package machine

import "testing"

func TestParseLsLa(t *testing.T) {
	out := `total 8
drwxr-xr-x 2 root root 4096 Jan 1 12:00 .
drwxr-xr-x 3 root root 4096 Jan 1 12:00 ..
-rwxr--r-- 1 root root  128 Jan 1 12:00 clean.sh
lrwxrwxrwx 1 root root    9 Jan 1 12:00 link -> clean.sh
drwxr-xr-x 2 root root 4096 Jan 1 12:00 logs
`
	entries := parseLsLa("/tmp", out, false)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d: %+v", len(entries), entries)
	}
	foundLink := false
	for _, e := range entries {
		if e.Name == "link" {
			foundLink = true
			if e.Type != "链接" || e.LinkTarget != "clean.sh" {
				t.Fatalf("link entry: %+v", e)
			}
		}
		if e.Name == "logs" && !e.IsDir {
			t.Fatalf("logs should be dir")
		}
	}
	if !foundLink {
		t.Fatal("missing link entry")
	}
}
