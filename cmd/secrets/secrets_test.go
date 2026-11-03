package secrets

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/cmd/clitest"

	"gotest.tools/v3/assert"
)

func run(t *testing.T, stdin string, args ...string) string {
	t.Helper()
	out, err := clitest.Run(SecretsCmd, stdin, args...)
	assert.NilError(t, err)

	return out
}

func TestGenerate(t *testing.T) {
	assert.Equal(t, len(strings.TrimSpace(run(t, "", "generate", "master", "--length", "16", "--raw"))), 24)

	keyFile := filepath.Join(t.TempDir(), "master.key")
	output := run(t, "", "generate", "master", "--output", keyFile)
	assert.Assert(t, strings.Contains(output, "✓ Master key written"), output)
	assert.Equal(t, strings.Count(output, keyFile), 1)
	assert.Equal(t, strings.Count(output, "✓"), 1)

	content, err := os.ReadFile(keyFile)
	assert.NilError(t, err)
	assert.Equal(t, len(content), 45)

	hash := strings.TrimSpace(run(t, "testpassword123", "generate", "login", "--raw"))
	assert.Assert(t, strings.HasPrefix(hash, "$argon2id$v=19$m=65536,t=3,p=4$"), hash)
}

func TestEncryptDecrypt(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "master.key")
	masterKey := base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))
	assert.NilError(t, os.WriteFile(keyFile, []byte(masterKey), 0o600))

	encrypted := strings.TrimSpace(run(t, "", "encrypt", "test-secret", "--master", keyFile, "--raw"))
	assert.Equal(t, strings.Count(encrypted, "$"), 1)
	assert.Assert(t, !strings.Contains(encrypted, "Encrypted"))

	assert.Equal(t, run(t, "", "decrypt", encrypted, "--master", keyFile, "--raw"), "test-secret")

	salt, cipher, _ := strings.Cut(encrypted, "$")
	assert.Equal(t, run(t, salt+"\n  \t$"+cipher+"\n", "decrypt", "-", "--master", keyFile, "--raw"), "test-secret")

	styled := run(t, "", "decrypt", encrypted, "--master", keyFile)
	assert.Assert(t, strings.Contains(styled, "Decrypted") && strings.Contains(styled, "test-secret"), styled)
}

func TestStyledOutputs(t *testing.T) {
	assert.Assert(t, strings.Contains(run(t, "", "generate", "master"), "Store this key securely"))
	assert.Assert(t, strings.Contains(run(t, "pw", "generate", "login"), "WS_AUTH_PASSWORD_HASHED"))

	root := t.TempDir()
	t.Setenv("WS__INTERNAL_SECRETS_ROOT", root)
	t.Setenv("WS_SECRETS_MASTER_KEY", "k")

	assert.Assert(t, strings.Contains(run(t, "", "materialize"), "✓ Master key projected to "+root))
	assert.Equal(t, run(t, "", "materialize"), "")
}

func TestErrors(t *testing.T) {
	cases := map[string][]string{
		"invalid key length":   {"generate", "master", "--length", "0"},
		"master key not found": {"encrypt", "x"},
		"invalid encrypted":    {"decrypt", "nodollar", "--master", "MTIzNDU2Nzg5MDEyMzQ1Ng=="},
		"empty password":       {"generate", "login"},
	}

	for want, args := range cases {
		t.Setenv("WS_SECRETS_MASTER_KEY", "")
		t.Setenv("WS__INTERNAL_SECRETS_ROOT", t.TempDir())
		_, err := clitest.Run(SecretsCmd, "", args...)
		assert.ErrorContains(t, err, want)
	}
}
