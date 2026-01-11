package secrets

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gotest.tools/v3/assert"
)

func resetCommandFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		flag.Value.Set(flag.DefValue)
		flag.Changed = false
	})

	for _, c := range cmd.Commands() {
		resetCommandFlags(c)
	}
}

func TestSecretsCommand(t *testing.T) {
	t.Run("Generate", func(t *testing.T) {
		resetCommandFlags(SecretsCmd)

		buffer := new(bytes.Buffer)
		SecretsCmd.SetOut(buffer)
		SecretsCmd.SetErr(buffer)
		SecretsCmd.SetArgs([]string{"generate", "--length", "16", "--raw"})

		err := SecretsCmd.Execute()
		assert.NilError(t, err)

		output := buffer.String()
		assert.Equal(t, len(strings.TrimSpace(output)), 24)
	})

	t.Run("EncryptRaw", func(t *testing.T) {
		resetCommandFlags(SecretsCmd)

		keyFile := filepath.Join(t.TempDir(), "master.key")
		masterKey := base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))
		err := os.WriteFile(keyFile, []byte(masterKey), 0600)
		assert.NilError(t, err)

		buffer := new(bytes.Buffer)
		SecretsCmd.SetOut(buffer)
		SecretsCmd.SetErr(buffer)
		SecretsCmd.SetArgs([]string{"encrypt", "test-secret", "--master", keyFile, "--raw"})

		err = SecretsCmd.Execute()
		assert.NilError(t, err)

		output := strings.TrimSpace(buffer.String())
		assert.Assert(t, strings.Count(output, "$") == 2)
		assert.Assert(t, !strings.Contains(output, "Encrypted"))
	})

	t.Run("DecryptRaw", func(t *testing.T) {
		resetCommandFlags(SecretsCmd)

		keyFile := filepath.Join(t.TempDir(), "master.key")
		masterKey := base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))
		err := os.WriteFile(keyFile, []byte(masterKey), 0600)
		assert.NilError(t, err)

		encryptBuffer := new(bytes.Buffer)
		SecretsCmd.SetOut(encryptBuffer)
		SecretsCmd.SetErr(encryptBuffer)
		SecretsCmd.SetArgs([]string{"encrypt", "test-secret", "--master", keyFile, "--raw"})

		err = SecretsCmd.Execute()
		assert.NilError(t, err)

		encrypted := strings.TrimSpace(encryptBuffer.String())

		resetCommandFlags(SecretsCmd)

		decryptBuffer := new(bytes.Buffer)
		SecretsCmd.SetOut(decryptBuffer)
		SecretsCmd.SetErr(decryptBuffer)
		SecretsCmd.SetArgs([]string{"decrypt", encrypted, "--master", keyFile, "--raw"})

		err = SecretsCmd.Execute()
		assert.NilError(t, err)

		output := decryptBuffer.String()
		assert.Equal(t, "test-secret", output)
		assert.Assert(t, !strings.Contains(output, "Decrypted"))
	})

	t.Run("DecryptMultiline", func(t *testing.T) {
		resetCommandFlags(SecretsCmd)

		keyFile := filepath.Join(t.TempDir(), "master.key")
		masterKey := base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))
		err := os.WriteFile(keyFile, []byte(masterKey), 0600)
		assert.NilError(t, err)

		encryptBuffer := new(bytes.Buffer)
		SecretsCmd.SetOut(encryptBuffer)
		SecretsCmd.SetErr(encryptBuffer)
		SecretsCmd.SetArgs([]string{"encrypt", "test-secret", "--master", keyFile, "--raw"})

		err = SecretsCmd.Execute()
		assert.NilError(t, err)

		encrypted := strings.TrimSpace(encryptBuffer.String())
		parts := strings.Split(encrypted, "$")
		assert.Equal(t, 3, len(parts))

		multilineEncrypted := parts[0] + "\n  \t$" + parts[1] + "\n  \t$" + parts[2] + "\n"

		resetCommandFlags(SecretsCmd)

		decryptBuffer := new(bytes.Buffer)
		decryptInputBuffer := bytes.NewBufferString(multilineEncrypted)
		SecretsCmd.SetIn(decryptInputBuffer)
		SecretsCmd.SetOut(decryptBuffer)
		SecretsCmd.SetErr(decryptBuffer)
		SecretsCmd.SetArgs([]string{"decrypt", "-", "--master", keyFile, "--raw"})

		err = SecretsCmd.Execute()
		assert.NilError(t, err)

		output := decryptBuffer.String()
		assert.Equal(t, "test-secret", output)
	})
}
