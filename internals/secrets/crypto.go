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
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
)

func newSalt() []byte {
	salt := make([]byte, 16)
	rand.Read(salt)
	return salt
}

func argonKey(secret, salt []byte) []byte {
	return argon2.IDKey(secret, salt, argonTime, argonMemory, argonThreads, argonKeyLen)
}

func Encrypt(plainText []byte, masterKey []byte) (string, error) {
	salt := newSalt()
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

	filePath := encrypted[len("file:"):]
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
	key := argonKey(masterKey, salt)
	defer clear(key)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	return cipher.NewGCM(block)
}

func HashPasswordForWorkspace(password string) (string, error) {
	salt := newSalt()

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(argonKey([]byte(password), salt)),
	), nil
}
