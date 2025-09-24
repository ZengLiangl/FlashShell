package crypto

import (
	"testing"
)

func TestEncryptDecryptSensitiveData(t *testing.T) {
	// 测试数据
	testData := &SensitiveData{
		Name:     "测试服务器",
		Host:     "101.200.180.32",
		Port:     22,
		Username: "root",
	}

	// 测试加密
	encryptedStr, err := EncryptSensitiveData(testData)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	if encryptedStr == "" {
		t.Fatal("加密结果为空")
	}

	t.Logf("加密结果: %s", encryptedStr)

	// 测试解密
	decryptedData, err := DecryptSensitiveData(encryptedStr)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}

	// 验证解密结果
	if decryptedData.Name != testData.Name {
		t.Errorf("名称不匹配: 期望 %s, 实际 %s", testData.Name, decryptedData.Name)
	}

	if decryptedData.Host != testData.Host {
		t.Errorf("主机地址不匹配: 期望 %s, 实际 %s", testData.Host, decryptedData.Host)
	}

	if decryptedData.Port != testData.Port {
		t.Errorf("端口不匹配: 期望 %d, 实际 %d", testData.Port, decryptedData.Port)
	}

	if decryptedData.Username != testData.Username {
		t.Errorf("用户名不匹配: 期望 %s, 实际 %s", testData.Username, decryptedData.Username)
	}

	if decryptedData.Password != testData.Password {
		t.Errorf("密码不匹配: 期望 %s, 实际 %s", testData.Password, decryptedData.Password)
	}

	if string(decryptedData.KeyData) != string(testData.KeyData) {
		t.Errorf("密钥数据不匹配: 期望 %s, 实际 %s", string(testData.KeyData), string(decryptedData.KeyData))
	}

	t.Log("加密解密测试通过")
}

func TestEncryptDecryptEmptyData(t *testing.T) {
	// 测试空数据
	_, err := EncryptSensitiveData(nil)
	if err == nil {
		t.Fatal("空数据加密应该失败")
	}

	// 测试空字符串解密
	decryptedData, err := DecryptSensitiveData("")
	if err != nil {
		t.Fatalf("空字符串解密失败: %v", err)
	}

	if decryptedData == nil {
		t.Fatal("空字符串解密应该返回空结构体")
	}

	t.Log("空数据处理测试通过")
}

func TestEncryptDecryptInvalidData(t *testing.T) {
	// 测试无效的加密数据
	_, err := DecryptSensitiveData("invalid_base64_data")
	if err == nil {
		t.Fatal("无效数据解密应该失败")
	}

	// 测试无效的Base64数据
	_, err = DecryptSensitiveData("not_base64_string")
	if err == nil {
		t.Fatal("无效Base64数据解密应该失败")
	}

	t.Log("无效数据处理测试通过")
}
