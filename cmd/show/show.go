package show

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/kloudkit/ws-cli/cmd/cli"
	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/logger"
	"github.com/kloudkit/ws-cli/internals/net"
	"github.com/kloudkit/ws-cli/internals/path"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var osExit = os.Exit

func exit(code int) error {
	osExit(code)
	return nil
}

var envCmd = cli.New("env <KEY>", "0.2.0", "Display the resolved value of a workspace environment variable",
	"Resolve a setting by its dotted key (server.port) and print it with its source and description. --value emits just the value for scripts, --as bool|int|list validates the shape, --check tests whether it is set so a startup script can guard on it.",
	func(cmd *cobra.Command, args []string) error {
		dotted := args[0]

		config.SetDeprecationWriter(cmd.ErrOrStderr())

		group, prop, ok := strings.Cut(dotted, ".")
		if !ok || strings.HasPrefix(dotted, "WS_") {
			fmt.Fprintf(cmd.ErrOrStderr(),
				"Use dotted key [%s] instead of [%s]\n", formatGroupProp(dotted), dotted)
			return exit(2)
		}

		key := config.RuntimeKey(group, prop)

		check, value, orSkip, deprecated := cli.Bool(cmd, "check"), cli.Bool(cmd, "value"), cli.Bool(cmd, "or-skip"), cli.String(cmd, "deprecated")

		if check && !value {
			return runCheck(cmd, key, deprecated)
		}

		propMeta, exists, err := config.LookupProperty(key)
		if err != nil {
			return err
		}
		if !exists {
			fmt.Fprintf(cmd.ErrOrStderr(), "Unknown env var [%s]\n", dotted)
			return exit(2)
		}

		if value && check {
			return runValueChecked(cmd, key, deprecated, orSkip)
		}

		if value {
			return runValue(cmd, key, orSkip)
		}

		if asType := cli.String(cmd, "as"); asType != "" {
			return runAs(cmd, key, asType, orSkip)
		}

		return runPretty(cmd, dotted, key, propMeta)
	})

func skipBreadcrumb(cmd *cobra.Command, key string) {
	logger.Log(cmd.ErrOrStderr(), "debug", fmt.Sprintf("Skipped: env [%s] not set", key), 1, true)
}

func runCheck(cmd *cobra.Command, preferred, deprecated string) error {
	switch config.Check(preferred, deprecated) {
	case config.CheckPreferredSet:
		return nil
	case config.CheckDeprecatedOnly:
		fmt.Fprintln(cmd.ErrOrStderr(), config.DeprecationLine(deprecated, preferred))
	case config.CheckBothSet:
		fmt.Fprintln(cmd.ErrOrStderr(), config.BothSetLine(deprecated, preferred))
		return exit(2)
	}
	return exit(1)
}

func runValue(cmd *cobra.Command, key string, orSkip bool) error {
	value, err := config.ResolveKey(key)
	if err != nil {
		return err
	}
	if orSkip && value == "" {
		skipBreadcrumb(cmd, key)
		return exit(1)
	}
	fmt.Fprintln(cmd.OutOrStdout(), value)
	return nil
}

func runValueChecked(cmd *cobra.Command, key, deprecated string, orSkip bool) error {
	switch config.Check(key, deprecated) {
	case config.CheckPreferredSet, config.CheckDeprecatedOnly:
		return runValue(cmd, key, false)
	case config.CheckBothSet:
		fmt.Fprintln(cmd.ErrOrStderr(), config.BothSetLine(deprecated, key))
		return exit(2)
	}
	if orSkip {
		skipBreadcrumb(cmd, key)
	}
	return exit(1)
}

func runAs(cmd *cobra.Command, key, asType string, orSkip bool) error {
	switch asType {
	case "bool":
		return runBool(cmd, key, orSkip)
	case "int":
		return runInt(cmd, key)
	case "list":
		return runList(cmd, key, cli.String(cmd, "delimiter"), cli.String(cmd, "validate"))
	}
	return fmt.Errorf("invalid --as value %q (accepted: bool, int, list)", asType)
}

func runBool(cmd *cobra.Command, key string, orSkip bool) error {
	value, err := config.ResolveKey(key)
	if err != nil {
		return err
	}

	if orSkip {
		value = strings.TrimSpace(value)
		if value == "" {
			skipBreadcrumb(cmd, key)
			return exit(1)
		}
	}

	parsed, err := config.ParseBool(value)
	if err != nil {
		return err
	}
	if !parsed {
		osExit(1)
	}
	return nil
}

func runInt(cmd *cobra.Command, key string) error {
	value, err := config.ResolveKey(key)
	if err != nil {
		return err
	}
	parsed, err := config.ParseInt(value)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), parsed)
	return nil
}

func runList(cmd *cobra.Command, key, delimiter, validate string) error {
	items, err := config.ResolveListKey(key, delimiter)
	if err != nil {
		return err
	}

	if validate != "" {
		re, err := regexp.Compile("^(?:" + validate + ")$")
		if err != nil {
			return fmt.Errorf("invalid --validate pattern %q: %w", validate, err)
		}
		for _, item := range items {
			if !re.MatchString(item) {
				fmt.Fprintf(cmd.ErrOrStderr(), "Rejected: invalid item [%s]\n", item)
				return exit(3)
			}
		}
	}

	for _, item := range items {
		fmt.Fprintln(cmd.OutOrStdout(), item)
	}
	return nil
}

