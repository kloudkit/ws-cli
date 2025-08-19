package log

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var ColorEnabled = true

var (
	infoStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	debugStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
)

func timestamp() string {
	return time.Now().UTC().Format("[2006-01-02T15:04:05.000Z]")
}

func formatLevel(level string) string {
	if !ColorEnabled {
		return level
	}

	switch level {
	case "info":
		return infoStyle.Render(level)
	case "error":
		return errorStyle.Render(level)
	case "warn":
		return warnStyle.Render(level)
	case "debug":
		return debugStyle.Render(level)
	default:
		return level
	}
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
		level = lipgloss.NewStyle().Width(5).Render(formatLevel(level)) + " "
	}

	if indent > 0 {
		prefix = strings.Repeat("  ", indent) + "- "
	}

	fmt.Fprintln(writer, stamp+level+prefix+message)
}
