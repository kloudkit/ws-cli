package styles

import (
	catppuccin "github.com/catppuccin/go"
	"github.com/charmbracelet/lipgloss"
	ilog "github.com/kloudkit/ws-cli/internals/log"
)

var Frappe = catppuccin.Frappe

var (
	colorText    = Frappe.Text().Hex
	colorSubtext = Frappe.Subtext1().Hex

	colorInfo    = Frappe.Blue().Hex
	colorSuccess = Frappe.Green().Hex
	colorWarning = Frappe.Yellow().Hex
	colorError   = Frappe.Red().Hex
	colorMuted   = Frappe.Overlay0().Hex
	colorAccent  = Frappe.Teal().Hex
	colorHeader  = Frappe.Mauve().Hex
	colorBorder  = Frappe.Surface2().Hex
)

var (
	noColorStyle = lipgloss.NewStyle()

	noColorTableHeaderStyle = lipgloss.NewStyle().
				Align(lipgloss.Center).
				PaddingLeft(1).
				PaddingRight(1)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorHeader)).
			Bold(true)

	subHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorInfo)).
			Bold(true)

	keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorAccent))

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorText))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorInfo))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorSuccess))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorWarning))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorError))

	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMuted))

	tableBorderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorBorder))

	tableHeaderStyle = headerStyle.
				Align(lipgloss.Center).
				PaddingLeft(1).
				PaddingRight(1)

	tableCellStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorText)).
			PaddingLeft(1).
			PaddingRight(1)

	tableKeyCellStyle = keyStyle.
				PaddingLeft(1).
				PaddingRight(1)

	tableRowTitleStyle = tableKeyCellStyle.
				Bold(true)
)

func HeaderStyle() lipgloss.Style {
	if ilog.ColorEnabled {
		return headerStyle
	}

	return noColorStyle
}

func SubHeaderStyle() lipgloss.Style {
	if ilog.ColorEnabled {
		return subHeaderStyle
	}

	return noColorStyle
}

func KeyStyle() lipgloss.Style {
	if ilog.ColorEnabled {
		return keyStyle
	}

	return noColorStyle
}

func ValueStyle() lipgloss.Style {
	if ilog.ColorEnabled {
		return valueStyle
	}

	return noColorStyle
}

func InfoStyle() lipgloss.Style {
	if ilog.ColorEnabled {
		return infoStyle
	}

	return noColorStyle
}

func SuccessStyle() lipgloss.Style {
	if ilog.ColorEnabled {
		return successStyle
	}

	return noColorStyle
}

func WarningStyle() lipgloss.Style {
	if ilog.ColorEnabled {
		return warningStyle
	}

	return noColorStyle
}

func ErrorStyle() lipgloss.Style {
	if ilog.ColorEnabled {
		return errorStyle
	}

	return noColorStyle
}

func MutedStyle() lipgloss.Style {
	if ilog.ColorEnabled {
		return mutedStyle
	}

	return noColorStyle
}

func TableBorderStyle() lipgloss.Style {
	if ilog.ColorEnabled {
		return tableBorderStyle
	}

	return noColorStyle
}

func TableHeaderStyle() lipgloss.Style {
	if ilog.ColorEnabled {
		return tableHeaderStyle
	}

	return noColorTableHeaderStyle
}

func TableCellStyle() lipgloss.Style {
	return tableCellStyle
}

func TableKeyCellStyle() lipgloss.Style {
	if ilog.ColorEnabled {
		return tableKeyCellStyle
	}

	return tableCellStyle
}

func TableRowTitleStyle() lipgloss.Style {
	if ilog.ColorEnabled {
		return tableRowTitleStyle
	}

	return tableCellStyle
}
