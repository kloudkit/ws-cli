package info

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kloudkit/ws-cli/internals/env"
	"github.com/kloudkit/ws-cli/internals/styles"
)

func showEnvironment(writer io.Writer) {
	allVars := env.GetAll()
	var wsVars [][]string
	for key, value := range allVars {
		if strings.HasPrefix(key, "WS_") {
			wsVars = append(wsVars, []string{key, value})
		}
	}

	fmt.Fprintf(writer, "%s\n", styles.TitleWithCount("Workspace Variables", len(wsVars)))

	sort.Slice(wsVars, func(i, j int) bool {
		return wsVars[i][0] < wsVars[j][0]
	})

	fmt.Fprintf(writer, "%s\n\n", styles.Table().Rows(wsVars...).Render())
}

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Display effective workspace environment variables",
	RunE: func(cmd *cobra.Command, args []string) error {
		showEnvironment(cmd.OutOrStdout())
		return nil
	},
}

func init() {
	InfoCmd.AddCommand(envCmd)
}
