package show

import (
	"fmt"
	"os"
	"strings"

	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var osExit = os.Exit

var envCmd = &cobra.Command{
	Use:   "env <KEY>",
	Short: "Display the resolved value of a workspace environment variable",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		config.SetDeprecationWriter(cmd.ErrOrStderr())

		check, _ := cmd.Flags().GetBool("check")
		if check {
			deprecated, _ := cmd.Flags().GetString("deprecated")
			return runCheck(cmd, key, deprecated)
		}

		prop, exists, err := config.LookupProperty(key)
		if err != nil {
			return err
		}
		if !exists {
			fmt.Fprintf(cmd.ErrOrStderr(), "Unknown env var [%s]\n", key)
			osExit(2)
			return nil
		}

		value, _ := cmd.Flags().GetBool("value")
		if value {
			return runValue(cmd, key)
		}

		asType, _ := cmd.Flags().GetString("as")
		if asType != "" {
			return runAs(cmd, key, asType)
		}

		return runPretty(cmd, key, prop)
	},
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

func runValue(cmd *cobra.Command, key string) error {
	value, err := config.ResolveKey(key)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), value)
	return nil
}

func runAs(cmd *cobra.Command, key, asType string) error {
	switch asType {
	case "bool":
		return runBool(cmd, key)
	case "int":
		return runInt(cmd, key)
	case "list":
		delimiter, _ := cmd.Flags().GetString("delimiter")
		return runList(cmd, key, delimiter)
	}
	return fmt.Errorf("invalid --as value %q (accepted: bool, int, list)", asType)
}

func runBool(_ *cobra.Command, key string) error {
	value, err := config.ResolveKey(key)
	if err != nil {
		return err
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

func runList(cmd *cobra.Command, key, delimiter string) error {
	items, err := config.ResolveListKey(key, delimiter)
	if err != nil {
		return err
	}
	for _, item := range items {
		fmt.Fprintln(cmd.OutOrStdout(), item)
	}
	return nil
}

func runPretty(cmd *cobra.Command, key string, prop config.Property) error {
	out := cmd.OutOrStdout()
	value, source, err := config.ResolveKeyWithSource(key)
	if err != nil {
		return err
	}

	styles.PrintTitle(out, "Workspace Environment")
	fmt.Fprintf(out, "  %s %s\n\n",
		styles.Key().Render(key),
		styles.Muted().Render("("+formatGroupProp(key)+")"))

	if prop.Description != "" {
		fmt.Fprintf(out, "  %s\n\n", styles.Value().Render(prop.Description))
	}

	if prop.LongDescription != "" {
		if err := styles.RenderMarkdown(out, prop.LongDescription); err != nil {
			return err
		}
	}

	styles.PrintKeyValue(out, "Value", value)
	styles.PrintKeyValue(out, "Source", source.Label())

	return nil
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
	envCmd.Flags().String("as", "", "Validate and emit as one of: bool, int, list (mutex with --value/--check)")
	envCmd.Flags().Bool("check", false, "Check whether the variable (or its --deprecated alias) is set")
	envCmd.Flags().String("delimiter", "", "Override delimiter for --as=list (defaults to YAML delimiter or space)")
	envCmd.Flags().String("deprecated", "", "Deprecated alias paired with --check")

	envCmd.MarkFlagsMutuallyExclusive("value", "as", "check")

	ShowCmd.AddCommand(envCmd)
}
