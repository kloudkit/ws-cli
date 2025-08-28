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

func init() {
	showCmd.Flags().Bool("local", false, "Show the local version of the template (if applied)")

	TemplateCmd.AddCommand(showCmd)
}

func runShow(cmd *cobra.Command, args []string) error {
	templateName := args[0]
	local, _ := cmd.Flags().GetBool("local")

	content, err := template.ShowTemplate(templateName, local)
	if err != nil {
		return err
	}

	if styles.ColorEnabled {
		if local {
			fmt.Fprintf(cmd.OutOrStdout(), "%sLocal template '%s':%s\n\n", styles.ColorHeader, templateName, styles.ColorText)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "%sTemplate '%s':%s\n\n", styles.ColorHeader, templateName, styles.ColorText)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s%s%s\n", styles.ColorMuted, content, styles.ColorText)
	} else {
		if local {
			fmt.Fprintf(cmd.OutOrStdout(), "Local template '%s':\n\n", templateName)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Template '%s':\n\n", templateName)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", content)
	}

	return nil
}
