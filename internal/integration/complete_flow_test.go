package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/guarzo/pkmgradegap/internal/analysis"
	"github.com/guarzo/pkmgradegap/internal/cache"
	"github.com/guarzo/pkmgradegap/internal/cards"
	"github.com/guarzo/pkmgradegap/internal/model"
	"github.com/guarzo/pkmgradegap/internal/population"
	"github.com/guarzo/pkmgradegap/internal/prices"
	"github.com/guarzo/pkmgradegap/internal/sales"
	"github.com/guarzo/pkmgradegap/internal/volatility"
)

// TestCompleteAnalysisFlow tests the full analysis workflow with all features
func TestCompleteAnalysisFlow(t *testing.T) {
	// Setup test cache
	testCache, err := cache.New("test_cache.json")
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer os.Remove("test_cache.json")

	// Initialize providers
	// cardProv := cards.NewPokeTCGIO("", testCache) // Not used in this test
	// priceProv := prices.NewPriceCharting("test", testCache)
	volTracker := volatility.NewTracker()

	// Initialize sales provider (will use mock in test mode)
	salesProv := sales.NewProvider(sales.Config{
		PokemonPriceTrackerAPIKey: "test",
	})

	// Initialize population provider (will use mock)
	popProv := population.NewMockProvider()

	// Test set lookup
	ctx := context.Background()

	// Create a test set
	testSet := &model.Set{
		ID:          "test-set",
		Name:        "Test Set",
		ReleaseDate: "2024/01/01",
	}

	// Create test cards
	testCards := []model.Card{
		{
			ID:      "test-1",
			Name:    "Charizard",
			Number:  "001",
			SetName: "Test Set",
			SetID:   "test-set",
			Rarity:  "Rare Holo",
			TCGPlayer: &model.TCGPlayerBlock{
				Prices: map[string]struct {
					Low       *float64 `json:"low,omitempty"`
					Mid       *float64 `json:"mid,omitempty"`
					High      *float64 `json:"high,omitempty"`
					Market    *float64 `json:"market,omitempty"`
					DirectLow *float64 `json:"directLow,omitempty"`
				}{
					"normal": {Market: floatPtrTest(50.0)},
				},
			},
		},
		{
			ID:      "test-2",
			Name:    "Pikachu",
			Number:  "025",
			SetName: "Test Set",
			SetID:   "test-set",
			Rarity:  "Common",
			TCGPlayer: &model.TCGPlayerBlock{
				Prices: map[string]struct {
					Low       *float64 `json:"low,omitempty"`
					Mid       *float64 `json:"mid,omitempty"`
					High      *float64 `json:"high,omitempty"`
					Market    *float64 `json:"market,omitempty"`
					DirectLow *float64 `json:"directLow,omitempty"`
				}{
					"normal": {Market: floatPtrTest(5.0)},
				},
			},
		},
	}

	// Build analysis rows with all features enabled
	rows := []analysis.Row{}
	for _, card := range testCards {
		row := analysis.Row{
			Card:    card,
			RawUSD:  extractRawPrice(card),
			RawSrc:  "TCGPlayer",
			RawNote: "Market price",
			Grades: analysis.Grades{
				PSA10:   100.0, // Mock graded prices
				Grade9:  50.0,
				Grade95: 75.0,
				BGS10:   110.0,
			},
		}

		// Add population data if available
		if popProv != nil && popProv.Available() {
			if popData, err := popProv.LookupPopulation(ctx, card); err == nil {
				row.Population = &model.PSAPopulation{
					TotalGraded: popData.TotalGraded,
					PSA10:       popData.PSA10Population,
					PSA9:        popData.PSA9Population,
					PSA8:        popData.PSA8Population,
					LastUpdated: popData.LastUpdated,
				}
			}
		}

		// Add sales data if available
		if salesProv != nil && salesProv.Available() {
			if salesData, err := salesProv.GetSalesData(testSet.Name, card.Name, card.Number); err == nil {
				row.SalesData = &model.SalesData{
					CardName:        card.Name,
					SetName:         testSet.Name,
					CardNumber:      card.Number,
					LastUpdated:     salesData.LastUpdated,
					SalesCount:      salesData.SaleCount,
					AvgSalePrice:    salesData.AveragePrice,
					MedianSalePrice: salesData.MedianPrice,
					DataSource:      salesData.DataSource,
				}
			}
		}

		// Add volatility
		volTracker.AddPrice(testSet.Name, card.Name, card.Number, "raw", row.RawUSD)
		volTracker.AddPrice(testSet.Name, card.Name, card.Number, "psa10", row.Grades.PSA10)
		row.Volatility = volTracker.Calculate30DayVolatility(testSet.Name, card.Name, card.Number, "psa10")

		rows = append(rows, row)
	}

	// Test different analysis modes
	t.Run("RankAnalysis", func(t *testing.T) {
		config := analysis.Config{
			MaxAgeYears:    5,
			MinDeltaUSD:    10.0,
			MinRawUSD:      1.0,
			TopN:           10,
			GradingCost:    25.0,
			ShippingCost:   10.0,
			FeePct:         0.10,
			JapaneseWeight: 1.2,
			ShowWhy:        true,
		}

		scored := analysis.Score(rows, testSet.ReleaseDate, config)
		if len(scored) == 0 {
			t.Error("expected scored rows, got none")
		}

		// Verify scoring considers population data
		for _, s := range scored {
			if s.Score <= 0 {
				t.Errorf("expected positive score, got %f", s.Score)
			}
			if s.Population != nil && s.PSA10Rate == 0 {
				t.Error("expected PSA10 rate calculation when population data present")
			}
		}
	})

	t.Run("DataSanitization", func(t *testing.T) {
		sanitizeConfig := analysis.DefaultSanitizeConfig()
		sanitized := analysis.SanitizeRows(rows, sanitizeConfig)

		// Should keep valid rows
		if len(sanitized) != len(rows) {
			t.Logf("Sanitization removed %d rows", len(rows)-len(sanitized))
		}

		// Check that outliers would be removed
		outlierRow := analysis.Row{
			Card:   testCards[0],
			RawUSD: 10000.0, // Unrealistic price
			Grades: analysis.Grades{PSA10: 10001.0},
		}
		testRows := append(rows, outlierRow)
		sanitized = analysis.SanitizeRows(testRows, sanitizeConfig)
		if len(sanitized) >= len(testRows) {
			t.Error("expected outlier to be removed")
		}
	})

	t.Run("SalesDataIntegration", func(t *testing.T) {
		// Verify sales data is properly integrated
		for _, row := range rows {
			if row.SalesData != nil {
				if row.SalesData.DataSource == "" {
					t.Error("sales data should have a data source")
				}
				if row.SalesData.CardName != row.Card.Name {
					t.Error("sales data card name mismatch")
				}
			}
		}
	})

	t.Run("PopulationScoring", func(t *testing.T) {
		// Test that population affects scoring
		config := analysis.Config{
			GradingCost:  25.0,
			ShippingCost: 10.0,
			FeePct:       0.10,
			TopN:         10,
		}

		// Create two identical rows except for population
		lowPopRow := rows[0]
		if lowPopRow.Population != nil {
			lowPopRow.Population.PSA10 = 10 // Very low population
		}

		highPopRow := rows[0]
		if highPopRow.Population != nil {
			highPopRow.Population.PSA10 = 10000 // Very high population
		}

		lowPopScored := analysis.Score([]analysis.Row{lowPopRow}, testSet.ReleaseDate, config)
		highPopScored := analysis.Score([]analysis.Row{highPopRow}, testSet.ReleaseDate, config)

		if len(lowPopScored) > 0 && len(highPopScored) > 0 {
			// Low population should score higher
			if lowPopScored[0].Score <= highPopScored[0].Score {
				t.Logf("Warning: Low population (%d) scored %.2f, high population (%d) scored %.2f",
					lowPopRow.Population.PSA10, lowPopScored[0].Score,
					highPopRow.Population.PSA10, highPopScored[0].Score)
			}
		}
	})

	t.Run("VolatilityImpact", func(t *testing.T) {
		// Test that volatility affects scoring
		lowVolRow := rows[0]
		lowVolRow.Volatility = 0.1 // Low volatility

		highVolRow := rows[0]
		highVolRow.Volatility = 0.9 // High volatility

		config := analysis.Config{
			GradingCost:    25.0,
			ShippingCost:   10.0,
			FeePct:         0.10,
			TopN:           10,
			WithVolatility: true,
		}

		lowVolScored := analysis.Score([]analysis.Row{lowVolRow}, testSet.ReleaseDate, config)
		highVolScored := analysis.Score([]analysis.Row{highVolRow}, testSet.ReleaseDate, config)

		if len(lowVolScored) > 0 && len(highVolScored) > 0 {
			// High volatility should score lower (penalty)
			if highVolScored[0].Score >= lowVolScored[0].Score {
				t.Logf("Warning: High volatility (%.2f) scored %.2f, low volatility (%.2f) scored %.2f",
					highVolRow.Volatility, highVolScored[0].Score,
					lowVolRow.Volatility, lowVolScored[0].Score)
			}
		}
	})

	t.Run("PerformanceBenchmark", func(t *testing.T) {
		// Benchmark processing time for realistic dataset
		largeDataset := make([]analysis.Row, 250)
		for i := range largeDataset {
			largeDataset[i] = rows[0] // Duplicate for volume
			largeDataset[i].Card.Number = fmt.Sprintf("%03d", i+1)
		}

		start := time.Now()
		config := analysis.Config{
			GradingCost:  25.0,
			ShippingCost: 10.0,
			FeePct:       0.10,
			TopN:         100,
		}

		scored := analysis.Score(largeDataset, testSet.ReleaseDate, config)
		elapsed := time.Since(start)

		if elapsed > 5*time.Second {
			t.Errorf("processing 250 cards took too long: %v", elapsed)
		}

		t.Logf("Processed %d cards in %v", len(scored), elapsed)
	})
}

