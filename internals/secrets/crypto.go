package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	Argon2Time    = 3
	Argon2Memory  = 64 * 1024 // 64MB
	Argon2Threads = 4
	Argon2KeyLen  = 32
	SaltLen       = 16
	NonceLen      = 12
)

func Encrypt(plainText []byte, masterKey []byte) (string, error) {
	salt := make([]byte, SaltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	key := argon2.IDKey(masterKey, salt, Argon2Time, Argon2Memory, Argon2Threads, Argon2KeyLen)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	cipherText := aesGCM.Seal(nonce, nonce, plainText, nil)

	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedCipherText := base64.RawStdEncoding.EncodeToString(cipherText)

	return fmt.Sprintf("argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", Argon2Memory, Argon2Time, Argon2Threads, encodedSalt, encodedCipherText), nil
}

func Decrypt(encodedValue string, masterKey []byte) ([]byte, error) {
	parts := strings.Split(encodedValue, "$")
	if len(parts) != 5 {
		return nil, fmt.Errorf("invalid encoded format")
	}

	if parts[0] != "argon2id" {
		return nil, fmt.Errorf("unsupported algorithm: %s", parts[0])
	}

	encodedSalt := parts[3]
	encodedCipherText := parts[4]

	salt, err := base64.RawStdEncoding.DecodeString(encodedSalt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	cipherTextWithNonce, err := base64.RawStdEncoding.DecodeString(encodedCipherText)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	key := argon2.IDKey(masterKey, salt, Argon2Time, Argon2Memory, Argon2Threads, Argon2KeyLen)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(cipherTextWithNonce) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, cipherText := cipherTextWithNonce[:nonceSize], cipherTextWithNonce[nonceSize:]
	plainText, err := aesGCM.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plainText, nil
}
