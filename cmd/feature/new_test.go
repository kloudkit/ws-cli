package feature

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/cmd/clitest"

	"github.com/kloudkit/ws-cli/internals/features"
	"gotest.tools/v3/assert"
)

func TestScaffold(t *testing.T) {
	for args, want := range map[string]string{"": "Install <feature>", "redis": "Install redis"} {
		out, err := clitest.Run(FeatureCmd, "", append([]string{"new"}, strings.Fields(args)...)...)
		assert.NilError(t, err)
		assert.Assert(t, strings.Contains(out, "hosts: workspace"))
		assert.Assert(t, strings.Contains(out, "gather_facts: false"))

		file := filepath.Join(t.TempDir(), "scaffold.yaml")
		assert.NilError(t, os.WriteFile(file, []byte(out), 0o644))

		feature, err := features.ParseFeatureFile(file)
		assert.NilError(t, err)
		assert.Assert(t, feature.Name != "")
		assert.Equal(t, feature.Description, want)
	}
}
