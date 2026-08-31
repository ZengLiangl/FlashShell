package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
)

// EncryptText 加密任意文本，返回 base64(nonce||ciphertext+tag)
func EncryptText(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	key, err := requireDEK()
	if err != nil {
		return "", err
	}
	return encryptWithKey(plain, key)
}

// DecryptText 解密；优先当前 DEK，失败则尝试遗留硬编码密钥（便于迁移读）
func DecryptText(encryptedStr string) (string, error) {
	if encryptedStr == "" {
		return "", nil
	}
	key, err := requireDEK()
	if err != nil {
		// 锁定时仍允许用遗留密钥只读？否——锁定应全面拒绝凭据访问
		return "", err
	}
	plain, err := decryptWithKey(encryptedStr, key)
	if err == nil {
		return plain, nil
	}
	// 迁移期：旧密文
	if p, e2 := decryptWithKey(encryptedStr, legacyKey); e2 == nil {
		return p, nil
	}
	return "", err
}

// EncryptSensitiveData 加密敏感数据
func EncryptSensitiveData(data *SensitiveData) (string, error) {
	if data == nil {
		return "", ErrEmptyData
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("序列化数据失败: %v", err)
	}
	key, err := requireDEK()
	if err != nil {
		return "", err
	}
	ciphertext, err := encryptBytes(jsonData, key)
	if err != nil {
		return "", fmt.Errorf("加密失败: %v", err)
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptSensitiveData 解密敏感数据（支持遗留密钥）
func DecryptSensitiveData(encryptedStr string) (*SensitiveData, error) {
	if encryptedStr == "" {
		return &SensitiveData{}, nil
	}
	key, err := requireDEK()
	if err != nil {
		return nil, err
	}
	data, err := decryptSensitiveWith(encryptedStr, key)
	if err == nil {
		return data, nil
	}
	if data, e2 := decryptSensitiveWith(encryptedStr, legacyKey); e2 == nil {
		return data, nil
	}
	return nil, err
}

func decryptSensitiveWith(encryptedStr string, key []byte) (*SensitiveData, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedStr)
	if err != nil {
		return nil, fmt.Errorf("%w: base64解码失败: %v", ErrInvalidData, err)
	}
	plaintext, err := decryptBytes(ciphertext, key)
	if err != nil {
		return nil, fmt.Errorf("解密失败: %v", err)
	}
	data := &SensitiveData{}
	if err := json.Unmarshal(plaintext, data); err != nil {
		return nil, fmt.Errorf("%w: JSON解析失败: %v", ErrInvalidData, err)
	}
	return data, nil
}

func encryptWithKey(plain string, key []byte) (string, error) {
	ct, err := encryptBytes([]byte(plain), key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ct), nil
}

func decryptWithKey(encryptedStr string, key []byte) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedStr)
	if err != nil {
		return "", fmt.Errorf("%w: base64解码失败: %v", ErrInvalidData, err)
	}
	plaintext, err := decryptBytes(ciphertext, key)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func encryptBytes(data, key []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, ErrEmptyData
	}
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, ErrInvalidKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidKey, err)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建GCM失败: %v", err)
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("生成随机数失败: %v", err)
	}
	return aesGCM.Seal(nonce, nonce, data, nil), nil
}

func decryptBytes(ciphertext, key []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, ErrEmptyData
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidKey, err)
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建GCM失败: %v", err)
	}
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("%w: 需要至少 %d 字节", ErrDataTooShort, nonceSize)
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptFailed, err)
	}
	return plaintext, nil
}

// ReencryptCiphertext 用旧解密函数解出明文再以新加密（迁移/重置用）
func ReencryptCiphertext(enc string, oldDecrypt, newEncrypt func(string) (string, error)) (string, error) {
	if enc == "" {
		return "", nil
	}
	plain, err := oldDecrypt(enc)
	if err != nil {
		// 尝试当前 DecryptText / 遗留
		plain, err = DecryptText(enc)
		if err != nil {
			data, err2 := DecryptSensitiveData(enc)
			if err2 != nil {
				return enc, nil // 解不开则原样保留
			}
			b, _ := json.Marshal(data)
			plain = string(b)
			out, err3 := newEncrypt(plain)
			return out, err3
		}
	}
	// plain 可能是 JSON SensitiveData 或任意文本
	if looksLikeSensitiveJSON(plain) {
		return newEncrypt(plain)
	}
	return newEncrypt(plain)
}

func looksLikeSensitiveJSON(s string) bool {
	var m map[string]any
	return json.Unmarshal([]byte(s), &m) == nil && (m["password"] != nil || m["host"] != nil || m["username"] != nil)
}

// IsLocked 是否处于锁定（仅主密码模式；基础模式可从 keyring 静默恢复 DEK）
func IsLocked() bool {
	globalVault.mu.RLock()
	defer globalVault.mu.RUnlock()
	if !globalVault.hasMaster {
		return false
	}
	return !globalVault.unlocked
}
