package marketplace

import (
	"testing"
	"time"

	"github.com/guarzo/pkmgradegap/internal/cache"
)

func TestPriceChartingMarketplace_Available(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected bool
	}{
		{
			name:     "Valid API key",
			apiKey:   "valid-api-key",
			expected: true,
		},
		{
			name:     "Empty API key",
			apiKey:   "",
			expected: false,
		},
		{
			name:     "Test API key",
			apiKey:   "test",
			expected: false,
		},
		{
			name:     "Mock API key",
			apiKey:   "mock",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPriceChartingMarketplace(tt.apiKey, nil)
			if p == nil && tt.expected {
				t.Errorf("Expected provider to be created but got nil")
			} else if p != nil && p.Available() != tt.expected {
				t.Errorf("Available() = %v, want %v", p.Available(), tt.expected)
			}
		})
	}
}

func TestPriceChartingMarketplace_GetProviderName(t *testing.T) {
	// Use a temporary cache for testing
	testCache, _ := cache.New("/tmp/test_marketplace_cache.json")
	p := NewPriceChartingMarketplace("valid-key", testCache)
	if p != nil {
		name := p.GetProviderName()
		expected := "PriceCharting Marketplace"
		if name != expected {
			t.Errorf("GetProviderName() = %v, want %v", name, expected)
		}
	}
}

func TestCompetitionAnalyzer_CalculateListingVelocity(t *testing.T) {
	analyzer := &CompetitionAnalyzer{}

	tests := []struct {
		name     string
		listings *MarketListings
		expected float64
	}{
		{
			name: "No listings",
			listings: &MarketListings{
				Listings: []Listing{},
			},
			expected: 0,
		},
		{
			name: "Listings with dates",
			listings: &MarketListings{
				Listings: []Listing{
					{ListedDate: time.Now().AddDate(0, 0, -10)}, // 10 days ago
					{ListedDate: time.Now().AddDate(0, 0, -5)},  // 5 days ago
					{ListedDate: time.Now().AddDate(0, 0, -1)},  // 1 day ago
				},
			},
			expected: 0.6, // Approximately 3 listings over 5 days average
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.calculateListingVelocity(tt.listings)
			// Allow some tolerance for time-based calculations
			if tt.expected == 0 && result != 0 {
				t.Errorf("calculateListingVelocity() = %v, want %v", result, tt.expected)
			} else if tt.expected > 0 && (result < tt.expected*0.5 || result > tt.expected*1.5) {
				t.Errorf("calculateListingVelocity() = %v, want approximately %v", result, tt.expected)
			}
		})
	}
}

func TestCompetitionAnalyzer_CalculateOptimalPrice(t *testing.T) {
	analyzer := &CompetitionAnalyzer{}

	tests := []struct {
		name     string
		listings *MarketListings
		stats    *PriceStats
		expected int
	}{
		{
			name: "Low competition",
			listings: &MarketListings{
				TotalListings:    2,
				LowestPriceCents: 1000,
			},
			stats: &PriceStats{
				Median:       1200,
				Percentile25: 1000,
				Percentile75: 1400,
			},
			expected: 1300, // Price higher with low competition
		},
		{
			name: "High competition",
			listings: &MarketListings{
				TotalListings:    15,
				LowestPriceCents: 1000,
			},
			stats: &PriceStats{
				Median:       1200,
				Percentile25: 1000,
				Percentile75: 1400,
			},
			expected: 1100, // Price lower with high competition
		},
		{
			name: "Medium competition",
			listings: &MarketListings{
				TotalListings:    5,
				LowestPriceCents: 1000,
			},
			stats: &PriceStats{
				Median:       1200,
				Percentile25: 1000,
				Percentile75: 1400,
			},
			expected: 1200, // Use median for medium competition
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.calculateOptimalPrice(tt.listings, tt.stats)
			if result != tt.expected {
				t.Errorf("calculateOptimalPrice() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCompetitionAnalyzer_DetectAnomalies(t *testing.T) {
	analyzer := &CompetitionAnalyzer{}

	listings := &MarketListings{
		Listings: []Listing{
			{PriceCents: 100, SellerID: "seller1"}, // Way below average
			{PriceCents: 500, SellerID: "seller2"},
			{PriceCents: 600, SellerID: "seller3"},
			{PriceCents: 2000, SellerID: "seller4"}, // Way above average
		},
	}

	stats := &PriceStats{
		Mean:              550,
		Percentile25:      400,
		Percentile75:      700,
		StandardDeviation: 200, // Add standard deviation for anomaly detection
	}

	anomalies := analyzer.detectAnomalies(listings, stats)

	if len(anomalies) == 0 {
		t.Error("Expected to detect anomalies but got none")
	}

	// Check for underpriced anomaly
	foundUnderpriced := false
	foundOverpriced := false
	for _, anomaly := range anomalies {
		if anomaly.Type == "UNDERPRICED" && anomaly.PriceCents == 100 {
			foundUnderpriced = true
		}
		if anomaly.Type == "OVERPRICED" && anomaly.PriceCents == 2000 {
			foundOverpriced = true
		}
	}

	if !foundUnderpriced {
		t.Error("Expected to find underpriced anomaly")
	}
	if !foundOverpriced {
		t.Error("Expected to find overpriced anomaly")
	}
}

func TestMarketTimingAnalyzer_DetermineTrend(t *testing.T) {
	analyzer := &MarketTimingAnalyzer{}

	tests := []struct {
		name     string
		data     []PricePoint
		expected string
	}{
		{
			name:     "No data",
			data:     []PricePoint{},
			expected: "NEUTRAL",
		},
		{
			name: "Increasing prices",
			data: []PricePoint{
				{Date: time.Now().AddDate(0, 0, -30), PriceCents: 1000},
				{Date: time.Now().AddDate(0, 0, -20), PriceCents: 1100},
				{Date: time.Now().AddDate(0, 0, -10), PriceCents: 1200},
				{Date: time.Now(), PriceCents: 1300},
			},
			expected: "BULLISH",
		},
		{
			name: "Decreasing prices",
			data: []PricePoint{
				{Date: time.Now().AddDate(0, 0, -30), PriceCents: 1300},
				{Date: time.Now().AddDate(0, 0, -20), PriceCents: 1200},
				{Date: time.Now().AddDate(0, 0, -10), PriceCents: 1100},
				{Date: time.Now(), PriceCents: 1000},
			},
			expected: "BEARISH",
		},
		{
			name: "Stable prices",
			data: []PricePoint{
				{Date: time.Now().AddDate(0, 0, -30), PriceCents: 1000},
				{Date: time.Now().AddDate(0, 0, -20), PriceCents: 1020},
				{Date: time.Now().AddDate(0, 0, -10), PriceCents: 1010},
				{Date: time.Now(), PriceCents: 1015},
			},
			expected: "NEUTRAL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.determineTrend(tt.data)
			if result != tt.expected {
				t.Errorf("determineTrend() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test is removed as enricher is in a different package (prices)
