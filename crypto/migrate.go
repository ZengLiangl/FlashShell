package crypto

import (
	"encoding/json"
	"strings"
)

// MigrateLegacyCiphertext 若密文仍为遗留硬编码密钥所加密，则用当前 DEK 重加密。
// 解不开或已是当前 DEK 则原样返回。
func MigrateLegacyCiphertext(enc string) (string, bool, error) {
	if strings.TrimSpace(enc) == "" {
		return enc, false, nil
	}
	key, err := requireDEK()
	if err != nil {
		return enc, false, err
	}
	// 已能用当前 DEK 解密 → 无需迁移
	if _, err := decryptWithKey(enc, key); err == nil {
		return enc, false, nil
	}
	if data, err := decryptSensitiveWith(enc, key); err == nil && data != nil {
		return enc, false, nil
	}
	// 试遗留密钥（SensitiveData）
	if data, err := decryptSensitiveWith(enc, legacyKey); err == nil {
		out, err := EncryptSensitiveData(data)
		return out, true, err
	}
	// 试遗留密钥（纯文本）
	if plain, err := decryptWithKey(enc, legacyKey); err == nil {
		// 若是 SensitiveData JSON，走结构体加密保持格式
		var sd SensitiveData
		if json.Unmarshal([]byte(plain), &sd) == nil && (sd.Host != "" || sd.Username != "" || sd.Password != "") {
			out, err := EncryptSensitiveData(&sd)
			return out, true, err
		}
		out, err := EncryptText(plain)
		return out, true, err
	}
	return enc, false, nil
}
