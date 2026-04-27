package vault

import (
	"fmt"
	"slices"
	"strings"

	internalSecrets "github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var decryptCmd = &cobra.Command{
	Use:   "decrypt",
	Short: "Decrypt vault secrets and write to destinations",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		inputFile, _ := cmd.Flags().GetString("input")
		masterKeyFlag, _ := cmd.Flags().GetString("master")
		keys, _ := cmd.Flags().GetStringArray("key")
		force, _ := cmd.Flags().GetBool("force")
		raw, _ := cmd.Flags().GetBool("raw")
		stdout, _ := cmd.Flags().GetBool("stdout")
		modeOverride, _ := cmd.Flags().GetString("mode")

		vaultPath, err := internalSecrets.ResolveVaultPath(inputFile)
		if err != nil {
			return err
		}

		masterKey, err := internalSecrets.ResolveMasterKey(masterKeyFlag)
		if err != nil {
			return err
		}

		vault, err := internalSecrets.LoadVault(vaultPath)
		if err != nil {
			return err
		}

		opts := internalSecrets.ProcessOptions{
			MasterKey:    masterKey,
			Keys:         keys,
			Stdout:       stdout,
			Raw:          raw,
			Force:        force,
			ModeOverride: modeOverride,
		}

		results, err := internalSecrets.ProcessVault(vault, opts)
		if err != nil {
			return err
		}

		if stdout {
			printStdoutResults(cmd, results, raw)
			return nil
		}

		if raw {
			return nil
		}

		printVaultSuccess(cmd, results)
		return nil
	},
}

func init() {
	decryptCmd.Flags().StringArray("key", []string{}, "Decrypt only specified key")
	decryptCmd.Flags().Bool("stdout", false, "Output decrypted values to stdout")
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func printStdoutResults(cmd *cobra.Command, results map[string]string, raw bool) {
	for _, key := range sortedKeys(results) {
		value := results[key]
		output := internalSecrets.FormatSecretForStdout(key, value, raw)
		fmt.Fprint(cmd.OutOrStdout(), output)
	}
}

func printVaultSuccess(cmd *cobra.Command, results map[string]string) {
	fmt.Fprintln(cmd.OutOrStdout(), styles.Success().Render("✓ Vault processed successfully"))
	for _, key := range sortedKeys(results) {
		dest := results[key]
		displayDest := dest
		if after, ok := strings.CutPrefix(dest, "env:"); ok {
			displayDest = fmt.Sprintf("env:%s", after)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  %s → %s\n",
			styles.Code().Render(key),
			styles.Muted().Render(displayDest))
	}
}
