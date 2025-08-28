package template

import (
	"fmt"

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

	listItems := make([]any, len(names))
	for i, name := range names {
		config := templates[name]
		resolvedPath := path.ResolveConfigPath(config.SourcePath)
		location := styles.Muted().Render(fmt.Sprintf("(%s)", path.ShortenHomePath(resolvedPath)))
		listItems[i] = fmt.Sprintf("%s %s", styles.Key().Render(name), location)
	}

	l := styles.List(listItems...)

	fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n", styles.Success().Render("Available Templates"))
	fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n", l.String())
	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.Muted().Render("Use 'ws-cli template apply <name>' to apply a template"))
	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.Muted().Render("Use 'ws-cli template show <name>' to view template contents"))

	return nil
}

func init() {
	TemplateCmd.AddCommand(listCmd)
}
