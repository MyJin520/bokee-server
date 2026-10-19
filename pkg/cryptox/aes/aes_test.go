package aes

import (
	"bytes"
	"crypto/aes"
	"testing"
)

// TestAESCommon 通用测试：正常字符串加密解密
func TestAESCommon(t *testing.T) {
	// 测试明文
	plainText := "hello aes-256-cbc 测试数据 123456"
	plainBytes := []byte(plainText)

	// 1. 原始字节加密解密
	encryptedBytes, err := Encrypt(plainBytes, SecretKey)
	if err != nil {
		t.Fatalf("字节加密失败: %v", err)
	}
	if len(encryptedBytes) == 0 {
		t.Fatal("加密结果为空")
	}

	decryptedBytes, err := Decrypt(encryptedBytes, SecretKey)
	if err != nil {
		t.Fatalf("字节解密失败: %v", err)
	}
	if string(decryptedBytes) != plainText {
		t.Errorf("解密结果不匹配\n期望: %s\n实际: %s", plainText, string(decryptedBytes))
	}
	t.Log("✅ 通用字节加解密测试通过")

	// 2. Base64 加密解密
	encryptedBase64, err := EncryptToBase64(plainBytes, SecretKey)
	if err != nil {
		t.Fatalf("Base64加密失败: %v", err)
	}
	t.Logf("加密后Base64: %s", encryptedBase64)

	decryptedBase64Bytes, err := DecryptFromBase64(encryptedBase64, SecretKey)
	if err != nil {
		t.Fatalf("Base64解密失败: %v", err)
	}
	if string(decryptedBase64Bytes) != plainText {
		t.Errorf("Base64解密结果不匹配\n期望: %s\n实际: %s", plainText, string(decryptedBase64Bytes))
	}
	t.Log("✅ Base64加解密测试通过")
}

// TestAESEmpty 空字符串测试
func TestAESEmpty(t *testing.T) {
	plainText := ""
	plainBytes := []byte(plainText)

	// 加密
	encBase64, err := EncryptToBase64(plainBytes, SecretKey)
	if err != nil {
		t.Fatalf("空字符串加密失败: %v", err)
	}

	// 解密
	decBytes, err := DecryptFromBase64(encBase64, SecretKey)
	if err != nil {
		t.Fatalf("空字符串解密失败: %v", err)
	}

	if string(decBytes) != plainText {
		t.Error("空字符串解密结果错误")
	}
	t.Log("✅ 空字符串测试通过")
}

// TestAESLongText 超长文本测试
func TestAESLongText(t *testing.T) {
	// 生成长文本
	var longText string
	for i := 0; i < 100; i++ {
		longText += "这是一段超长的测试文本，用于验证AES加密对大数据的处理能力 "
	}

	encBase64, err := EncryptToBase64([]byte(longText), SecretKey)
	if err != nil {
		t.Fatal(err)
	}

	decBytes, err := DecryptFromBase64(encBase64, SecretKey)
	if err != nil {
		t.Fatal(err)
	}

	if string(decBytes) != longText {
		t.Error("超长文本解密结果不匹配")
	}
	t.Log("✅ 超长文本测试通过")
}

// TestAESSpecialChar 特殊字符测试
func TestAESSpecialChar(t *testing.T) {
	plainText := "!@#$%^&*()_+-=[]{}|;':\",./<>?~`中文😀"
	encBase64, err := EncryptToBase64([]byte(plainText), SecretKey)
	if err != nil {
		t.Fatal(err)
	}

	decBytes, err := DecryptFromBase64(encBase64, SecretKey)
	if err != nil {
		t.Fatal(err)
	}

	if string(decBytes) != plainText {
		t.Errorf("特殊字符解密失败\n期望: %s\n实际: %s", plainText, string(decBytes))
	}
	t.Log("✅ 特殊字符测试通过")
}

// TestPKCS5Padding 填充与去填充单独测试
func TestPKCS5Padding(t *testing.T) {
	blockSize := aes.BlockSize // 16
	testCases := []struct {
		name  string
		input []byte
	}{
		{"长度正好16", bytes.Repeat([]byte("a"), 16)},
		{"长度15", bytes.Repeat([]byte("b"), 15)},
		{"长度1", []byte("c")},
		{"空字节", []byte{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			padded := PKCS5Padding(tc.input, blockSize)
			unpadded := PKCS5UnPadding(padded)
			if string(unpadded) != string(tc.input) {
				t.Errorf("填充/去填充不匹配\n输入: %s\n输出: %s", tc.input, unpadded)
			}
		})
	}
	t.Log("✅ PKCS5填充测试通过")
}
