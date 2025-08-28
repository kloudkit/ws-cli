package styles

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

func TableBorderStyle() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().Foreground(lipgloss.Color(ColorBorder)),
	)
}

func TableHeaderStyle() lipgloss.Style {
	return WithColor(
		Header().
			Align(lipgloss.Center).
			Padding(0, 1),
		lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorText)).
			Padding(0, 1),
	)
}

func TableCellStyle() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorText)).
			Padding(0, 1),
	)
}

func TableKeyCellStyle() lipgloss.Style {
	return WithColor(
		Key().
			Padding(0, 1).
			Bold(true),
		TableCellStyle(),
	)
}

func TableRowTitleStyle() lipgloss.Style {
	return WithColor(
		TableKeyCellStyle().
			Bold(true),
		TableCellStyle(),
	)
}

func Table(headers ...string) *table.Table {
	return table.New().
		Headers(headers...).
		Border(lipgloss.RoundedBorder()).
		BorderStyle(TableBorderStyle()).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return TableHeaderStyle()
			}

			if col == 0 {
				return TableRowTitleStyle()
			}

			return TableCellStyle().Width(65)
		})
}
