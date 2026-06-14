package feature

import (
	"testing"

	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
)

func _buildVars(t *testing.T, args ...string) map[string]any {
	t.Helper()

	cmd := &cobra.Command{Use: "install"}
	addInstallFlags(cmd)
	assert.NilError(t, cmd.ParseFlags(args))

	return buildVars(cmd)
}

func TestBuildVarsSkipFlagsLowerToExtraVars(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want map[string]any
	}{
		{
			name: "no flags emits no skip keys",
			args: nil,
			want: map[string]any{},
		},
		{
			name: "skip-extensions alone",
			args: []string{"--skip-extensions"},
			want: map[string]any{"skip_extensions": "true"},
		},
		{
			name: "skip-completion alone",
			args: []string{"--skip-completion"},
			want: map[string]any{"skip_completion": "true"},
		},
		{
			name: "skip-repository alone",
			args: []string{"--skip-repository"},
			want: map[string]any{"skip_repository": "true"},
		},
		{
			name: "all three at once",
			args: []string{"--skip-extensions", "--skip-completion", "--skip-repository"},
			want: map[string]any{
				"skip_extensions": "true",
				"skip_completion": "true",
				"skip_repository": "true",
			},
		},
		{
			name: "skip flag composes with --opt without clobber",
			args: []string{"--skip-extensions", "--opt", "version=14"},
			want: map[string]any{"skip_extensions": "true", "version": "14"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.DeepEqual(t, _buildVars(t, tc.args...), tc.want)
		})
	}
}

func TestBuildVarsSkipFlagWinsOverOptCollision(t *testing.T) {
	vars := _buildVars(t, "--opt", "skip_extensions=false", "--skip-extensions")

	assert.DeepEqual(t, vars, map[string]any{"skip_extensions": "true"})
}