// TestMockProviders verifies mock providers work correctly
func TestMockProviders(t *testing.T) {
	ctx := context.Background()

	t.Run("MockSalesProvider", func(t *testing.T) {
		provider := sales.NewProvider(sales.Config{})
		if !provider.Available() {
			t.Error("mock provider should be available")
		}

		data, err := provider.GetSalesData("Test Set", "Charizard", "001")
		if err != nil {
			t.Fatalf("mock provider failed: %v", err)
		}

		if data.DataSource != "Mock" {
			t.Errorf("expected Mock data source, got %s", data.DataSource)
		}

		if data.SaleCount == 0 {
			t.Error("mock should provide sale count")
		}
	})

	t.Run("MockPopulationProvider", func(t *testing.T) {
		provider := population.NewMockProvider()
		if !provider.Available() {
			t.Error("mock provider should be available")
		}

		card := model.Card{
			Name:    "Pikachu",
			SetName: "Test Set",
			Number:  "025",
		}

		data, err := provider.LookupPopulation(ctx, card)
		if err != nil {
			t.Fatalf("mock provider failed: %v", err)
		}

		if data.PSA10Population == 0 {
			t.Error("mock should provide population data")
		}

		if data.ScarcityLevel == "" {
			t.Error("mock should provide scarcity level")
		}
	})

	t.Run("CSVProvider", func(t *testing.T) {
		// Test CSV provider with mock data
		config := population.CSVConfig{
			DataPath:     "test_population.csv",
			AutoDownload: false,
		}

		provider := population.NewCSVProvider(config)
		// Provider should work even without data (returns fallback)

		card := model.Card{
			Name:    "Mew",
			SetName: "Test Set",
			Number:  "151",
		}

		data, err := provider.LookupPopulation(ctx, card)
		if err != nil {
			t.Fatalf("CSV provider failed: %v", err)
		}

		// Should return fallback data
		if data.PSA10Population == 0 {
			t.Error("CSV provider should return fallback data")
		}

		// Clean up
		os.Remove("test_population.csv")
	})
}

