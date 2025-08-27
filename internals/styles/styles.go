package styles

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	NoColorStyle = lipgloss.NewStyle()

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorHeader)).
			Bold(true)

	subHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorInfo)).
			Bold(true)

	keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorAccent))

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorText))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorInfo))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSuccess))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorWarning))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorError))

	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMuted))
)

func HeaderStyle() lipgloss.Style {
	if ColorEnabled {
		return headerStyle
	}

	return NoColorStyle
}

func SubHeaderStyle() lipgloss.Style {
	if ColorEnabled {
		return subHeaderStyle
	}

	return NoColorStyle
}

func KeyStyle() lipgloss.Style {
	if ColorEnabled {
		return keyStyle
	}

	return NoColorStyle
}

func ValueStyle() lipgloss.Style {
	if ColorEnabled {
		return valueStyle
	}

	return NoColorStyle
}

func InfoStyle() lipgloss.Style {
	if ColorEnabled {
		return infoStyle
	}

	return NoColorStyle
}

func SuccessStyle() lipgloss.Style {
	if ColorEnabled {
		return successStyle
	}

	return NoColorStyle
}

func WarningStyle() lipgloss.Style {
	if ColorEnabled {
		return warningStyle
	}

	return NoColorStyle
}

func ErrorStyle() lipgloss.Style {
	if ColorEnabled {
		return errorStyle
	}

	return NoColorStyle
}

func MutedStyle() lipgloss.Style {
	if ColorEnabled {
		return mutedStyle
	}

	return NoColorStyle
}
