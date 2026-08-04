package machine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"FlashDock/define"

	"golang.org/x/crypto/ssh"
)

// defaultKeyPreference 无显式密钥时扫描 ~/.ssh/id_* 的偏好顺序
var defaultKeyPreference = []string{
	"id_ed25519",
	"id_ecdsa",
	"id_rsa",
}

func buildSSHAuth(machine *define.Machine, sensitive *define.SensitiveData) ([]ssh.AuthMethod, error) {
	var auth []ssh.AuthMethod
	if machine.KeyFile != "" {
		key, err := loadPrivateKey(machine.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("加载私钥失败: %w", err)
		}
		auth = append(auth, ssh.PublicKeys(key))
	} else {
		for _, signer := range loadDefaultPrivateKeys() {
			auth = append(auth, ssh.PublicKeys(signer))
		}
	}
	if sensitive != nil && sensitive.Password != "" {
		auth = append(auth, ssh.Password(sensitive.Password))
	}
	if len(auth) == 0 {
		return nil, fmt.Errorf("未配置认证方式")
	}
	return auth, nil
}

func loadDefaultPrivateKeys() []ssh.Signer {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	sshDir := filepath.Join(home, ".ssh")
	var signers []ssh.Signer
	for _, name := range defaultKeyPreference {
		path := filepath.Join(sshDir, name)
		key, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			continue
		}
		signers = append(signers, signer)
	}
	return signers
}

func loadPrivateKey(keyPath string) (ssh.Signer, error) {
	if strings.HasPrefix(keyPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		keyPath = filepath.Join(homeDir, keyPath[2:])
	}
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(key)
}