func runPretty(cmd *cobra.Command, dotted, key string, prop config.Property) error {
	out := cmd.OutOrStdout()
	value, source, err := config.ResolveKeyWithSource(key)
	if err != nil {
		return err
	}

	styles.PrintTitle(out, dotted, key)

	for i, text := range []string{prop.Description, prop.LongDescription} {
		if text == "" {
			continue
		}
		if i > 0 {
			fmt.Fprintln(out)
		}
		if err := styles.RenderMarkdown(out, text); err != nil {
			return err
		}
	}

	display := value
	switch {
	case value == "":
		display = styles.Aside("unset")
	case prop.Secret:
		display = styles.Muted.Render("<redacted>")
	}

	fmt.Fprintln(out)
	styles.PrintKeyValues(out,
		styles.Pair{Key: "Value", Value: display},
		styles.Pair{Key: "Source", Value: sourceLabel(prop, source, value)},
	)

	return nil
}

func sourceLabel(prop config.Property, source config.ResolveSource, value string) string {
	label := source.Label()
	if source == config.SourceEnv && prop.Default != nil && value == *prop.Default {
		label += " " + styles.Aside("matches declared")
	}
	return label
}

func formatGroupProp(key string) string {
	s := strings.ToLower(strings.TrimPrefix(key, "WS_"))
	if group, prop, ok := strings.Cut(s, "_"); ok {
		return group + "." + prop
	}
	return s
}

func init() {
	ShowCmd.PersistentFlags().Bool("raw", false, "Output raw value without styling")
	pathVscodeCmd.Flags().Bool("workspace", false, "Get the workspace settings")
	envCmd.Args = cobra.ExactArgs(1)
	envCmd.Flags().Bool("value", false, "Emit the raw resolved value as a single line")
	envCmd.Flags().String("as", "", "Validate and emit as one of: bool, int, list (mutex with --value)")
	envCmd.Flags().Bool("check", false, "Check whether the variable (or its --deprecated alias) is set")
	envCmd.Flags().String("delimiter", "", "Override delimiter for --as=list (defaults to YAML delimiter or space)")
	envCmd.Flags().String("deprecated", "", "Deprecated alias paired with --check")
	envCmd.Flags().Bool("or-skip", false, "Exit 1 (not error) on the natural absence of the chosen projection")
	envCmd.Flags().String("validate", "", "Anchored regex each --as=list token must full-match; rejects fail-closed")

	envCmd.MarkFlagsMutuallyExclusive("value", "as")
	envCmd.MarkFlagsMutuallyExclusive("as", "check")
}

var pathVscodeCmd = cli.New("vscode-settings", "0.2.0", "Display the VS Code settings path",
	"Print the path to the VS Code settings file — the user file by default, or the folder's with --workspace.",
	func(cmd *cobra.Command, args []string) error {
		if cli.Bool(cmd, "workspace") {
			return emit(cmd, "VS Code Settings Path", "Workspace", "Path", "/workspace/.vscode/settings.json")
		}

		return emit(cmd, "VS Code Settings Path", "User", "Path", path.GetHomeDirectory("/.local/share/ws-server/User/settings.json"))
	})

func makeValueCmd(use, short, long, title, label string, getter func() (string, error)) *cobra.Command {
	return cli.New(use, "0.2.0", short, long, func(cmd *cobra.Command, args []string) error {
		value, err := getter()
		if err != nil {
			return err
		}

		return emit(cmd, title, "", label, value)
	})
}

var ShowCmd = cli.Group("show", "0.2.0", "Display information about the current workspace instance",
	"Resolve and print facts about this workspace instance — settings, IP addresses, and paths. --raw drops the styling for use in scripts.",
	`# Resolve a setting by its dotted key
ws show env server.port

# Reverse-tunnel a local port to the workspace node
ws_node_ip=$(ws show ip node --raw); ssh -N -R "3001:${ws_node_ip}:3001" "${ws_node_ip}"`,
	envCmd,
	cli.Group("ip", "0.2.0", "Display IP addresses",
		"Print the workspace's IP addresses — the internal container address or the node it runs on.", "",
		makeValueCmd("internal", "Display the internal IP address", "Print the workspace container's internal IP address.", "Internal IP Address", "Address", net.GetInternalIP),
		makeValueCmd("node", "Display the node/host IP address", "Print the IP address of the node hosting the workspace.", "Node IP Address", "Address", net.GetNodeIP),
	),
	cli.Group("path", "0.2.0", "Display various paths",
		"Print well-known workspace paths — the home root or the VS Code settings file.", "",
		makeValueCmd("home", "Display the workspace home path", "Print the workspace home (server root) path.", "Workspace Home Path", "Path", func() (string, error) {
			return config.MustResolve("server", "root"), nil
		}),
		pathVscodeCmd,
	),
)

func emit(cmd *cobra.Command, title, kind, label, value string) error {
	out := cmd.OutOrStdout()

	if cli.Bool(cmd, "raw") {
		fmt.Fprintln(out, value)
		return nil
	}

	var pairs []styles.Pair
	if kind != "" {
		pairs = append(pairs, styles.Pair{Key: "Type", Value: kind})
	}

	styles.PrintTitle(out, title)
	styles.PrintKeyValues(out, append(pairs, styles.Pair{Key: label, Value: value})...)

	return nil
}
