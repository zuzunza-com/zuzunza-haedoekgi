package progress

import (
	"fmt"
	"os"
	"time"
)

type Reporter struct {
	start time.Time
}

func New() *Reporter {
	return &Reporter{start: time.Now()}
}

func (r *Reporter) Callback(written, total int64, limited bool) {
	elapsed := time.Since(r.start).Seconds()
	if elapsed < 0.001 {
		elapsed = 0.001
	}
	speed := float64(written) / elapsed
	line := fmt.Sprintf("\r다운로드 %s", formatBytes(written))
	if total > 0 {
		line += fmt.Sprintf(" / %s (%.0f%%)", formatBytes(total), float64(written)/float64(total)*100)
	}
	line += fmt.Sprintf(" · %.1f KB/s", speed/1024)
	if limited {
		line += " · 대역폭 제한"
	}
	fmt.Fprint(os.Stderr, line)
}

func formatBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
