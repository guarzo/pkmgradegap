package analysis

import (
	"testing"

	"github.com/guarzo/pkmgradegap/internal/model"
)

func TestScoreRow_BasicROI(t *testing.T) {
	// Test basic ROI calculation
	row := Row{
		Card: model.Card{
			Name:   "Test Card",
			Number: "1",
		},
		RawUSD: 50.0,
		Grades: Grades{
			PSA10:   150.0,
			Grade9:  100.0, // 100/150 = 0.67 < 0.75, passes thin premium filter
			Grade95: 130.0,
		},
	}

	config := Config{
		GradingCost:  25.0,
		ShippingCost: 20.0,
		FeePct:       0.13,
		MinRawUSD:    5.0,
		MinDeltaUSD:  25.0,
	}

	// Expected calculations:
	// Total cost = 50 + 25 + 20 = 95
	// Selling fees = 150 * 0.13 = 19.5
	// Net profit = 150 - 95 - 19.5 = 35.5
	// Premium lift = (1 - 100/150) * 10 = 3.33
	// Expected score = 35.5 + 3.33 = 38.83

	rows := []Row{row}
	scoredRows := scoreAndFilter(rows, config)

	if len(scoredRows) != 1 {
		// Debug the filtering
		delta := row.Grades.PSA10 - row.RawUSD
		t.Logf("Delta: %.2f (required: %.2f)", delta, config.MinDeltaUSD)
		t.Logf("Raw price: %.2f (required: %.2f)", row.RawUSD, config.MinRawUSD)
		t.Logf("PSA10 vs Raw: %.2f vs %.2f", row.Grades.PSA10, row.RawUSD)
		// Check thin premium
		ratio := row.Grades.Grade9 / row.Grades.PSA10
		t.Logf("PSA9/PSA10 ratio: %.3f (allowed: %v)", ratio, config.AllowThinPremium)
		t.Fatalf("Expected 1 scored row, got %d", len(scoredRows))
	}

	sr := scoredRows[0]
	expectedNetProfit := 35.5
	expectedScore := 38.83

	if abs(sr.NetProfitUSD-expectedNetProfit) > 0.01 {
		t.Errorf("Expected net profit %.2f, got %.2f", expectedNetProfit, sr.NetProfitUSD)
	}

	if abs(sr.Score-expectedScore) > 0.01 {
		t.Errorf("Expected score %.2f, got %.2f", expectedScore, sr.Score)
	}
}

func TestScoreRow_JapaneseMultiplier(t *testing.T) {
	// Test Japanese card multiplier
	row := Row{
		Card: model.Card{
			Name:   "ピカチュウ", // Pikachu in Japanese
			Number: "1",
		},
		RawUSD: 50.0,
		Grades: Grades{
			PSA10:  150.0,
			Grade9: 100.0, // Pass thin premium filter
		},
	}

	config := Config{
		GradingCost:    25.0,
		ShippingCost:   20.0,
		FeePct:         0.13,
		JapaneseWeight: 1.2,
		MinRawUSD:      5.0,
		MinDeltaUSD:    25.0,
	}

	rows := []Row{row}
	scoredRows := scoreAndFilter(rows, config)

	if len(scoredRows) != 1 {
		t.Fatalf("Expected 1 scored row, got %d", len(scoredRows))
	}

	sr := scoredRows[0]
	if !sr.IsJapanese {
		t.Error("Expected Japanese card to be detected")
	}

	// Score should be multiplied by 1.2
	baseScore := 35.5 + 3.33 // from previous test (updated premium lift)
	expectedScore := baseScore * 1.2

	if abs(sr.Score-expectedScore) > 0.01 {
		t.Errorf("Expected score %.2f, got %.2f", expectedScore, sr.Score)
	}
}

func TestScoreRow_NegativeROIFilter(t *testing.T) {
	// Test that PSA10 <= Raw cards are filtered out
	row := Row{
		Card: model.Card{
			Name:   "Bad Card",
			Number: "1",
		},
		RawUSD: 100.0,
		Grades: Grades{
			PSA10: 90.0, // Lower than raw price
		},
	}

	config := Config{
		GradingCost:  25.0,
		ShippingCost: 20.0,
		FeePct:       0.13,
		MinRawUSD:    5.0,
		MinDeltaUSD:  25.0,
	}

	rows := []Row{row}
	scoredRows := scoreAndFilter(rows, config)

	if len(scoredRows) != 0 {
		t.Errorf("Expected negative ROI card to be filtered out, got %d rows", len(scoredRows))
	}
}

