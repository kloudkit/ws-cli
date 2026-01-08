package secrets

import (
	"fmt"

	"github.com/kloudkit/ws-cli/internals/io"
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
		modeStr, _ := cmd.Flags().GetString("mode")
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

		finalOutput := internalSecrets.EncodeWithPrefix([]byte(encrypted))

		if outputFile == "" {
			fmt.Fprintln(cmd.OutOrStdout(), finalOutput)
			return nil
		}

		if err := io.WriteSecureFile(outputFile, []byte(finalOutput+"\n"), modeStr, force); err != nil {
			return err
		}

		if !raw {
			fmt.Fprintln(cmd.OutOrStdout(), styles.Success().Render(fmt.Sprintf("✓ Encrypted value written to %s", outputFile)))
		}
		return nil
	},
}

func init() {
	encryptCmd.Flags().String("output", "", "Write output to file instead of stdout")
}
