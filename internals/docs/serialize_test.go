package docs

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
)

func TestSerializeOmitsAbsolutePathDefaults(t *testing.T) {
	root := &cobra.Command{Use: "demo", Run: func(*cobra.Command, []string) {}}
	root.Flags().String("source", "/home/runner/.ws/seed.d", "Seed source directory")
	root.Flags().String("bind", "0.0.0.0", "Bind address")
	root.Flags().Int("port", 38080, "Port")

	out, err := Serialize(root)
	assert.NilError(t, err)

	rendered := string(out)
	assert.Assert(t, !strings.Contains(rendered, "/home/runner/.ws/seed.d"),
		"env-resolved absolute-path defaults must be omitted:\n%s", rendered)
	assert.Assert(t, strings.Contains(rendered, "default: 0.0.0.0"))
	assert.Assert(t, strings.Contains(rendered, "default: \"38080\""))
}

func TestSerializeSortsChildrenAndOmitsHidden(t *testing.T) {
	root := &cobra.Command{Use: "demo"}
	root.AddCommand(&cobra.Command{Use: "zebra", Run: func(*cobra.Command, []string) {}})
	root.AddCommand(&cobra.Command{Use: "alpha", Run: func(*cobra.Command, []string) {}})
	root.AddCommand(&cobra.Command{Use: "ghost", Hidden: true, Run: func(*cobra.Command, []string) {}})

	out, err := Serialize(root)
	assert.NilError(t, err)

	rendered := string(out)
	assert.Assert(t, strings.Index(rendered, "demo alpha") < strings.Index(rendered, "demo zebra"),
		"children must sort by name:\n%s", rendered)
	assert.Assert(t, !strings.Contains(rendered, "ghost"), "hidden commands must be omitted")
}
