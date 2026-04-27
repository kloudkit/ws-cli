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

	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.TitleWithCount("Templates Available", len(names)))

	listItems := make([]any, len(names))
	for i, name := range names {
		config := templates[name]
		resolvedPath := path.ResolveConfigPath(config.SourcePath)
		location := styles.Muted().Render(fmt.Sprintf("(%s)", path.ShortenHomePath(resolvedPath)))

		localIndicator := ""
		if _, err := os.Stat(config.OutputName); err == nil {
			localIndicator = styles.Success().Render("(applied)")
		}

		listItems[i] = fmt.Sprintf("%s %s %s", styles.Key().Render(name), location, localIndicator)
	}

	l := styles.List(listItems...)

	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", l.String())

	styles.PrintHints(cmd.OutOrStdout(), [][]string{
		{"ws-cli template apply <name>", "Apply a template"},
		{"ws-cli template show <name>", "View template contents"},
		{"ws-cli template show --local <name>", "View applied template"},
	})

	return nil
}

func init() {
	TemplateCmd.AddCommand(listCmd)
}
