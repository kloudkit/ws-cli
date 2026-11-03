package seed

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestEnsureBlock(t *testing.T) {
	tests := []struct{ name, existing, body, comment, want, err string }{
		{name: "AppendsWhenAbsent", body: "export FOO=1\n", want: "# >>> ws-seed >>>\nexport FOO=1\n# <<< ws-seed <<<\n"},
		{name: "AddsTrailingNewlineToBody", body: "x", want: "# >>> ws-seed >>>\nx\n# <<< ws-seed <<<\n"},
		{name: "CustomCommentPrefix", body: "const x = 1;\n", comment: "//", want: "// >>> ws-seed >>>\nconst x = 1;\n// <<< ws-seed <<<\n"},
		{name: "CustomCommentReplacesBody", existing: "// >>> ws-seed >>>\nold\n// <<< ws-seed <<<\n", body: "new\n", comment: "//", want: "// >>> ws-seed >>>\nnew\n// <<< ws-seed <<<\n"},
		{name: "AppendsAfterExisting", existing: "base\n", body: "line\n", want: "base\n# >>> ws-seed >>>\nline\n# <<< ws-seed <<<\n"},
		{name: "AddsNewlineBeforeMarker", existing: "base", body: "line\n", want: "base\n# >>> ws-seed >>>\nline\n# <<< ws-seed <<<\n"},
		{name: "ReplacesBodyPreservingSurround", existing: "head\n# >>> ws-seed >>>\nold\n# <<< ws-seed <<<\ntail\n", body: "new\n", want: "head\n# >>> ws-seed >>>\nnew\n# <<< ws-seed <<<\ntail\n"},
		{name: "DuplicateBeginRejected", existing: "# >>> ws-seed >>>\n# >>> ws-seed >>>\n# <<< ws-seed <<<\n", body: "x\n", err: "duplicate begin marker"},
		{name: "DuplicateEndRejected", existing: "# >>> ws-seed >>>\n# <<< ws-seed <<<\n# <<< ws-seed <<<\n", body: "x\n", err: "duplicate end marker"},
		{name: "EndBeforeBeginRejected", existing: "# <<< ws-seed <<<\nbody\n# >>> ws-seed >>>\n", body: "x\n", err: "markers out of order"},
		{name: "BeginWithoutEndRejected", existing: "# >>> ws-seed >>>\norphan\n", body: "x\n", err: "markers out of order"},
		{name: "EndWithoutBeginRejected", existing: "orphan\n# <<< ws-seed <<<\n", body: "x\n", err: "markers out of order"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ensureBlock([]byte(tt.existing), []byte(tt.body), tt.comment)
			assertEdit(t, got, err, tt.want, tt.err)
		})
	}
}

func TestEnsureLine(t *testing.T) {
	tests := []struct{ name, existing, body, want, err string }{
		{name: "AppendsWhenAbsent", existing: "A=1\n", body: "FOO=bar\n", want: "A=1\nFOO=bar\n"},
		{name: "ReplacesInPlaceWhenPresent", existing: "FOO=old\n", body: "FOO=new\n", want: "FOO=new\n"},
		{name: "PreservesSurroundingLines", existing: "A=1\nFOO=old\nB=2\n", body: "FOO=new\n", want: "A=1\nFOO=new\nB=2\n"},
		{name: "MatchesIndentedKey", existing: "  FOO=old\n", body: "FOO=new\n", want: "FOO=new\n"},
		{name: "ReplacesUnterminatedLastLine", existing: "A=1\nFOO=old", body: "FOO=new\n", want: "A=1\nFOO=new\n"},
		{name: "Idempotent", existing: "FOO=new\n", body: "FOO=new\n", want: "FOO=new\n"},
		{name: "AddsNewlineBeforeAppend", existing: "A=1", body: "FOO=bar\n", want: "A=1\nFOO=bar\n"},
		{name: "CreatesFromEmpty", body: "FOO=bar\n", want: "FOO=bar\n"},
		{name: "KeyWithNoSeparatorAppendsDistinct", existing: "keepme\n", body: "addme\n", want: "keepme\naddme\n"},
		{name: "KeyWithNoSeparatorReplacesExact", existing: "solo\n", body: "solo\n", want: "solo\n"},
		{name: "NoSeparatorDoesNotMatchKeyed", existing: "FOO=1\n", body: "FOO\n", want: "FOO=1\nFOO\n"},
		{name: "MultiLineRejected", body: "A=1\nB=2\n", err: "single line"},
		{name: "EmptyRejected", existing: "x\n", body: "\n", err: "requires content"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ensureLine([]byte(tt.existing), []byte(tt.body))
			assertEdit(t, got, err, tt.want, tt.err)
		})
	}

	t.Run("MultipleKeysAccumulate", func(t *testing.T) {
		got := []byte("base\n")
		for _, line := range []string{"FOO=1\n", "BAR=2\n", "FOO=9\n"} {
			var err error
			got, err = ensureLine(got, []byte(line))
			assert.NilError(t, err)
		}

		assert.Equal(t, string(got), "base\nFOO=9\nBAR=2\n")
	})
}

