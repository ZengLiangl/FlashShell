package crypto

import (
	"encoding/base64"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "flashshell-crypto-*")
	if err != nil {
		panic(err)
	}
	SetConfigHomeFunc(func() string { return dir })
	_ = os.Setenv("FLASH_SHELL_FORCE_FILE_KEYRING", "1")
	if err := InitVault(); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func TestEncryptDecryptSensitiveData(t *testing.T) {
	testData := &SensitiveData{
		Name:     "测试服务器",
		Host:     "your-server.com",
		Port:     22,
		Username: "root",
		Password: "secret",
	}
	encryptedStr, err := EncryptSensitiveData(testData)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	if encryptedStr == "" {
		t.Fatal("加密结果为空")
	}
	decryptedData, err := DecryptSensitiveData(encryptedStr)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}
	if decryptedData.Password != testData.Password {
		t.Errorf("密码不匹配")
	}
	if decryptedData.Host != testData.Host {
		t.Errorf("主机不匹配")
	}
}

func TestEncryptDecryptEmptyData(t *testing.T) {
	_, err := EncryptSensitiveData(nil)
	if err == nil {
		t.Fatal("空数据加密应该失败")
	}
	decryptedData, err := DecryptSensitiveData("")
	if err != nil {
		t.Fatalf("空字符串解密失败: %v", err)
	}
	if decryptedData == nil {
		t.Fatal("空字符串解密应该返回空结构体")
	}
}

func TestEncryptDecryptInvalidData(t *testing.T) {
	_, err := DecryptSensitiveData("invalid_base64_data!!!")
	if err == nil {
		t.Fatal("无效数据解密应该失败")
	}
}

func TestMasterPasswordWrap(t *testing.T) {
	st := GetStatus()
	if !st.Unlocked {
		t.Fatal("应已解锁")
	}
	if err := SetMasterPassword("TestPass-w0rd!", "TestPass-w0rd!"); err != nil {
		t.Fatalf("启用主密码: %v", err)
	}
	Lock()
	if !IsLocked() {
		t.Fatal("锁定后应 IsLocked")
	}
	if err := Unlock("wrong"); err == nil {
		t.Fatal("错误密码应失败")
	}
	if err := Unlock("TestPass-w0rd!"); err != nil {
		t.Fatalf("正确密码解锁: %v", err)
	}
	if err := DisableMasterPassword("TestPass-w0rd!"); err != nil {
		t.Fatalf("关闭主密码: %v", err)
	}
	if GetStatus().HasMasterPassword {
		t.Fatal("应已关闭主密码")
	}
}

func TestWriteKeyringDoesNotDualWriteWhenKeyringOK(t *testing.T) {
	// 本测试在 FORCE_FILE 下只能验证文件后端；非强制时由集成环境验证 purge
	if forceFileKeyring() {
		st := GetStatus()
		if st.KeyringBackend != "file" {
			t.Fatalf("force file backend expected, got %q", st.KeyringBackend)
		}
	}
}

func TestLegacyMigrate(t *testing.T) {
	ct, err := encryptBytes([]byte(`{"name":"x","host":"h","port":22,"username":"u","password":"p"}`), legacyKey)
	if err != nil {
		t.Fatal(err)
	}
	enc := base64.StdEncoding.EncodeToString(ct)
	out, ok, err := MigrateLegacyCiphertext(enc)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("应迁移")
	}
	data, err := DecryptSensitiveData(out)
	if err != nil {
		t.Fatal(err)
	}
	if data.Password != "p" {
		t.Fatalf("密码 %q", data.Password)
	}
}
