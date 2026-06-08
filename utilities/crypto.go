package utilities

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// decodeKey turns the base64 config key into a 32-byte AES-256 key.
func decodeKey(keyB64 string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return nil, errors.New("failed to decode encryption key")
	}
	if len(key) != 32 {
		return nil, errors.New("encryption key must be 32 bytes")
	}
	return key, nil
}

// EncryptString encrypts plaintext with AES-256-GCM using the base64 key and returns
// a base64 string of nonce||ciphertext. The output contains no ':' so it is safe to
// embed in the prefixed Strava code field.
func EncryptString(plaintext string, keyB64 string) (string, error) {
	key, err := decodeKey(keyB64)
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

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString reverses EncryptString. It returns an error when the input is not
// valid ciphertext for this key (e.g. a legacy plaintext value), letting callers fall
// back appropriately.
func DecryptString(ciphertextB64 string, keyB64 string) (string, error) {
	key, err := decodeKey(keyB64)
	if err != nil {
		return "", err
	}

	data, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", errors.New("failed to decode ciphertext")
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
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.New("failed to decrypt")
	}

	return string(plaintext), nil
}
