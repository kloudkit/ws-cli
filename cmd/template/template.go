package template

import (
	"github.com/spf13/cobra"
)

var TemplateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage static configuration files",
	Long:  "Copy and manage configuration files stored in global locations",
}
