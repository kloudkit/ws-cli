package seed

import (
	"github.com/kloudkit/ws-cli/internals/seed"
	"github.com/spf13/cobra"
)

var rotateCmd = &cobra.Command{
	Use:          "rotate",
	Short:        "Re-encrypt managed secrets under a new master key",
	SilenceUsage: true,
	RunE:         runRotate,
}

func runRotate(cmd *cobra.Command, args []string) error {
	source, _ := cmd.Flags().GetString("source")
	master, _ := cmd.Flags().GetString("master")
	newMaster, _ := cmd.Flags().GetString("new-master")

	resolved, err := seed.ResolveSource(source)
	if err != nil {
		return err
	}

	return seed.Rotate(seed.RotateOptions{
		Source:       resolved,
		MasterKey:    master,
		NewMasterKey: newMaster,
		Out:          cmd.OutOrStdout(),
		Styled:       isTerminal(cmd.OutOrStdout()),
	})
}

func init() {
	rotateCmd.Flags().String("master", "", "Current master key or path to key file")
	rotateCmd.Flags().String("new-master", "", "New master key or path to key file")

	SeedCmd.AddCommand(rotateCmd)
}
