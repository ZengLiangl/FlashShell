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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zalando/go-keyring"
	"golang.org/x/crypto/argon2"
)

const (
	keyringService     = "com.flashshell.dek"
	keyringDEK         = "dek"
	keyringWrappedDEK  = "wrapped-dek"
	vaultMetaFile      = "vault_meta.yaml"
	secretsFallbackFile = "secrets.enc"

	argonMemoryKiB = 19 * 1024 // 19 MiB OWASP
	argonTime      = 2
	argonThreads   = 1
	argonKeyLen    = 32
	saltLen        = 16
	dekLen         = 32
)

var (
	ErrEmptyData     = errors.New("数据为空")
	ErrInvalidKey    = errors.New("无效的加密密钥")
	ErrInvalidData   = errors.New("无效的数据格式")
	ErrDataTooShort  = errors.New("数据长度不足")
	ErrDecryptFailed = errors.New("解密失败")
	ErrLocked        = errors.New("凭据库已锁定，请先解锁")
	ErrBadPassword   = errors.New("主密码错误")
	ErrRateLimited   = errors.New("尝试次数过多，请稍后再试")
	ErrNoMaster      = errors.New("未启用主密码")
	ErrHasMaster     = errors.New("已启用主密码")
)

// 遗留硬编码密钥：仅用于从旧版本密文迁移，新加密一律走随机 DEK。
var legacyKey = []byte("McCALaLSmfHtNIfnGSTFlrUmOkhD+Exd")

