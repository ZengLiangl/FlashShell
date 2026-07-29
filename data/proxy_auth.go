package data

import "FlashDock/crypto"

// SetProxyPassword 加密并写入代理密码；空字符串清除。
func (p *ProxySettings) SetProxyPassword(password string) error {
	if p == nil {
		return nil
	}
	p.Password = password
	if password == "" {
		p.EncryptedPassword = ""
		return nil
	}
	encrypted, err := crypto.EncryptSensitiveData(&crypto.SensitiveData{
		Name:     "proxy",
		Username: p.User,
		Password: password,
	})
	if err != nil {
		return err
	}
	p.EncryptedPassword = encrypted
	return nil
}

// GetProxyPassword 读取明文代理密码（优先内存缓存，其次解密落盘字段）。
func (p *ProxySettings) GetProxyPassword() (string, error) {
	if p == nil {
		return "", nil
	}
	if p.Password != "" {
		return p.Password, nil
	}
	if p.EncryptedPassword == "" {
		return "", nil
	}
	data, err := crypto.DecryptSensitiveData(p.EncryptedPassword)
	if err != nil {
		return "", err
	}
	p.Password = data.Password
	return p.Password, nil
}

// HydrateProxyPassword 解密密码到 Password，供前端表单展示。
func (p *ProxySettings) HydrateProxyPassword() {
	if p == nil {
		return
	}
	pw, err := p.GetProxyPassword()
	if err != nil {
		p.Password = ""
		return
	}
	p.Password = pw
}

// PrepareProxyPasswordForSave 将 Password 加密写入 EncryptedPassword。
func (p *ProxySettings) PrepareProxyPasswordForSave() error {
	if p == nil {
		return nil
	}
	return p.SetProxyPassword(p.Password)
}
