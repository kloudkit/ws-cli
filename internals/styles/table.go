package styles

import (
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/x/ansi"
)

const minColumnWidth = 8

type Table struct {
	headers []string
	rows    [][]string
}

func NewTable(headers ...string) *Table {
	return &Table{headers: headers}
}

func (t *Table) Rows(rows ...[]string) *Table {
	t.rows = append(t.rows, rows...)
	return t
}

func (t *Table) Render() string {
	return t.RenderWidth(TerminalWidth())
}

func (t *Table) RenderWidth(maxWidth int) string {
	widths := t.columnWidths(maxWidth - len(indent))

	wrapped := make([][]string, len(t.rows))
	for i, row := range t.rows {
		wrapped[i] = make([]string, len(widths))
		for j := range widths {
			if j < len(row) {
				wrapped[i][j] = wrapCell(row[j], widths[j])
			}
		}
	}

	tbl := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(Surface2)).
		StyleFunc(func(row, col int) lipgloss.Style {
			base := lipgloss.NewStyle().Padding(0, 1).Width(widths[col] + 2)

			switch {
			case row == table.HeaderRow:
				return base.Inherit(Header())
			case col == 0:
				return base.Inherit(Key())
			default:
				return base.Inherit(Value())
			}
		}).
		Rows(wrapped...)

	if len(t.headers) > 0 {
		tbl.Headers(t.headers...)
	}

	lines := strings.Split(tbl.Render(), "\n")
	for i, line := range lines {
		lines[i] = indent + line
	}

	return strings.Join(lines, "\n")
}

func (t *Table) columnWidths(available int) []int {
	columns := len(t.headers)
	for _, row := range t.rows {
		columns = max(columns, len(row))
	}

	widths := make([]int, columns)
	for _, row := range append([][]string{t.headers}, t.rows...) {
		for i, cell := range row {
			for line := range strings.SplitSeq(cell, "\n") {
				widths[i] = max(widths[i], ansi.StringWidth(line))
			}
		}
	}

	if available <= 0 {
		return widths
	}

	total := columns + 1
	for _, w := range widths {
		total += w + 2
	}

	order := make([]int, columns)
	for i := range order {
		order[i] = i
	}

	for i := 1; i < len(order); i++ {
		for j := i; j > 0 && widths[order[j]] > widths[order[j-1]]; j-- {
			order[j], order[j-1] = order[j-1], order[j]
		}
	}

	for _, floor := range []int{minColumnWidth, 1} {
		for _, col := range order {
			if total <= available {
				return widths
			}

			shrink := min(total-available, widths[col]-floor)
			if shrink > 0 {
				widths[col] -= shrink
				total -= shrink
			}
		}
	}

	return widths
}

func wrapCell(text string, width int) string {
	var out []string
	for line := range strings.SplitSeq(text, "\n") {
		out = append(out, wrapLine(line, width)...)
	}

	return strings.Join(out, "\n")
}

func wrapLine(text string, width int) []string {
	if ansi.StringWidth(text) <= width {
		return []string{text}
	}

	var tokens []string
	if strings.Contains(text, " ") {
		for i, word := range strings.Fields(text) {
			if i > 0 {
				word = " " + word
			}
			tokens = append(tokens, word)
		}
	} else {
		tokens = strings.SplitAfter(text, "/")
	}

	var lines []string
	current := ""

	for _, token := range tokens {
		if current != "" && ansi.StringWidth(current+token) > width {
			lines = append(lines, current)
			token = strings.TrimPrefix(token, " ")
			current = ""
		}

		current += token

		for ansi.StringWidth(current) > width {
			lines = append(lines, ansi.Truncate(current, width, ""))
			current = ansi.TruncateLeft(current, width, "")
		}
	}

	if current != "" || len(lines) == 0 {
		lines = append(lines, current)
	}

	return lines
}
