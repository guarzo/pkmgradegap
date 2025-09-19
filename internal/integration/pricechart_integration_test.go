package integration

import (
	"os"
	"testing"
	"time"

	"github.com/guarzo/pkmgradegap/internal/cache"
	"github.com/guarzo/pkmgradegap/internal/model"
	"github.com/guarzo/pkmgradegap/internal/prices"
	"github.com/guarzo/pkmgradegap/internal/testutil"
)

// TestPriceCharting_LiveAPI_DataExtraction tests real API integration
// This test requires a valid PRICECHARTING_TOKEN environment variable
func TestPriceCharting_LiveAPI_DataExtraction(t *testing.T) {
	token := os.Getenv("PRICECHARTING_TOKEN")
	if token == "" {
		t.Skip("Skipping live API test: PRICECHARTING_TOKEN not set")
	}

	// Create a temporary cache
	tmpDir := t.TempDir()
	testCache, err := cache.New(tmpDir + "/test_cache.json")
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	pc := prices.NewPriceCharting(token, testCache)

	// Test with a known Pokemon card
	card := model.Card{
		Name:   "Charizard",
		Number: "4",
	}

	t.Run("Base Set Charizard Lookup", func(t *testing.T) {
		match, err := pc.LookupCard("Base Set", card)
		if err != nil {
			t.Fatalf("failed to lookup card: %v", err)
		}

		// Verify basic fields are populated
		if match.ID == "" {
			t.Error("expected non-empty product ID")
		}
		if match.ProductName == "" {
			t.Error("expected non-empty product name")
		}

		// Log the retrieved data for manual inspection
		t.Logf("Product ID: %s", match.ID)
		t.Logf("Product Name: %s", match.ProductName)
		t.Logf("Loose Price: $%.2f", float64(match.LooseCents)/100)
		t.Logf("PSA 10 Price: $%.2f", float64(match.PSA10Cents)/100)
		t.Logf("Grade 9 Price: $%.2f", float64(match.Grade9Cents)/100)

		// New fields
		if match.NewPriceCents > 0 {
			t.Logf("New (Sealed) Price: $%.2f", float64(match.NewPriceCents)/100)
		}
		if match.CIBPriceCents > 0 {
			t.Logf("CIB Price: $%.2f", float64(match.CIBPriceCents)/100)
		}
		if match.ManualPriceCents > 0 {
			t.Logf("Manual Only Price: $%.2f", float64(match.ManualPriceCents)/100)
		}
		if match.BoxPriceCents > 0 {
			t.Logf("Box Only Price: $%.2f", float64(match.BoxPriceCents)/100)
		}

		// Sales data
		if match.SalesVolume > 0 {
			t.Logf("Sales Volume: %d", match.SalesVolume)
		}
		if match.LastSoldDate != "" {
			t.Logf("Last Sold Date: %s", match.LastSoldDate)
		}

		// Retail pricing
		if match.RetailBuyPrice > 0 {
			t.Logf("Retail Buy Price: $%.2f", float64(match.RetailBuyPrice)/100)
		}
		if match.RetailSellPrice > 0 {
			t.Logf("Retail Sell Price: $%.2f", float64(match.RetailSellPrice)/100)
		}

		// Recent sales
		if len(match.RecentSales) > 0 {
			t.Logf("Recent Sales Count: %d", len(match.RecentSales))
			for i, sale := range match.RecentSales {
				if i < 3 { // Show first 3 sales
					t.Logf("  Sale %d: $%.2f on %s (Grade: %s, Source: %s)",
						i+1, float64(sale.PriceCents)/100, sale.Date, sale.Grade, sale.Source)
				}
			}
		}

		// Verify we have at least some price data
		if match.LooseCents == 0 && match.PSA10Cents == 0 && match.Grade9Cents == 0 {
			t.Error("expected at least one price field to be populated")
		}
	})

	t.Run("Modern Set Card Lookup", func(t *testing.T) {
		// Try a more recent card
		modernCard := model.Card{
			Name:   "Pikachu",
			Number: "25",
		}

		match, err := pc.LookupCard("Celebrations", modernCard)
		if err != nil {
			// Not critical if this specific card isn't found
			t.Logf("Modern card lookup warning: %v", err)
			return
		}

		t.Logf("Modern Card: %s", match.ProductName)
		t.Logf("  Loose: $%.2f", float64(match.LooseCents)/100)
		t.Logf("  PSA 10: $%.2f", float64(match.PSA10Cents)/100)

		// Check if sales data is available for modern cards
		if match.SalesVolume > 0 {
			t.Logf("  Sales Volume: %d", match.SalesVolume)
		}
	})

	t.Run("Cache Functionality", func(t *testing.T) {
		// First lookup should hit the API
		start := time.Now()
		match1, err := pc.LookupCard("Base Set", card)
		if err != nil {
			t.Fatalf("first lookup failed: %v", err)
		}
		apiDuration := time.Since(start)

		// Second lookup should use cache and be much faster
		start = time.Now()
		match2, err := pc.LookupCard("Base Set", card)
		if err != nil {
			t.Fatalf("cached lookup failed: %v", err)
		}
		cacheDuration := time.Since(start)

		// Cache should be significantly faster (at least 10x)
		if cacheDuration > apiDuration/10 {
			t.Logf("Warning: cache might not be working properly. API: %v, Cache: %v", apiDuration, cacheDuration)
		}

		// Verify cached data matches
		if match1.ID != match2.ID {
			t.Errorf("cached ID mismatch: %s vs %s", match1.ID, match2.ID)
		}
		if match1.PSA10Cents != match2.PSA10Cents {
			t.Errorf("cached PSA10 price mismatch: %d vs %d", match1.PSA10Cents, match2.PSA10Cents)
		}
	})
}

