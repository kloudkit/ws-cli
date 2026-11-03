package styles

import (
	"fmt"
	"io"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const indent = "  "

type Pair struct {
	Key   string
	Value string
}

type DescriptionItem struct {
	Name        string
	Description string
}

type Hint struct {
	Command     string
	Description string
}

func Check(value bool) string {
	if value {
		return "✓"
	}

	return ""
}

func line(w io.Writer, style lipgloss.Style, text string) {
	fmt.Fprintf(w, "%s%s\n", indent, style.Render(text))
}

func pad(text string, width int) string {
	return text + strings.Repeat(" ", width-ansi.StringWidth(text))
}

func widest[T any](items []T, key func(T) string) int {
	width := 0
	for _, item := range items {
		width = max(width, ansi.StringWidth(key(item)))
	}

	return width
}

func PrintTitle(w io.Writer, title string, extras ...any) {
	title = Header.Render(title)
	for _, extra := range extras {
		if n, ok := extra.(int); ok {
			title += " " + Count.Render(fmt.Sprintf("(%d)", n))
		} else {
			title += " " + Muted.Render(fmt.Sprint(extra))
		}
	}

	fmt.Fprintf(w, "\n%s%s\n\n", indent, title)
}

func PrintSubHeader(w io.Writer, text string) { line(w, SubHeader, text) }

func PrintMuted(w io.Writer, text string) { line(w, Muted, text) }

func PrintEmpty(w io.Writer, message string) {
	fmt.Fprintln(w)
	line(w, Muted, "○ "+message)
}

func PrintWarning(w io.Writer, message string) { line(w, Warning, "⚠ "+message) }

func PrintSuccess(w io.Writer, message string, details ...Pair) {
	if len(details) == 0 {
		line(w, Success, "✓ "+message)
		return
	}

	fmt.Fprintf(w, "\n%s%s\n\n", indent, Success.Bold(true).Render("✓ "+message))
	PrintKeyValues(w, details...)
}

func PrintText(w io.Writer, text string) {
	for row := range strings.SplitSeq(text, "\n") {
		if row == "" {
			fmt.Fprintln(w)
		} else {
			line(w, Value, row)
		}
	}
}

func PrintCode(w io.Writer, text string) {
	gutter := Border.Render("│")

	for row := range strings.SplitSeq(strings.TrimRight(text, "\n"), "\n") {
		if row = strings.TrimRight(row, " \t"); row == "" {
			fmt.Fprintf(w, "%s%s\n", indent, gutter)
		} else {
			fmt.Fprintf(w, "%s%s %s\n", indent, gutter, Value.Render(row))
		}
	}
}

func PrintKeyValues(w io.Writer, pairs ...Pair) {
	width := widest(pairs, func(p Pair) string { return p.Key })
	continuation := indent + strings.Repeat(" ", width+2)

	for _, pair := range pairs {
		rows := strings.Split(pair.Value, "\n")
		fmt.Fprintf(w, "%s%s  %s\n", indent, Key.Render(pad(pair.Key, width)), Value.Render(rows[0]))

		for _, row := range rows[1:] {
			if row == "" {
				fmt.Fprintln(w)
			} else {
				fmt.Fprintf(w, "%s%s\n", continuation, Value.Render(row))
			}
		}
	}
}

func PrintList(w io.Writer, items []DescriptionItem) {
	width := widest(items, func(i DescriptionItem) string { return i.Name })
	bullet, dash := Muted.Render("•"), Muted.Render("—")

	for _, item := range items {
		fmt.Fprintf(w, "%s%s %s %s %s\n", indent, bullet, Key.Render(pad(item.Name, width)), dash, Value.Render(item.Description))
	}
}

func PrintBullets(w io.Writer, names []string) {
	for _, name := range names {
		fmt.Fprintf(w, "%s%s %s\n", indent, Muted.Render("•"), Value.Render(name))
	}
}

func PrintHints(w io.Writer, hints ...Hint) {
	width := widest(hints, func(h Hint) string { return h.Command })

	fmt.Fprintf(w, "\n%s%s\n", indent, Muted.Render("Quick actions"))

	for _, hint := range hints {
		fmt.Fprintf(w, "%s%s%s  %s\n", indent, indent, Key.Render(pad(hint.Command, width)), Muted.Render(hint.Description))
	}
}
