package progress

import (
	"strings"
	"testing"
	"time"
)

func TestNewIndicator(t *testing.T) {
	tests := []struct {
		name    string
		message string
		total   int
		enabled bool
	}{
		{
			name:    "enabled indicator",
			message: "Processing",
			total:   100,
			enabled: true,
		},
		{
			name:    "disabled indicator",
			message: "Processing",
			total:   100,
			enabled: false,
		},
		{
			name:    "indeterminate progress",
			message: "Loading",
			total:   0,
			enabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indicator := NewIndicator(tt.message, tt.total, tt.enabled)

			if indicator.message != tt.message {
				t.Errorf("expected message %q, got %q", tt.message, indicator.message)
			}
			if indicator.total != tt.total {
				t.Errorf("expected total %d, got %d", tt.total, indicator.total)
			}
			if indicator.enabled != tt.enabled {
				t.Errorf("expected enabled %v, got %v", tt.enabled, indicator.enabled)
			}
		})
	}
}

func TestProgressBar(t *testing.T) {
	indicator := NewIndicator("Test", 100, true)

	tests := []struct {
		percentage float64
		expected   string
	}{
		{0.0, "▓░░░░░░░░░░░░░░░░░░░░░░░░░░░░░"},
		{50.0, "███████████████▓░░░░░░░░░░░░░░"},
		{100.0, "██████████████████████████████"},
	}

	for _, tt := range tests {
		result := indicator.createProgressBar(tt.percentage)
		if result != tt.expected {
			t.Errorf("progress bar for %.1f%%: expected %q, got %q", tt.percentage, tt.expected, result)
		}
	}
}

func TestSpinner(t *testing.T) {
	indicator := NewIndicator("Test", 0, true)

	// Test different elapsed times produce different spinner states
	elapsed1 := 0 * time.Millisecond
	elapsed2 := 100 * time.Millisecond
	elapsed3 := 200 * time.Millisecond

	spinner1 := indicator.getSpinner(elapsed1)
	spinner2 := indicator.getSpinner(elapsed2)
	spinner3 := indicator.getSpinner(elapsed3)

	// Should all be valid spinner characters
	validSpinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	isValid := func(s string) bool {
		for _, valid := range validSpinners {
			if s == valid {
				return true
			}
		}
		return false
	}

	if !isValid(spinner1) {
		t.Errorf("invalid spinner character: %s", spinner1)
	}
	if !isValid(spinner2) {
		t.Errorf("invalid spinner character: %s", spinner2)
	}
	if !isValid(spinner3) {
		t.Errorf("invalid spinner character: %s", spinner3)
	}

	// Different elapsed times should produce different spinners
	if spinner1 == spinner2 && spinner2 == spinner3 {
		t.Errorf("spinner should change over time")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{50 * time.Millisecond, "50ms"},
		{1500 * time.Millisecond, "1.5s"},
		{90 * time.Second, "1.5m"},
		{3600 * time.Second, "1.0h"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.duration)
		if result != tt.expected {
			t.Errorf("formatDuration(%v): expected %q, got %q", tt.duration, tt.expected, result)
		}
	}
}

func TestSimpleConstructor(t *testing.T) {
	// Test with quiet=false (should be enabled)
	indicator1 := Simple("Test message", false)
	if !indicator1.enabled {
		t.Errorf("expected indicator to be enabled when quiet=false")
	}

	// Test with quiet=true (should be disabled)
	indicator2 := Simple("Test message", true)
	if indicator2.enabled {
		t.Errorf("expected indicator to be disabled when quiet=true")
	}
}

func TestWithTotalConstructor(t *testing.T) {
	// Test with quiet=false (should be enabled)
	indicator1 := WithTotal("Processing", 100, false)
	if !indicator1.enabled {
		t.Errorf("expected indicator to be enabled when quiet=false")
	}
	if indicator1.total != 100 {
		t.Errorf("expected total 100, got %d", indicator1.total)
	}

	// Test with quiet=true (should be disabled)
	indicator2 := WithTotal("Processing", 50, true)
	if indicator2.enabled {
		t.Errorf("expected indicator to be disabled when quiet=true")
	}
}

func TestDisabledIndicatorNoOutput(t *testing.T) {
	// This test verifies that disabled indicators don't produce output
	// We can't easily test stderr output in unit tests, but we can test
	// that the methods don't panic and return quickly

	indicator := NewIndicator("Test", 100, false)

	// These should all be no-ops for disabled indicators
	indicator.Start()
	indicator.Update(50)
	indicator.Finish()
	indicator.FinishWithError(nil)

	// Test passes if no panic occurs
}

func TestProgressBarVisualConsistency(t *testing.T) {
	indicator := NewIndicator("Test", 100, true)

	// Test edge cases
	tests := []float64{0, 0.1, 33.33, 66.67, 99.9, 100}

	for _, percentage := range tests {
		bar := indicator.createProgressBar(percentage)

		// All bars should be the same length
		const expectedLength = 30
		if len([]rune(bar)) != expectedLength {
			t.Errorf("progress bar at %.1f%% has wrong length: expected %d chars, got %d",
				percentage, expectedLength, len([]rune(bar)))
		}

		// Bar should only contain valid characters
		validChars := []string{"█", "▓", "░"}
		for _, char := range strings.Split(bar, "") {
			if char == "" {
				continue
			}
			valid := false
			for _, validChar := range validChars {
				if char == validChar {
					valid = true
					break
				}
			}
			if !valid {
				t.Errorf("progress bar contains invalid character: %q", char)
			}
		}
	}
}

func TestIndicatorMessagePreservation(t *testing.T) {
	message := "Loading important data"
	indicator := NewIndicator(message, 100, true)

	if indicator.message != message {
		t.Errorf("message not preserved: expected %q, got %q", message, indicator.message)
	}

	// Message should be preserved after operations
	indicator.Update(50)
	if indicator.message != message {
		t.Errorf("message changed after update: expected %q, got %q", message, indicator.message)
	}
}

func BenchmarkProgressBar(b *testing.B) {
	indicator := NewIndicator("Benchmark", 100, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		percentage := float64(i % 101) // 0-100%
		_ = indicator.createProgressBar(percentage)
	}
}

func BenchmarkSpinner(b *testing.B) {
	indicator := NewIndicator("Benchmark", 0, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		elapsed := time.Duration(i%1000) * time.Millisecond
		_ = indicator.getSpinner(elapsed)
	}
}
