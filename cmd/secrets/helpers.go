package secrets

import (
	"fmt"
	"io"

	internalIO "github.com/kloudkit/ws-cli/internals/io"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

type outputConfig struct {
	file  string
	mode  string
	force bool
	raw   bool
}

func getOutputConfig(cmd *cobra.Command) outputConfig {
	outputFile, _ := cmd.Flags().GetString("output")
	modeStr, _ := cmd.Flags().GetString("mode")
	force, _ := cmd.Flags().GetBool("force")
	raw, _ := cmd.Flags().GetBool("raw")

	return outputConfig{
		file:  outputFile,
		mode:  modeStr,
		force: force,
		raw:   raw,
	}
}

func handleOutput(cmd *cobra.Command, cfg outputConfig, value string, title string, successMsg string, addNewline bool) error {
	if cfg.file == "" {
		return writeToStdout(cmd.OutOrStdout(), cfg.raw, value, title, addNewline)
	}
	return writeToFile(cmd.OutOrStdout(), cfg, value, successMsg)
}

func handleCustomOutput(cmd *cobra.Command, cfg outputConfig, value string, successMsg string, styledOutput func(io.Writer)) error {
	if cfg.file == "" {
		if cfg.raw {
			fmt.Fprintln(cmd.OutOrStdout(), value)
			return nil
		}
		styledOutput(cmd.OutOrStdout())
		return nil
	}
	return writeToFile(cmd.OutOrStdout(), cfg, value, successMsg)
}

func writeToStdout(out io.Writer, raw bool, value string, title string, addNewline bool) error {
	if raw {
		if addNewline {
			fmt.Fprintln(out, value)
		} else {
			fmt.Fprint(out, value)
		}
		return nil
	}
	styles.PrintTitle(out, title)
	styles.PrintKeyCode(out, "Value", value)
	return nil
}

func writeToFile(out io.Writer, cfg outputConfig, value string, successMsg string) error {
	if err := internalIO.WriteSecureFile(cfg.file, []byte(value+"\n"), cfg.mode, cfg.force); err != nil {
		return err
	}
	if !cfg.raw {
		styles.PrintSuccessWithDetailsCode(out, successMsg, [][]string{
			{"Output", cfg.file},
		})
	}
	return nil
}
