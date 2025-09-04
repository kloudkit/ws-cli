package styles

import (
	"image/color"
	"strconv"

	"github.com/charmbracelet/fang"
	lipgloss "github.com/charmbracelet/lipgloss/v2"
)

const (
	Base     = "#303446"
	Surface0 = "#414559"
	Surface1 = "#51576d"
	Surface2 = "#626880"
	Overlay0 = "#737994"
	Overlay1 = "#838ba7"
	Text     = "#c6d0f5"
	Subtext1 = "#b5bfe2"
	Blue     = "#8caaee"
	Green    = "#a6d189"
	Yellow   = "#e5c890"
	Red      = "#e78284"
	Teal     = "#81c8be"
	Mauve    = "#ca9ee6"
	Peach    = "#ef9f76"
)

func hexToColor(hex string) color.Color {
	if hex[0] == '#' {
		hex = hex[1:]
	}

	val, _ := strconv.ParseUint(hex, 16, 32)

	return color.RGBA{uint8(val >> 16), uint8(val >> 8), uint8(val), 255}
}

func FrappeColorScheme(c lipgloss.LightDarkFunc) fang.ColorScheme {
	baseColor := hexToColor(Base)
	textColor := hexToColor(Text)
	redColor := hexToColor(Red)

	return fang.ColorScheme{
		Base:           baseColor,
		Title:          hexToColor(Mauve),
		Description:    textColor,
		Codeblock:      hexToColor(Surface0),
		Program:        hexToColor(Teal),
		DimmedArgument: hexToColor(Overlay0),
		Comment:        hexToColor(Overlay1),
		Flag:           hexToColor(Green),
		FlagDefault:    hexToColor(Yellow),
		Command:        hexToColor(Blue),
		QuotedString:   hexToColor(Peach),
		Argument:       redColor,
		Help:           hexToColor(Subtext1),
		Dash:           hexToColor(Surface2),
		ErrorHeader:    [2]color.Color{redColor, baseColor},
		ErrorDetails:   textColor,
	}
}
