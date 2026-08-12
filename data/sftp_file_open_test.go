package data

import "testing"

func TestNormalizeSftpDefaultOpener(t *testing.T) {
	cases := map[string]string{
		"":               "ask",
		"ask":            "ask",
		"builtin-editor": "builtin-editor",
		"system-app":     "system-app",
		"other":          "ask",
	}
	for in, want := range cases {
		if got := NormalizeSftpDefaultOpener(in); got != want {
			t.Fatalf("NormalizeSftpDefaultOpener(%q)=%q want %q", in, got, want)
		}
	}
}

func TestNormalizeSftpFileExtension(t *testing.T) {
	cases := map[string]string{
		"":     "file",
		".":    "file",
		".go":  "go",
		"GO":   "go",
		" md ": "md",
	}
	for in, want := range cases {
		if got := NormalizeSftpFileExtension(in); got != want {
			t.Fatalf("NormalizeSftpFileExtension(%q)=%q want %q", in, got, want)
		}
	}
}

func TestNormalizeSftpFileAssociations(t *testing.T) {
	got := NormalizeSftpFileAssociations(map[string]SftpFileAssociation{
		".go": {OpenerType: "builtin-editor"},
		"md": {
			OpenerType: "system-app",
			SystemApp:  &SftpSystemApp{Path: "/Applications/Visual Studio Code.app", Name: "Visual Studio Code"},
		},
		"bin": {OpenerType: "system-app"}, // missing app → drop
		"x":   {OpenerType: "invalid"},
	})
	if len(got) != 2 {
		t.Fatalf("len=%d want 2: %#v", len(got), got)
	}
	if got["go"].OpenerType != "builtin-editor" {
		t.Fatalf("go assoc: %#v", got["go"])
	}
	if got["md"].SystemApp == nil || got["md"].SystemApp.Path == "" {
		t.Fatalf("md assoc: %#v", got["md"])
	}
}

func TestSftpAutoSyncEnabled(t *testing.T) {
	if !SftpAutoSyncEnabled(nil) {
		t.Fatal("nil config should default true")
	}
	off := false
	if SftpAutoSyncEnabled(&GlobalConfig{SftpAutoSync: &off}) {
		t.Fatal("explicit false should be false")
	}
}
