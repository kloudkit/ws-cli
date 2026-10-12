package seed

import (
	"fmt"
	"strings"

	"github.com/kloudkit/ws-cli/internals/seed"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:         "ls",
	Short:       "List seed destinations and their behaviors",
	Long:        "List what apply would write — each destination with its operation and whether it carries a secret or a template — without touching the filesystem.",
	Annotations: map[string]string{"since": "0.5.0"},
	RunE:        runLs,
}

func runLs(cmd *cobra.Command, args []string) error {
	source, _ := cmd.Flags().GetString("source")

	resolved, err := seed.ResolveSource(source, cmd.Flags().Changed("source"))
	if err != nil {
		return err
	}

	plan, err := seed.BuildPlan(resolved, false)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()

	if len(plan.Ops) == 0 {
		styles.PrintEmpty(out, fmt.Sprintf("No seed entries in [%s]", resolved))
		return nil
	}

	styles.PrintTitleCount(out, "Seed Entries", len(plan.Ops))

	items := make([]styles.DescriptionItem, len(plan.Ops))
	for i, op := range plan.Ops {
		items[i] = styles.DescriptionItem{Name: op.Dest, Description: describe(op)}
	}

	styles.PrintList(out, items)

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
