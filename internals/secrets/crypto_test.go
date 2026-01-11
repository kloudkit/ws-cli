package secrets

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestEncryptDecrypt(t *testing.T) {
	masterKey := make([]byte, 32)
	plainText := "secret data"

	encrypted, err := Encrypt([]byte(plainText), masterKey)
	assert.NilError(t, err)
	assert.Assert(t, strings.Count(encrypted, "$") == 2)

	decrypted, err := Decrypt(encrypted, masterKey)
	assert.NilError(t, err)
	assert.Equal(t, plainText, string(decrypted))
}

func TestDecryptErrors(t *testing.T) {
	tests := []struct {
		name          string
		encoded       string
		masterKey     []byte
		errorContains string
	}{
		{
			name:          "invalid format",
			encoded:       "invalid",
			masterKey:     make([]byte, 32),
			errorContains: "invalid encrypted format",
		},
		{
			name: "wrong key",
			encoded: func() string {
				key1 := []byte("12345678901234567890123456789012")
				enc, _ := Encrypt([]byte("data"), key1)
				return enc
			}(),
			masterKey:     []byte("22345678901234567890123456789012"),
			errorContains: "message authentication failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decrypt(tt.encoded, tt.masterKey)
			assert.ErrorContains(t, err, tt.errorContains)
		})
	}
}

func TestDecryptUnsupportedVersion(t *testing.T) {
	masterKey := make([]byte, 32)

	encryptedValue := "999$dGVzdA$dGVzdA"

	_, err := Decrypt(encryptedValue, masterKey)
	assert.ErrorContains(t, err, "unsupported algorithm version")
}
