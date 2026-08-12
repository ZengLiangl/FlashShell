package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

var (
	// 加密密钥 - 32字节的密钥
	key = []byte("McCALaLSmfHtNIfnGSTFlrUmOkhD+Exd")

	// 错误定义
	ErrEmptyData     = errors.New("数据为空")
	ErrInvalidKey    = errors.New("无效的加密密钥")
	ErrInvalidData   = errors.New("无效的数据格式")
	ErrDataTooShort  = errors.New("数据长度不足")
	ErrDecryptFailed = errors.New("解密失败")
)

// SensitiveData 敏感信息结构体
type SensitiveData struct {
	Name     string `json:"name"`     // 名称
	Host     string `json:"host"`     // 主机地址
	Port     int    `json:"port"`     // 端口
	Username string `json:"username"` // 用户名
	Password string `json:"password"` // 密码
	// KeyPassphrase 私钥口令（可选）
	KeyPassphrase string `json:"key_passphrase,omitempty"`
	KeyData       []byte `json:"key_data"` // 密钥文件内容
}

// 加密数据
func encrypt(data []byte) ([]byte, error) {
	if len(data) == 0 {
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

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("生成随机数失败: %v", err)
	}

	return aesGCM.Seal(nonce, nonce, data, nil), nil
}

// 解密数据
func decrypt(ciphertext []byte) ([]byte, error) {
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

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptFailed, err)
	}

	return plaintext, nil
}

// EncryptSensitiveData 加密敏感数据
func EncryptSensitiveData(data *SensitiveData) (string, error) {
	if data == nil {
		return "", ErrEmptyData
	}

	// 序列化敏感数据
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("序列化数据失败: %v", err)
	}

	// 加密
	ciphertext, err := encrypt(jsonData)
	if err != nil {
		return "", fmt.Errorf("加密失败: %v", err)
	}

	// 转换为base64
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptSensitiveData 解密敏感数据
func DecryptSensitiveData(encryptedStr string) (*SensitiveData, error) {
	if encryptedStr == "" {
		return &SensitiveData{}, nil
	}

	// base64解码
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedStr)
	if err != nil {
		return nil, fmt.Errorf("%w: base64解码失败: %v", ErrInvalidData, err)
	}

	// 解密
	plaintext, err := decrypt(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("解密失败: %v", err)
	}

	// 反序列化
	data := &SensitiveData{}
	if err := json.Unmarshal(plaintext, data); err != nil {
		return nil, fmt.Errorf("%w: JSON解析失败: %v", ErrInvalidData, err)
	}

	return data, nil
}
