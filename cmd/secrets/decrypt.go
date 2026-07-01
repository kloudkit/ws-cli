package secrets

import (
	internalIO "github.com/kloudkit/ws-cli/internals/io"
	internalSecrets "github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/spf13/cobra"
)

var decryptCmd = &cobra.Command{
	Use:   "decrypt <encrypted|->",
	Short: "Decrypt an encrypted value",
	Long:  "Decrypt a value produced by encrypt, under the master key. Reads from the argument or stdin (-); writes the plaintext to stdout, or a file with --output.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := getOutputConfig(cmd)
		masterKeyFlag, _ := cmd.Flags().GetString("master")

		masterKey, err := internalSecrets.ResolveMasterKey(masterKeyFlag)
		if err != nil {
			return err
		}

		input, err := internalIO.ReadInput(args[0], cmd.InOrStdin())
		if err != nil {
			return err
		}

		input = internalSecrets.NormalizeEncrypted(input)

		decrypted, err := internalSecrets.Decrypt(input, masterKey)
		if err != nil {
			return err
		}

		return handleOutput(cmd, cfg, string(decrypted), "Decrypted Value", "Secret decrypted successfully", false)
	},
}
