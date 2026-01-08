package secrets

import (
	"fmt"
	"strings"

	internalIO "github.com/kloudkit/ws-cli/internals/io"
	internalSecrets "github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var decryptCmd = &cobra.Command{
	Use:   "decrypt <encrypted|->",
	Short: "Decrypt an encrypted value",
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

		input, err := internalIO.ReadInput(args[0], cmd.InOrStdin())
		if err != nil {
			return err
		}

		input = strings.ReplaceAll(input, "\r", "")
		input = strings.ReplaceAll(input, "\n", "")
		input = strings.ReplaceAll(input, " ", "")
		input = strings.ReplaceAll(input, "\t", "")

		decrypted, err := internalSecrets.Decrypt(input, masterKey)
		if err != nil {
			return err
		}

		if outputFile == "" {
			if raw {
				fmt.Fprint(cmd.OutOrStdout(), string(decrypted))
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.Header().Render("Decrypted Value"))
				fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", styles.Code().Render(string(decrypted)))
			}
			return nil
		}

		if err := internalIO.WriteSecureFile(outputFile, decrypted, modeStr, force); err != nil {
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
