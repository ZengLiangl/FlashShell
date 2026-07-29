package data

import "testing"

func TestProxyPasswordRoundTrip(t *testing.T) {
	p := &ProxySettings{User: "alice"}
	if err := p.SetProxyPassword("s3cret"); err != nil {
		t.Fatalf("SetProxyPassword: %v", err)
	}
	if p.EncryptedPassword == "" {
		t.Fatal("expected encrypted password")
	}
	p.Password = ""
	got, err := p.GetProxyPassword()
	if err != nil {
		t.Fatalf("GetProxyPassword: %v", err)
	}
	if got != "s3cret" {
		t.Fatalf("got %q, want s3cret", got)
	}
}

func TestProxyPasswordClear(t *testing.T) {
	p := &ProxySettings{User: "bob"}
	_ = p.SetProxyPassword("x")
	if err := p.SetProxyPassword(""); err != nil {
		t.Fatal(err)
	}
	if p.EncryptedPassword != "" {
		t.Fatalf("expected cleared encrypted password, got %q", p.EncryptedPassword)
	}
}
