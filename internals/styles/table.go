package styles

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

var (
	tableCellStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorText)).
		PaddingLeft(1).
		PaddingRight(1)
)

func TableBorderStyle() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().Foreground(lipgloss.Color(ColorBorder)),
	)
}

func TableHeaderStyle() lipgloss.Style {
	return WithColor(
		HeaderStyle().
			Align(lipgloss.Center).
			PaddingLeft(1).
			PaddingRight(1),
		lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorText)).
			PaddingLeft(1).
			PaddingRight(1),
	)
}

func TableCellStyle() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorText)).
			PaddingLeft(1).
			PaddingRight(1),
	)
}

func TableKeyCellStyle() lipgloss.Style {
	return WithColor(
		KeyStyle().
			PaddingLeft(1).
			PaddingRight(1),
		tableCellStyle,
	)
}

func TableRowTitleStyle() lipgloss.Style {
  return WithColor(
    TableKeyCellStyle().
			Bold(true),
    tableCellStyle,
  )
}

func Table(headers ...string) *table.Table {
	return table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(TableBorderStyle())
}
