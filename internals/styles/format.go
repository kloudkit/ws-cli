package styles

import (
	"fmt"
	"strings"
	"time"
)

func FormatBytes(bytes uint64) string {
	const unit = 1024

	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func FormatPercent(used, total uint64) string {
	if total == 0 {
		return "0%"
	}

	return fmt.Sprintf("%.1f%%", float64(used)/float64(total)*100)
}

func FormatDuration(duration time.Duration) string {
	units := []struct {
		n    int
		name string
	}{
		{int(duration.Hours() / 24), "day"},
		{int(duration.Hours()) % 24, "hour"},
		{int(duration.Minutes()) % 60, "minute"},
	}

	var parts []string

	for _, u := range units {
		switch {
		case u.n == 1:
			parts = append(parts, "1 "+u.name)
		case u.n > 1:
			parts = append(parts, fmt.Sprintf("%d %ss", u.n, u.name))
		}
	}

	if len(parts) == 0 {
		return "less than a minute"
	}

	return strings.Join(parts, ", ")
}

func FormatCPUTime(seconds float64) string {
	switch {
	case seconds < 60:
		return fmt.Sprintf("%.1fs", seconds)
	case seconds < 3600:
		return fmt.Sprintf("%.1fm", seconds/60)
	default:
		return fmt.Sprintf("%.1fh", seconds/3600)
	}
}
