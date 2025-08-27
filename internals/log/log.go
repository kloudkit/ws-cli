package log

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/kloudkit/ws-cli/internals/styles"
)

var ColorEnabled = true

func timestamp() string {
	return time.Now().UTC().Format("[2006-01-02T15:04:05.000Z]")
}

var styleMap = map[string]lipgloss.Style{
	"info":  styles.InfoStyle(),
	"error": styles.ErrorStyle(),
	"warn":  styles.WarningStyle(),
	"debug": styles.MutedStyle(),
}

func formatLevel(level string) string {
	if style, ok := styleMap[level]; ok {
		return style.Width(5).Render(level)
	}

	return level
}

var Pipe = func(reader io.Reader, writer io.Writer, level string, indent int, withStamp bool) {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()
		Log(writer, level, line, indent, withStamp)
	}
}

var Log = func(writer io.Writer, level, message string, indent int, withStamp bool) {
	stamp := ""
	prefix := ""

	if withStamp {
		stamp = timestamp() + " "
	}

	if len(level) > 0 {
		level = formatLevel(level) + " "
	}

	if indent > 0 {
		prefix = strings.Repeat("  ", indent) + "- "
	}

	fmt.Fprintln(writer, stamp+level+prefix+message)
}
