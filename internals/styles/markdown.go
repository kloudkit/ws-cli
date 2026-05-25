package styles

import (
	"io"
	"regexp"
	"strings"

	"charm.land/glamour/v2"
)

var trailingPad = regexp.MustCompile(`(?:\x1b\[[\d;]*m|[ \t])+$`)

func RenderMarkdown(w io.Writer, src string) error {
	if strings.TrimSpace(src) == "" {
		return nil
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(CatppuccinFrappeStyleConfig),
	)
	if err != nil {
		return err
	}
	rendered, err := renderer.Render(src)
	if err != nil {
		return err
	}
	rendered = trimTrailingSpaces(rendered)
	rendered = strings.Trim(rendered, "\n") + "\n"
	_, err = io.WriteString(w, rendered)
	return err
}

func trimTrailingSpaces(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = trailingPad.ReplaceAllString(line, "")
	}
	return strings.Join(lines, "\n")
}
