package secrets

import (
	"fmt"
	"io"

	internalIO "github.com/kloudkit/ws-cli/internals/io"
	internalSecrets "github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Generate a workspace password hash for authentication",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := getOutputConfig(cmd)
		return generateLoginHash(cmd, cfg)
	},
}

func generateLoginHash(cmd *cobra.Command, cfg outputConfig) error {
	password, err := internalIO.ReadPasswordFromReader(cmd.InOrStdin())
	if err != nil {
		return err
	}

	hash, err := internalSecrets.HashPasswordForWorkspace(password)
	if err != nil {
		return err
	}

	return handleCustomOutput(cmd, cfg, hash, "✓ Password hash written to "+cfg.file, func(out io.Writer) {
		fmt.Fprintf(out, "%s\n", styles.Header().Render("Workspace Password Hash"))
		fmt.Fprintf(out, "  %s\n", styles.Code().Render(hash))
		fmt.Fprintf(out, "%s\n", styles.Muted().Render("💡 Use this hash for WS_AUTH_PASSWORD_HASHED"))
	})
}
