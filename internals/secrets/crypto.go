package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	Argon2Time    = 3
	Argon2Memory  = 64 * 1024 // 64MB
	Argon2Threads = 4
	Argon2KeyLen  = 32
	SaltLen       = 16
)

func Encrypt(plainText []byte, masterKey []byte) (string, error) {
	salt := make([]byte, SaltLen)
	rand.Read(salt)

	aesGCM, err := deriveKeyAndGCM(masterKey, salt)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	rand.Read(nonce)

	cipherText := aesGCM.Seal(nonce, nonce, plainText, nil)

	return fmt.Sprintf("%s$%s",
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(cipherText)), nil
}

func NormalizeEncrypted(encrypted string) string {
	return strings.Join(strings.Fields(encrypted), "")
}

func ResolveEncryptedValue(encrypted string) (string, error) {
	if !strings.HasPrefix(encrypted, "file:") {
		return encrypted, nil
	}

	filePath := strings.TrimPrefix(encrypted, "file:")
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read encrypted file %s: %w", filePath, err)
	}

	return NormalizeEncrypted(string(content)), nil
}

func Decrypt(encodedValue string, masterKey []byte) ([]byte, error) {
	parts := strings.Split(encodedValue, "$")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid encrypted format")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	cipherTextWithNonce, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	aesGCM, err := deriveKeyAndGCM(masterKey, salt)
	if err != nil {
		return nil, err
	}

	nonceSize := aesGCM.NonceSize()
	if len(cipherTextWithNonce) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	return aesGCM.Open(nil, cipherTextWithNonce[:nonceSize], cipherTextWithNonce[nonceSize:], nil)
}

func deriveKeyAndGCM(masterKey, salt []byte) (cipher.AEAD, error) {
	key := argon2.IDKey(masterKey, salt, Argon2Time, Argon2Memory, Argon2Threads, Argon2KeyLen)
	defer clear(key)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	return cipher.NewGCM(block)
}

func HashPasswordForWorkspace(password string) (string, error) {
	salt := make([]byte, SaltLen)
	rand.Read(salt)

	hash := argon2.IDKey([]byte(password), salt, Argon2Time, Argon2Memory, Argon2Threads, Argon2KeyLen)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, Argon2Memory, Argon2Time, Argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}
