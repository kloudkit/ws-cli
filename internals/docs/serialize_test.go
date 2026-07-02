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

func TestSerializeEmitsSinceAnnotation(t *testing.T) {
	root := &cobra.Command{Use: "demo"}
	root.AddCommand(&cobra.Command{
		Use:         "tagged",
		Annotations: map[string]string{"since": "0.4.1"},
		Run:         func(*cobra.Command, []string) {},
	})
	root.AddCommand(&cobra.Command{Use: "plain", Run: func(*cobra.Command, []string) {}})

	out, err := Serialize(root)
	assert.NilError(t, err)

	rendered := string(out)
	assert.Assert(t, strings.Contains(rendered, "since: 0.4.1"),
		"since annotation must be emitted:\n%s", rendered)
	assert.Equal(t, strings.Count(rendered, "since:"), 1,
		"unannotated commands must omit since:\n%s", rendered)
}

func TestSerializeEmitsDeprecatedAnnotation(t *testing.T) {
	root := &cobra.Command{Use: "demo"}
	root.AddCommand(&cobra.Command{
		Use:         "tagged",
		Annotations: map[string]string{"deprecated": "0.5.0"},
		Run:         func(*cobra.Command, []string) {},
	})
	root.AddCommand(&cobra.Command{Use: "plain", Run: func(*cobra.Command, []string) {}})

	out, err := Serialize(root)
	assert.NilError(t, err)

	rendered := string(out)
	assert.Assert(t, strings.Contains(rendered, "deprecated: 0.5.0"),
		"deprecated annotation must be emitted:\n%s", rendered)
	assert.Equal(t, strings.Count(rendered, "deprecated:"), 1,
		"unannotated commands must omit deprecated:\n%s", rendered)
}

func TestSerializeDeprecatedSupersedesSince(t *testing.T) {
	root := &cobra.Command{
		Use:         "demo",
		Annotations: map[string]string{"since": "0.2.0", "deprecated": "0.5.0"},
		Run:         func(*cobra.Command, []string) {},
	}

	out, err := Serialize(root)
	assert.NilError(t, err)

	rendered := string(out)
	assert.Assert(t, strings.Contains(rendered, "deprecated: 0.5.0"),
		"deprecated annotation must be emitted:\n%s", rendered)
	assert.Assert(t, !strings.Contains(rendered, "since:"),
		"a deprecated command must drop since:\n%s", rendered)
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
