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

func PrintErrorWithOptions(writer io.Writer, message string, options [][]string) {
	PrintError(writer, message)
	fmt.Fprintln(writer)
	for _, opt := range options {
		fmt.Fprintf(writer, "  %s %s\n", Code().Render(opt[0]), Muted().Render(opt[1]))
	}
}

func PrintSuccessWithDetails(writer io.Writer, message string, details [][]string) {
	PrintSuccess(writer, message)
	for _, detail := range details {
		if len(detail) >= 2 {
			PrintKeyValue(writer, detail[0], detail[1])
		}
	}
}

func PrintSuccessWithDetailsCode(writer io.Writer, message string, details [][]string) {
	PrintSuccess(writer, message)
	for _, detail := range details {
		if len(detail) >= 2 {
			PrintKeyCode(writer, detail[0], detail[1])
		}
	}
}

func PrintHints(writer io.Writer, hints [][]string) {
	fmt.Fprintf(writer, "\n%s\n", Muted().Render("Quick actions:"))
	for _, hint := range hints {
		if len(hint) >= 2 {
			fmt.Fprintf(writer, "  %s %s\n", Code().Render(hint[0]), Muted().Render(hint[1]))
		}
	}
}
