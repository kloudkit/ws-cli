package styles

import (
	"io"
	"strings"

	"charm.land/glamour/v2"
)

func RenderMarkdown(w io.Writer, src string) error {
	if strings.TrimSpace(src) == "" {
		return nil
	}
	rendered, err := glamour.RenderWithEnvironmentConfig(src)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, rendered)
	return err
}
