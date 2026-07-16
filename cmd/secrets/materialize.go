package secrets

import (
	internalSecrets "github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var materializeCmd = &cobra.Command{
	Use:         "materialize",
	Annotations: map[string]string{"since": "next"},
	Short:       "Project the configured master key to its conventional secret path",
	Long:        "Persist WS_SECRETS_MASTER_KEY to /run/secrets/workspace/secrets/master_key so the key outlives the editor's environment scrub. A no-op when the key is unset or the path already holds one.",
	Args:        cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := internalSecrets.MaterializeMasterKey()
		if err != nil {
			return err
		}

		if path != "" {
			styles.PrintSuccess(cmd.OutOrStdout(), "Master key projected to "+path)
		}

		return nil
	},
}