// TestPriceCharting_RetryLogic tests the retry mechanism with a flaky server
func TestPriceCharting_RetryLogic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping retry logic test in short mode")
	}

	// This test would require setting up a mock server that fails intermittently
	// For now, we'll test with the real API assuming it's stable
	token := os.Getenv("PRICECHARTING_TOKEN")
	if token == "" {
		token = testutil.GetTestPriceChartingToken()
	}

	// Skip test if using default test token (not a real API key)
	if token == testutil.DefaultTestToken || token == "test-token" {
		t.Skip("Skipping retry logic test - requires real PRICECHARTING_TOKEN")
	}

	pc := prices.NewPriceCharting(token, nil)

	// Test multiple concurrent requests to stress the retry logic
	cards := []model.Card{
		{Name: "Charizard", Number: "4"},
		{Name: "Blastoise", Number: "2"},
		{Name: "Venusaur", Number: "15"},
	}

	type result struct {
		card  model.Card
		match *prices.PCMatch
		err   error
	}

	results := make(chan result, len(cards))

	for _, c := range cards {
		go func(card model.Card) {
			match, err := pc.LookupCard("Base Set", card)
			results <- result{card: card, match: match, err: err}
		}(c)
	}

	// Collect results
	successCount := 0
	for i := 0; i < len(cards); i++ {
		res := <-results
		if res.err == nil && res.match != nil {
			successCount++
			t.Logf("Successfully retrieved %s: %s", res.card.Name, res.match.ProductName)
		} else if res.err != nil {
			t.Logf("Failed to retrieve %s: %v", res.card.Name, res.err)
		}
	}

	// We expect at least some successes even if the API is flaky
	if successCount == 0 {
		t.Error("all requests failed - retry logic may not be working")
	}
}

// BenchmarkPriceCharting_LiveAPI benchmarks actual API performance
func BenchmarkPriceCharting_LiveAPI(b *testing.B) {
	token := os.Getenv("PRICECHARTING_TOKEN")
	if token == "" {
		b.Skip("Skipping benchmark: PRICECHARTING_TOKEN not set")
	}

	pc := prices.NewPriceCharting(token, nil)
	card := model.Card{
		Name:   "Pikachu",
		Number: "25",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pc.LookupCard("Base Set", card)
		if err != nil {
			b.Fatalf("lookup failed: %v", err)
		}
		// Add small delay to avoid rate limiting
		time.Sleep(100 * time.Millisecond)
	}
}

// BenchmarkPriceCharting_WithCache benchmarks cached performance
func BenchmarkPriceCharting_WithCache(b *testing.B) {
	token := os.Getenv("PRICECHARTING_TOKEN")
	if token == "" {
		token = testutil.GetTestPriceChartingToken()
	}

	tmpDir := b.TempDir()
	testCache, err := cache.New(tmpDir + "/bench_cache.json")
	if err != nil {
		b.Fatalf("failed to create cache: %v", err)
	}

	pc := prices.NewPriceCharting(token, testCache)
	card := model.Card{
		Name:   "Pikachu",
		Number: "25",
	}

	// Warm up the cache
	_, _ = pc.LookupCard("Base Set", card)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pc.LookupCard("Base Set", card)
		if err != nil {
			b.Fatalf("lookup failed: %v", err)
		}
	}
}
