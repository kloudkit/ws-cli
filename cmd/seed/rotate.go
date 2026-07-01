package seed

import (
	"github.com/kloudkit/ws-cli/internals/seed"
	"github.com/spf13/cobra"
)

var rotateCmd = &cobra.Command{
	Use:          "rotate",
	Short:        "Re-encrypt managed secrets under a new master key",
	Long:         "Re-encrypt every managed secret from the old master key (--master) to a new one (--new-master), in place. All-or-nothing: it verifies every secret decrypts before writing anything.",
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
