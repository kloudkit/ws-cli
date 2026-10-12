package template

import (
	"fmt"
	"strings"

	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/kloudkit/ws-cli/internals/template"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:         "show <template>",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "Display the contents of a configuration template",
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

	out := cmd.OutOrStdout()

	styles.PrintTitle(out, headerText+" "+templateName)

	for line := range strings.SplitSeq(strings.TrimRight(content, "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			fmt.Fprintln(out)
			continue
		}

		fmt.Fprintf(out, "    %s\n", styles.Value().Render(line))
	}

	return nil
}

func init() {
	showCmd.Flags().Bool("local", false, "Show the local version of the template (if applied)")

	TemplateCmd.AddCommand(showCmd)
}
