package show

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/logger"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var osExit = os.Exit

var envCmd = &cobra.Command{
	Use:   "env <KEY>",
	Short: "Display the resolved value of a workspace environment variable",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dotted := args[0]

		config.SetDeprecationWriter(cmd.ErrOrStderr())

		group, prop, ok := strings.Cut(dotted, ".")
		if !ok || strings.HasPrefix(dotted, "WS_") {
			fmt.Fprintf(cmd.ErrOrStderr(),
				"Use dotted key [%s] instead of [%s]\n", formatGroupProp(dotted), dotted)
			osExit(2)
			return nil
		}

		key := config.RuntimeKey(group, prop)

		check, _ := cmd.Flags().GetBool("check")
		value, _ := cmd.Flags().GetBool("value")
		orSkip, _ := cmd.Flags().GetBool("or-skip")
		deprecated, _ := cmd.Flags().GetString("deprecated")

		if check && !value {
			return runCheck(cmd, key, deprecated)
		}

		propMeta, exists, err := config.LookupProperty(key)
		if err != nil {
			return err
		}
		if !exists {
			fmt.Fprintf(cmd.ErrOrStderr(), "Unknown env var [%s]\n", dotted)
			osExit(2)
			return nil
		}

		if value && check {
			return runValueChecked(cmd, key, deprecated, orSkip)
		}

		if value {
			return runValue(cmd, key, orSkip)
		}

		asType, _ := cmd.Flags().GetString("as")
		if asType != "" {
			return runAs(cmd, key, asType, orSkip)
		}

		return runPretty(cmd, dotted, key, propMeta)
	},
}

func skipBreadcrumb(cmd *cobra.Command, key string) {
	logger.Log(cmd.ErrOrStderr(), "debug", fmt.Sprintf("Skipped: env [%s] not set", key), 1, true)
}

func runCheck(cmd *cobra.Command, preferred, deprecated string) error {
	switch config.Check(preferred, deprecated) {
	case config.CheckPreferredSet:
		return nil
	case config.CheckDeprecatedOnly:
		fmt.Fprintln(cmd.ErrOrStderr(), config.DeprecationLine(deprecated, preferred))
		osExit(1)
	case config.CheckBothSet:
		fmt.Fprintln(cmd.ErrOrStderr(), config.BothSetLine(deprecated, preferred))
		osExit(2)
	case config.CheckUnset:
		osExit(1)
	}
	return nil
}

func runValue(cmd *cobra.Command, key string, orSkip bool) error {
	value, err := config.ResolveKey(key)
	if err != nil {
		return err
	}
	if orSkip && value == "" {
		skipBreadcrumb(cmd, key)
		osExit(1)
		return nil
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
		osExit(2)
	case config.CheckUnset:
		if orSkip {
			skipBreadcrumb(cmd, key)
		}
		osExit(1)
	}
	return nil
}

func runAs(cmd *cobra.Command, key, asType string, orSkip bool) error {
	switch asType {
	case "bool":
		return runBool(cmd, key, orSkip)
	case "int":
		return runInt(cmd, key)
	case "list":
		delimiter, _ := cmd.Flags().GetString("delimiter")
		validate, _ := cmd.Flags().GetString("validate")
		return runList(cmd, key, delimiter, validate)
	}
	return fmt.Errorf("invalid --as value %q (accepted: bool, int, list)", asType)
}

func runBool(cmd *cobra.Command, key string, orSkip bool) error {
	value, err := config.ResolveKey(key)
	if err != nil {
		return err
	}

	if !orSkip {
		parsed, err := config.ParseBool(value)
		if err != nil {
			return err
		}
		if !parsed {
			osExit(1)
		}
		return nil
	}

	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		skipBreadcrumb(cmd, key)
		osExit(1)
		return nil
	}
	parsed, err := config.ParseBool(trimmed)
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
				osExit(3)
				return nil
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

	styles.PrintTitle(out, "Workspace Environment")
	fmt.Fprintf(out, "  %s %s\n",
		styles.Key().Render(dotted),
		styles.Muted().Render("("+key+")"))

	if prop.Description != "" {
		if err := styles.RenderMarkdown(out, prop.Description); err != nil {
			return err
		}
	}

	if prop.LongDescription != "" {
		fmt.Fprintln(out)
		if err := styles.RenderMarkdown(out, prop.LongDescription); err != nil {
			return err
		}
	}

	fmt.Fprintln(out)
	displayValue := value
	if prop.Secret && value != "" {
		displayValue = styles.Muted().Render("<redacted>")
	}
	styles.PrintKeyValue(out, "Value", displayValue)
	styles.PrintKeyValue(out, "Source", sourceLabel(prop, source, value))

	return nil
}

func sourceLabel(prop config.Property, source config.ResolveSource, value string) string {
	label := source.Label()
	if source == config.SourceEnv && prop.Default != nil && value == *prop.Default {
		label += " (matches declared)"
	}
	return label
}

func formatGroupProp(key string) string {
	s := strings.TrimPrefix(key, "WS_")
	parts := strings.SplitN(s, "_", 2)
	if len(parts) != 2 {
		return strings.ToLower(s)
	}
	return strings.ToLower(parts[0] + "." + parts[1])
}

func init() {
	envCmd.Flags().Bool("value", false, "Emit the raw resolved value as a single line")
	envCmd.Flags().String("as", "", "Validate and emit as one of: bool, int, list (mutex with --value)")
	envCmd.Flags().Bool("check", false, "Check whether the variable (or its --deprecated alias) is set")
	envCmd.Flags().String("delimiter", "", "Override delimiter for --as=list (defaults to YAML delimiter or space)")
	envCmd.Flags().String("deprecated", "", "Deprecated alias paired with --check")
	envCmd.Flags().Bool("or-skip", false, "Exit 1 (not error) on the natural absence of the chosen projection")
	envCmd.Flags().String("validate", "", "Anchored regex each --as=list token must full-match; rejects fail-closed")

	// --value --check is now permitted ("emit value, but only when set"); --as
	// stays exclusive with both.
	envCmd.MarkFlagsMutuallyExclusive("value", "as")
	envCmd.MarkFlagsMutuallyExclusive("as", "check")

	ShowCmd.AddCommand(envCmd)
}
