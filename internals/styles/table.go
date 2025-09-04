package styles

import (
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/lipgloss/v2/table"
)

func TableBorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Surface2)
}

func TableHeaderStyle() lipgloss.Style {
	return Header().
		Align(lipgloss.Center).
		Padding(0, 1)
}

func TableCellStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Text).
		Padding(0, 1)
}

func TableKeyCellStyle() lipgloss.Style {
	return Key().
		Padding(0, 1).
		Bold(true)
}

func TableRowTitleStyle() lipgloss.Style {
	return TableKeyCellStyle().
		Bold(true)
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
