package data

import (
	"fmt"

	"FlashDock/crypto"

	"github.com/google/uuid"
)

// GlobalAccount 全局 SSH 帐号
type GlobalAccount struct {
	ID                string `yaml:"id" json:"id"`
	Name              string `yaml:"name" json:"name"`
	User              string `yaml:"user" json:"user"`
	EncryptedPassword string `yaml:"encrypted_password,omitempty" json:"encrypted_password,omitempty"`
	password          string `yaml:"-" json:"-"`
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

// GlobalAccountDTO 返回给前端的帐号信息（含密码，用于填充表单）
type GlobalAccountDTO struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
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
		result = append(result, GlobalAccountDTO{
			ID:       account.ID,
			Name:     account.Name,
			User:     account.User,
			Password: password,
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
