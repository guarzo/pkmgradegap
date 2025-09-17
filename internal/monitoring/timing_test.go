package monitoring

import (
	"testing"
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
)

func TestCalculateTrend(t *testing.T) {
	tests := []struct {
		prices   []float64
		expected MarketTrend
	}{
		{[]float64{10, 12, 14, 16, 18}, TrendUp},   // Clear upward trend
		{[]float64{18, 16, 14, 12, 10}, TrendDown}, // Clear downward trend
		{[]float64{10, 11, 10, 11, 10}, TrendFlat}, // Flat/sideways
		{[]float64{10, 10.1, 10.2}, TrendFlat},     // Very small changes
	}

	for i, test := range tests {
		result := calculateTrend(test.prices)
		if result != test.expected {
			t.Errorf("Test %d: expected %s, got %s for prices %v", i, test.expected, result, test.prices)
		}
	}
}

func TestCalculateConfidence(t *testing.T) {
	// Opposite trends should give high confidence
	rawPrices := []float64{20, 18, 16, 14, 12}        // Downward
	psa10Prices := []float64{100, 105, 110, 115, 120} // Upward

	confidence := calculateConfidence(rawPrices, psa10Prices)

	if confidence < 70 {
		t.Errorf("Expected high confidence (>70) for opposite trends, got %.1f", confidence)
	}

	// Same trends should give lower confidence
	sameTrendRaw := []float64{20, 22, 24, 26, 28}
	sameTrendPSA10 := []float64{100, 105, 110, 115, 120}

	lowConfidence := calculateConfidence(sameTrendRaw, sameTrendPSA10)

	if lowConfidence >= confidence {
		t.Errorf("Expected lower confidence for same trends, got %.1f vs %.1f", lowConfidence, confidence)
	}
}

func TestMarketAnalyzer(t *testing.T) {
	// Create test snapshots with price history
	snapshots := []*Snapshot{
		{
			Timestamp: time.Now().Add(-72 * time.Hour),
			Cards: map[string]*SnapshotCardData{
				"001-Test Card": {
					Card:        model.Card{Name: "Test Card", Number: "001"},
					RawPriceUSD: 20.00,
					PSA10Price:  100.00,
				},
			},
		},
		{
			Timestamp: time.Now().Add(-48 * time.Hour),
			Cards: map[string]*SnapshotCardData{
				"001-Test Card": {
					Card:        model.Card{Name: "Test Card", Number: "001"},
					RawPriceUSD: 18.00,
					PSA10Price:  105.00,
				},
			},
		},
		{
			Timestamp: time.Now().Add(-24 * time.Hour),
			Cards: map[string]*SnapshotCardData{
				"001-Test Card": {
					Card:        model.Card{Name: "Test Card", Number: "001"},
					RawPriceUSD: 16.00,
					PSA10Price:  110.00,
				},
			},
		},
	}

	analyzer := NewMarketAnalyzer(snapshots)
	rec := analyzer.AnalyzeCard("001-Test Card")

	if rec == nil {
		t.Fatal("Expected recommendation, got nil")
	}

	// Raw prices trending down, PSA10 up - should recommend BUY
	if rec.Action != "BUY" {
		t.Errorf("Expected BUY recommendation, got %s", rec.Action)
	}

	if rec.Confidence < 50 {
		t.Errorf("Expected reasonable confidence (>50), got %.1f", rec.Confidence)
	}
}

func TestCalculateVolatility(t *testing.T) {
	// Low volatility prices
	lowVol := []float64{10.0, 10.1, 9.9, 10.2, 9.8}
	volatility := calculateVolatility(lowVol)

	if volatility > 5 {
		t.Errorf("Expected low volatility (<5), got %.2f", volatility)
	}

	// High volatility prices
	highVol := []float64{10.0, 15.0, 8.0, 12.0, 6.0}
	highVolatility := calculateVolatility(highVol)

	if highVolatility <= volatility {
		t.Errorf("Expected higher volatility for second set, got %.2f vs %.2f", highVolatility, volatility)
	}
}
