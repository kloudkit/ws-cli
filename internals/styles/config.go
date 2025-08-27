package styles

import "github.com/charmbracelet/lipgloss"

var ColorEnabled = true

func WithColor(style lipgloss.Style, fallback ...lipgloss.Style) lipgloss.Style {
	if ColorEnabled {
		return style
	}

	if len(fallback) > 0 {
		return fallback[0]
	}

	return lipgloss.NewStyle()
}
