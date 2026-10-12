package secrets

import (
	"crypto/rand"
	"encoding/base64"
	"errors"

	"github.com/spf13/cobra"
)

var masterCmd = &cobra.Command{
	Use:         "master",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "Generate a cryptographically secure master key",
	Long:        "Generate a random master key, printed base64-encoded — the key encrypt, decrypt, and the seed engine use. --length sets the byte size (default 32).",
	Args:        cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		keyLength, _ := cmd.Flags().GetInt("length")
		if keyLength <= 0 {
			return errors.New("invalid key length")
		}

		key := make([]byte, keyLength)
		rand.Read(key)

		encodedKey := base64.StdEncoding.EncodeToString(key)
		return emit(cmd, encodedKey, "Master key written", true,
			printKey("Master Key", encodedKey, "Store this key securely. You need it to encrypt and decrypt secrets."))
	},
}

func init() {
	masterCmd.Flags().Int("length", 32, "Key length in bytes")
}
