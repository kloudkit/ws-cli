package styles

import (
	"image/color"

	"charm.land/fang/v2"
	lipgloss "charm.land/lipgloss/v2"
)

var (
	Base     = color.RGBA{48, 52, 70, 255}
	Surface0 = color.RGBA{65, 69, 89, 255}
	Surface2 = color.RGBA{98, 104, 128, 255}
	Overlay0 = color.RGBA{115, 121, 148, 255}
	Overlay1 = color.RGBA{131, 139, 167, 255}
	Text     = color.RGBA{198, 208, 245, 255}
	Subtext1 = color.RGBA{181, 191, 226, 255}
	Blue     = color.RGBA{140, 170, 238, 255}
	Green    = color.RGBA{166, 209, 137, 255}
	Yellow   = color.RGBA{229, 200, 144, 255}
	Red      = color.RGBA{231, 130, 132, 255}
	Teal     = color.RGBA{129, 200, 190, 255}
	Mauve    = color.RGBA{202, 158, 230, 255}
	Peach    = color.RGBA{239, 159, 118, 255}
)

var (
	Header    = lipgloss.NewStyle().Foreground(Mauve).Bold(true)
	SubHeader = lipgloss.NewStyle().Foreground(Blue).Bold(true)
	Key       = lipgloss.NewStyle().Foreground(Teal).Bold(true)
	Value     = lipgloss.NewStyle().Foreground(Text)
	Info      = lipgloss.NewStyle().Foreground(Blue)
	Success   = lipgloss.NewStyle().Foreground(Green)
	Warning   = lipgloss.NewStyle().Foreground(Yellow)
	Error     = lipgloss.NewStyle().Foreground(Red).Bold(true)
	Muted     = lipgloss.NewStyle().Foreground(Overlay0)
	Count     = lipgloss.NewStyle().Foreground(Overlay1)
	Border    = lipgloss.NewStyle().Foreground(Surface2)
)

func FrappeColorScheme(lipgloss.LightDarkFunc) fang.ColorScheme {
	return fang.ColorScheme{
		Base:           Base,
		Title:          Mauve,
		Description:    Text,
		Codeblock:      Surface0,
		Program:        Teal,
		DimmedArgument: Overlay0,
		Comment:        Overlay1,
		Flag:           Green,
		FlagDefault:    Yellow,
		Command:        Blue,
		QuotedString:   Peach,
		Argument:       Red,
		Help:           Subtext1,
		Dash:           Surface2,
		ErrorHeader:    [2]color.Color{Red, Base},
		ErrorDetails:   Text,
	}
}
