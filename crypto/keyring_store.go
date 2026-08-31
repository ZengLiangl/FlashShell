package crypto

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/zalando/go-keyring"
	"gopkg.in/yaml.v3"
)

func (v *vault) metaPath() string {
	return filepath.Join(v.home(), vaultMetaFile)
}

func (v *vault) fallbackPath() string {
	return filepath.Join(v.home(), secretsFallbackFile)
}

func (v *vault) loadMetaLocked() metaFile {
	var m metaFile
	b, err := os.ReadFile(v.metaPath())
	if err != nil {
		return m
	}
	_ = yaml.Unmarshal(b, &m)
	return m
}

func (v *vault) saveMetaLocked() error {
	m := metaFile{
		HasMasterPassword: v.hasMaster,
		IdleLockMinutes:   v.idleMinutes,
	}
	b, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(v.metaPath(), b, 0600)
}

type fallbackDoc struct {
	DEK        string `json:"dek,omitempty"`
	WrappedDEK string `json:"wrappedDek,omitempty"`
}

func forceFileKeyring() bool {
	return strings.TrimSpace(os.Getenv("FLASH_SHELL_FORCE_FILE_KEYRING")) != ""
}

// writeKeyring 对齐 Reeve：优先只写 OS keyring；成功则删除磁盘回落，避免 DEK 双份落盘。
// 仅当 keyring 不可用（或测试强制）时才写 secrets.enc。
func (v *vault) writeKeyring(user, value string) error {
	if !forceFileKeyring() {
		if err := keyring.Set(keyringService, user, value); err == nil {
			v.backend = "keyring"
			_ = v.purgeFallbackFile()
			return nil
		}
	}
	v.backend = "file"
	return v.writeFallback(user, value)
}

func (v *vault) readKeyring(user string) (value, backend string, err error) {
	if !forceFileKeyring() {
		val, e := keyring.Get(keyringService, user)
		if e == nil && val != "" {
			// 历史版本曾双写 secrets.enc：读到 keyring 后清掉磁盘副本
			_ = v.purgeFallbackFile()
			return val, "keyring", nil
		}
	}
	val, e := v.readFallback(user)
	if e != nil {
		return "", "file", e
	}
	return val, "file", nil
}

func (v *vault) deleteKeyring(user string) error {
	if !forceFileKeyring() {
		_ = keyring.Delete(keyringService, user)
	}
	return v.deleteFallback(user)
}

// purgeFallbackFile 删除可还原 DEK 的磁盘回落（keyring 可用时必须清）
func (v *vault) purgeFallbackFile() error {
	path := v.fallbackPath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (v *vault) readFallback(user string) (string, error) {
	doc, err := v.loadFallbackDoc()
	if err != nil {
		return "", err
	}
	var enc string
	switch user {
	case keyringDEK:
		enc = doc.DEK
	case keyringWrappedDEK:
		enc = doc.WrappedDEK
	}
	if enc == "" {
		return "", fmt.Errorf("fallback empty")
	}
	plain, err := openFallbackBlob(enc)
	if err != nil {
		return "", err
	}
	return plain, nil
}

func (v *vault) writeFallback(user, value string) error {
	doc, _ := v.loadFallbackDoc()
	enc, err := sealFallbackBlob(value)
	if err != nil {
		return err
	}
	switch user {
	case keyringDEK:
		doc.DEK = enc
		doc.WrappedDEK = ""
	case keyringWrappedDEK:
		doc.WrappedDEK = enc
		doc.DEK = ""
	}
	b, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	return os.WriteFile(v.fallbackPath(), b, 0600)
}

func (v *vault) deleteFallback(user string) error {
	doc, err := v.loadFallbackDoc()
	if err != nil {
		return nil
	}
	switch user {
	case keyringDEK:
		doc.DEK = ""
	case keyringWrappedDEK:
		doc.WrappedDEK = ""
	}
	if doc.DEK == "" && doc.WrappedDEK == "" {
		return v.purgeFallbackFile()
	}
	b, _ := json.Marshal(doc)
	return os.WriteFile(v.fallbackPath(), b, 0600)
}

func (v *vault) loadFallbackDoc() (fallbackDoc, error) {
	var doc fallbackDoc
	b, err := os.ReadFile(v.fallbackPath())
	if err != nil {
		return doc, err
	}
	_ = json.Unmarshal(b, &doc)
	return doc, nil
}

func machineFallbackKey() []byte {
	host, _ := os.Hostname()
	sum := sha256.Sum256([]byte("flashshell-dek-fallback|" + host + "|" + runtime.GOOS + "|" + runtime.GOARCH))
	return sum[:]
}

func sealFallbackBlob(plain string) (string, error) {
	ct, err := gcmSeal(machineFallbackKey(), []byte(plain))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ct), nil
}

func openFallbackBlob(enc string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	pt, err := gcmOpen(machineFallbackKey(), raw)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}
