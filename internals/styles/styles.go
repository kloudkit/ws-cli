package styles

import "charm.land/lipgloss/v2"

func Header() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Mauve).
		Bold(true)
}

func SubHeader() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Blue).
		Bold(true)
}

func Key() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Teal).
		Bold(true)
}

func Value() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Text)
}

func Info() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Blue)
}

func Success() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Green)
}

func Warning() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Yellow)
}

func Error() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Red).
		Bold(true)
}

func Muted() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Overlay0)
}

func Count() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Overlay1)
}
