package mcp

import (
	"os"
	"path/filepath"

	"FlashDock/data"
)

const (
	dirName            = "mcp"
	settingsFile       = "settings.yaml"
	tokensFile         = "tokens.yaml"
	auditFile          = "audit.jsonl"
	vaultFile          = "vault.yaml"           // 服务凭据 installed_services
	sensitiveVaultFile = "sensitive_vault.yaml" // 出口脱敏被动捕获
	falsePositivesFile = "false_positives.yaml" // 敏感库误报样本
	sitesFile          = "sites.yaml"
	deploysFile        = "deploys.yaml"
	historyFile        = "deploy_history.yaml"
	runtimeFile        = "runtime.json"
	userSkillsDir      = "skills-user"
	experienceDir      = "experience"
	runbooksDir        = "runbooks"
)

func homeDir() (string, error) {
	root, err := data.ConfigHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, dirName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

func join(elem ...string) string {
	return filepath.Join(elem...)
}

func mustSubdir(name string) string {
	root, err := homeDir()
	if err != nil {
		return ""
	}
	p := filepath.Join(root, name)
	_ = os.MkdirAll(p, 0700)
	return p
}
