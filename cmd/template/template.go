package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kloudkit/ws-cli/cmd/cli"
	"github.com/kloudkit/ws-cli/internals/path"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/kloudkit/ws-cli/internals/template"
	"github.com/spf13/cobra"
)

var applyCmd = cli.New("apply <template>", "0.2.0", "Apply a configuration template to the current project",
	"Copy a template into the project — a shared config like ruff or markdownlint — to --path (the current directory by default). --force overwrites an existing file.",
	func(cmd *cobra.Command, args []string) error {
		name, targetPath := args[0], cli.String(cmd, "path")

		if err := template.ApplyTemplate(name, targetPath, cli.Bool(cmd, "force")); err != nil {
			return err
		}

		config := template.SupportedTemplates[name]
		target, _ := filepath.Abs(filepath.Join(targetPath, config.OutputName))
		out := cmd.OutOrStdout()

		styles.PrintSuccess(out, "Template applied",
			styles.Pair{Key: "Template", Value: name},
			styles.Pair{Key: "Source", Value: sourcePath(config)},
			styles.Pair{Key: "Target", Value: target},
		)
		styles.PrintHints(out, styles.Hint{Command: "ws template show --local " + name, Description: "View applied template"})

		return nil
	})

var listCmd = cli.New("list", "0.2.0", "List all available configuration templates",
	"List the templates you can apply, with their source paths and whether one is already applied here.",
	func(cmd *cobra.Command, args []string) error {
		names := template.GetTemplateNames()
		out := cmd.OutOrStdout()

		items := make([]styles.DescriptionItem, len(names))
		for i, name := range names {
			config := template.SupportedTemplates[name]
			items[i] = styles.DescriptionItem{Name: name, Description: sourcePath(config)}

			if _, err := os.Stat(config.OutputName); err == nil {
				items[i].Description += " " + styles.Success.Render("(applied)")
			}
		}

		styles.PrintTitle(out, "Templates Available", len(names))
		styles.PrintList(out, items)
		styles.PrintHints(out,
			styles.Hint{Command: "ws template apply <name>", Description: "Apply a template"},
			styles.Hint{Command: "ws template show <name>", Description: "View template contents"},
			styles.Hint{Command: "ws template show --local <name>", Description: "View applied template"},
		)

		return nil
	})

var showCmd = cli.New("show <template>", "0.2.0", "Display the contents of a configuration template",
	fmt.Sprintf("Display the contents of a configuration template.\n\n  Available templates: %s", strings.Join(template.GetTemplateNames(), ", ")),
	func(cmd *cobra.Command, args []string) error {
		local := cli.Bool(cmd, "local")

		content, err := template.ShowTemplate(args[0], local)
		if err != nil {
			return err
		}

		title := "Template "
		if local {
			title = "Local Template "
		}

		styles.PrintTitle(cmd.OutOrStdout(), title+args[0])
		styles.PrintCode(cmd.OutOrStdout(), content)

		return nil
	})

var TemplateCmd = cli.Group("template", "0.2.0", "Manage static configuration files",
	"Copy shared configuration files (linters, formatters) from their global locations into a project, and inspect what they hold.",
	`# List available templates
ws template list

# Apply the ruff config to the current project
ws template apply ruff`,
	applyCmd, listCmd, showCmd)

func sourcePath(config template.Config) string {
	return path.ShortenHomePath(path.ResolveConfigPath(config.SourcePath))
}

func templateCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return template.GetTemplateNames(), cobra.ShellCompDirectiveNoFileComp
}

func init() {
	applyCmd.Flags().String("path", ".", "Target directory path")
	applyCmd.Flags().BoolP("force", "f", false, "Overwrite existing files")
	showCmd.Flags().Bool("local", false, "Show the local version of the template (if applied)")
	applyCmd.Args, showCmd.Args = cobra.ExactArgs(1), cobra.ExactArgs(1)
	applyCmd.ValidArgsFunction, showCmd.ValidArgsFunction = templateCompletion, templateCompletion
	listCmd.Aliases = []string{"ls"}
}
