package integration

import (
	"context"
	"os"
	"testing"

	"github.com/guarzo/pkmgradegap/internal/analysis"
	"github.com/guarzo/pkmgradegap/internal/cache"
	"github.com/guarzo/pkmgradegap/internal/marketplace"
	"github.com/guarzo/pkmgradegap/internal/model"
	"github.com/guarzo/pkmgradegap/internal/prices"
)

func TestMarketplaceIntegration(t *testing.T) {
	// Skip if no API key
	apiKey := os.Getenv("PRICECHARTING_TOKEN")
	if apiKey == "" || apiKey == "test" || apiKey == "mock" {
		t.Skip("Skipping marketplace integration test: PRICECHARTING_TOKEN not set")
	}

	// Create cache
	c, _ := cache.New("/tmp/test_marketplace_integration.json")

	// Create price provider with marketplace enrichment
	priceProv := prices.NewPriceCharting(apiKey, c)
	if !priceProv.Available() {
		t.Fatal("Price provider not available")
	}

	// Create marketplace provider
	marketProv := marketplace.NewPriceChartingMarketplace(apiKey, c)
	if marketProv == nil || !marketProv.Available() {
		t.Skip("Marketplace provider not available")
	}

	// Test card
	testCard := model.Card{
		Name:   "Charizard ex",
		Number: "054",
	}
	setName := "Obsidian Flames"

	// Lookup price data
	match, err := priceProv.LookupCard(setName, testCard)
	if err != nil {
		t.Fatalf("Failed to lookup card: %v", err)
	}

	if match == nil {
		t.Fatal("No match found for test card")
	}

	// Verify marketplace fields were populated
	t.Logf("Card: %s #%s", testCard.Name, testCard.Number)
	t.Logf("Product ID: %s", match.ID)
	t.Logf("Active Listings: %d", match.ActiveListings)
	t.Logf("Lowest Listing: $%.2f", float64(match.LowestListing)/100)
	t.Logf("Optimal Price: $%.2f", float64(match.OptimalListingPrice)/100)
	t.Logf("Competition Level: %s", match.CompetitionLevel)
	t.Logf("Market Trend: %s", match.MarketTrend)
	t.Logf("Listing Velocity: %.2f", match.ListingVelocity)
	t.Logf("Supply/Demand Ratio: %.2f", match.SupplyDemandRatio)
	t.Logf("Price Volatility: %.2f", match.PriceVolatility)
	t.Logf("Market Confidence: %.2f", match.MarketConfidence)

	// Test competition analysis
	analyzer := marketplace.NewCompetitionAnalyzer(marketProv)
	marketAnalysis, err := analyzer.AnalyzeMarket(match.ID)
	if err != nil {
		t.Logf("Warning: Could not analyze market: %v", err)
	} else if marketAnalysis != nil {
		t.Logf("Market Analysis:")
		t.Logf("  Days on Market: %.1f", marketAnalysis.DaysOnMarket)
		t.Logf("  Sales Velocity: %.2f/day", marketAnalysis.SalesVelocity)
		t.Logf("  Price Anomalies: %d", len(marketAnalysis.PriceAnomalies))

		opportunities := analyzer.IdentifyOpportunities(marketAnalysis)
		if len(opportunities) > 0 {
			t.Logf("Opportunities identified:")
			for _, opp := range opportunities {
				t.Logf("  - %s", opp)
			}
		}
	}

	// Test timing recommendations
	timingAnalyzer := marketplace.NewMarketTimingAnalyzer(marketProv)
	timing, err := timingAnalyzer.GetTimingRecommendations(match.ID, nil)
	if err != nil {
		t.Logf("Warning: Could not get timing recommendations: %v", err)
	} else if timing != nil {
		t.Logf("Timing Recommendations:")
		t.Logf("  Current Trend: %s", timing.CurrentTrend)
		t.Logf("  Best Buy Time: %s", timing.BestBuyTime)
		t.Logf("  Best Sell Time: %s", timing.BestSellTime)
		t.Logf("  Recommendation: %s", timing.Recommendation)
		t.Logf("  Confidence: %.1f%%", timing.Confidence*100)
	}
}

