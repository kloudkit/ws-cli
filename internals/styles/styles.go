package styles

import "github.com/charmbracelet/lipgloss"

func HeaderStyle() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorHeader)).
			Bold(true),
	)
}

func SubHeaderStyle() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorInfo)).
			Bold(true),
	)
}

func KeyStyle() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().Foreground(lipgloss.Color(ColorAccent)),
	)
}

func ValueStyle() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().Foreground(lipgloss.Color(ColorText)),
	)
}

func InfoStyle() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().Foreground(lipgloss.Color(ColorInfo)),
	)
}

func SuccessStyle() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSuccess)),
	)
}

func WarningStyle() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWarning)),
	)
}

func ErrorStyle() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().Foreground(lipgloss.Color(ColorError)),
	)
}

func MutedStyle() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().Foreground(lipgloss.Color(ColorMuted)),
	)
}
