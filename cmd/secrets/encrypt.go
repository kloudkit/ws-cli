package secrets

import (
	"fmt"
	"strings"

	internalIO "github.com/kloudkit/ws-cli/internals/io"
	internalSecrets "github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var encryptCmd = &cobra.Command{
	Use:   "encrypt <plaintext|->",
	Short: "Encrypt a plaintext value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		outputFile, _ := cmd.Flags().GetString("output")
		masterKeyFlag, _ := cmd.Flags().GetString("master")
		modeStr, _ := cmd.Flags().GetString("mode")
		force, _ := cmd.Flags().GetBool("force")
		raw, _ := cmd.Flags().GetBool("raw")

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

		if outputFile == "" {
			if raw {
				fmt.Fprintln(cmd.OutOrStdout(), encrypted)
			} else {
				styles.PrintTitle(cmd.OutOrStdout(), "Encrypted Value")
				styles.PrintKeyCode(cmd.OutOrStdout(), "Value", encrypted)
			}
			return nil
		}

		if err := internalIO.WriteSecureFile(outputFile, []byte(encrypted+"\n"), modeStr, force); err != nil {
			return err
		}

		if !raw {
			styles.PrintSuccessWithDetailsCode(cmd.OutOrStdout(), "Secret encrypted successfully", [][]string{
				{"Output", outputFile},
			})
		}
		return nil
	},
}

func init() {
	encryptCmd.Flags().String("output", "", "Write output to file instead of stdout")
}
