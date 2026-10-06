package secrets

import (
	internalIO "github.com/kloudkit/ws-cli/internals/io"
	internalSecrets "github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:         "login",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "Generate a workspace password hash for authentication",
	Long:        "Prompt for a password and print its hash for the workspace server login (WS_AUTH_PASSWORD_HASHED). Store the hash, never the password.",
	Args:        cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		password, err := internalIO.ReadPasswordFromReader(cmd.InOrStdin())
		if err != nil {
			return err
		}

		hash, err := internalSecrets.HashPasswordForWorkspace(password)
		if err != nil {
			return err
		}

		file, _ := cmd.Flags().GetString("output")

		return emit(cmd, hash, "Password hash written to "+file, true,
			printKey("Workspace Password Hash", hash, "Use this hash for WS_AUTH_PASSWORD_HASHED"))
	},
}
