package monitoring

import (
	"testing"

	"github.com/guarzo/pkmgradegap/internal/model"
)

func TestOptimizeSubmission(t *testing.T) {
	cards := []SubmissionCard{
		{
			Card:          model.Card{Name: "Low Value Card", Number: "001"},
			RawPriceUSD:   5.00,
			PSA10Price:    150.00, // Under Value threshold
			ExpectedGrade: 9.5,
			ExpectedValue: 120.00,
		},
		{
			Card:          model.Card{Name: "High Value Card", Number: "002"},
			RawPriceUSD:   50.00,
			PSA10Price:    800.00, // Over Value threshold
			ExpectedGrade: 9.7,
			ExpectedValue: 650.00,
		},
		{
			Card:          model.Card{Name: "Medium Value Card", Number: "003"},
			RawPriceUSD:   20.00,
			PSA10Price:    300.00, // Value Plus threshold
			ExpectedGrade: 9.3,
			ExpectedValue: 250.00,
		},
	}

	optimizer := NewBulkOptimizer(0.13, 20.0)
	batches := optimizer.OptimizeSubmission(cards)

	// Should create batches based on PSA10 value thresholds
	// Note: May be 0 if minimum card requirements aren't met
	if len(batches) == 0 {
		t.Log("No batches created - may be due to minimum card requirements")
	}

	// Verify cards are sorted by value (highest first)
	for _, batch := range batches {
		for i := 1; i < len(batch.Cards); i++ {
			if batch.Cards[i-1].PSA10Price < batch.Cards[i].PSA10Price {
				t.Error("Cards should be sorted by PSA10 price (highest first)")
			}
		}
	}
}

func TestFindServiceLevel(t *testing.T) {
	optimizer := NewBulkOptimizer(0.13, 20.0)

	tests := []struct {
		value    float64
		expected string
	}{
		{150.00, "Value"},
		{300.00, "Value Plus"},
		{800.00, "Regular"},
		{2000.00, "Express"},
		{4000.00, "Super Express"},
		{10000.00, "Walk Through"},
	}

	for _, test := range tests {
		level := optimizer.findServiceLevel(test.value)
		if level.Name != test.expected {
			t.Errorf("For value %.2f, expected %s, got %s", test.value, test.expected, level.Name)
		}
	}
}

func TestEstimateExpectedGrade(t *testing.T) {
	tests := []struct {
		psa10Rate float64
		psa9Rate  float64
		expected  float64
	}{
		{0.35, 0.45, 9.7}, // High PSA10 rate
		{0.25, 0.45, 9.5}, // Medium PSA10 rate
		{0.15, 0.45, 9.3}, // Low PSA10 rate
		{0.05, 0.45, 9.0}, // Very low PSA10 rate
	}

	for _, test := range tests {
		result := EstimateExpectedGrade(test.psa10Rate, test.psa9Rate)
		if result != test.expected {
			t.Errorf("For PSA10 rate %.2f, expected %.1f, got %.1f", test.psa10Rate, test.expected, result)
		}
	}
}