package vault

import (
	"fmt"

	internalSecrets "github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var rotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Re-encrypt vault secrets with a new master key",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		inputFile, _ := cmd.Flags().GetString("input")
		masterKeyFlag, _ := cmd.Flags().GetString("master")
		newMasterFlag, _ := cmd.Flags().GetString("new-master")
		raw, _ := cmd.Flags().GetBool("raw")

		vaultPath, err := internalSecrets.ResolveVaultPath(inputFile)
		if err != nil {
			return err
		}

		oldKey, err := internalSecrets.ResolveMasterKey(masterKeyFlag)
		if err != nil {
			return err
		}

		newKey, err := internalSecrets.ResolveMasterKey(newMasterFlag)
		if err != nil {
			return fmt.Errorf("new master key: %w", err)
		}

		vault, err := internalSecrets.LoadRawVault(vaultPath)
		if err != nil {
			return err
		}

		fileRefs, err := internalSecrets.RotateVault(vault, oldKey, newKey)
		if err != nil {
			return err
		}

		if err := internalSecrets.SaveVault(vaultPath, vault); err != nil {
			return err
		}

		if raw {
			fmt.Fprintf(cmd.OutOrStdout(), "%d\n", len(vault.Secrets))
			return nil
		}

		if len(fileRefs) > 0 {
			fmt.Fprintln(cmd.OutOrStdout(), styles.Warning().Render("⚠ The following secrets had file: references and are now inlined:"))
			for _, name := range fileRefs {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", styles.Code().Render(name))
			}
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s\n",
			styles.Success().Render(fmt.Sprintf("✓ Rotated %d secret(s)", len(vault.Secrets))))

		return nil
	},
}

func init() {
	rotateCmd.Flags().String("new-master", "", "New master key or path to key file")
	rotateCmd.MarkFlagRequired("new-master")
}
