package seed

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func decodeBack(t *testing.T, data []byte, dest string) map[string]any {
	t.Helper()

	c, err := codecFor(dest)
	assert.NilError(t, err)

	out := map[string]any{}
	assert.NilError(t, c.unmarshal(data, &out))

	return out
}

func TestMergeContent(t *testing.T) {
	tests := []struct {
		name     string
		dest     string
		existing string
		fragment string
	}{
		{"json", "config.json", `{"keep":1,"list":[1,2,3]}`, `{"list":[9],"add":true}`},
		{"yaml", "config.yaml", "keep: 1\nlist: [1, 2, 3]\n", "list: [9]\nadd: true\n"},
		{"toml", "config.toml", "keep = 1\nlist = [1, 2, 3]\n", "list = [9]\nadd = true\n"},
	}

	for _, tt := range tests {
		t.Run("ListReplace/"+tt.name, func(t *testing.T) {
			merged, err := mergeContent([]byte(tt.existing), []byte(tt.fragment), tt.dest)
			assert.NilError(t, err)

			out := decodeBack(t, merged, tt.dest)

			list, ok := out["list"].([]any)
			assert.Assert(t, ok)
			assert.Equal(t, len(list), 1)
			assert.Assert(t, out["keep"] != nil)
			assert.Assert(t, out["add"] != nil)
		})
	}

	t.Run("ScalarVsMapConflict", func(t *testing.T) {
		_, err := mergeContent([]byte(`{"k":"scalar"}`), []byte(`{"k":{"nested":1}}`), "config.json")
		assert.ErrorContains(t, err, "merge conflict at key")
	})

	t.Run("MapVsScalarConflict", func(t *testing.T) {
		_, err := mergeContent([]byte(`{"k":{"nested":1}}`), []byte(`{"k":"scalar"}`), "config.json")
		assert.ErrorContains(t, err, "merge conflict at key")
	})

	t.Run("JSONNumberFidelity", func(t *testing.T) {
		merged, err := mergeContent([]byte(`{"n":1}`), []byte(`{"n":2}`), "config.json")
		assert.NilError(t, err)
		assert.Assert(t, strings.Contains(string(merged), `"n": 2`))
	})

	t.Run("LargeIntExistingSide", func(t *testing.T) {
		merged, err := mergeContent([]byte(`{"big":9007199254740993}`), []byte(`{"x":1}`), "config.json")
		assert.NilError(t, err)

		out := string(merged)
		assert.Assert(t, strings.Contains(out, "9007199254740993"))
		assert.Assert(t, !strings.Contains(out, "e+"))
		assert.Assert(t, !strings.Contains(out, "E"))
	})

	t.Run("LargeIntFragmentSide", func(t *testing.T) {
		merged, err := mergeContent([]byte(`{"x":1}`), []byte(`{"big":9223372036854775807}`), "config.json")
		assert.NilError(t, err)

		out := string(merged)
		assert.Assert(t, strings.Contains(out, "9223372036854775807"))
		assert.Assert(t, !strings.Contains(out, "e+"))
	})

	t.Run("YamlTomlLargeIntControl", func(t *testing.T) {
		cases := []struct {
			dest     string
			existing string
			fragment string
		}{
			{"config.yaml", "keep: 1\n", "big: 9223372036854775807\n"},
			{"config.toml", "keep = 1\n", "big = 9223372036854775807\n"},
		}

		for _, tc := range cases {
			merged, err := mergeContent([]byte(tc.existing), []byte(tc.fragment), tc.dest)
			assert.NilError(t, err)
			assert.Assert(t, strings.Contains(string(merged), "9223372036854775807"))
		}
	})

	t.Run("NegativeZeroFloatStillMerge", func(t *testing.T) {
		merged, err := mergeContent([]byte(`{"x":1}`), []byte(`{"neg":-5,"zero":0,"frac":1.5}`), "config.json")
		assert.NilError(t, err)

		out := string(merged)
		assert.Assert(t, strings.Contains(out, `"neg": -5`))
		assert.Assert(t, strings.Contains(out, `"zero": 0`))
		assert.Assert(t, strings.Contains(out, `"frac": 1.5`))
	})

	t.Run("UnknownExtensionRejected", func(t *testing.T) {
		_, err := mergeContent([]byte("a"), []byte("b"), "config.ini")
		assert.ErrorContains(t, err, "cannot infer merge format")
	})
}
