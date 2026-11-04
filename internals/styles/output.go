package styles

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	chromastyles "github.com/alecthomas/chroma/v2/styles"
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

const titleWidth = 64

func PrintTitle(w io.Writer, title string, extras ...any) {
	width := titleWidth
	if terminal := TerminalWidth(); terminal > 0 {
		width = min(width, terminal-2*len(indent))
	}

	printTitle(w, width, title, extras)
}

func printTitle(w io.Writer, width int, title string, extras []any) {
	tail := make([]string, len(extras))
	for i, extra := range extras {
		if n, ok := extra.(int); ok {
			tail[i] = Count.Render(strconv.Itoa(n))
		} else {
			tail[i] = Muted.Render(fmt.Sprint(extra))
		}
	}

	suffix := strings.Join(tail, " ")
	if suffix != "" {
		suffix = " " + suffix
	}

	rule := max(width-ansi.StringWidth(title)-ansi.StringWidth(suffix)-1, 3)
	fmt.Fprintf(w, "\n%s%s %s%s\n\n", indent, Header.Render(title), Border.Render(strings.Repeat("─", rule)), suffix)
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

func Aside(text string) string { return Muted.Render("(" + text + ")") }

func PrintCode(w io.Writer, text, filename string, first int) {
	rows := highlight(text, filename)
	width := len(strconv.Itoa(first + len(rows) - 1))

	for i, row := range rows {
		number := Muted.Render(fmt.Sprintf("%*d", width, first+i))
		if row == "" {
			fmt.Fprintf(w, "%s%s %s\n", indent, number, Border.Render("│"))
		} else {
			fmt.Fprintf(w, "%s%s %s %s\n", indent, number, Border.Render("│"), row)
		}
	}
}

func highlight(text, filename string) []string {
	raw := strings.Split(strings.TrimRight(text, "\n"), "\n")
	for i, row := range raw {
		raw[i] = strings.TrimRight(row, " \t")
	}

	var tokens []chroma.Token
	if lexer := lexers.Match(filename); lexer != nil {
		tokens, _ = chroma.Tokenise(chroma.Coalesce(lexer), nil, strings.Join(raw, "\n"))
	}

	if len(tokens) == 0 {
		for i, row := range raw {
			if row != "" {
				raw[i] = Value.Render(row)
			}
		}

		return raw
	}

	theme, rows := chromastyles.Get("catppuccin-frappe"), []string{""}
	for _, token := range tokens {
		style := Value
		if entry := theme.Get(token.Type); entry.Colour.IsSet() {
			style = lipgloss.NewStyle().Foreground(lipgloss.Color(entry.Colour.String())).Italic(entry.Italic == chroma.Yes).Bold(entry.Bold == chroma.Yes)
		}

		for i, part := range strings.Split(token.Value, "\n") {
			if i > 0 {
				rows = append(rows, "")
			}

			if part != "" {
				rows[len(rows)-1] += style.Render(part)
			}
		}
	}

	return rows[:len(raw)]
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
