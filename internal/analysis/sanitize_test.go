package analysis

import (
	"math"
	"testing"

	"github.com/guarzo/pkmgradegap/internal/model"
)

func TestSanitizePrice(t *testing.T) {
	config := DefaultSanitizeConfig()

	tests := []struct {
		name     string
		price    float64
		rarity   string
		expected float64
		desc     string
	}{
		// Valid prices
		{"valid_common", 50.00, "common", 50.00, "Valid common price"},
		{"valid_rare", 1500.00, "rare", 1500.00, "Valid rare price"},
		{"valid_ultra", 8000.00, "ultrarare", 8000.00, "Valid ultra rare price"},

		// Outlier prices (should return 0)
		{"outlier_69420", 69420.69, "common", 0, "69420 pattern detected"},
		{"outlier_69420_int", 69420.00, "rare", 0, "69420 integer pattern"},
		{"outlier_12345", 12345.67, "common", 0, "Test value 12345.67"},
		{"outlier_99999", 99999.99, "rare", 0, "Test value 99999.99"},

		// Price caps
		{"cap_common", 600.00, "common", 0, "Common exceeds 500 cap"},
		{"cap_rare", 6000.00, "rare", 0, "Rare exceeds 5000 cap"},
		{"cap_ultra", 20000.00, "ultrarare", 0, "Ultra rare exceeds 15000 cap"},

		// Invalid prices
		{"negative", -10.00, "common", 0, "Negative price"},
		{"nan", math.NaN(), "rare", 0, "NaN price"},
		{"inf", math.Inf(1), "rare", 0, "Infinite price"},

		// Penny cards
		{"penny_card", 0.03, "common", 0, "Below minimum threshold"},
		{"at_minimum", 0.05, "common", 0.05, "At minimum threshold"},
		{"above_minimum", 0.10, "common", 0.10, "Above minimum threshold"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePrice(tt.price, tt.rarity, config)
			if math.Abs(result-tt.expected) > 0.001 {
				t.Errorf("%s: expected %.2f, got %.2f", tt.desc, tt.expected, result)
			}
		})
	}
}

func TestSanitizePriceWithPennyCards(t *testing.T) {
	config := &SanitizeConfig{
		MinPriceUSD:      0.05,
		EnablePennyCards: true,
		OutlierThreshold: 10.0,
	}

	tests := []struct {
		price    float64
		expected float64
		desc     string
	}{
		{0.01, 0.01, "Penny card enabled: $0.01"},
		{0.03, 0.03, "Penny card enabled: $0.03"},
		{0.05, 0.05, "At minimum: $0.05"},
		{0.10, 0.10, "Above minimum: $0.10"},
	}

	for _, tt := range tests {
		result := SanitizePrice(tt.price, "common", config)
		if math.Abs(result-tt.expected) > 0.001 {
			t.Errorf("%s: expected %.2f, got %.2f", tt.desc, tt.expected, result)
		}
	}
}

func TestSanitizeRow(t *testing.T) {
	config := DefaultSanitizeConfig()

	row := Row{
		Card: model.Card{
			Name:   "Test Card",
			Number: "001",
			Rarity: "rare",
		},
		RawUSD: 69420.69, // Outlier
		Grades: Grades{
			PSA10:   2000.00,  // Valid
			Grade9:  1500.00,  // Valid
			Grade95: 99999.99, // Test value
			BGS10:   6000.00,  // Exceeds rare cap
		},
	}

	sanitized := SanitizeRow(row, config)

	// Check that outliers were zeroed
	if sanitized.RawUSD != 0 {
		t.Errorf("Expected RawUSD to be 0, got %.2f", sanitized.RawUSD)
	}

	// Check valid prices remain
	if sanitized.Grades.PSA10 != 2000.00 {
		t.Errorf("Expected PSA10 to be 2000.00, got %.2f", sanitized.Grades.PSA10)
	}
	if sanitized.Grades.Grade9 != 1500.00 {
		t.Errorf("Expected Grade9 to be 1500.00, got %.2f", sanitized.Grades.Grade9)
	}

	// Check outliers were zeroed
	if sanitized.Grades.Grade95 != 0 {
		t.Errorf("Expected Grade95 to be 0, got %.2f", sanitized.Grades.Grade95)
	}
	if sanitized.Grades.BGS10 != 0 {
		t.Errorf("Expected BGS10 to be 0, got %.2f", sanitized.Grades.BGS10)
	}
}

func TestSanitizeRows(t *testing.T) {
	config := DefaultSanitizeConfig()

	rows := []Row{
		{
			Card:   model.Card{Name: "Valid Card", Rarity: "rare"},
			RawUSD: 100.00,
			Grades: Grades{PSA10: 500.00},
		},
		{
			Card:   model.Card{Name: "Outlier Card", Rarity: "common"},
			RawUSD: 69420.00,
			Grades: Grades{PSA10: 69420.00},
		},
		{
			Card:   model.Card{Name: "Penny Card", Rarity: "common"},
			RawUSD: 0.02,
			Grades: Grades{PSA10: 10.00},
		},
		{
			Card:   model.Card{Name: "All Invalid", Rarity: "rare"},
			RawUSD: -10.00,
			Grades: Grades{PSA10: math.NaN()},
		},
	}

	sanitized := SanitizeRows(rows, config)

	// Should keep valid card and penny card with valid PSA10
	if len(sanitized) != 2 {
		t.Errorf("Expected 2 sanitized rows, got %d", len(sanitized))
	}

	// Verify first card is valid
	if sanitized[0].Card.Name != "Valid Card" {
		t.Errorf("Expected first card to be 'Valid Card', got %s", sanitized[0].Card.Name)
	}

	// Verify penny card kept (has valid PSA10)
	if len(sanitized) > 1 && sanitized[1].Card.Name != "Penny Card" {
		t.Errorf("Expected second card to be 'Penny Card', got %s", sanitized[1].Card.Name)
	}
}

