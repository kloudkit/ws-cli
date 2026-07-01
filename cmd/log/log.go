package log

import (
	"fmt"

	"github.com/kloudkit/ws-cli/internals/logger"
	"github.com/spf13/cobra"
)

var LogCmd = &cobra.Command{
	Use:   "log",
	Short: "Log messages",
	Long:  "Emit styled, level-tagged log lines — the same formatting the startup scripts use. --pipe runs each line of piped input through the logger.",
}

var debugCmd = createCommand("debug", "debugging")
var errorCmd = createCommand("error", "error")
var infoCmd = createCommand("info", "information")
var warnCmd = createCommand("warn", "warning")
var stampCmd = &cobra.Command{
	Use:   "stamp",
	Short: "Log the current timestamp",
	Long:  "Print just the current timestamp in the workspace log style — handy for marking phases in a startup log.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		withPipe, _ := cmd.Flags().GetBool("pipe")

		if withPipe {
			logger.Pipe(cmd.InOrStdin(), cmd.OutOrStdout(), "", 0, true)
		} else {
			logger.Log(cmd.OutOrStdout(), "", "", 0, true)
		}
		return nil
	},
}

func createCommand(short, long string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("%s message", short),
		Short: fmt.Sprintf("Log %s messages", long),
		Long:  fmt.Sprintf("Emit a log line at %s level in the workspace style — the same formatting the startup scripts use. --indent nests it under a preceding line, --stamp prefixes a timestamp.", short),
		Args:  validate,
		RunE:  execute(short),
	}

	cmd.Flags().IntP("indent", "i", 0, "Desired prefixed indentation")
	cmd.Flags().BoolP("stamp", "s", false, "Prefix message with current timestamp")

	return cmd
}

func execute(level string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		indentation, _ := cmd.Flags().GetInt("indent")
		withPipe, _ := cmd.Flags().GetBool("pipe")
		withStamp, _ := cmd.Flags().GetBool("stamp")

		if withPipe {
			logger.Pipe(cmd.InOrStdin(), cmd.OutOrStdout(), level, indentation, withStamp)
		} else {
			logger.Log(cmd.OutOrStdout(), level, args[0], indentation, withStamp)
		}
		return nil
	}
}

func validate(cmd *cobra.Command, args []string) error {
	withPipe, _ := cmd.Flags().GetBool("pipe")

	if withPipe {
		return cobra.NoArgs(cmd, args)
	}

	return cobra.ExactArgs(1)(cmd, args)
}

func init() {
	LogCmd.PersistentFlags().BoolP("pipe", "p", false, "Loop through piped output")

	LogCmd.AddCommand(infoCmd, errorCmd, warnCmd, debugCmd, stampCmd)
}
