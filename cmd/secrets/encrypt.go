package secrets

import (
	"fmt"
	"strings"

	internalIO "github.com/kloudkit/ws-cli/internals/io"
	internalSecrets "github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/spf13/cobra"
)

var encryptCmd = &cobra.Command{
	Use:   "encrypt <plaintext|->",
	Short: "Encrypt a plaintext value",
	Long:  "Encrypt a value under the master key. Reads the plaintext from the argument or stdin (-); writes the ciphertext to stdout, or a file with --output.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := getOutputConfig(cmd)
		masterKeyFlag, _ := cmd.Flags().GetString("master")

		masterKey, err := internalSecrets.ResolveMasterKey(masterKeyFlag)
		if err != nil {
			return err
		}

		plaintext, err := internalIO.ReadInput(args[0], cmd.InOrStdin())
		if err != nil {
			return err
		}

		plaintext = strings.TrimSpace(plaintext)

		encrypted, err := internalSecrets.Encrypt([]byte(plaintext), masterKey)
		if err != nil {
			return fmt.Errorf("encryption failed: %w", err)
		}

		return handleOutput(cmd, cfg, encrypted, "Encrypted Value", "Secret encrypted successfully", true)
	},
}
