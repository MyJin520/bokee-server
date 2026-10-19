package aes

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
)

var (
	SecretKey = []byte("2985BCFDB5FE43129843DB59825F8647")
)

// PKCS5Padding 对明文进行 PKCS5 填充，使其长度满足块大小的整数倍。
// 参数 plaintext: 待填充的明文字节切片。
// 参数 blockSize: AES 块大小（通常为 16 字节）。
// 返回值: 填充后的字节切片。
func PKCS5Padding(plaintext []byte, blockSize int) []byte {
	padding := blockSize - len(plaintext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(plaintext, padtext...)
}

// PKCS5UnPadding 去除 PKCS5 填充，还原原始明文。
// 参数 origData: 已解密的字节切片（包含填充）。
// 返回值: 去除填充后的原始明文字节切片。
func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	padding := int(origData[length-1])
	return origData[:(length - padding)]
}

// Encrypt 使用 AES-CBC 模式对明文进行加密。
// 参数 origData: 待加密的明文字节切片。
// 参数 key: 加密密钥，长度必须为 16、24 或 32 字节，分别对应 AES-128、AES-192、AES-256。
// 返回值: 加密后的密文字节切片；若密钥无效则返回错误。
func Encrypt(origData, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()
	origData = PKCS5Padding(origData, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	encrypted := make([]byte, len(origData))
	blockMode.CryptBlocks(encrypted, origData)
	return encrypted, nil
}

// EncryptToBase64 加密明文并将结果编码为 RawURL 格式的 Base64 字符串。
// 参数 origData: 待加密的明文字节切片。
// 参数 key: 加密密钥。
// 返回值: Base64 编码后的密文字符串；若加密失败则返回错误。
func EncryptToBase64(origData, key []byte) (string, error) {
	encrypted, err := Encrypt(origData, key)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encrypted), nil
}

// Decrypt 使用 AES-CBC 模式对密文进行解密。
// 参数 encrypted: 待解密的密文字节切片。
// 参数 key: 解密密钥，必须与加密时使用的密钥相同。
// 返回值: 解密后的明文字节切片；若密钥无效或数据被篡改则返回错误。
func Decrypt(encrypted, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(encrypted))
	blockMode.CryptBlocks(origData, encrypted)
	origData = PKCS5UnPadding(origData)
	return origData, nil
}

// DecryptFromBase64 从 RawURL 格式的 Base64 字符串解密数据。
// 参数 data: Base64 编码的密文字符串。
// 参数 key: 解密密钥。
// 返回值: 解密后的明文字节切片；若解码或解密失败则返回错误。
func DecryptFromBase64(data string, key []byte) ([]byte, error) {
	crypted, err := base64.RawURLEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	return Decrypt(crypted, key)
}