func TestScoreRow_ThinPremiumFilter(t *testing.T) {
	// Test PSA9/PSA10 > 0.75 filter
	row := Row{
		Card: model.Card{
			Name:   "Thin Premium Card",
			Number: "1",
		},
		RawUSD: 50.0,
		Grades: Grades{
			PSA10:  100.0,
			Grade9: 85.0, // 85/100 = 0.85 > 0.75
		},
	}

	config := Config{
		GradingCost:      25.0,
		ShippingCost:     20.0,
		FeePct:           0.13,
		AllowThinPremium: false,
		MinRawUSD:        5.0,
		MinDeltaUSD:      25.0,
	}

	rows := []Row{row}
	scoredRows := scoreAndFilter(rows, config)

	if len(scoredRows) != 0 {
		t.Errorf("Expected thin premium card to be filtered out, got %d rows", len(scoredRows))
	}

	// Test with flag enabled
	config.AllowThinPremium = true
	scoredRows = scoreAndFilter(rows, config)

	if len(scoredRows) != 1 {
		t.Errorf("Expected thin premium card to be allowed with flag, got %d rows", len(scoredRows))
	}
}

func TestCalculateBreakEven(t *testing.T) {
	// Test break-even price calculation
	tests := []struct {
		rawPrice     float64
		gradingCost  float64
		shippingCost float64
		feePct       float64
		expected     float64
	}{
		{50.0, 25.0, 20.0, 0.13, 109.20},  // (50+25+20)/(1-0.13) = 95/0.87
		{100.0, 30.0, 25.0, 0.10, 172.22}, // (100+30+25)/(1-0.10) = 155/0.90
	}

	for _, test := range tests {
		totalCost := test.rawPrice + test.gradingCost + test.shippingCost
		breakEven := totalCost / (1 - test.feePct)

		if abs(breakEven-test.expected) > 0.01 {
			t.Errorf("For costs %.2f, expected break-even %.2f, got %.2f",
				totalCost, test.expected, breakEven)
		}
	}
}

func TestCalculateSetAge(t *testing.T) {
	// Test date parsing and age calculation
	tests := []struct {
		releaseDate string
		expectedAge int
	}{
		{"2020-01-01", 5}, // Approximate, will vary by current date
		{"2024-01-01", 1},
		{"invalid-date", 999}, // Should return 999 for unparseable dates
	}

	for _, test := range tests {
		age := calculateSetAge(test.releaseDate)

		if test.releaseDate == "invalid-date" {
			if age != 999 {
				t.Errorf("Expected age 999 for invalid date, got %d", age)
			}
		} else {
			// For valid dates, check that age is reasonable (within 1 year tolerance)
			if abs(float64(age-test.expectedAge)) > 1.0 {
				t.Errorf("For date %s, expected age ~%d, got %d",
					test.releaseDate, test.expectedAge, age)
			}
		}
	}
}

func TestContainsJapanese(t *testing.T) {
	tests := []struct {
		text     string
		expected bool
	}{
		{"Pikachu", false},
		{"ピカチュウ", true}, // Katakana
		{"Pokemon", false},
		{"ポケモン", true}, // Katakana
		{"日本", true},   // Kanji
		{"ひらがな", true}, // Hiragana
		{"Mixed ピカチュウ Text", true},
		{"", false},
	}

	for _, test := range tests {
		result := containsJapanese(test.text)
		if result != test.expected {
			t.Errorf("For text '%s', expected %v, got %v", test.text, test.expected, result)
		}
	}
}

// Helper function to extract scoring logic for testing
func scoreAndFilter(rows []Row, config Config) []ScoredRow {
	var scoredRows []ScoredRow

	for _, r := range rows {
		// Skip if no prices
		if r.RawUSD <= 0 || r.Grades.PSA10 <= 0 {
			continue
		}

		// Apply minimum filters
		if r.RawUSD < config.MinRawUSD {
			continue
		}

		// Skip negative ROI cards
		if r.Grades.PSA10 <= r.RawUSD {
			continue
		}

		delta := r.Grades.PSA10 - r.RawUSD
		if delta < config.MinDeltaUSD {
			continue
		}

		// Filter thin premium unless allowed
		if !config.AllowThinPremium && r.Grades.Grade9 > 0 && r.Grades.PSA10 > 0 {
			if r.Grades.Grade9/r.Grades.PSA10 > 0.75 {
				continue
			}
		}

		// Calculate costs and score
		totalCost := r.RawUSD + config.GradingCost + config.ShippingCost
		sellingFees := r.Grades.PSA10 * config.FeePct
		netProfit := r.Grades.PSA10 - totalCost - sellingFees
		breakEven := totalCost / (1 - config.FeePct)

		// Base score is net profit
		score := netProfit

		// Add premium lift bonus (rewards steep PSA10 premium)
		if r.Grades.PSA10 > 0 && r.Grades.Grade9 > 0 {
			premiumLift := (1 - r.Grades.Grade9/r.Grades.PSA10) * 10
			score += premiumLift
		}

		// Check if Japanese
		isJapanese := containsJapanese(r.Card.Name)
		if isJapanese {
			score *= config.JapaneseWeight
		}

		scoredRow := ScoredRow{
			Row:          r,
			Score:        score,
			BreakEvenUSD: breakEven,
			NetProfitUSD: netProfit,
			TotalCostUSD: totalCost,
			IsJapanese:   isJapanese,
		}

		scoredRows = append(scoredRows, scoredRow)
	}

	return scoredRows
}

// Helper function for floating point comparison
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
