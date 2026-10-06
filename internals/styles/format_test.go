package styles

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		in   time.Duration
		want string
	}{
		{30 * time.Second, "less than a minute"},
		{5 * time.Minute, "5 minutes"},
		{23*time.Hour + 59*time.Minute, "23 hours, 59 minutes"},
		{24*time.Hour + time.Minute, "1 day, 1 minute"},
		{50 * time.Hour, "2 days, 2 hours"},
	}

	for _, tt := range tests {
		assert.Equal(t, FormatDuration(tt.in), tt.want)
	}
}
