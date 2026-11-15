package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
)

const referenceFixture = `envs:
  features:
    properties:
      store_url:
        type: string
        default: null
      offline:
        type: boolean
        default: false
      token:
        type: string
        default: null
        secret: true
  ai:
    properties:
      mcp_allow_write:
        type: boolean
        default: false
      mcp_tools:
        type: string
        default: "*"
        delimiter: ","
`

const referenceWithoutMcp = `envs:
  features:
    properties:
      store_url:
        type: string
        default: null
`

func useReference(t *testing.T, content string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "env.reference.yaml")
	assert.NilError(t, os.WriteFile(path, []byte(content), 0o644))
	t.Setenv("WS__INTERNAL_ENV_REFERENCE", path)

	for _, key := range []string{"WS_AI_MCP_ALLOW_WRITE", "WS_AI_MCP_TOOLS", "WS_FEATURES_STORE_URL", "WS_FEATURES_OFFLINE", "WS_FEATURES_TOKEN"} {
		t.Setenv(key, "")
	}
}

func leaf(use string, annotations map[string]string) *cobra.Command {
	return &cobra.Command{Use: use, Short: "Short " + use, Annotations: annotations, RunE: func(*cobra.Command, []string) error { return nil }}
}

func group(use string, children ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{Use: use}
	cmd.AddCommand(children...)

	return cmd
}

func read(extra ...string) map[string]string {
	annotations := map[string]string{AnnotationAccess: string(Read)}
	for i := 0; i+1 < len(extra); i += 2 {
		annotations[extra[i]] = extra[i+1]
	}

	return annotations
}

func syntheticTree() *cobra.Command {
	logs := leaf("logs", read(AnnotationFlags, "tail,level"))
	logs.Flags().Bool("follow", false, "Follow the log")
	logs.Flags().Int("tail", 0, "Lines from the end")
	logs.Flags().String("level", "", "Level filter")

	reap := leaf("reap <zip>", read())
	reap.Hidden = true

	open := leaf("open <file>", map[string]string{AnnotationAccess: string(Write)})
	open.Long = "Open a file in the editor."

	return group("ws-cli",
		group("info", leaf("version", read()), leaf("env", nil)),
		group("editor", leaf("list", read()), open, reap),
		group("feature", leaf("store", read(AnnotationWhen, "features.store_url"))),
		group("show", leaf("env <KEY>", read()), group("path", leaf("vscode-settings", read()))),
		logs,
	)
}

func names(tools []Tool) string {
	listed := make([]string, len(tools))
	for i, tool := range tools {
		listed[i] = tool.Name
	}

	return strings.Join(listed, " ")
}

