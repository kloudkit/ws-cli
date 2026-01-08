package template

import (
	"fmt"

	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/kloudkit/ws-cli/internals/template"
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:               "apply <template>",
	Short:             "Apply a configuration template to the current project",
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

	config, _ := template.GetTemplate(templateName)
	sourcePath := template.SupportedTemplates[templateName].SourcePath

	styles.PrintSuccess(cmd.OutOrStdout(), "Template applied successfully")
	styles.PrintKeyValue(cmd.OutOrStdout(), "Template", templateName)
	styles.PrintKeyCode(cmd.OutOrStdout(), "Source", sourcePath)
	styles.PrintKeyCode(cmd.OutOrStdout(), "Target", fmt.Sprintf("%s/%s", targetPath, config.OutputName))

	styles.PrintHints(cmd.OutOrStdout(), [][]string{
		{"ws-cli template show --local " + templateName, "View applied template"},
	})

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
