package secrets

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestEncryptDecrypt(t *testing.T) {
	masterKey := make([]byte, 32) // Use a dummy 32-byte key
	plainText := "secret data"

	encrypted, err := Encrypt([]byte(plainText), masterKey)
	assert.NilError(t, err)
	assert.Assert(t, strings.HasPrefix(encrypted, "argon2id$"))

	decrypted, err := Decrypt(encrypted, masterKey)
	assert.NilError(t, err)
	assert.Equal(t, plainText, string(decrypted))
}

func TestDecryptInvalidFormat(t *testing.T) {
	masterKey := make([]byte, 32)
	_, err := Decrypt("invalid", masterKey)
	assert.ErrorContains(t, err, "invalid encoded format")
}

func TestDecryptUnsupportedAlgorithm(t *testing.T) {
	masterKey := make([]byte, 32)
	// parts needs to be 6
	encoded := "sha256$v=1$m=1,t=1,p=1$salt$cipher"
	_, err := Decrypt(encoded, masterKey)
	assert.ErrorContains(t, err, "unsupported algorithm")
}

func TestDecryptWrongKey(t *testing.T) {
	key1 := []byte("12345678901234567890123456789012")
	key2 := []byte("22345678901234567890123456789012")
	plainText := "data"

	encrypted, err := Encrypt([]byte(plainText), key1)
	assert.NilError(t, err)

	_, err = Decrypt(encrypted, key2)
	assert.ErrorContains(t, err, "message authentication failed")
}
