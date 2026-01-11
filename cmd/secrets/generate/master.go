package generate

import (
	"github.com/spf13/cobra"
)

var masterCmd = &cobra.Command{
	Use:   "master",
	Short: "Generate a cryptographically secure master key",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := getOutputConfig(cmd)
		keyLength, _ := cmd.Flags().GetInt("length")
		return generateMasterKey(cmd, cfg, keyLength)
	},
}

func init() {
	masterCmd.Flags().Int("length", 32, "Key length in bytes")
}
