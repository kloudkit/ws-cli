package seed

import (
	"io"
	"os"

	"github.com/kloudkit/ws-cli/internals/seed"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var applyCmd = &cobra.Command{
	Use:          "apply [dest...]",
	Short:        "Project seed content onto the filesystem",
	SilenceUsage: true,
	RunE:         runApply,
}

func runApply(cmd *cobra.Command, args []string) error {
	source, _ := cmd.Flags().GetString("source")
	force, _ := cmd.Flags().GetBool("force")
	master, _ := cmd.Flags().GetString("master")

	resolved, err := seed.ResolveSource(source)
	if err != nil {
		return err
	}

	return seed.Apply(seed.Options{
		Source:    resolved,
		Force:     force,
		Dests:     args,
		MasterKey: master,
		Out:       cmd.OutOrStdout(),
		Styled:    isTerminal(cmd.OutOrStdout()),
	})
}

func isTerminal(out io.Writer) bool {
	file, ok := out.(*os.File)

	return ok && term.IsTerminal(int(file.Fd()))
}

func init() {
	applyCmd.Flags().Bool("force", false, "Overwrite existing destinations")
	applyCmd.Flags().String("master", "", "Master key or path to key file")

	SeedCmd.AddCommand(applyCmd)
}
