package seed

import (
	"strings"

	"github.com/kloudkit/ws-cli/internals/seed"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:         "ls",
	Short:       "List seed destinations and their behaviors",
	Long:        "List what apply would write — each destination with its operation and whether it carries a secret or a template — without touching the filesystem.",
	Annotations: map[string]string{"since": "next"},
	RunE:        runLs,
}

func runLs(cmd *cobra.Command, args []string) error {
	source, _ := cmd.Flags().GetString("source")

	resolved, err := seed.ResolveSource(source)
	if err != nil {
		return err
	}

	plan, err := seed.BuildPlan(resolved, false)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	for _, op := range plan.Ops {
		styles.PrintKeyValue(out, op.Dest, describe(op))
	}

	return nil
}

func describe(op seed.ResolvedOp) string {
	parts := []string{string(op.Op)}

	if op.Secret {
		parts = append(parts, "secret")
	}

	if op.Template {
		parts = append(parts, "template")
	}

	return strings.Join(parts, " ")
}

func init() {
	SeedCmd.AddCommand(lsCmd)
}
