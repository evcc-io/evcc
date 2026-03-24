package requests

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
)

func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS5Trimming(encrypt []byte) []byte {
	padding := encrypt[len(encrypt)-1]
	return encrypt[:len(encrypt)-int(padding)]
}

func Decrypt(cipherText, hexKey, hexIV string) string {
	if len(cipherText) == 0 || len(hexKey) == 0 || len(hexIV) == 0 {
		return ""
	}

	secretKey, _ := hex.DecodeString(hexKey)
	ivParameter, _ := hex.DecodeString(hexIV)
	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return ""
	}

	decrypter := cipher.NewCBCDecrypter(block, ivParameter)
	text, _ := hex.DecodeString(cipherText)
	decrypted := make([]byte, len(text))
	decrypter.CryptBlocks(decrypted, text)

	return string(PKCS5Trimming(decrypted))
}

func Encrypt(plainText, hexKey, hexIV string) string {
	if len(plainText) == 0 || len(hexKey) == 0 || len(hexIV) == 0 {
		return ""
	}

	secretKey, _ := hex.DecodeString(hexKey)
	ivParameter, _ := hex.DecodeString(hexIV)
	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return ""
	}

	content := PKCS5Padding([]byte(plainText), block.BlockSize())
	encrypter := cipher.NewCBCEncrypter(block, ivParameter)

	ciphertext := make([]byte, len(content))
	encrypter.CryptBlocks(ciphertext, content)

	return hex.EncodeToString(ciphertext)
}
