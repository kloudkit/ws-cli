package template

import (
	"fmt"
	"os"

	"github.com/kloudkit/ws-cli/internals/path"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/kloudkit/ws-cli/internals/template"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all available configuration templates",
	Aliases: []string{"ls"},
	RunE:    runList,
}

func runList(cmd *cobra.Command, args []string) error {
	templates := template.SupportedTemplates
	names := template.GetTemplateNames()

	fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n\n", styles.InfoBadge().Render(fmt.Sprintf("%d TEMPLATES AVAILABLE", len(names))))

	listItems := make([]any, len(names))
	for i, name := range names {
		config := templates[name]
		resolvedPath := path.ResolveConfigPath(config.SourcePath)
		location := styles.Muted().Render(fmt.Sprintf("(%s)", path.ShortenHomePath(resolvedPath)))

		localIndicator := ""
		if _, err := os.Stat(config.OutputName); err == nil {
			localIndicator = styles.InfoBadge().Render("APPLIED")
		}

		listItems[i] = fmt.Sprintf("%s %s %s", styles.Key().Render(name), location, localIndicator)
	}

	l := styles.List(listItems...)

	fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n", l.String())

	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.Muted().Render("🚀 Quick actions:"))
	fmt.Fprintf(cmd.OutOrStdout(), "  %s %s\n", styles.Code().Render("ws-cli template apply <name>"), styles.Muted().Render("- Apply a template"))
	fmt.Fprintf(cmd.OutOrStdout(), "  %s %s\n", styles.Code().Render("ws-cli template show <name>"), styles.Muted().Render("- View template contents"))
	fmt.Fprintf(cmd.OutOrStdout(), "  %s %s\n", styles.Code().Render("ws-cli template show --local <name>"), styles.Muted().Render("- View applied template"))

	return nil
}

func init() {
	TemplateCmd.AddCommand(listCmd)
}
