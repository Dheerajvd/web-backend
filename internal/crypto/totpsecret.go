package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

// DeriveMFAKey returns a 32-byte AES key from the JWT secret (do not use JWT_SECRET shorter than 32 chars in prod).
func DeriveMFAKey(jwtSecret string) []byte {
	sum := sha256.Sum256([]byte("web-manager:mfa:totp:v1:" + jwtSecret))
	return sum[:]
}

// EncryptAESGCM encodes nonce||ciphertext as base64 (raw std).
func EncryptAESGCM(plaintext, key []byte) (string, error) {
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
	out := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.RawStdEncoding.EncodeToString(out), nil
}

func DecryptAESGCM(encoded string, key []byte) ([]byte, error) {
	raw, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(raw) < ns {
		return nil, errors.New("ciphertext too short")
	}
	return gcm.Open(nil, raw[:ns], raw[ns:], nil)
}
