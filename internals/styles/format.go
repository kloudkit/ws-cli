package styles

import "fmt"

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
