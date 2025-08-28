package template

import (
	"fmt"
	"strings"

	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/kloudkit/ws-cli/internals/template"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <template>",
	Short: "Display the contents of a configuration template",
	Long: fmt.Sprintf(`Display the contents of a configuration template.

  Available templates: %s`, strings.Join(template.GetTemplateNames(), ", ")),
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: templateCompletion,
	RunE:              runShow,
}

func runShow(cmd *cobra.Command, args []string) error {
	templateName := args[0]
	local, _ := cmd.Flags().GetBool("local")

	content, err := template.ShowTemplate(templateName, local)
	if err != nil {
		return err
	}

	headerText := "Template"
	if local {
		headerText = "Local Template"
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n\n", styles.SuccessBadge().Render(fmt.Sprintf("%s '%s'", headerText, templateName)))
	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.Code().MarginLeft(2).Render(content))

	return nil
}

func init() {
	showCmd.Flags().Bool("local", false, "Show the local version of the template (if applied)")

	TemplateCmd.AddCommand(showCmd)
}
