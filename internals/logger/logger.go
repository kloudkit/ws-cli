package logger

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/kloudkit/ws-cli/internals/config"
	"github.com/kloudkit/ws-cli/internals/styles"
)

var (
	levelRegex = regexp.MustCompile(`^\[[^\]]+\] (debug|info|warn|error)\b`)
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

func Pipe(reader io.Reader, writer io.Writer, level string, indent int, withStamp bool) {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		Log(writer, level, scanner.Text(), indent, withStamp)
	}
}

func Log(writer io.Writer, level, message string, indent int, withStamp bool) {
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

func NewReader(tailLines int, levelFilter, target string) (*Reader, error) {
	if target == "" {
		target = "main"
	}

	logPath := filepath.Join(
		config.MustResolve("logging", "dir"),
		config.MustResolve("logging", target+"_file"),
	)

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("log file not found at %s", logPath)
	}

	return &Reader{logPath, levelFilter, tailLines}, nil
}

func (r *Reader) shouldIncludeLine(plain string) bool {
	if r.levelFilter == "" {
		return true
	}

	matches := levelRegex.FindStringSubmatch(plain)

	return matches != nil && strings.EqualFold(matches[1], r.levelFilter)
}

func colorLevel(plain string) string {
	loc := levelRegex.FindStringSubmatchIndex(plain)
	if loc == nil {
		return plain
	}

	level := plain[loc[2]:loc[3]]

	return plain[:loc[2]] + styleMap[level].Render(level) + plain[loc[3]:]
}

func plainLine(line string) string {
	return ansi.Strip(strings.TrimRight(line, "\r\n"))
}

func (r *Reader) ReadLogs(writer io.Writer, follow bool) error {
	file, err := os.Open(r.logPath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	defer file.Close()

	if r.tailLines > 0 {
		offset, err := r.tailOffset(file)
		if err != nil {
			return err
		}

		if _, err := file.Seek(offset, io.SeekStart); err != nil {
			return err
		}
	}

	reader := bufio.NewReader(file)

	pending, err := r.drain(reader, writer, "", true)
	if err != nil || !follow {
		return err
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		if pending, err = r.drain(reader, writer, pending, false); err != nil {
			return err
		}
	}

	return nil
}

func (r *Reader) drain(reader *bufio.Reader, writer io.Writer, pending string, flush bool) (string, error) {
	for {
		chunk, err := reader.ReadString('\n')
		pending += chunk

		if err == io.EOF {
			if flush && pending != "" {
				r.emit(writer, pending)
				return "", nil
			}

			return pending, nil
		}

		if err != nil {
			return pending, err
		}

		r.emit(writer, pending)
		pending = ""
	}
}

func (r *Reader) emit(writer io.Writer, line string) {
	plain := plainLine(line)

	if r.shouldIncludeLine(plain) {
		fmt.Fprintln(writer, colorLevel(plain))
	}
}

func (r *Reader) tailOffset(file *os.File) (int64, error) {
	stat, err := file.Stat()
	if err != nil {
		return 0, err
	}

	size := stat.Size()
	if size == 0 {
		return 0, nil
	}

	last := make([]byte, 1)
	if _, err := file.ReadAt(last, size-1); err != nil {
		return 0, err
	}

	end := size
	if last[0] == '\n' {
		end--
	}

	const chunkSize = 8192

	var pending []byte
	pos, count := end, 0

	for {
		for {
			i := bytes.LastIndexByte(pending, '\n')
			if i < 0 {
				break
			}

			if r.shouldIncludeLine(plainLine(string(pending[i+1:]))) {
				count++
				if count == r.tailLines {
					return pos + int64(i) + 1, nil
				}
			}

			pending = pending[:i]
		}

		if pos == 0 {
			return 0, nil
		}

		readSize := min(pos, chunkSize)
		pos -= readSize

		chunk := make([]byte, readSize, int(readSize)+len(pending))
		if _, err := file.ReadAt(chunk, pos); err != nil {
			return 0, err
		}

		pending = append(chunk, pending...)
	}
}

func Strip(reader io.Reader, writer io.Writer) error {
	buffered := bufio.NewReader(reader)

	for {
		line, err := buffered.ReadString('\n')
		if len(line) > 0 {
			if _, werr := io.WriteString(writer, ansi.Strip(line)); werr != nil {
				return werr
			}
		}

		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}
	}
}
