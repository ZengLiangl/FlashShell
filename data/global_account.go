package data

import (
	"fmt"

	"FlashDock/crypto"

	"github.com/google/uuid"
)

// GlobalAccount 全局 SSH 帐号
type GlobalAccount struct {
	ID                    string `yaml:"id" json:"id"`
	Name                  string `yaml:"name" json:"name"`
	User                  string `yaml:"user" json:"user"`
	KeyFile               string `yaml:"keyFile,omitempty" json:"keyFile,omitempty"`
	EncryptedPassword     string `yaml:"encrypted_password,omitempty" json:"encrypted_password,omitempty"`
	EncryptedKeyPassphrase string `yaml:"encrypted_key_passphrase,omitempty" json:"encrypted_key_passphrase,omitempty"`
	password              string `yaml:"-" json:"-"`
	keyPassphrase         string `yaml:"-" json:"-"`
}

// EnsureID 确保帐号有唯一 ID
func (a *GlobalAccount) EnsureID() {
	if a != nil && a.ID == "" {
		a.ID = uuid.NewString()
	}
}

// SetPassword 加密并保存密码
func (a *GlobalAccount) SetPassword(password string) error {
	a.password = password
	if password == "" {
		a.EncryptedPassword = ""
		return nil
	}
	encrypted, err := crypto.EncryptSensitiveData(&crypto.SensitiveData{
		Name:     a.Name,
		Username: a.User,
		Password: password,
	})
	if err != nil {
		return err
	}
	a.EncryptedPassword = encrypted
	return nil
}

// GetPassword 获取明文密码
func (a *GlobalAccount) GetPassword() (string, error) {
	if a.password != "" {
		return a.password, nil
	}
	if a.EncryptedPassword == "" {
		return "", nil
	}
	data, err := crypto.DecryptSensitiveData(a.EncryptedPassword)
	if err != nil {
		return "", err
	}
	a.password = data.Password
	return a.password, nil
}

// SetKeyPassphrase 加密并保存私钥口令
func (a *GlobalAccount) SetKeyPassphrase(passphrase string) error {
	a.keyPassphrase = passphrase
	if passphrase == "" {
		a.EncryptedKeyPassphrase = ""
		return nil
	}
	encrypted, err := crypto.EncryptSensitiveData(&crypto.SensitiveData{
		Name:          a.Name,
		Username:      a.User,
		KeyPassphrase: passphrase,
	})
	if err != nil {
		return err
	}
	a.EncryptedKeyPassphrase = encrypted
	return nil
}

// GetKeyPassphrase 获取私钥口令
func (a *GlobalAccount) GetKeyPassphrase() (string, error) {
	if a.keyPassphrase != "" {
		return a.keyPassphrase, nil
	}
	if a.EncryptedKeyPassphrase == "" {
		return "", nil
	}
	data, err := crypto.DecryptSensitiveData(a.EncryptedKeyPassphrase)
	if err != nil {
		return "", err
	}
	a.keyPassphrase = data.KeyPassphrase
	return a.keyPassphrase, nil
}

// GlobalAccountDTO 返回给前端的帐号信息（含密码，用于填充表单）
type GlobalAccountDTO struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	User          string `json:"user"`
	Password      string `json:"password"`
	KeyFile       string `json:"keyFile,omitempty"`
	KeyPassphrase string `json:"keyPassphrase,omitempty"`
}

func (gcm *GlobalConfigManager) ensureGlobalAccountIDs() bool {
	if gcm.config == nil {
		return false
	}
	changed := false
	for i := range gcm.config.GlobalAccounts {
		if gcm.config.GlobalAccounts[i].ID == "" {
			gcm.config.GlobalAccounts[i].EnsureID()
			changed = true
		}
	}
	return changed
}

func (gcm *GlobalConfigManager) GetGlobalAccounts() []GlobalAccountDTO {
	if gcm.config == nil {
		return []GlobalAccountDTO{}
	}
	result := make([]GlobalAccountDTO, 0, len(gcm.config.GlobalAccounts))
	for _, account := range gcm.config.GlobalAccounts {
		password, _ := account.GetPassword()
		passphrase, _ := account.GetKeyPassphrase()
		result = append(result, GlobalAccountDTO{
			ID:            account.ID,
			Name:          account.Name,
			User:          account.User,
			Password:      password,
			KeyFile:       account.KeyFile,
			KeyPassphrase: passphrase,
		})
	}
	return result
}

func (gcm *GlobalConfigManager) SaveGlobalAccounts(accounts []GlobalAccount) error {
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return err
		}
	}
	for i := range accounts {
		accounts[i].EnsureID()
	}
	gcm.config.GlobalAccounts = accounts
	return gcm.SaveGlobalConfig(gcm.config)
}

func (gcm *GlobalConfigManager) GetGlobalAccountByID(id string) *GlobalAccount {
	if gcm.config == nil || id == "" {
		return nil
	}
	for i := range gcm.config.GlobalAccounts {
		if gcm.config.GlobalAccounts[i].ID == id {
			return &gcm.config.GlobalAccounts[i]
		}
	}
	return nil
}

func (gcm *GlobalConfigManager) GetGlobalAccountCredentials(id string) (user, password string, err error) {
	account := gcm.GetGlobalAccountByID(id)
	if account == nil {
		return "", "", fmt.Errorf("未找到全局帐号: %s", id)
	}
	password, err = account.GetPassword()
	if err != nil {
		return "", "", err
	}
	return account.User, password, nil
}