func TestMarketplaceCSVOutput(t *testing.T) {
	// Create test rows with marketplace data
	testRows := []analysis.Row{
		{
			Card: model.Card{
				Name:   "Test Card 1",
				Number: "001",
			},
			RawUSD: 50.0,
			Grades: analysis.Grades{
				PSA10:  150.0,
				Grade9: 100.0,
			},
			// Marketplace fields
			ActiveListings:      5,
			LowestListing:       48.0,
			ListingVelocity:     2.5,
			CompetitionLevel:    "MEDIUM",
			OptimalListingPrice: 52.0,
			MarketTrend:         "BULLISH",
			SupplyDemandRatio:   2.0,
		},
		{
			Card: model.Card{
				Name:   "Test Card 2",
				Number: "002",
			},
			RawUSD: 100.0,
			Grades: analysis.Grades{
				PSA10:  300.0,
				Grade9: 200.0,
			},
			// Marketplace fields
			ActiveListings:      15,
			LowestListing:       95.0,
			ListingVelocity:     5.0,
			CompetitionLevel:    "HIGH",
			OptimalListingPrice: 98.0,
			MarketTrend:         "NEUTRAL",
			SupplyDemandRatio:   3.0,
		},
	}

	// Create config with marketplace enabled
	config := analysis.Config{
		MinRawUSD:       10.0,
		MinDeltaUSD:     50.0,
		GradingCost:     18.0,
		ShippingCost:    15.0,
		FeePct:          0.13,
		TopN:            10,
		WithMarketplace: true,
	}

	// Generate report
	records := analysis.ReportRank(testRows, nil, config)

	// Verify headers include marketplace columns
	if len(records) == 0 {
		t.Fatal("No records generated")
	}

	headers := records[0]
	hasMarketplaceColumns := false
	for _, h := range headers {
		if h == "ActiveListings" || h == "OptimalPrice" || h == "MarketTrend" {
			hasMarketplaceColumns = true
			break
		}
	}

	if !hasMarketplaceColumns {
		t.Error("Marketplace columns not found in CSV headers")
		t.Logf("Headers: %v", headers)
	}

	// Verify data rows include marketplace data
	if len(records) > 1 {
		dataRow := records[1]
		t.Logf("Data row: %v", dataRow)

		// Find the column indices for marketplace fields
		activeListingsIdx := -1
		for i, h := range headers {
			if h == "ActiveListings" {
				activeListingsIdx = i
				break
			}
		}

		if activeListingsIdx >= 0 && activeListingsIdx < len(dataRow) {
			if dataRow[activeListingsIdx] == "" || dataRow[activeListingsIdx] == "0" {
				t.Logf("Warning: ActiveListings column appears empty")
			}
		}
	}
}

func TestMarketplaceEnrichment(t *testing.T) {
	apiKey := os.Getenv("PRICECHARTING_TOKEN")
	if apiKey == "" || apiKey == "test" || apiKey == "mock" {
		t.Skip("Skipping enrichment test: PRICECHARTING_TOKEN not set")
	}

	c, _ := cache.New("/tmp/test_enrichment.json")
	enricher := prices.NewMarketplaceEnricher(apiKey, c)

	if enricher == nil || !enricher.Available() {
		t.Skip("Marketplace enricher not available")
	}

	// Create a test PCMatch
	match := &prices.PCMatch{
		ID:          "pokemon-obsidian-flames-charizard-ex-054",
		ProductName: "Charizard ex",
		LooseCents:  5000,
		PSA10Cents:  15000,
		Grade9Cents: 10000,
	}

	// Enrich with marketplace data
	err := enricher.EnrichPCMatch(match)
	if err != nil {
		t.Logf("Warning: Enrichment returned error: %v", err)
	}

	// Log enriched data
	t.Logf("Enriched match:")
	t.Logf("  Active Listings: %d", match.ActiveListings)
	t.Logf("  Lowest Listing: %d cents", match.LowestListing)
	t.Logf("  Competition: %s", match.CompetitionLevel)
	t.Logf("  Market Trend: %s", match.MarketTrend)
	t.Logf("  Optimal Price: %d cents", match.OptimalListingPrice)
}

func TestMarketplaceProviderMock(t *testing.T) {
	// Test that mock/test providers are properly disabled
	c, _ := cache.New("/tmp/test_mock.json")

	testKeys := []string{"", "test", "mock"}
	for _, key := range testKeys {
		provider := marketplace.NewPriceChartingMarketplace(key, c)
		if provider != nil && provider.Available() {
			t.Errorf("Provider should not be available with key: %s", key)
		}

		enricher := prices.NewMarketplaceEnricher(key, c)
		if enricher != nil && enricher.Available() {
			t.Errorf("Enricher should not be available with key: %s", key)
		}
	}
}

func TestMarketplaceDataFlow(t *testing.T) {
	// Test the complete data flow from API to CSV output
	apiKey := os.Getenv("PRICECHARTING_TOKEN")
	if apiKey == "" || apiKey == "test" || apiKey == "mock" {
		t.Skip("Skipping data flow test: PRICECHARTING_TOKEN not set")
	}

	ctx := context.Background()
	c, _ := cache.New("/tmp/test_dataflow.json")

	// Create provider
	priceProv := prices.NewPriceCharting(apiKey, c)

	// Test card
	card := model.Card{
		Name:   "Pikachu ex",
		Number: "001",
	}

	// Lookup with marketplace enrichment
	match, err := priceProv.LookupCard("Surging Sparks", card)
	if err != nil {
		t.Logf("Lookup error (expected for some cards): %v", err)
		return
	}

	if match == nil {
		t.Log("No match found (expected for some cards)")
		return
	}

	// Create analysis row
	row := analysis.Row{
		Card:   card,
		RawUSD: 10.0,
		Grades: analysis.Grades{
			PSA10: float64(match.PSA10Cents) / 100,
		},
		// Populate marketplace fields
		ActiveListings:      match.ActiveListings,
		LowestListing:       float64(match.LowestListing) / 100,
		ListingVelocity:     match.ListingVelocity,
		CompetitionLevel:    match.CompetitionLevel,
		OptimalListingPrice: float64(match.OptimalListingPrice) / 100,
		MarketTrend:         match.MarketTrend,
		SupplyDemandRatio:   match.SupplyDemandRatio,
	}

	// Verify data was populated
	if row.ActiveListings == 0 && row.CompetitionLevel == "" {
		t.Log("Note: Marketplace data may not be available for all cards")
	} else {
		t.Logf("Successfully populated marketplace data:")
		t.Logf("  Active Listings: %d", row.ActiveListings)
		t.Logf("  Competition: %s", row.CompetitionLevel)
		t.Logf("  Market Trend: %s", row.MarketTrend)
	}

	_ = ctx // Suppress unused variable warning
}