func TestBuildTools(t *testing.T) {
	const readTools = "editor_list info_version logs show_env show_path_vscode_settings"

	cases := map[string]struct {
		settings Settings
		want     string
	}{
		"Default":             {Settings{}, readTools},
		"WritesEnabled":       {Settings{AllowWrite: true}, "editor_list editor_open info_version logs show_env show_path_vscode_settings"},
		"TwoGroups":           {Settings{Tools: []string{"info", "editor"}}, "editor_list info_version"},
		"GroupAndTool":        {Settings{AllowWrite: true, Tools: []string{"info", "editor_open"}}, "editor_open info_version"},
		"NestedGroup":         {Settings{Tools: []string{"show_path"}}, "show_path_vscode_settings"},
		"UnknownEntry":        {Settings{AllowWrite: true, Tools: []string{"bogus"}}, ""},
		"PartialName":         {Settings{Tools: []string{"inf", "show_path_vscode", "log"}}, ""},
		"EmptyList":           {Settings{Tools: []string{}}, ""},
		"WriteNeedsBothKnobs": {Settings{Tools: []string{"editor_open"}}, ""},
		"NeverATool":          {Settings{AllowWrite: true, Tools: []string{"info_env", "editor_reap", "info"}}, "info_version"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			useReference(t, referenceFixture)

			tools, _, err := BuildTools(syntheticTree(), tc.settings)
			assert.NilError(t, err)
			assert.Equal(t, names(tools), tc.want)
		})
	}

	t.Run("Shape", func(t *testing.T) {
		useReference(t, referenceFixture)

		tools, _, err := BuildTools(syntheticTree(), Settings{AllowWrite: true})
		assert.NilError(t, err)

		byName := map[string]Tool{}
		for _, tool := range tools {
			byName[tool.Name] = tool
		}

		open := byName["editor_open"]
		assert.Equal(t, open.Access, Write)
		assert.Equal(t, open.Title, "Short open <file>")
		assert.Equal(t, open.Description, "Open a file in the editor.")
		assert.DeepEqual(t, open.Command, []string{"editor", "open"})
		assert.Equal(t, string(open.Schema()),
			`{"additionalProperties":false,"properties":{"args":{"description":"Positional arguments: \u003cfile\u003e","items":{"type":"string"},"type":"array"}},"type":"object"}`)

		assert.Equal(t, byName["info_version"].Description, "Short version")
		assert.Equal(t, string(byName["info_version"].Schema()), `{"additionalProperties":false,"properties":{},"type":"object"}`)
		assert.DeepEqual(t, byName["show_path_vscode_settings"].Command, []string{"show", "path", "vscode-settings"})
		assert.Equal(t, string(byName["logs"].Schema()),
			`{"additionalProperties":false,"properties":{"level":{"description":"Level filter","type":"string"},"tail":{"description":"Lines from the end","type":"integer"}},"type":"object"}`)
	})

	t.Run("NewFlagStaysHiddenUntilListed", func(t *testing.T) {
		useReference(t, referenceFixture)

		root := syntheticTree()
		logs, _, _ := root.Find([]string{"logs"})
		logs.Flags().String("since", "", "Start time")

		tools, _, err := BuildTools(root, Settings{Tools: []string{"logs"}})
		assert.NilError(t, err)
		assert.Assert(t, !strings.Contains(string(tools[0].Schema()), "since"))

		logs.Annotations[AnnotationFlags] = "tail,level,since"
		tools, _, err = BuildTools(root, Settings{Tools: []string{"logs"}})
		assert.NilError(t, err)
		assert.Assert(t, strings.Contains(string(tools[0].Schema()), `"since"`))
	})

	t.Run("Rejects", func(t *testing.T) {
		useReference(t, referenceFixture)

		slice := leaf("slice", read(AnnotationFlags, "items"))
		slice.Flags().StringSlice("items", nil, "Items")

		cases := map[string]struct {
			root *cobra.Command
			want string
		}{
			"DuplicateName":   {group("ws-cli", group("a-b", leaf("c", read())), group("a", leaf("b-c", read()))), "tool name [a_b_c] is already taken"},
			"UnknownFlag":     {group("ws-cli", leaf("logs", read(AnnotationFlags, "tail"))), "unknown flag [tail]"},
			"UnsupportedFlag": {group("ws-cli", slice), "cannot expose flag [items] of type stringSlice"},
			"RepeatedFlag": {group("ws-cli", func() *cobra.Command {
				c := leaf("x", read(AnnotationFlags, "n,n"))
				c.Flags().Int("n", 0, "")
				return c
			}()), "names flag [n] twice"},
			"BadAccess":     {group("ws-cli", leaf("x", map[string]string{AnnotationAccess: "admin"})), `must be "read" or "write"`},
			"MalformedTerm": {group("ws-cli", leaf("x", read(AnnotationWhen, "features"))), `malformed term "features"`},
			"EmptyTerm":     {group("ws-cli", leaf("x", read(AnnotationWhen, "features.store_url,"))), `malformed term ""`},
			"StrayOption":   {group("ws-cli", leaf("x", map[string]string{AnnotationFlags: "tail"})), "needs [mcp]"},
			"MalformedOnAFilteredTool": {group("ws-cli", func() *cobra.Command {
				c := leaf("x", read(AnnotationWhen, "!!a.b"))
				c.Annotations[AnnotationAccess] = "write"
				return c
			}()), `malformed term "!!a.b"`},
		}

		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				_, _, err := BuildTools(tc.root, Settings{})
				assert.ErrorContains(t, err, tc.want)
			})
		}
	})

	t.Run("UnmatchedEntries", func(t *testing.T) {
		useReference(t, referenceFixture)

		entries := []string{"bogus", "Info", "show-path", "editor_open", "feature_store", "info", "bogus", "info_env", "editor_reap"}

		tools, warnings, err := BuildTools(syntheticTree(), Settings{Tools: entries})
		assert.NilError(t, err)
		assert.Equal(t, names(tools), "info_version")
		assert.DeepEqual(t, warnings, []string{
			"Ignoring [WS_AI_MCP_TOOLS] entry [Info]: it names no tool or group",
			"Ignoring [WS_AI_MCP_TOOLS] entry [bogus]: it names no tool or group",
			"Ignoring [WS_AI_MCP_TOOLS] entry [editor_reap]: it names no tool or group",
			"Ignoring [WS_AI_MCP_TOOLS] entry [info_env]: it names no tool or group",
			"Ignoring [WS_AI_MCP_TOOLS] entry [show-path]: it names no tool or group",
		})

		_, warnings, err = BuildTools(syntheticTree(), Settings{})
		assert.NilError(t, err)
		assert.Equal(t, len(warnings), 0)
	})

	t.Run("PersistentFlag", func(t *testing.T) {
		useReference(t, referenceFixture)

		root := group("ws-cli", group("editor", leaf("list", read(AnnotationFlags, "raw"))))
		editor, _, _ := root.Find([]string{"editor"})
		editor.PersistentFlags().Bool("raw", false, "Raw output")

		tools, _, err := BuildTools(root, Settings{})
		assert.NilError(t, err)
		assert.DeepEqual(t, tools[0].Params, []Param{{Name: "raw", Type: "boolean", Description: "Raw output"}})
	})
}

