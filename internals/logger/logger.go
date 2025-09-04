package logger

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/kloudkit/ws-cli/internals/env"
	"github.com/kloudkit/ws-cli/internals/styles"
)

var (
	logRegex   = regexp.MustCompile(`^\[.*?\]\s+(\w+)\s*(.*)$`)
	timeFormat = "[2006-01-02T15:04:05.000Z]"
	styleMap   = map[string]lipgloss.Style{
		"info":  styles.Info(),
		"error": styles.Error(),
		"warn":  styles.Warning(),
		"debug": styles.Muted(),
	}
)

type Reader struct {
	logPath     string
	levelFilter string
	tailLines   int
}

func timestamp() string {
	return time.Now().UTC().Format(timeFormat)
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
		Log(writer, level, scanner.Text(), indent, withStamp)
	}
}

var Log = func(writer io.Writer, level, message string, indent int, withStamp bool) {
	var parts []string
	if withStamp {
		parts = append(parts, timestamp())
	}

	if len(level) > 0 {
		parts = append(parts, formatLevel(level))
	}

	if indent > 0 {
		parts = append(parts, strings.Repeat("  ", indent)+"- "+message)
	} else {
		parts = append(parts, message)
	}

	fmt.Fprintln(writer, strings.Join(parts, " "))
}

func NewReader(tailLines int, levelFilter string) (*Reader, error) {
	logPath := filepath.Join(env.String("WS_LOGGING_DIR", "/var/log/workspace"), env.String("WS_LOGGING_MAIN_FILE", "workspace.log"))

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("log file not found at %s", logPath)
	}

	return &Reader{logPath, levelFilter, tailLines}, nil
}

func (r *Reader) shouldIncludeLine(line string) bool {
	if r.levelFilter == "" {
		return true
	}

	if matches := logRegex.FindStringSubmatch(line); len(matches) == 3 {
		return strings.EqualFold(matches[1], r.levelFilter)
	}

	return true
}

func (r *Reader) ReadLogs(writer io.Writer) error {
	return r.processLogs(writer, false)
}

func (r *Reader) FollowLogs(writer io.Writer) error {
	return r.processLogs(writer, true)
}

func (r *Reader) processLogs(writer io.Writer, follow bool) error {
	file, err := os.Open(r.logPath)

	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	defer file.Close()

	if r.tailLines > 0 && !follow {
		if err := r.seekToTail(file); err != nil {
			return err
		}
	}

	r.scanAndWrite(file, writer)

	if !follow {
		return nil
	}

	file.Seek(0, io.SeekEnd)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		r.scanAndWrite(file, writer)
	}

	return nil
}

func (r *Reader) scanAndWrite(file *os.File, writer io.Writer) {
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if r.shouldIncludeLine(line) {
			fmt.Fprintln(writer, line)
		}
	}
}

func (r *Reader) seekToTail(file *os.File) error {
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	if stat.Size() == 0 {
		return nil
	}

	buf := make([]byte, min(stat.Size(), 8192))
	lines, pos := 0, stat.Size()

	for lines < r.tailLines && pos > 0 {
		readSize := int64(len(buf))
		if pos < readSize {
			readSize = pos
		}
		pos -= readSize
		if _, err := file.Seek(pos, 0); err != nil {
			return err
		}
		n, err := file.Read(buf[:readSize])
		if err != nil {
			return err
		}
		for i := n - 1; i >= 0; i-- {
			if buf[i] == '\n' {
				lines++
				if lines == r.tailLines {
					file.Seek(pos+int64(i)+1, 0)
					return nil
				}
			}
		}
	}

	file.Seek(0, 0)

	return nil
}
