package secrets

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/kloudkit/ws-cli/internals/path"
	internalSecrets "github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var encryptCmd = &cobra.Command{
	Use:   "encrypt <plaintext>",
	Short: "Encrypt a plaintext value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		plaintext := args[0]
		outputFile, _ := cmd.Flags().GetString("output")
		masterKeyFlag, _ := cmd.Flags().GetString("master")
		force, _ := cmd.Flags().GetBool("force")
		raw, _ := cmd.Flags().GetBool("raw")

		masterKey, err := internalSecrets.ResolveMasterKey(masterKeyFlag)
		if err != nil {
			return err
		}

		encrypted, err := internalSecrets.Encrypt([]byte(plaintext), masterKey)
		if err != nil {
			return fmt.Errorf("encryption failed: %w", err)
		}

		// Requirement: Output encoded as Base64 with base64: prefix
		finalOutput := "base64:" + base64.StdEncoding.EncodeToString([]byte(encrypted))

		if outputFile == "" {
			fmt.Fprintln(cmd.OutOrStdout(), finalOutput)
			return nil
		}

		// Write to file
		if !path.CanOverride(outputFile, force) {
			return fmt.Errorf("file %s exists, use --force to overwrite", outputFile)
		}

		if err := os.WriteFile(outputFile, []byte(finalOutput+"\n"), 0644); err != nil {
			return fmt.Errorf("failed to write to output file: %w", err)
		}

		if !raw {
			fmt.Fprintln(cmd.OutOrStdout(), styles.Success().Render(fmt.Sprintf("Encrypted value written to %s", outputFile)))
		}
		return nil
	},
}

func init() {
	encryptCmd.Flags().String("output", "", "Write output to file instead of stdout")
}