func TestMergeContent(t *testing.T) {
	for _, tt := range []struct{ dest, existing, fragment string }{
		{"config.json", `{"keep":1,"list":[1,2,3]}`, `{"list":[9],"add":true}`},
		{"config.yaml", "keep: 1\nlist: [1, 2, 3]\n", "list: [9]\nadd: true\n"},
		{"config.toml", "keep = 1\nlist = [1, 2, 3]\n", "list = [9]\nadd = true\n"},
	} {
		t.Run("ListReplace/"+tt.dest, func(t *testing.T) {
			merged, err := mergeContent([]byte(tt.existing), []byte(tt.fragment), tt.dest)
			assert.NilError(t, err)

			c, err := codecFor(tt.dest)
			assert.NilError(t, err)
			out, err := c.decode(merged)
			assert.NilError(t, err)

			assert.Equal(t, len(out["list"].([]any)), 1)
			assert.Assert(t, out["keep"] != nil && out["add"] != nil)
		})
	}

	for _, tt := range []struct {
		name, dest, existing, fragment string
		want                           []string
	}{
		{"JSONNumberFidelity", "config.json", `{"n":1}`, `{"n":2}`, []string{`"n": 2`}},
		{"LargeIntExistingSide", "config.json", `{"big":9007199254740993}`, `{"x":1}`, []string{"9007199254740993"}},
		{"LargeIntFragmentSide", "config.json", `{"x":1}`, `{"big":9223372036854775807}`, []string{"9223372036854775807"}},
		{"LargeIntYAML", "config.yaml", "keep: 1\n", "big: 9223372036854775807\n", []string{"9223372036854775807"}},
		{"LargeIntTOML", "config.toml", "keep = 1\n", "big = 9223372036854775807\n", []string{"9223372036854775807"}},
		{"NumberShapes", "config.json", `{"x":1}`, `{"neg":-5,"zero":0,"frac":1.5}`, []string{`"neg": -5`, `"zero": 0`, `"frac": 1.5`}},
		{"NullExisting", "config.yaml", "~\n", "a: 1\n", []string{"a: 1"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			merged, err := mergeContent([]byte(tt.existing), []byte(tt.fragment), tt.dest)
			assert.NilError(t, err)

			for _, want := range tt.want {
				assert.Assert(t, strings.Contains(string(merged), want), string(merged))
			}
			assert.Assert(t, !strings.Contains(string(merged), "e+"))
		})
	}

	for _, tt := range []struct{ name, existing, fragment, dest, err string }{
		{"ScalarVsMapConflict", `{"k":"scalar"}`, `{"k":{"nested":1}}`, "config.json", "merge conflict at key"},
		{"MapVsScalarConflict", `{"k":{"nested":1}}`, `{"k":"scalar"}`, "config.json", "merge conflict at key"},
		{"UnknownExtensionRejected", "a", "b", "config.ini", "cannot infer merge format"},
		{"BadExistingRejected", "{", `{}`, "config.json", "failed to decode existing"},
		{"BadFragmentRejected", `{}`, "{", "config.json", "failed to decode fragment"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mergeContent([]byte(tt.existing), []byte(tt.fragment), tt.dest)
			assert.ErrorContains(t, err, tt.err)
		})
	}
}

func assertEdit(t *testing.T, got []byte, err error, want, wantErr string) {
	t.Helper()

	if wantErr != "" {
		assert.ErrorContains(t, err, wantErr)
		return
	}

	assert.NilError(t, err)
	assert.Equal(t, string(got), want)
}
