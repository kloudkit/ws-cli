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
		lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorAccent)).
			Bold(true),
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
		lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorError)).
			Bold(true),
	)
}

func Muted() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().Foreground(lipgloss.Color(ColorMuted)),
	)
}

func Badge() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorBase)).
			Background(lipgloss.Color(BgAccent)).
			Align(lipgloss.Center).
			Bold(true).
			Padding(0, 2),
	)
}

func SuccessBadge() lipgloss.Style {
	return WithColor(Badge().Background(lipgloss.Color(BgSuccess)))
}

func WarningBadge() lipgloss.Style {
	return WithColor(Badge().Background(lipgloss.Color(BgWarning)))
}

func ErrorBadge() lipgloss.Style {
	return WithColor(Badge().Background(lipgloss.Color(BgError)))
}

func InfoBadge() lipgloss.Style {
	return WithColor(Badge().Background(lipgloss.Color(BgInfo)))
}

func TipBadge() lipgloss.Style {
	return WithColor(Badge().Background(lipgloss.Color(ColorHeader)))
}

func Highlighted() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().
			Background(lipgloss.Color(BgMuted)).
			Padding(0, 1).
			Margin(0, 0),
	)
}

func Code() lipgloss.Style {
	return WithColor(
		lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorAccent)).
			Background(lipgloss.Color(BgAccent)).
			Padding(0, 1),
	)
}
