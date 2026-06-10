package feature

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kloudkit/ws-cli/internals/features"
	"gotest.tools/v3/assert"
)

func _runNew(t *testing.T, args ...string) string {
	t.Helper()

	var out bytes.Buffer
	FeatureCmd.SetOut(&out)
	FeatureCmd.SetErr(&out)
	FeatureCmd.SetArgs(append([]string{"new"}, args...))
	assert.NilError(t, FeatureCmd.Execute())

	return out.String()
}

func _parseScaffold(t *testing.T, output string) *features.Feature {
	t.Helper()

	file := filepath.Join(t.TempDir(), "scaffold.yaml")
	assert.NilError(t, os.WriteFile(file, []byte(output), 0644))

	feature, err := features.ParseFeatureFile(file)
	assert.NilError(t, err)

	return feature
}

func TestScaffoldRoundTrips(t *testing.T) {
	feature := _parseScaffold(t, _runNew(t))

	assert.Assert(t, feature.Name != "")
	assert.Assert(t, feature.Description != "")
}

func TestScaffoldHasRequiredPlayShape(t *testing.T) {
	out := _runNew(t)

	assert.Assert(t, strings.Contains(out, "hosts: workspace"))
	assert.Assert(t, strings.Contains(out, "gather_facts: false"))
}

func TestScaffoldNameArgSeedsName(t *testing.T) {
	feature := _parseScaffold(t, _runNew(t, "redis"))

	assert.Equal(t, "Install redis", feature.Description)
}

func TestScaffoldNoArgPlaceholder(t *testing.T) {
	feature := _parseScaffold(t, _runNew(t))

	assert.Equal(t, "Install <feature>", feature.Description)
}
