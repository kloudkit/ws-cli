package feature

import (
	"testing"

	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
)

func TestBuildVars(t *testing.T) {
	cases := [][]string{nil, {"--skip-extensions", "--skip-completion", "--skip-repository"}}
	wants := []map[string]any{{}, {"skip_extensions": "true", "skip_completion": "true", "skip_repository": "true"}}

	for _, section := range skippableSections {
		cases = append(cases, []string{"--" + section.flag})
		wants = append(wants, map[string]any{section.key: "true"})
	}

	cases = append(cases, []string{"--skip-extensions", "--opt", "version=14"}, []string{"--opt", "skip_extensions=false", "--skip-extensions"})
	wants = append(wants, map[string]any{"skip_extensions": "true", "version": "14"}, map[string]any{"skip_extensions": "true"})

	for i, args := range cases {
		cmd := &cobra.Command{Use: "install"}
		addInstallFlags(cmd)
		assert.NilError(t, cmd.ParseFlags(args))
		assert.DeepEqual(t, buildVars(cmd), wants[i])
	}
}
