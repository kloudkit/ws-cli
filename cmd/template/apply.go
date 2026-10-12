package template

import (
	"path/filepath"

	"github.com/kloudkit/ws-cli/internals/path"

	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/kloudkit/ws-cli/internals/template"
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:               "apply <template>",
	Annotations:       map[string]string{"since": "0.2.0"},
	Short:             "Apply a configuration template to the current project",
	Long:              "Copy a template into the project — a shared config like ruff or markdownlint — to --path (the current directory by default). --force overwrites an existing file.",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: templateCompletion,
	RunE:              runApply,
}

func runApply(cmd *cobra.Command, args []string) error {
	templateName := args[0]
	targetPath, _ := cmd.Flags().GetString("path")
	force, _ := cmd.Flags().GetBool("force")

	if err := template.ApplyTemplate(templateName, targetPath, force); err != nil {
		return err
	}

	config := template.SupportedTemplates[templateName]
	out := cmd.OutOrStdout()

	target := filepath.Join(targetPath, config.OutputName)
	if abs, err := filepath.Abs(target); err == nil {
		target = abs
	}

	styles.PrintSuccess(out, "Template applied")
	styles.PrintKeyValues(out,
		styles.Pair{Key: "Template", Value: templateName},
		styles.Pair{Key: "Source", Value: path.ShortenHomePath(path.ResolveConfigPath(config.SourcePath))},
		styles.Pair{Key: "Target", Value: target},
	)

	styles.PrintHints(out, styles.Hint{Command: "ws template show --local " + templateName, Description: "View applied template"})

	return nil
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

	TemplateCmd.AddCommand(applyCmd)
}
