package styles

import "github.com/charmbracelet/lipgloss"

func Header() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(Mauve)).
		Bold(true)
}

func SubHeader() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(Blue)).
		Bold(true)
}

func Key() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(Teal)).
		Bold(true)
}

func Value() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(Text))
}

func Info() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(Blue))
}

func Success() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(Green))
}

func Warning() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(Yellow))
}

func Error() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(Red)).
		Bold(true)
}

func Muted() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(Overlay0))
}

func Badge() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(Base)).
		Background(lipgloss.Color(Surface1)).
		Align(lipgloss.Center).
		Bold(true).
		Padding(0, 2)
}

func SuccessBadge() lipgloss.Style {
	return Badge().Background(lipgloss.Color(Green))
}

func WarningBadge() lipgloss.Style {
	return Badge().Background(lipgloss.Color(Yellow))
}

func ErrorBadge() lipgloss.Style {
	return Badge().Background(lipgloss.Color(Red))
}

func InfoBadge() lipgloss.Style {
	return Badge().Background(lipgloss.Color(Blue))
}

func TipBadge() lipgloss.Style {
	return Badge().Background(lipgloss.Color(Mauve))
}

func Highlighted() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(Surface0)).
		Padding(0, 1).
		Margin(0, 0)
}

func Code() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(Teal)).
		Background(lipgloss.Color(Surface1)).
		Padding(0, 1)
}