func TestResolveSettings(t *testing.T) {
	const (
		allowKey = "WS_AI_MCP_ALLOW_WRITE"
		toolsKey = "WS_AI_MCP_TOOLS"
	)

	cases := map[string]struct {
		reference string
		env       map[string]string
		want      Settings
		warnings  []string
	}{
		"Unset":           {referenceFixture, nil, Settings{}, nil},
		"WritesEnabled":   {referenceFixture, map[string]string{allowKey: "true"}, Settings{AllowWrite: true}, nil},
		"WritesDisabled":  {referenceFixture, map[string]string{allowKey: "0"}, Settings{}, nil},
		"Unparsable":      {referenceFixture, map[string]string{allowKey: "maybe"}, Settings{}, []string{"Ignoring [WS_AI_MCP_ALLOW_WRITE], write tools stay off:"}},
		"MissingAndUnset": {referenceWithoutMcp, nil, Settings{}, nil},
		"MissingFromSchema": {referenceWithoutMcp, map[string]string{allowKey: "true", toolsKey: "info"}, Settings{}, []string{
			"Ignoring [WS_AI_MCP_ALLOW_WRITE]: it is not declared in this workspace",
			"Ignoring [WS_AI_MCP_TOOLS]: it is not declared in this workspace",
		}},
		"TwoGroups":       {referenceFixture, map[string]string{toolsKey: "info,editor"}, Settings{Tools: []string{"info", "editor"}}, nil},
		"Padded":          {referenceFixture, map[string]string{toolsKey: " info , editor_open "}, Settings{Tools: []string{"info", "editor_open"}}, nil},
		"Bogus":           {referenceFixture, map[string]string{toolsKey: "bogus"}, Settings{Tools: []string{"bogus"}}, nil},
		"OnlySeparators":  {referenceFixture, map[string]string{toolsKey: " , "}, Settings{Tools: []string{}}, nil},
		"UnparsableTools": {referenceFixture, map[string]string{allowKey: "true", toolsKey: "file:/nope"}, Settings{AllowWrite: true, Tools: []string{}}, []string{"Cannot read [WS_AI_MCP_TOOLS], serving no tools:"}},
		"Star":            {referenceFixture, map[string]string{toolsKey: "*"}, Settings{}, nil},
		"StarAmongOthers": {referenceFixture, map[string]string{toolsKey: "info, *"}, Settings{}, nil},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			useReference(t, tc.reference)
			for key, value := range tc.env {
				t.Setenv(key, value)
			}

			settings, warnings := ResolveSettings()
			assert.DeepEqual(t, settings, tc.want)
			assert.Equal(t, len(warnings), len(tc.warnings), "%v", warnings)

			for i, want := range tc.warnings {
				assert.Assert(t, strings.HasPrefix(warnings[i], want), warnings[i])
			}
		})
	}
}

