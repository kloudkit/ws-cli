package secrets

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/kloudkit/ws-cli/internals/path"
	internalSecrets "github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var decryptCmd = &cobra.Command{
	Use:   "decrypt <encrypted>",
	Short: "Decrypt an encrypted value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := args[0]
		outputFile, _ := cmd.Flags().GetString("output")
		masterKeyFlag, _ := cmd.Flags().GetString("master")
		force, _ := cmd.Flags().GetBool("force")
		raw, _ := cmd.Flags().GetBool("raw")

		masterKey, err := internalSecrets.ResolveMasterKey(masterKeyFlag)
		if err != nil {
			return err
		}

		// Handle base64: prefix
		var encryptedString string
		if strings.HasPrefix(input, "base64:") {
			encoded := strings.TrimPrefix(input, "base64:")
			decodedBytes, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				return fmt.Errorf("failed to decode base64 input: %w", err)
			}
			encryptedString = string(decodedBytes)
		} else {
			encryptedString = input
		}

		decrypted, err := internalSecrets.Decrypt(encryptedString, masterKey)
		if err != nil {
			return err
		}

		if outputFile == "" {
			fmt.Fprint(cmd.OutOrStdout(), string(decrypted))
			return nil
		}

		// Write to file
		if !path.CanOverride(outputFile, force) {
			return fmt.Errorf("file %s exists, use --force to overwrite", outputFile)
		}

		// Determine file mode - if we knew the type we could set it, but for generic decrypt use 0600 for safety
		if err := os.WriteFile(outputFile, decrypted, 0600); err != nil {
			return fmt.Errorf("failed to write to output file: %w", err)
		}

		if !raw {
			fmt.Fprintln(cmd.OutOrStdout(), styles.Success().Render(fmt.Sprintf("Decrypted value written to %s", outputFile)))
		}
		return nil
	},
}

func init() {
	decryptCmd.Flags().String("output", "", "Write output to file instead of stdout")
}
