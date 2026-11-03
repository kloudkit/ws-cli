package log

import (
	"fmt"

	"github.com/kloudkit/ws-cli/cmd/cli"
	"github.com/kloudkit/ws-cli/internals/logger"
	"github.com/spf13/cobra"
)

func logCmd(level, use, short, long string) *cobra.Command {
	cmd := cli.New(use, "0.2.0", short, long, func(cmd *cobra.Command, args []string) error {
		stamp, indent := level == "" || cli.Bool(cmd, "stamp"), cli.Int(cmd, "indent")

		if cli.Bool(cmd, "pipe") {
			logger.Pipe(cmd.InOrStdin(), cmd.OutOrStdout(), level, indent, stamp)
		} else {
			logger.Log(cmd.OutOrStdout(), level, append(args, "")[0], indent, stamp)
		}

		return nil
	})

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if level == "" || cli.Bool(cmd, "pipe") {
			return cobra.NoArgs(cmd, args)
		}

		return cobra.ExactArgs(1)(cmd, args)
	}

	if level != "" {
		cmd.Flags().IntP("indent", "i", 0, "Desired prefixed indentation")
		cmd.Flags().BoolP("stamp", "s", false, "Prefix message with current timestamp")
	}

	return cmd
}

func levelCmd(level, name string) *cobra.Command {
	return logCmd(level, level+" message", "Log "+name+" messages",
		fmt.Sprintf("Emit a log line at %s level in the workspace style — the same formatting the startup scripts use. --indent nests it under a preceding line, --stamp prefixes a timestamp.", level))
}

var stripCmd = cli.New("strip", "0.5.3", "Strip ANSI escapes from stdin",
	"Copy stdin to stdout line by line with every ANSI escape sequence removed. The entrypoint uses it to keep the workspace log file plain while the console stays colored.",
	func(cmd *cobra.Command, args []string) error { return logger.Strip(cmd.InOrStdin(), cmd.OutOrStdout()) })

var LogCmd = cli.Group("log", "0.2.0", "Log messages",
	"Emit styled, level-tagged log lines — the same formatting the startup scripts use. --pipe runs each line of piped input through the logger.",
	`# Emit an info-level line
ws log info "starting up"

# Nest a warning under it with a timestamp
ws log warn "slow response" --indent 1 --stamp`,
	levelCmd("info", "information"), levelCmd("error", "error"), levelCmd("warn", "warning"), levelCmd("debug", "debugging"),
	logCmd("", "stamp", "Log the current timestamp", "Print just the current timestamp in the workspace log style — handy for marking phases in a startup log."),
	stripCmd)

func init() {
	stripCmd.Hidden = true
	stripCmd.Args = cobra.NoArgs
	LogCmd.PersistentFlags().BoolP("pipe", "p", false, "Loop through piped output")
}
