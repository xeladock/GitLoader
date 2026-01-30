package crypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// var SecretKey = []byte("F/x31AP9n3/auxKX0THSRCfwgpCXJqWj8V1n98EeNhw=")
var SecretKey = []byte{
	0xF3, 0x7F, 0xD3, 0x10, 0x3F, 0xD9, 0xF7, 0xFE,
	0xB5, 0xF0, 0x4C, 0xD4, 0x4A, 0x31, 0x47, 0xC2,
	0x7C, 0x60, 0xA0, 0x70, 0x97, 0x26, 0xA5, 0xA3,
	0xF1, 0x5D, 0x67, 0xF7, 0xC7, 0x1E, 0x36, 0x1C,
}

// Замени на реальный ключ
// шифрование
func Encrypt(plaintext string, key []byte) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}
func Decrypt(encrypted string, key []byte) (string, error) {
	if encrypted == "" {
		return "", nil
	}
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
