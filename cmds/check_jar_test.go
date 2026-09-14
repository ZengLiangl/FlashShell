package cmds

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func writeTestJar(t *testing.T, dir string, files map[string]string) string {
	t.Helper()
	path := filepath.Join(dir, "app.jar")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCheckJarRejectsMissingMainClass(t *testing.T) {
	dir := t.TempDir()
	jar := writeTestJar(t, dir, map[string]string{
		"BOOT-INF/classes/com/taj/web/Foo.class": "class",
		"BOOT-INF/classes/application.yml":       "spring:\n  profiles:\n    active: dev\n",
	})
	_, err := CheckJar(checkJarSpec{
		JarPath:    jar,
		Classes:    []string{"com.taj.DromaraApplication"},
		MinClasses: 1,
	})
	if err == nil {
		t.Fatal("want missing main class")
	}
}

func TestCheckJarRejectsPlaceholderAndSmallJar(t *testing.T) {
	dir := t.TempDir()
	jar := writeTestJar(t, dir, map[string]string{
		"BOOT-INF/classes/com/taj/DromaraApplication.class": "class-bytes",
		"BOOT-INF/classes/application.yml":                  "spring:\n  profiles:\n    active: @profiles.active@\n",
	})
	_, err := CheckJar(checkJarSpec{
		JarPath: jar,
		Classes: []string{"com.taj.DromaraApplication"},
	})
	if err == nil {
		t.Fatal("want placeholder error")
	}
	_, err = CheckJar(checkJarSpec{
		JarPath:    jar,
		Classes:    []string{"com.taj.DromaraApplication"},
		MinBytes:   1 << 20,
		AllowPlaceholder: true,
	})
	if err == nil {
		t.Fatal("want min-bytes error")
	}
}

func TestCheckJarOK(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"BOOT-INF/classes/com/taj/DromaraApplication.class": "class-bytes",
		"BOOT-INF/classes/application.yml":                  "spring:\n  profiles:\n    active: dev\n",
	}
	for i := 0; i < 8; i++ {
		files["BOOT-INF/classes/com/taj/C"+string(rune('A'+i))+".class"] = "x"
	}
	jar := writeTestJar(t, dir, files)
	msg, err := CheckJar(checkJarSpec{
		JarPath:    jar,
		Classes:    []string{"com.taj.DromaraApplication"},
		MinClasses: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if msg == "" {
		t.Fatal("want success message")
	}
}

func TestParseCheckJarArgs(t *testing.T) {
	spec, err := parseCheckJarArgs([]string{
		"check-jar", `D:\a\taj-admin.jar`,
		"--class", "com.taj.DromaraApplication",
		"--min-bytes", "6000000",
		"--min-classes", "800",
	})
	if err != nil {
		t.Fatal(err)
	}
	if spec.JarPath != `D:\a\taj-admin.jar` || spec.MinBytes != 6000000 || spec.MinClasses != 800 {
		t.Fatalf("parsed %+v", spec)
	}
	if len(spec.Classes) != 1 || spec.Classes[0] != "com.taj.DromaraApplication" {
		t.Fatalf("classes %+v", spec.Classes)
	}
}
