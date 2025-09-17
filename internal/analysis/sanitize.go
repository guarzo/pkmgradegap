package analysis

import (
	"fmt"
	"math"
	"strings"

	"github.com/guarzo/pkmgradegap/internal/model"
)

// PriceCaps defines maximum reasonable prices by rarity
var PriceCaps = map[string]float64{
	"common":     500.00,
	"uncommon":   1000.00,
	"rare":       5000.00,
	"holorare":   10000.00,
	"ultrarare":  15000.00,
	"secretrare": 20000.00,
	"specialart": 25000.00,
	"default":    30000.00,
}

// SanitizeConfig holds configuration for price sanitization
type SanitizeConfig struct {
	MinPriceUSD      float64            // Minimum price threshold (default 0.05)
	EnablePennyCards bool               // Allow cards under minimum price
	OutlierThreshold float64            // Statistical outlier threshold (default 10.0 std devs)
	CustomCaps       map[string]float64 // Override default price caps
}

// DefaultSanitizeConfig returns default sanitization settings
func DefaultSanitizeConfig() *SanitizeConfig {
	return &SanitizeConfig{
		MinPriceUSD:      0.05,
		EnablePennyCards: false,
		OutlierThreshold: 10.0,
		CustomCaps:       nil,
	}
}

// SanitizePrice validates and sanitizes a single price value
func SanitizePrice(price float64, rarity string, config *SanitizeConfig) float64 {
	if config == nil {
		config = DefaultSanitizeConfig()
	}

	// Check for obvious invalid prices (69420 pattern, negative, NaN)
	if isInvalidPrice(price) {
		return 0
	}

	// Apply rarity-based price cap
	cap := getCapForRarity(rarity, config)
	if price > cap {
		return 0 // Return 0 for outliers rather than capping
	}

	// Filter penny cards unless explicitly enabled
	if !config.EnablePennyCards && price < config.MinPriceUSD {
		return 0
	}

	return price
}

// SanitizeRow sanitizes all prices in a Row
func SanitizeRow(row Row, config *SanitizeConfig) Row {
	if config == nil {
		config = DefaultSanitizeConfig()
	}

	rarity := strings.ToLower(row.Card.Rarity)

	// Sanitize raw price
	row.RawUSD = SanitizePrice(row.RawUSD, rarity, config)

	// Sanitize graded prices
	row.Grades.PSA10 = SanitizePrice(row.Grades.PSA10, rarity, config)
	row.Grades.Grade9 = SanitizePrice(row.Grades.Grade9, rarity, config)
	row.Grades.Grade95 = SanitizePrice(row.Grades.Grade95, rarity, config)
	row.Grades.BGS10 = SanitizePrice(row.Grades.BGS10, rarity, config)

	return row
}

// SanitizeRows sanitizes a slice of rows
func SanitizeRows(rows []Row, config *SanitizeConfig) []Row {
	sanitized := make([]Row, 0, len(rows))
	for _, row := range rows {
		clean := SanitizeRow(row, config)
		// Only include rows that have valid prices after sanitization
		if clean.RawUSD > 0 || clean.Grades.PSA10 > 0 {
			sanitized = append(sanitized, clean)
		}
	}
	return sanitized
}

// ExtractUngradedUSDWithSanitization extracts and sanitizes ungraded USD prices
func ExtractUngradedUSDWithSanitization(c model.Card, config *SanitizeConfig) (value float64, source string, note string) {
	if config == nil {
		config = DefaultSanitizeConfig()
	}

	// Priority: TCGPlayer market > TCGPlayer mid > Cardmarket trend (EUR converted)
	if c.TCGPlayer != nil && c.TCGPlayer.Prices != nil {
		// Try market prices first
		typeOrder := []string{"normal", "holofoil", "reverseHolofoil", "1stEditionHolofoil", "1stEditionNormal"}

		// First pass: market prices
		for _, t := range typeOrder {
			if p, ok := c.TCGPlayer.Prices[t]; ok && p.Market != nil && *p.Market > 0 {
				price := SanitizePrice(*p.Market, c.Rarity, config)
				if price > 0 {
					return round2(price), "tcgplayer.market", "USD"
				}
			}
		}

		// Second pass: mid prices if no valid market price
		for _, t := range typeOrder {
			if p, ok := c.TCGPlayer.Prices[t]; ok && p.Mid != nil && *p.Mid > 0 {
				price := SanitizePrice(*p.Mid, c.Rarity, config)
				if price > 0 {
					return round2(price), "tcgplayer.mid", "USD"
				}
			}
		}
	}

	// Fallback to Cardmarket EUR (converted) if explicitly enabled
	if config.EnablePennyCards && c.Cardmarket != nil {
		if trend := c.Cardmarket.Prices.TrendPrice; trend != nil && *trend > 0 {
			// Convert EUR to USD (approximate rate)
			usdPrice := *trend * 1.1
			price := SanitizePrice(usdPrice, c.Rarity, config)
			if price > 0 {
				return round2(price), "cardmarket.trend", "EUR→USD"
			}
		}
	}

	return 0, "", ""
}

// Helper functions

func isInvalidPrice(price float64) bool {
	// Check for NaN or Inf
	if math.IsNaN(price) || math.IsInf(price, 0) {
		return true
	}

	// Check for negative
	if price < 0 {
		return true
	}

	// Check for 69420 pattern (common test/joke value)
	priceStr := formatPrice(price)
	if strings.Contains(priceStr, "69420") {
		return true
	}

	// Check for other common test values
	testValues := []float64{12345.67, 99999.99, 11111.11, 88888.88}
	for _, test := range testValues {
		if math.Abs(price-test) < 0.01 {
			return true
		}
	}

	return false
}

func getCapForRarity(rarity string, config *SanitizeConfig) float64 {
	r := strings.ToLower(rarity)

	// Check custom caps first
	if config.CustomCaps != nil {
		if cap, ok := config.CustomCaps[r]; ok {
			return cap
		}
	}

	// Map various rarity names to standard categories
	switch {
	case strings.Contains(r, "common") && !strings.Contains(r, "uncommon"):
		return PriceCaps["common"]
	case strings.Contains(r, "uncommon"):
		return PriceCaps["uncommon"]
	case strings.Contains(r, "secret"):
		return PriceCaps["secretrare"]
	case strings.Contains(r, "ultra"):
		return PriceCaps["ultrarare"]
	case strings.Contains(r, "holo"):
		return PriceCaps["holorare"]
	case strings.Contains(r, "special") || strings.Contains(r, "illustration"):
		return PriceCaps["specialart"]
	case strings.Contains(r, "rare"):
		return PriceCaps["rare"]
	default:
		return PriceCaps["default"]
	}
}

func formatPrice(price float64) string {
	return strings.ReplaceAll(strings.TrimRight(strings.TrimRight(
		fmt.Sprintf("%.2f", price), "0"), "."), ",", "")
}
