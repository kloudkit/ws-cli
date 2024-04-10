package log

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var LogCmd = &cobra.Command{
	Use:   "log",
	Short: "Log messages to the console",
}

var debugCmd = createCommand("debug", "debugging")
var errorCmd = createCommand("error", "error")
var infoCmd = createCommand("info", "information")
var warnCmd = createCommand("warn", "warning")
var stampCmd = &cobra.Command{
	Use:   "stamp",
	Short: "Log the current timestamp to the console",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
    withPipe, _ := cmd.Flags().GetBool("pipe")

		if withPipe {
			pipe(cmd.InOrStdin(), cmd.OutOrStdout(), "", 0, true)
		} else {
			log(cmd.OutOrStdout(), "", "", 0, true)
		}
	},
}

func createCommand(short, long string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("%s message", short),
		Short: fmt.Sprintf("Log a %s messages to the console", long),
		Args:  validate,
		Run:   execute(short),
	}

	cmd.Flags().IntP("indent", "i", 0, "Desired prefixed indentation")
	cmd.Flags().BoolP("stamp", "s", false, "Prefix message with current timestamp")

	return cmd
}

func timestamp() string {
	return time.Now().UTC().Format("[2006-01-02T15:04:05.000Z]")
}

func execute(level string) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		indentation, _ := cmd.Flags().GetInt("indent")
		withPipe, _ := cmd.Flags().GetBool("pipe")
		withStamp, _ := cmd.Flags().GetBool("stamp")

		if withPipe {
			pipe(cmd.InOrStdin(), cmd.OutOrStdout(), level, indentation, withStamp)
		} else {
			log(cmd.OutOrStdout(), level, args[0], indentation, withStamp)
		}
	}
}

func pipe(reader io.Reader, writer io.Writer, level string, indent int, withStamp bool) {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()

		log(writer, level, line, indent, withStamp)
	}
}

func log(writer io.Writer, level, message string, indent int, withStamp bool) {
  stamp := ""
  prefix := ""

  if withStamp {
    stamp = timestamp() + " "
  }

  if len(level) > 0 {
    level = fmt.Sprintf("%-5s ", level)
  }

	if indent > 0 {
		prefix = strings.Repeat("  ", indent) + "- "
	}

	fmt.Fprintln(writer, stamp+level+prefix+message)
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
