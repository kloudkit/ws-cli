package info

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/kloudkit/ws-cli/internals/env"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

func showEnvironment(writer io.Writer) {
	fmt.Fprintln(writer, styles.Header().Render("Environment Variables"))

	var envVars [][]string
	for key, value := range env.GetAll() {
		if strings.HasPrefix(key, "WS_") {
			envVars = append(envVars, []string{key, value})
		}
	}

	if len(envVars) == 0 {
		fmt.Fprintln(writer, styles.Warning().Render("  No environment variables found"))
		return
	}

	sort.Slice(envVars, func(i, j int) bool {
		return envVars[i][0] < envVars[j][0]
	})

	t := styles.Table("Variable", "Value").
		Rows(envVars...)

	fmt.Fprintln(writer)
	fmt.Fprintln(writer, t.Render())
}

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Display effective workspace environment variables",
	Run: func(cmd *cobra.Command, args []string) {
		showEnvironment(cmd.OutOrStdout())
	},
}

func init() {
	InfoCmd.AddCommand(envCmd)
}
