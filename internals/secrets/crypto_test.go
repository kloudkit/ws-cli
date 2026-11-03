package secrets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestEncryptDecrypt(t *testing.T) {
	key := []byte("12345678901234567890123456789012")

	encrypted, err := Encrypt([]byte("secret data"), key)
	assert.NilError(t, err)
	assert.Equal(t, strings.Count(encrypted, "$"), 1)

	decrypted, err := Decrypt(NormalizeEncrypted(" "+encrypted[:10]+"\n"+encrypted[10:]+"\n"), key)
	assert.NilError(t, err)
	assert.Equal(t, string(decrypted), "secret data")

	for input, want := range map[string]string{
		encrypted:  "message authentication failed",
		"invalid":  "invalid encrypted format",
		"!!$AAAA":  "failed to decode salt",
		"AAAA$!!":  "failed to decode ciphertext",
		"AAAA$AAA": "ciphertext too short",
	} {
		other := []byte("22345678901234567890123456789012")
		if input != encrypted {
			other = key
		}

		_, err = Decrypt(input, other)
		assert.ErrorContains(t, err, want)
	}
}

func TestResolveEncryptedValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cipher")
	assert.NilError(t, os.WriteFile(path, []byte("ab\ncd\n"), 0o600))

	for input, want := range map[string]string{"inline$value": "inline$value", "file:" + path: "abcd"} {
		got, err := ResolveEncryptedValue(input)
		assert.NilError(t, err)
		assert.Equal(t, got, want)
	}

	_, err := ResolveEncryptedValue("file:" + path + ".missing")
	assert.ErrorContains(t, err, "failed to read encrypted file")
}

func TestHashPasswordForWorkspace(t *testing.T) {
	hash, err := HashPasswordForWorkspace("testpassword123")
	assert.NilError(t, err)
	assert.Assert(t, strings.HasPrefix(hash, "$argon2id$v=19$m=65536,t=3,p=4$"))

	hash2, _ := HashPasswordForWorkspace("testpassword123")
	assert.Assert(t, hash != hash2)
}
