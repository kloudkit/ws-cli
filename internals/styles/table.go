package styles

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

func Table(headers ...string) *table.Table {
	return table.New().
		Headers(headers...).
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(Surface2)).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return Header().Align(lipgloss.Center).Padding(0, 1)
			}

			if col == 0 {
				return Key().Padding(0, 1)
			}

			return lipgloss.NewStyle().Foreground(Text).Padding(0, 1).Width(65)
		})
}
