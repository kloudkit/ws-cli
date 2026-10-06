package secrets

import (
	"fmt"
	"io"

	internalIO "github.com/kloudkit/ws-cli/internals/io"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

// emit writes value to --output when set; otherwise it prints value as-is under
// --raw (with a trailing newline when newline is set), or renders it via styled.
func emit(cmd *cobra.Command, value, successMsg string, newline bool, styled func(io.Writer)) error {
	out := cmd.OutOrStdout()
	file, _ := cmd.Flags().GetString("output")
	raw, _ := cmd.Flags().GetBool("raw")

	switch {
	case file != "":
		mode, _ := cmd.Flags().GetString("mode")
		force, _ := cmd.Flags().GetBool("force")

		if err := internalIO.WriteSecureFile(file, []byte(value+"\n"), mode, force); err != nil {
			return err
		}

		if !raw {
			styles.PrintSuccess(out, successMsg)
			styles.PrintKeyCode(out, "Output", file)
		}
	case raw && newline:
		fmt.Fprintln(out, value)
	case raw:
		fmt.Fprint(out, value)
	default:
		styled(out)
	}

	return nil
}

func printValue(title, value string) func(io.Writer) {
	return func(out io.Writer) {
		styles.PrintTitle(out, title)
		styles.PrintKeyCode(out, "Value", value)
	}
}

func printKey(header, value, hint string) func(io.Writer) {
	return func(out io.Writer) {
		fmt.Fprintln(out, styles.Header().Render(header))
		fmt.Fprintln(out, "  "+styles.Code().Render(value))
		fmt.Fprintln(out, styles.Muted().Render("💡 "+hint))
	}
}
