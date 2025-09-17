package progress

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Indicator provides progress indication for long-running operations
type Indicator struct {
	enabled    bool
	message    string
	total      int
	current    int
	startTime  time.Time
	lastUpdate time.Time
}

// NewIndicator creates a new progress indicator
func NewIndicator(message string, total int, enabled bool) *Indicator {
	return &Indicator{
		enabled:   enabled,
		message:   message,
		total:     total,
		startTime: time.Now(),
	}
}

// Start begins the progress indication
func (p *Indicator) Start() {
	if !p.enabled {
		return
	}

	p.startTime = time.Now()
	p.lastUpdate = p.startTime
	fmt.Fprintf(os.Stderr, "%s...\n", p.message)
}

// Update increments progress and shows current status
func (p *Indicator) Update(current int) {
	if !p.enabled {
		return
	}

	p.current = current
	now := time.Now()

	// Only update display every 100ms to avoid flickering
	if now.Sub(p.lastUpdate) < 100*time.Millisecond && current < p.total {
		return
	}
	p.lastUpdate = now

	elapsed := now.Sub(p.startTime)

	if p.total > 0 {
		percentage := float64(current) / float64(p.total) * 100
		bar := p.createProgressBar(percentage)

		// Estimate time remaining
		var eta string
		if current > 0 {
			rate := float64(current) / elapsed.Seconds()
			remaining := float64(p.total-current) / rate
			eta = fmt.Sprintf(" ETA: %s", formatDuration(time.Duration(remaining)*time.Second))
		}

		fmt.Fprintf(os.Stderr, "\r%s [%s] %d/%d (%.1f%%)%s",
			p.message, bar, current, p.total, percentage, eta)
	} else {
		// Indeterminate progress - just show spinner
		spinner := p.getSpinner(elapsed)
		fmt.Fprintf(os.Stderr, "\r%s %s (%d processed)", p.message, spinner, current)
	}
}

// Finish completes the progress indication
func (p *Indicator) Finish() {
	if !p.enabled {
		return
	}

	elapsed := time.Now().Sub(p.startTime)

	if p.total > 0 {
		fmt.Fprintf(os.Stderr, "\r%s ✓ Completed %d items in %s\n",
			p.message, p.total, formatDuration(elapsed))
	} else {
		fmt.Fprintf(os.Stderr, "\r%s ✓ Completed %d items in %s\n",
			p.message, p.current, formatDuration(elapsed))
	}
}

// FinishWithError completes the progress indication with an error
func (p *Indicator) FinishWithError(err error) {
	if !p.enabled {
		return
	}

	elapsed := time.Now().Sub(p.startTime)
	fmt.Fprintf(os.Stderr, "\r%s ✗ Failed after %s: %v\n",
		p.message, formatDuration(elapsed), err)
}

// createProgressBar creates a visual progress bar
func (p *Indicator) createProgressBar(percentage float64) string {
	const width = 30
	filled := int(percentage / 100.0 * width)

	var bar strings.Builder
	for i := 0; i < width; i++ {
		if i < filled {
			bar.WriteString("█")
		} else if i == filled && percentage < 100 {
			bar.WriteString("▓")
		} else {
			bar.WriteString("░")
		}
	}
	return bar.String()
}

// getSpinner returns a spinning character based on elapsed time
func (p *Indicator) getSpinner(elapsed time.Duration) string {
	spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	index := int(elapsed.Milliseconds()/100) % len(spinners)
	return spinners[index]
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	} else if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// Simple creates a simple progress indicator for known operations
func Simple(message string, quiet bool) *Indicator {
	return NewIndicator(message, 0, !quiet)
}

// WithTotal creates a progress indicator with a known total
func WithTotal(message string, total int, quiet bool) *Indicator {
	return NewIndicator(message, total, !quiet)
}
