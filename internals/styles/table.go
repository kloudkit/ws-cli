package styles

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

var (
	noColorTableHeaderStyle = lipgloss.NewStyle().
				Align(lipgloss.Center).
				PaddingLeft(1).
				PaddingRight(1)

	tableBorderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorBorder))

	tableCellStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorText)).
			PaddingLeft(1).
			PaddingRight(1)
)

func TableBorderStyle() lipgloss.Style {
	if ColorEnabled {
		return tableBorderStyle
	}

	return NoColorStyle
}

func TableHeaderStyle() lipgloss.Style {
	if ColorEnabled {
		return HeaderStyle().
			Align(lipgloss.Center).
			PaddingLeft(1).
			PaddingRight(1)
	}

	return noColorTableHeaderStyle
}

func TableCellStyle() lipgloss.Style {
	return tableCellStyle
}

func TableKeyCellStyle() lipgloss.Style {
	if ColorEnabled {
		return KeyStyle().
			PaddingLeft(1).
			PaddingRight(1)
	}

	return tableCellStyle
}

func TableRowTitleStyle() lipgloss.Style {
	if ColorEnabled {
		return TableKeyCellStyle().
			Bold(true)
	}

	return tableCellStyle
}

func Table(headers ...string) *table.Table {
	return table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(TableBorderStyle())
}
