package info

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kloudkit/ws-cli/internals/env"
	"github.com/kloudkit/ws-cli/internals/styles"
)

func showEnvironment(writer io.Writer) {
	allVars := env.GetAll()
	var wsVars [][]string
	for _, key := range slices.Sorted(maps.Keys(allVars)) {
		if strings.HasPrefix(key, "WS_") {
			wsVars = append(wsVars, []string{key, allVars[key]})
		}
	}

	fmt.Fprintf(writer, "%s\n", styles.TitleWithCount("Workspace Variables", len(wsVars)))

	fmt.Fprintf(writer, "%s\n\n", styles.Table().Rows(wsVars...).Render())
}

var envCmd = &cobra.Command{
	Use:         "env",
	Annotations: map[string]string{"since": "0.2.0"},
	Short:       "Display effective workspace environment variables",
	Long:        "Print every WS_* variable in effect, sorted — the resolved environment the workspace booted with.",
	RunE: func(cmd *cobra.Command, args []string) error {
		showEnvironment(cmd.OutOrStdout())
		return nil
	},
}

func init() {
	InfoCmd.AddCommand(envCmd)
}