func TestExtractUngradedUSDWithSanitization(t *testing.T) {
	config := DefaultSanitizeConfig()

	// Create test cards with various price patterns
	marketPrice := 69420.69
	midPrice := 50.00
	validPrice := 100.00

	card1 := model.Card{
		Name:   "Outlier Market Price",
		Rarity: "common",
		TCGPlayer: &model.TCGPlayerBlock{
			Prices: map[string]struct {
				Low       *float64 `json:"low,omitempty"`
				Mid       *float64 `json:"mid,omitempty"`
				High      *float64 `json:"high,omitempty"`
				Market    *float64 `json:"market,omitempty"`
				DirectLow *float64 `json:"directLow,omitempty"`
			}{
				"normal": {
					Market: &marketPrice, // Outlier
					Mid:    &midPrice,    // Valid fallback
				},
			},
		},
	}

	// Test with outlier market price - should fallback to mid
	value, source, _ := ExtractUngradedUSDWithSanitization(card1, config)
	if value != 50.00 {
		t.Errorf("Expected fallback to mid price 50.00, got %.2f", value)
	}
	if source != "tcgplayer.mid" {
		t.Errorf("Expected source 'tcgplayer.mid', got %s", source)
	}

	// Test with valid market price
	card2 := model.Card{
		Name:   "Valid Market Price",
		Rarity: "rare",
		TCGPlayer: &model.TCGPlayerBlock{
			Prices: map[string]struct {
				Low       *float64 `json:"low,omitempty"`
				Mid       *float64 `json:"mid,omitempty"`
				High      *float64 `json:"high,omitempty"`
				Market    *float64 `json:"market,omitempty"`
				DirectLow *float64 `json:"directLow,omitempty"`
			}{
				"holofoil": {
					Market: &validPrice,
				},
			},
		},
	}

	value2, source2, _ := ExtractUngradedUSDWithSanitization(card2, config)
	if value2 != 100.00 {
		t.Errorf("Expected market price 100.00, got %.2f", value2)
	}
	if source2 != "tcgplayer.market" {
		t.Errorf("Expected source 'tcgplayer.market', got %s", source2)
	}
}

func TestGetCapForRarity(t *testing.T) {
	config := DefaultSanitizeConfig()

	tests := []struct {
		rarity   string
		expected float64
	}{
		{"common", 500.00},
		{"Common", 500.00},
		{"COMMON", 500.00},
		{"uncommon", 1000.00},
		{"rare", 5000.00},
		{"Rare Holo", 10000.00},
		{"Ultra Rare", 15000.00},
		{"Secret Rare", 20000.00},
		{"Special Illustration Rare", 25000.00},
		{"unknown", 30000.00},
	}

	for _, tt := range tests {
		result := getCapForRarity(tt.rarity, config)
		if result != tt.expected {
			t.Errorf("Rarity '%s': expected cap %.2f, got %.2f", tt.rarity, tt.expected, result)
		}
	}
}

func TestCustomCaps(t *testing.T) {
	config := &SanitizeConfig{
		MinPriceUSD:      0.05,
		EnablePennyCards: false,
		OutlierThreshold: 10.0,
		CustomCaps: map[string]float64{
			"common": 100.00,   // Override default
			"mythic": 50000.00, // New category
		},
	}

	// Test overridden cap
	if cap := getCapForRarity("common", config); cap != 100.00 {
		t.Errorf("Expected custom cap 100.00 for common, got %.2f", cap)
	}

	// Test new category
	if cap := getCapForRarity("mythic", config); cap != 50000.00 {
		t.Errorf("Expected custom cap 50000.00 for mythic, got %.2f", cap)
	}

	// Test default still works
	if cap := getCapForRarity("rare", config); cap != 5000.00 {
		t.Errorf("Expected default cap 5000.00 for rare, got %.2f", cap)
	}
}

func BenchmarkSanitizePrice(b *testing.B) {
	config := DefaultSanitizeConfig()
	prices := []float64{100.00, 69420.69, 0.03, 5000.00, -10.00}
	rarities := []string{"common", "rare", "ultrarare", "holorare", "unknown"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		price := prices[i%len(prices)]
		rarity := rarities[i%len(rarities)]
		_ = SanitizePrice(price, rarity, config)
	}
}

func BenchmarkSanitizeRows(b *testing.B) {
	config := DefaultSanitizeConfig()
	rows := make([]Row, 100)
	for i := range rows {
		rows[i] = Row{
			Card: model.Card{
				Name:   "Test Card",
				Rarity: "rare",
			},
			RawUSD: float64(i * 10),
			Grades: Grades{
				PSA10:  float64(i * 50),
				Grade9: float64(i * 30),
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SanitizeRows(rows, config)
	}
}
