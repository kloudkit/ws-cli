package styles

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func OutputRaw(cmd *cobra.Command, value string) bool {
	raw, _ := cmd.Flags().GetBool("raw")
	if raw {
		fmt.Fprintln(cmd.OutOrStdout(), value)
		return true
	}
	return false
}

func PrintKeyValue(writer io.Writer, key, value string) {
	fmt.Fprintf(writer, "  %s %s\n", Key().Render(key+":"), Value().Render(value))
}

func PrintKeyCode(writer io.Writer, key, value string) {
	fmt.Fprintf(writer, "  %s %s\n", Key().Render(key+":"), Code().Render(value))
}

func PrintTitle(writer io.Writer, title string) {
	fmt.Fprintf(writer, "%s\n", Title().Render(title))
}

func PrintSuccess(writer io.Writer, message string) {
	fmt.Fprintf(writer, "%s\n", Success().Render("✓ "+message))
}

func PrintWarning(writer io.Writer, message string) {
	fmt.Fprintf(writer, "%s\n", Warning().Render("⚠ "+message))
}

func PrintError(writer io.Writer, message string) {
	fmt.Fprintf(writer, "%s\n", ErrorBadge().Render("ERROR"))
	fmt.Fprintf(writer, "%s\n", Error().Render(message))
}
