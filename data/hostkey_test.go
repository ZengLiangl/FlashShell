package data

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestHostKeyCallbackMismatchRevokesAndReturnsUnknown(t *testing.T) {
	IsolateConfigHome(t)
	m := NewHostKeyManager()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sshPub, err := ssh.NewPublicKey(priv.Public())
	if err != nil {
		t.Fatal(err)
	}
	actualFP := FingerprintSHA256(sshPub)

	if err := m.Trust("101.200.180.32", 22, "SHA256:stale"); err != nil {
		t.Fatal(err)
	}
	if !m.IsTrusted("101.200.180.32", 22) {
		t.Fatal("预置信任记录失败")
	}

	cb := m.Callback()
	err = cb("101.200.180.32:22", nil, sshPub)
	hk := ParseHostKeyUnknownError(err)
	if hk == nil {
		t.Fatalf("期望 HostKeyUnknownError，实际: %v", err)
	}
	if hk.Host != "101.200.180.32" || hk.Port != 22 {
		t.Fatalf("主机解析错误: %+v", hk)
	}
	if hk.Fingerprint != actualFP {
		t.Fatalf("指纹应为远端实际值，got %s want %s", hk.Fingerprint, actualFP)
	}
	if m.IsTrusted("101.200.180.32", 22) {
		t.Fatal("冲突后应已删除本地持久记录")
	}

	m.TrustSession("101.200.180.32", 22, "SHA256:stale-session")
	err = cb("101.200.180.32:22", nil, sshPub)
	if ParseHostKeyUnknownError(err) == nil {
		t.Fatalf("会话级冲突也应返回 HostKeyUnknownError，实际: %v", err)
	}
}