func TestWhen(t *testing.T) {
	t.Run("HidesAndShowsTheTool", func(t *testing.T) {
		useReference(t, referenceFixture)

		tools, _, err := BuildTools(syntheticTree(), Settings{Tools: []string{"feature"}})
		assert.NilError(t, err)
		assert.Equal(t, names(tools), "")

		t.Setenv("WS_FEATURES_STORE_URL", "https://store")
		tools, _, err = BuildTools(syntheticTree(), Settings{Tools: []string{"feature"}})
		assert.NilError(t, err)
		assert.Equal(t, names(tools), "feature_store")
	})

	t.Run("UnreadableReferenceIsAnError", func(t *testing.T) {
		t.Setenv("WS__INTERNAL_ENV_REFERENCE", filepath.Join(t.TempDir(), "missing.yaml"))

		_, _, err := BuildTools(syntheticTree(), Settings{Tools: []string{"feature"}})
		assert.ErrorContains(t, err, "ws-cli feature store: cannot read")
	})
}

func TestArgv(t *testing.T) {
	tool := Tool{
		Command: []string{"logs"},
		Params:  []Param{{Name: "tail", Type: "integer"}, {Name: "level", Type: "string"}, {Name: "plain", Type: "boolean"}},
		Args:    "<KEY>",
	}

	accepted := map[string][]string{
		``:                                  {"logs"},
		`null`:                              {"logs"},
		`{}`:                                {"logs"},
		`{"level":"error","tail":5}`:        {"logs", "--tail=5", "--level=error"},
		`{"plain":false,"tail":-1}`:         {"logs", "--tail=-1", "--plain=false"},
		`{"level":"--follow"}`:              {"logs", "--level=--follow"},
		`{"level":""}`:                      {"logs", "--level="},
		`{"args":["--value","-f","a b"]}`:   {"logs", "--", "--value", "-f", "a b"},
		`{"args":[]}`:                       {"logs"},
		`{"args":["x"],"level":"a=b c\"d"}`: {"logs", `--level=a=b c"d`, "--", "x"},
	}

	for arguments, want := range accepted {
		argv, err := tool.Argv([]byte(arguments))
		assert.NilError(t, err, arguments)
		assert.DeepEqual(t, argv, want)
	}

	rejected := map[string]string{
		`{"follow":true}`:          "unknown argument [follow] (accepted: tail, level, plain, args)",
		`{"--follow":true}`:        "unknown argument [--follow]",
		`{"tail":"5"}`:             "argument [tail] must be of type integer",
		`{"tail":5.5}`:             "argument [tail] must be of type integer",
		`{"tail":1e3}`:             "argument [tail] must be of type integer",
		`{"tail":null}`:            "argument [tail] must be of type integer",
		`{"level":5}`:              "argument [level] must be of type string",
		`{"level":null}`:           "argument [level] must be of type string",
		`{"level":["a"]}`:          "argument [level] must be of type string",
		`{"level":"a\u0000b"}`:     "argument [level] must be of type string",
		`{"plain":"true"}`:         "argument [plain] must be of type boolean",
		`{"plain":1}`:              "argument [plain] must be of type boolean",
		`{"args":"x"}`:             "argument [args] must be an array of strings",
		`{"args":[1]}`:             "argument [args] must be an array of strings",
		`{"args":["a",null]}`:      "argument [args] must be an array of strings",
		`{"args":null}`:            "argument [args] must be an array of strings",
		`{"args":["a\u0000"]}`:     "argument [args] must be an array of strings",
		`["logs"]`:                 "arguments must be a JSON object",
		`"logs"`:                   "arguments must be a JSON object",
		`{"tail":1,"zz":1,"aa":1}`: "unknown argument [aa]",
	}

	for arguments, want := range rejected {
		_, err := tool.Argv([]byte(arguments))
		assert.ErrorContains(t, err, want, arguments)
	}

	bare := Tool{Command: []string{"info", "version"}}
	_, err := bare.Argv([]byte(`{"args":["x"]}`))
	assert.ErrorContains(t, err, "unknown argument [args] (accepted: none)")
}
