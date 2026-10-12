package styles

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

type DescriptionItem struct {
	Name        string
	Description string
}

func PrintList(writer io.Writer, items []DescriptionItem) {
	width := 0
	for _, item := range items {
		width = max(width, ansi.StringWidth(item.Name))
	}

	bullet, dash := Muted().Render("•"), Muted().Render("—")

	for _, item := range items {
		name := item.Name + strings.Repeat(" ", width-ansi.StringWidth(item.Name))
		fmt.Fprintf(writer, "%s%s %s %s %s\n", indent, bullet, Key().Render(name), dash, Value().Render(item.Description))
	}
}

func PrintBullets(writer io.Writer, names []string) {
	bullet := Muted().Render("•")

	for _, name := range names {
		fmt.Fprintf(writer, "%s%s %s\n", indent, bullet, Value().Render(name))
	}
}
