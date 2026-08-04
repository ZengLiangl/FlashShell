package machine

import (
	"fmt"
	"time"

	"FlashDock/define"

	"golang.org/x/crypto/ssh"
)

// legacyKex 旧设备常用密钥交换
var legacyKex = []string{
	"diffie-hellman-group-exchange-sha256",
	"diffie-hellman-group-exchange-sha1",
	"diffie-hellman-group14-sha1",
	"diffie-hellman-group14-sha256",
	"diffie-hellman-group1-sha1",
	"ecdh-sha2-nistp256",
	"ecdh-sha2-nistp384",
	"ecdh-sha2-nistp521",
	"curve25519-sha256",
	"curve25519-sha256@libssh.org",
}

var legacyCiphers = []string{
	"aes128-ctr", "aes192-ctr", "aes256-ctr",
	"aes128-gcm@openssh.com", "aes256-gcm@openssh.com",
	"aes128-cbc", "aes192-cbc", "aes256-cbc",
	"3des-cbc",
}

var legacyMACs = []string{
	"hmac-sha2-256", "hmac-sha2-512",
	"hmac-sha1", "hmac-sha1-96",
	"hmac-md5", "hmac-md5-96",
}

var modernHostKeyAlgorithms = []string{
	"ssh-ed25519",
	"ecdsa-sha2-nistp256",
	"ecdsa-sha2-nistp384",
	"ecdsa-sha2-nistp521",
	"rsa-sha2-512",
	"rsa-sha2-256",
	"ssh-rsa",
}

var legacyHostKeyAlgorithms = []string{
	"ssh-rsa",
	"rsa-sha2-256",
	"rsa-sha2-512",
	"ssh-ed25519",
	"ecdsa-sha2-nistp256",
}

func sshAlgorithmConfig(machine *define.Machine) ssh.Config {
	cfg := ssh.Config{}
	if machine != nil && machine.LegacyAlgorithms {
		cfg.KeyExchanges = append([]string{}, legacyKex...)
		cfg.Ciphers = append([]string{}, legacyCiphers...)
		cfg.MACs = append([]string{}, legacyMACs...)
	}
	return cfg
}

func hostKeyAlgorithmsForMachine(machine *define.Machine) []string {
	if machine == nil {
		return nil
	}
	if machine.LegacyAlgorithms && machine.SkipEcdsaHostKey {
		return append([]string{}, legacyHostKeyAlgorithms...)
	}
	if machine.SkipEcdsaHostKey {
		algs := make([]string, 0, len(modernHostKeyAlgorithms))
		for _, alg := range modernHostKeyAlgorithms {
			if alg == "ecdsa-sha2-nistp256" || alg == "ecdsa-sha2-nistp384" || alg == "ecdsa-sha2-nistp521" {
				continue
			}
			algs = append(algs, alg)
		}
		return algs
	}
	return nil
}

func buildSSHClientConfig(user string, auth []ssh.AuthMethod, machine *define.Machine, timeout time.Duration) *ssh.ClientConfig {
	if timeout <= 0 {
		timeout = define.SSHHandshakeTimeout()
	}
	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: define.HostKeyCallback(),
		Timeout:         timeout,
		Config:          sshAlgorithmConfig(machine),
	}
	if algs := hostKeyAlgorithmsForMachine(machine); len(algs) > 0 {
		cfg.HostKeyAlgorithms = algs
	}
	return cfg
}

func requestSessionAgentForwarding(session *ssh.Session) error {
	ok, err := session.SendRequest("auth-agent-req@openssh.com", true, nil)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("服务端拒绝 Agent 转发")
	}
	return nil
}
