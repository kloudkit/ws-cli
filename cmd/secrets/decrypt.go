package secrets

import (
	"fmt"

	"github.com/kloudkit/ws-cli/internals/io"
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
		modeStr, _ := cmd.Flags().GetString("mode")
		force, _ := cmd.Flags().GetBool("force")
		raw, _ := cmd.Flags().GetBool("raw")

		masterKey, err := internalSecrets.ResolveMasterKey(masterKeyFlag)
		if err != nil {
			return err
		}

		encryptedBytes, err := internalSecrets.DecodeWithPrefix(input)
		if err != nil {
			return err
		}

		decrypted, err := internalSecrets.Decrypt(string(encryptedBytes), masterKey)
		if err != nil {
			return err
		}

		if outputFile == "" {
			fmt.Fprint(cmd.OutOrStdout(), string(decrypted))
			return nil
		}

		if err := io.WriteSecureFile(outputFile, decrypted, modeStr, force); err != nil {
			return err
		}

		if !raw {
			fmt.Fprintln(cmd.OutOrStdout(), styles.Success().Render(fmt.Sprintf("✓ Decrypted value written to %s", outputFile)))
		}
		return nil
	},
}

func init() {
	decryptCmd.Flags().String("output", "", "Write output to file instead of stdout")
}
