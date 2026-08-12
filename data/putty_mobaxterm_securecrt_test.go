package data

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParsePuttyRegContent(t *testing.T) {
	text := strings.Join([]string{
		`Windows Registry Editor Version 5.00`,
		``,
		`[HKEY_CURRENT_USER\Software\SimonTatham\PuTTY\Sessions\prod%20web]`,
		`"HostName"="10.0.0.1"`,
		`"UserName"="deploy"`,
		`"PortNumber"=dword:000008ae`,
		`"Protocol"="ssh"`,
		``,
		`[HKEY_CURRENT_USER\Software\SimonTatham\PuTTY\Sessions\serial-box]`,
		`"HostName"="com1"`,
		`"Protocol"="serial"`,
	}, "\n")

	sessions, err := ParsePuttyRegContent(text)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	if sessions[0].Name != "prod web" || sessions[0].Host != "10.0.0.1" || sessions[0].Port != 2222 || sessions[0].User != "deploy" {
		t.Fatalf("unexpected session: %+v", sessions[0])
	}
}

func TestParseMobaXtermContent(t *testing.T) {
	sessionValue := `#109#0%10.0.0.20%2222%deploy%%-1%-1%%%%%0%0%0%%%-1%0%0%0%%1080%%0%0%1%#MobaFont%10%0%0%-1#0# #-1`
	text := strings.Join([]string{
		`[Bookmarks]`,
		`SubRep=`,
		`ImgNum=42`,
		`root-server=#109#0%root.example.com%22%<default>%%-1%-1%%%%%0%0%0%%%-1%0%0%0%%1080%%0%0%1%#MobaFont%10%0%0%-1#0# #-1`,
		``,
		`[Bookmarks_1]`,
		`SubRep=Production\Linux`,
		`ImgNum=41`,
		`web-server=` + sessionValue,
	}, "\n")

	sessions, warnings, err := ParseMobaXtermContent(text)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	found := map[string]ParsedMobaXtermSession{}
	for _, s := range sessions {
		found[s.Name] = s
	}
	root := found["root-server"]
	if root.Host != "root.example.com" || root.User != "" || root.Port != 22 {
		t.Fatalf("root-server: %+v", root)
	}
	web := found["web-server"]
	if web.Host != "10.0.0.20" || web.Port != 2222 || web.User != "deploy" || web.Group != "Production/Linux" {
		t.Fatalf("web-server: %+v", web)
	}
}

func TestParseMobaXtermLegacyPathGroup(t *testing.T) {
	text := "[Bookmarks]\nLegacy\\server=deploy@legacy.example.com:2222#ssh\n"
	sessions, warnings, err := ParseMobaXtermContent(text)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	s := sessions[0]
	if s.Name != "server" || s.Host != "legacy.example.com" || s.Port != 2222 || s.User != "deploy" || s.Group != "Legacy" {
		t.Fatalf("unexpected: %+v", s)
	}
}

func TestParseSecureCRTContent(t *testing.T) {
	text := strings.Join([]string{
		`S:"Hostname"=secure.example.com`,
		`S:"Username"=operator`,
		`S:"Protocol Name"=SSH2`,
		`D:"[SSH2] Port"=000008ae`,
	}, "\n")
	sessions, err := ParseSecureCRTContent(text, "Secure Host")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	s := sessions[0]
	if s.Name != "Secure Host" || s.Host != "secure.example.com" || s.User != "operator" || s.Port != 2222 {
		t.Fatalf("unexpected: %+v", s)
	}
}

func TestParseSecureCRTSSH1Port(t *testing.T) {
	text := strings.Join([]string{
		`S:"Hostname"=ssh1.example.com`,
		`S:"Protocol Name"=SSH1`,
		`D:"[SSH1] Port"=000008af`,
		`D:"[SSH2] Port"=00000016`,
	}, "\n")
	sessions, err := ParseSecureCRTContent(text, "ssh1")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if sessions[0].Port != 2223 {
		t.Fatalf("expected port 2223, got %d", sessions[0].Port)
	}
}

func TestSecureCrtGroupFromRel(t *testing.T) {
	if g := secureCrtGroupFromRel(filepath.ToSlash(`Production/Linux/host.ini`)); g != "Production/Linux" {
		t.Fatalf("got %q", g)
	}
	if g := secureCrtGroupFromRel(filepath.ToSlash(`Sessions/Prod/host.ini`)); g != "Prod" {
		t.Fatalf("got %q", g)
	}
	if g := secureCrtGroupFromRel("host.ini"); g != "" {
		t.Fatalf("got %q", g)
	}
}

func TestMobaXtermDecodeGB18030(t *testing.T) {
	// "中文" in GB18030
	label := []byte{0xd6, 0xd0, 0xce, 0xc4}
	data := append([]byte("[Bookmarks]\nSubRep=\nImgNum=42\n"), label...)
	data = append(data, []byte("=#109#0%10.0.0.1%22%root\n")...)
	text := decodeMobaXtermBytes(data)
	sessions, _, err := ParseMobaXtermContent(text)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(sessions) != 1 || sessions[0].Name != "中文" {
		t.Fatalf("expected Chinese label, got %+v", sessions)
	}
}
