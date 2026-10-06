package styles

import (
	"fmt"
	"io"
)

func OutputRaw(w io.Writer, raw bool, value string) bool {
	if raw {
		fmt.Fprintln(w, value)
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
	fmt.Fprintln(writer, Title().Render(title))
}

func PrintSuccess(writer io.Writer, message string) {
	fmt.Fprintln(writer, Success().Render("✓ "+message))
}

func PrintWarning(writer io.Writer, message string) {
	fmt.Fprintln(writer, Warning().Render("⚠ "+message))
}

func PrintError(writer io.Writer, message string) {
	fmt.Fprintln(writer, ErrorBadge().Render("ERROR"))
	fmt.Fprintln(writer, Error().Render(message))
}

func PrintErrorWithOptions(writer io.Writer, message string, options [][]string) {
	PrintError(writer, message)
	fmt.Fprintln(writer)
	printPairs(writer, options)
}

func PrintHints(writer io.Writer, hints [][]string) {
	fmt.Fprintf(writer, "\n%s\n", Muted().Render("Quick actions:"))
	printPairs(writer, hints)
}

func printPairs(writer io.Writer, pairs [][]string) {
	for _, pair := range pairs {
		if len(pair) >= 2 {
			fmt.Fprintf(writer, "  %s %s\n", Code().Render(pair[0]), Muted().Render(pair[1]))
		}
	}
}
