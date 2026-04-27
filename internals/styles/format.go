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
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	var parts []string

	if days > 0 {
		parts = append(parts, fmt.Sprintf("%d days", days))
	}

	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d hours", hours))
	}

	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%d minutes", minutes))
	}

	if len(parts) == 0 {
		return "just now"
	}

	return strings.Join(parts, ", ") + " ago"
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
