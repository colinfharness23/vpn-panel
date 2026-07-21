package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

const credentialAAD = "commercial-residential-relay-v1"

func credentialKey() ([]byte, error) {
	settingService := &SettingService{}
	secret, err := settingService.GetSecret()
	if err != nil || len(secret) == 0 {
		return nil, errors.New("panel secret is unavailable")
	}
	sum := sha256.Sum256(append([]byte(credentialAAD+":"), secret...))
	return sum[:], nil
}

func ProtectCredential(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	key, err := credentialKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(value), []byte(credentialAAD))
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}

func UnprotectCredential(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	key, err := credentialKey()
	if err != nil {
		return "", err
	}
	data, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", errors.New("encrypted credential is truncated")
	}
	nonce := data[:gcm.NonceSize()]
	plain, err := gcm.Open(nil, nonce, data[gcm.NonceSize():], []byte(credentialAAD))
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
