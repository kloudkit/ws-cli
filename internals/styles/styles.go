package styles

import "github.com/charmbracelet/lipgloss"

func Header() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorHeader)).
			Bold(true),
	)
}

func SubHeader() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorInfo)).
			Bold(true),
	)
}

func Key() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().Foreground(lipgloss.Color(ColorAccent)),
	)
}

func Value() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().Foreground(lipgloss.Color(ColorText)),
	)
}

func Info() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().Foreground(lipgloss.Color(ColorInfo)),
	)
}

func Success() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSuccess)),
	)
}

func Warning() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWarning)),
	)
}

func Error() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().Foreground(lipgloss.Color(ColorError)),
	)
}

func Muted() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().Foreground(lipgloss.Color(ColorMuted)),
	)
}