// SensitiveData 敏感信息结构体
type SensitiveData struct {
	Name          string `json:"name"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	KeyPassphrase string `json:"key_passphrase,omitempty"`
	KeyData       []byte `json:"key_data"`
}

// Status 凭据库状态（给前端）
type Status struct {
	Unlocked           bool   `json:"unlocked"`
	HasMasterPassword  bool   `json:"hasMasterPassword"`
	Mode               string `json:"mode"` // basic | master
	IdleLockMinutes    int    `json:"idleLockMinutes"`
	KeyringBackend     string `json:"keyringBackend"` // keyring | file
	UnlockFailCount    int    `json:"unlockFailCount"`
	UnlockLockedUntil  string `json:"unlockLockedUntil,omitempty"`
	CredentialAlgo     string `json:"credentialAlgo"`
}

type metaFile struct {
	HasMasterPassword bool `yaml:"hasMasterPassword" json:"hasMasterPassword"`
	IdleLockMinutes   int  `yaml:"idleLockMinutes" json:"idleLockMinutes"`
}

type wrappedBlob struct {
	// V1: base64(salt[16] || nonce[12] || ct)
	V1 string `json:"v1"`
}

type vault struct {
	mu sync.RWMutex

	dek            []byte
	unlocked       bool
	hasMaster      bool
	idleMinutes    int
	backend        string
	failCount      int
	lockedUntil    time.Time
	lastActivity   time.Time
	configHomeFunc func() string
}

var globalVault = &vault{
	configHomeFunc: defaultConfigHome,
	lastActivity:   time.Now(),
}

func defaultConfigHome() string {
	if v := strings.TrimSpace(os.Getenv("FLASHSHELL_CONFIG_HOME")); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".flashshell")
}

// SetConfigHomeFunc 测试注入配置目录
func SetConfigHomeFunc(fn func() string) {
	if fn == nil {
		return
	}
	globalVault.mu.Lock()
	globalVault.configHomeFunc = fn
	globalVault.mu.Unlock()
}

func (v *vault) home() string {
	if v.configHomeFunc != nil {
		return v.configHomeFunc()
	}
	return defaultConfigHome()
}

// InitVault 启动时初始化：从 keyring 拉 DEK，或进入锁定，或生成新 DEK 并迁移遗留密文。
func InitVault() error {
	return globalVault.init()
}

func (v *vault) init() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	_ = os.MkdirAll(v.home(), 0700)
	meta := v.loadMetaLocked()
	v.hasMaster = meta.HasMasterPassword
	v.idleMinutes = meta.IdleLockMinutes

	if v.hasMaster {
		wrapped, backend, err := v.readKeyring(keyringWrappedDEK)
		v.backend = backend
		if err != nil || strings.TrimSpace(wrapped) == "" {
			// 元数据说有主密码但 keyring 丢了：保持锁定，等用户重置
			v.unlocked = false
			v.dek = nil
			return nil
		}
		_ = wrapped
		v.unlocked = false
		v.dek = nil
		// 若 Wrapped DEK 仅在磁盘回落，尝试升迁到 keyring
		v.tryPromoteFileToKeyringLocked(keyringWrappedDEK, wrapped, backend)
		return nil
	}

	plain, backend, err := v.readKeyring(keyringDEK)
	v.backend = backend
	if err == nil && strings.TrimSpace(plain) != "" {
		raw, err := base64.StdEncoding.DecodeString(plain)
		if err == nil && len(raw) == dekLen {
			v.dek = raw
			v.unlocked = true
			v.lastActivity = time.Now()
			v.tryPromoteFileToKeyringLocked(keyringDEK, plain, backend)
			return nil
		}
	}

	// 无 DEK：生成新的，写入 keyring，进入基础模式
	dek := make([]byte, dekLen)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return err
	}
	if err := v.writeKeyring(keyringDEK, base64.StdEncoding.EncodeToString(dek)); err != nil {
		return err
	}
	v.dek = dek
	v.unlocked = true
	v.hasMaster = false
	v.lastActivity = time.Now()
	_ = v.saveMetaLocked()
	return nil
}

// GetStatus 当前凭据库状态
func GetStatus() Status {
	return globalVault.status()
}

func (v *vault) status() Status {
	v.mu.RLock()
	defer v.mu.RUnlock()
	st := Status{
		Unlocked:          v.unlocked,
		HasMasterPassword: v.hasMaster,
		Mode:              "basic",
		IdleLockMinutes:   v.idleMinutes,
		KeyringBackend:    v.backend,
		UnlockFailCount:   v.failCount,
		CredentialAlgo:    "AES-256-GCM",
	}
	if v.hasMaster {
		st.Mode = "master"
	}
	if !v.lockedUntil.IsZero() && time.Now().Before(v.lockedUntil) {
		st.UnlockLockedUntil = v.lockedUntil.Format(time.RFC3339)
	}
	if st.KeyringBackend == "" {
		st.KeyringBackend = "file"
	}
	return st
}

// TouchActivity 用户活动，重置空闲计时
func TouchActivity() {
	globalVault.mu.Lock()
	globalVault.lastActivity = time.Now()
	globalVault.mu.Unlock()
}

// CheckIdleLock 若超时则锁定；返回是否刚锁定
func CheckIdleLock() bool {
	return globalVault.checkIdle()
}

func (v *vault) checkIdle() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.unlocked || v.idleMinutes <= 0 || !v.hasMaster {
		return false
	}
	if time.Since(v.lastActivity) < time.Duration(v.idleMinutes)*time.Minute {
		return false
	}
	v.clearDEKLocked()
	return true
}

// Lock 清空内存 DEK（主密码模式才有意义；基础模式也可调用，下次需从 keyring 再拉）
func Lock() {
	globalVault.mu.Lock()
	defer globalVault.mu.Unlock()
	globalVault.clearDEKLocked()
}

func (v *vault) tryPromoteFileToKeyringLocked(user, value, backend string) {
	if backend != "file" || forceFileKeyring() || strings.TrimSpace(value) == "" {
		return
	}
	if err := keyring.Set(keyringService, user, value); err != nil {
		return
	}
	v.backend = "keyring"
	_ = v.purgeFallbackFile()
}

func (v *vault) clearDEKLocked() {
	if len(v.dek) > 0 {
		for i := range v.dek {
			v.dek[i] = 0
		}
	}
	v.dek = nil
	v.unlocked = false
}

// Unlock 主密码解锁；基础模式无主密码时尝试从 keyring 恢复 DEK
func Unlock(masterPassword string) error {
	return globalVault.unlock(masterPassword)
}

func (v *vault) unlock(masterPassword string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if err := v.checkRateLocked(); err != nil {
		return err
	}
	if v.hasMaster {
		wrapped, backend, err := v.readKeyring(keyringWrappedDEK)
		v.backend = backend
		if err != nil || wrapped == "" {
			v.recordFailLocked()
			return ErrBadPassword
		}
		dek, err := unwrapDEK(wrapped, masterPassword)
		if err != nil {
			v.recordFailLocked()
			return ErrBadPassword
		}
		v.dek = dek
		v.unlocked = true
		v.failCount = 0
		v.lockedUntil = time.Time{}
		v.lastActivity = time.Now()
		return nil
	}
	// 基础模式
	plain, backend, err := v.readKeyring(keyringDEK)
	v.backend = backend
	if err != nil || plain == "" {
		return fmt.Errorf("无法从钥匙串读取 DEK: %w", err)
	}
	raw, err := base64.StdEncoding.DecodeString(plain)
	if err != nil || len(raw) != dekLen {
		return ErrInvalidKey
	}
	v.dek = raw
	v.unlocked = true
	v.lastActivity = time.Now()
	return nil
}

func (v *vault) checkRateLocked() error {
	if !v.lockedUntil.IsZero() && time.Now().Before(v.lockedUntil) {
		return ErrRateLimited
	}
	return nil
}

func (v *vault) recordFailLocked() {
	v.failCount++
	switch {
	case v.failCount >= 10:
		v.lockedUntil = time.Now().Add(5 * time.Minute)
	case v.failCount >= 3:
		v.lockedUntil = time.Now().Add(10 * time.Second)
	}
}

// SetMasterPassword 启用主密码：包装现有 DEK
func SetMasterPassword(password, confirm string) error {
	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("主密码不能为空")
	}
	if password != confirm {
		return fmt.Errorf("两次输入的主密码不一致")
	}
	return globalVault.setMaster(password)
}

func (v *vault) setMaster(password string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.hasMaster {
		return ErrHasMaster
	}
	if !v.unlocked || len(v.dek) != dekLen {
		return ErrLocked
	}
	blob, err := wrapDEK(v.dek, password)
	if err != nil {
		return err
	}
	if err := v.writeKeyring(keyringWrappedDEK, blob); err != nil {
		return err
	}
	_ = v.deleteKeyring(keyringDEK)
	v.hasMaster = true
	return v.saveMetaLocked()
}

// ChangeMasterPassword 修改主密码
func ChangeMasterPassword(oldPass, newPass, confirm string) error {
	if newPass != confirm {
		return fmt.Errorf("两次输入的新主密码不一致")
	}
	if strings.TrimSpace(newPass) == "" {
		return fmt.Errorf("新主密码不能为空")
	}
	return globalVault.changeMaster(oldPass, newPass)
}

func (v *vault) changeMaster(oldPass, newPass string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.hasMaster {
		return ErrNoMaster
	}
	if err := v.checkRateLocked(); err != nil {
		return err
	}
	wrapped, _, err := v.readKeyring(keyringWrappedDEK)
	if err != nil || wrapped == "" {
		return ErrBadPassword
	}
	dek, err := unwrapDEK(wrapped, oldPass)
	if err != nil {
		v.recordFailLocked()
		return ErrBadPassword
	}
	blob, err := wrapDEK(dek, newPass)
	if err != nil {
		zero(dek)
		return err
	}
	if err := v.writeKeyring(keyringWrappedDEK, blob); err != nil {
		zero(dek)
		return err
	}
	if v.unlocked {
		zero(v.dek)
		v.dek = dek
	} else {
		zero(dek)
	}
	v.failCount = 0
	v.lockedUntil = time.Time{}
	return nil
}

// DisableMasterPassword 关闭主密码，DEK 明文回 keyring
func DisableMasterPassword(password string) error {
	return globalVault.disableMaster(password)
}

func (v *vault) disableMaster(password string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.hasMaster {
		return ErrNoMaster
	}
	if err := v.checkRateLocked(); err != nil {
		return err
	}
	wrapped, _, err := v.readKeyring(keyringWrappedDEK)
	if err != nil || wrapped == "" {
		v.recordFailLocked()
		return ErrBadPassword
	}
	dek, err := unwrapDEK(wrapped, password)
	if err != nil {
		v.recordFailLocked()
		return ErrBadPassword
	}
	if err := v.writeKeyring(keyringDEK, base64.StdEncoding.EncodeToString(dek)); err != nil {
		zero(dek)
		return err
	}
	_ = v.deleteKeyring(keyringWrappedDEK)
	v.hasMaster = false
	v.dek = dek
	v.unlocked = true
	v.failCount = 0
	v.lockedUntil = time.Time{}
	v.lastActivity = time.Now()
	return v.saveMetaLocked()
}

// SetIdleLockMinutes 空闲自动锁定（仅主密码模式生效）；0=关闭
func SetIdleLockMinutes(minutes int) error {
	if minutes < 0 {
		minutes = 0
	}
	switch minutes {
	case 0, 5, 10, 15, 30, 60:
	default:
		if minutes > 0 && minutes < 5 {
			minutes = 5
		}
	}
	globalVault.mu.Lock()
	defer globalVault.mu.Unlock()
	globalVault.idleMinutes = minutes
	return globalVault.saveMetaLocked()
}

// ResetReencrypt 生成新 DEK 并回调重加密全部凭据
func ResetReencrypt(masterPassword string, reencrypt func(oldDecrypt, newEncrypt func(string) (string, error)) error) error {
	return globalVault.resetReencrypt(masterPassword, reencrypt)
}

func (v *vault) resetReencrypt(masterPassword string, reencrypt func(oldDecrypt, newEncrypt func(string) (string, error)) error) error {
	v.mu.Lock()
	if v.hasMaster {
		if err := v.checkRateLocked(); err != nil {
			v.mu.Unlock()
			return err
		}
		wrapped, _, err := v.readKeyring(keyringWrappedDEK)
		if err != nil || wrapped == "" {
			v.mu.Unlock()
			return ErrBadPassword
		}
		oldDEK, err := unwrapDEK(wrapped, masterPassword)
		if err != nil {
			v.recordFailLocked()
			v.mu.Unlock()
			return ErrBadPassword
		}
		zero(v.dek)
		v.dek = oldDEK
		v.unlocked = true
	} else if !v.unlocked || len(v.dek) != dekLen {
		v.mu.Unlock()
		return ErrLocked
	}
	oldDEK := append([]byte(nil), v.dek...)
	newDEK := make([]byte, dekLen)
	if _, err := io.ReadFull(rand.Reader, newDEK); err != nil {
		v.mu.Unlock()
		return err
	}
	v.mu.Unlock()

	oldDecrypt := func(s string) (string, error) {
		return decryptWithKey(s, oldDEK)
	}
	newEncrypt := func(s string) (string, error) {
		return encryptWithKey(s, newDEK)
	}
	if err := reencrypt(oldDecrypt, newEncrypt); err != nil {
		zero(oldDEK)
		zero(newDEK)
		return err
	}

	v.mu.Lock()
	defer v.mu.Unlock()
	zero(v.dek)
	v.dek = append([]byte(nil), newDEK...)
	zero(newDEK)
	zero(oldDEK)
	v.unlocked = true
	v.lastActivity = time.Now()
	if v.hasMaster {
		blob, err := wrapDEK(v.dek, masterPassword)
		if err != nil {
			return err
		}
		return v.writeKeyring(keyringWrappedDEK, blob)
	}
	return v.writeKeyring(keyringDEK, base64.StdEncoding.EncodeToString(v.dek))
}

// ResetWipeSecrets 清空 DEK 相关密钥材料并生成新 DEK（调用方负责清凭据字段）
func ResetWipeSecrets(masterPassword string) error {
	return globalVault.resetWipe(masterPassword)
}

// ResetForgotMasterPassword 忘记主密码：不校验旧密码，销毁 Wrapped DEK，生成新 DEK 并回到基础模式。
// 调用方必须先清空全部凭据密文；此操作不可恢复。
func ResetForgotMasterPassword() error {
	return globalVault.resetForgotMaster()
}

func (v *vault) resetForgotMaster() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	dek := make([]byte, dekLen)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return err
	}
	v.clearDEKLocked()
	v.dek = dek
	v.unlocked = true
	v.hasMaster = false
	v.failCount = 0
	v.lockedUntil = time.Time{}
	v.lastActivity = time.Now()
	_ = v.deleteKeyring(keyringWrappedDEK)
	_ = v.deleteKeyring(keyringDEK)
	if err := v.writeKeyring(keyringDEK, base64.StdEncoding.EncodeToString(dek)); err != nil {
		return err
	}
	return v.saveMetaLocked()
}

func (v *vault) resetWipe(masterPassword string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.hasMaster {
		if err := v.checkRateLocked(); err != nil {
			return err
		}
		wrapped, _, err := v.readKeyring(keyringWrappedDEK)
		if err != nil || wrapped == "" {
			return ErrBadPassword
		}
		if _, err := unwrapDEK(wrapped, masterPassword); err != nil {
			v.recordFailLocked()
			return ErrBadPassword
		}
	}
	dek := make([]byte, dekLen)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return err
	}
	v.clearDEKLocked()
	v.dek = dek
	v.unlocked = true
	v.hasMaster = false
	v.failCount = 0
	v.lockedUntil = time.Time{}
	v.lastActivity = time.Now()
	_ = v.deleteKeyring(keyringWrappedDEK)
	if err := v.writeKeyring(keyringDEK, base64.StdEncoding.EncodeToString(dek)); err != nil {
		return err
	}
	return v.saveMetaLocked()
}

func requireDEK() ([]byte, error) {
	return globalVault.requireDEK()
}

func (v *vault) requireDEK() ([]byte, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.unlocked && len(v.dek) == dekLen {
		return v.dek, nil
	}
	// 基础模式：尝试静默从 keyring 恢复
	if !v.hasMaster {
		plain, backend, err := v.readKeyring(keyringDEK)
		v.backend = backend
		if err == nil && plain != "" {
			raw, err := base64.StdEncoding.DecodeString(plain)
			if err == nil && len(raw) == dekLen {
				v.dek = raw
				v.unlocked = true
				v.lastActivity = time.Now()
				return v.dek, nil
			}
		}
	}
	return nil, ErrLocked
}

func wrapDEK(dek []byte, password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}
	kek := argon2.IDKey([]byte(password), salt, argonTime, argonMemoryKiB, argonThreads, argonKeyLen)
	ct, err := gcmSeal(kek, dek)
	zero(kek)
	if err != nil {
		return "", err
	}
	blob := append(append([]byte{}, salt...), ct...)
	payload, _ := json.Marshal(wrappedBlob{V1: base64.StdEncoding.EncodeToString(blob)})
	return string(payload), nil
}

func unwrapDEK(blobJSON, password string) ([]byte, error) {
	var wb wrappedBlob
	if err := json.Unmarshal([]byte(blobJSON), &wb); err != nil {
		return nil, ErrBadPassword
	}
	raw, err := base64.StdEncoding.DecodeString(wb.V1)
	if err != nil || len(raw) < saltLen+12+16 {
		return nil, ErrBadPassword
	}
	salt, rest := raw[:saltLen], raw[saltLen:]
	kek := argon2.IDKey([]byte(password), salt, argonTime, argonMemoryKiB, argonThreads, argonKeyLen)
	dek, err := gcmOpen(kek, rest)
	zero(kek)
	if err != nil || len(dek) != dekLen {
		return nil, ErrBadPassword
	}
	return dek, nil
}

func gcmSeal(key, plain []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plain, nil), nil
}

func gcmOpen(key, nonceAndCT []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(nonceAndCT) < ns {
		return nil, ErrDataTooShort
	}
	nonce, ct := nonceAndCT[:ns], nonceAndCT[ns:]
	return gcm.Open(nil, nonce, ct, nil)
}

func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}