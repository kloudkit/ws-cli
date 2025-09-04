package styles

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
)

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

func ErrorBadge() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Text).
		Background(Red).
		Align(lipgloss.Center).
		Bold(true).
		Padding(0, 2)
}

func Highlighted() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(Surface0).
		Padding(0, 1).
		Margin(0, 0)
}

func Code() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Teal).
		Background(Surface1).
		Padding(0, 1)
}

func Title() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Mauve).
		Bold(true).
		Transform(strings.ToUpper).
		Padding(1, 0).
		Margin(0, 2)
}

func TitleWithCount(title string, count int) string {
	titleStyle := Title().
		UnsetMargins().
		UnsetPadding()

	countRendered := lipgloss.NewStyle().
		Foreground(Overlay1).
		Bold(false).
		Render(fmt.Sprintf("(%d)", count))

	return lipgloss.NewStyle().
		Padding(1, 0).
		Margin(0, 2).
		Render(titleStyle.Render(title) + " " + countRendered)
}
