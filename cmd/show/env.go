package show

import (
	"fmt"
	"os"

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

		asList, _ := cmd.Flags().GetBool("list")
		asBool, _ := cmd.Flags().GetBool("bool")
		asInt, _ := cmd.Flags().GetBool("int")
		asCheck, _ := cmd.Flags().GetBool("check")
		raw, _ := cmd.Flags().GetBool("raw")
		delimiter, _ := cmd.Flags().GetString("delimiter")
		deprecated, _ := cmd.Flags().GetString("deprecated")

		switch {
		case asCheck:
			return runCheck(cmd, key, deprecated)
		case asBool:
			return runBool(cmd, key)
		case asInt:
			return runInt(cmd, key)
		case asList:
			return runList(cmd, key, delimiter)
		}

		value, err := config.ResolveKey(key)
		if err != nil {
			return err
		}

		if styles.OutputRaw(cmd.OutOrStdout(), raw, value) {
			return nil
		}

		styles.PrintTitle(cmd.OutOrStdout(), "Workspace Environment")
		styles.PrintKeyCode(cmd.OutOrStdout(), key, value)

		return nil
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

func init() {
	envCmd.Flags().Bool("list", false, "Output as newline-separated list (uses YAML delimiter or --delimiter)")
	envCmd.Flags().Bool("bool", false, "Coerce to boolean; exit 0 truthy, 1 falsy, 2 invalid")
	envCmd.Flags().Bool("int", false, "Coerce to integer; print canonical form or fail with exit 2")
	envCmd.Flags().Bool("check", false, "Check whether the variable (or its --deprecated alias) is set")
	envCmd.Flags().String("delimiter", "", "Override delimiter for --list (defaults to YAML delimiter or space)")
	envCmd.Flags().String("deprecated", "", "Deprecated alias paired with --check")

	envCmd.MarkFlagsMutuallyExclusive("list", "bool", "int", "check")

	ShowCmd.AddCommand(envCmd)
}
