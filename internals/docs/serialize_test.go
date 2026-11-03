package docs

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
)

func leaf(use string, annotations map[string]string) *cobra.Command {
	return &cobra.Command{Use: use, Annotations: annotations, Run: func(*cobra.Command, []string) {}}
}

func serialize(t *testing.T, root *cobra.Command, children ...*cobra.Command) string {
	t.Helper()
	root.AddCommand(children...)

	out, err := Serialize(root)
	assert.NilError(t, err)

	return string(out)
}

func TestSerializeOmitsAbsolutePathDefaults(t *testing.T) {
	root := leaf("demo", nil)
	root.Flags().String("source", "/home/runner/.ws/seed.d", "Seed source directory")
	root.Flags().String("bind", "0.0.0.0", "Bind address")
	root.Flags().Int("port", 38080, "Port")

	rendered := serialize(t, root)
	assert.Assert(t, !strings.Contains(rendered, "/home/runner/.ws/seed.d"), rendered)
	assert.Assert(t, strings.Contains(rendered, "default: 0.0.0.0"), rendered)
	assert.Assert(t, strings.Contains(rendered, `default: "38080"`), rendered)
}

func TestSerializeAnnotations(t *testing.T) {
	for _, key := range []string{"since", "deprecated"} {
		rendered := serialize(t, &cobra.Command{Use: "demo"}, leaf("tagged", map[string]string{key: "0.4.1"}), leaf("plain", nil))
		assert.Assert(t, strings.Contains(rendered, key+": 0.4.1"), rendered)
		assert.Equal(t, strings.Count(rendered, key+":"), 1, rendered)
	}

	rendered := serialize(t, leaf("demo", map[string]string{"since": "0.2.0", "deprecated": "0.5.0"}))
	assert.Assert(t, strings.Contains(rendered, "deprecated: 0.5.0"), rendered)
	assert.Assert(t, !strings.Contains(rendered, "since:"), rendered)
}

func TestSerializeSortsChildrenAndOmitsHidden(t *testing.T) {
	ghost := leaf("ghost", nil)
	ghost.Hidden = true

	rendered := serialize(t, &cobra.Command{Use: "demo"}, leaf("zebra", nil), leaf("alpha", nil), ghost)
	assert.Assert(t, strings.Index(rendered, "demo alpha") < strings.Index(rendered, "demo zebra"), rendered)
	assert.Assert(t, !strings.Contains(rendered, "ghost"), rendered)
}
