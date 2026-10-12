package styles

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

const indent = "  "

func OutputRaw(w io.Writer, raw bool, value string) bool {
	if raw {
		fmt.Fprintln(w, value)
		return true
	}
	return false
}

func Check(value bool) string {
	if value {
		return "✓"
	}

	return ""
}

func PrintTitle(writer io.Writer, title string) {
	fmt.Fprintf(writer, "\n%s%s\n\n", indent, Header().Render(title))
}

func PrintTitleCount(writer io.Writer, title string, count int) {
	fmt.Fprintf(writer, "\n%s%s %s\n\n", indent, Header().Render(title), Count().Render(fmt.Sprintf("(%d)", count)))
}

func PrintSubHeader(writer io.Writer, text string) {
	fmt.Fprintf(writer, "%s%s\n", indent, SubHeader().Render(text))
}

func PrintText(writer io.Writer, text string) {
	for line := range strings.SplitSeq(text, "\n") {
		if line == "" {
			fmt.Fprintln(writer)
			continue
		}

		fmt.Fprintf(writer, "%s%s\n", indent, Value().Render(line))
	}
}

func PrintMuted(writer io.Writer, text string) {
	fmt.Fprintf(writer, "%s%s\n", indent, Muted().Render(text))
}

type Pair struct {
	Key   string
	Value string
}

func PrintKeyValues(writer io.Writer, pairs ...Pair) {
	width := 0
	for _, pair := range pairs {
		width = max(width, ansi.StringWidth(pair.Key))
	}

	for _, pair := range pairs {
		key := pair.Key + strings.Repeat(" ", width-ansi.StringWidth(pair.Key))
		lines := strings.Split(pair.Value, "\n")
		fmt.Fprintf(writer, "%s%s  %s\n", indent, Key().Render(key), Value().Render(lines[0]))

		for _, line := range lines[1:] {
			if line == "" {
				fmt.Fprintln(writer)
				continue
			}

			fmt.Fprintf(writer, "%s%s%s\n", indent, strings.Repeat(" ", width+2), Value().Render(line))
		}
	}
}

func PrintSuccess(writer io.Writer, message string) {
	fmt.Fprintln(writer, Success().Render("✓ "+message))
}

func PrintWarning(writer io.Writer, message string) {
	fmt.Fprintln(writer, Warning().Render("⚠ "+message))
}

func PrintEmpty(writer io.Writer, message string) {
	fmt.Fprintf(writer, "%s%s\n", indent, Muted().Render("○ "+message))
}

type Hint struct {
	Command     string
	Description string
}

func PrintHints(writer io.Writer, hints ...Hint) {
	width := 0
	for _, hint := range hints {
		width = max(width, ansi.StringWidth(hint.Command))
	}

	fmt.Fprintf(writer, "\n%s%s\n", indent, Muted().Render("Quick actions"))

	for _, hint := range hints {
		command := hint.Command + strings.Repeat(" ", width-ansi.StringWidth(hint.Command))
		fmt.Fprintf(writer, "%s%s%s  %s\n", indent, indent, Key().Render(command), Muted().Render(hint.Description))
	}
}