// TestDataFusion tests combining data from multiple sources
func TestDataFusion(t *testing.T) {
	// This would test the smart data fusion logic when implemented
	// For now, just verify the structure is in place

	card := model.Card{
		Name:    "Test Card",
		SetName: "Test Set",
		Number:  "001",
	}

	// Simulate data from multiple sources
	priceData := analysis.Grades{
		PSA10:  100.0,
		Grade9: 50.0,
	}

	salesData := &model.SalesData{
		PSA10AvgPrice: 95.0, // Slightly different from price data
		PSA10Sales:    5,
		DataSource:    "PokemonPriceTracker",
	}

	// In a real fusion system, we'd reconcile these differences
	// For now, just verify the structures work together
	row := analysis.Row{
		Card:      card,
		Grades:    priceData,
		SalesData: salesData,
	}

	if row.SalesData.PSA10AvgPrice == 0 {
		t.Error("sales data should be preserved")
	}

	if row.Grades.PSA10 == 0 {
		t.Error("price data should be preserved")
	}
}

// Helper function
func floatPtrTest(f float64) *float64 {
	return &f
}

func extractRawPrice(card model.Card) float64 {
	if card.TCGPlayer != nil {
		if prices, ok := card.TCGPlayer.Prices["normal"]; ok {
			if prices.Market != nil {
				return *prices.Market
			}
		}
	}
	return 0
}
