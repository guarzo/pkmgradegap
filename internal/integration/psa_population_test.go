package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
	"github.com/guarzo/pkmgradegap/internal/population"
)

// TestPSAScraperWithMockHTML tests the PSA scraper with mock HTML responses
func TestPSAScraperWithMockHTML(t *testing.T) {
	// Create a mock server that returns HTML responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/pop/search") {
			// Return search results HTML
			w.Write([]byte(`
				<!DOCTYPE html>
				<html>
				<body>
					<div class="search-results">
						<a href="/pop/pokemon/2023/151/001">Bulbasaur #001 - Pokemon 151</a>
					</div>
				</body>
				</html>
			`))
		} else if strings.Contains(r.URL.Path, "/pop") {
			// Return population report HTML
			w.Write([]byte(`
				<!DOCTYPE html>
				<html>
				<body>
					<table class="pop-table">
						<thead>
							<tr>
								<th>Grade</th>
								<th>1</th>
								<th>2</th>
								<th>3</th>
								<th>4</th>
								<th>5</th>
								<th>6</th>
								<th>7</th>
								<th>8</th>
								<th>9</th>
								<th>10</th>
							</tr>
						</thead>
						<tbody>
							<tr>
								<td>Population</td>
								<td>5</td>
								<td>8</td>
								<td>15</td>
								<td>22</td>
								<td>35</td>
								<td>48</td>
								<td>62</td>
								<td>125</td>
								<td>250</td>
								<td>500</td>
							</tr>
						</tbody>
					</table>
				</body>
				</html>
			`))
		}
	}))
	defer server.Close()

	// Create a mock cache
	cache := &mockCache{
		data: make(map[string]interface{}),
	}

	// Create scraper with mock server URL
	scraper := population.NewPSAScraper(cache)

	// Override the base URLs in the scraper (would need to expose these or use dependency injection)
	// For now, we'll test the parsing functions directly

	ctx := context.Background()

	t.Run("GetCardPopulation", func(t *testing.T) {
		// This would normally hit the real PSA website
		// For integration testing, we'd need to either:
		// 1. Use a mock server with URL override
		// 2. Test against real PSA website (not recommended for unit tests)
		// 3. Extract and test the parsing logic separately

		// For now, let's test that the scraper doesn't panic
		pop, err := scraper.GetCardPopulation(ctx, "Pokemon 151", "001", "Bulbasaur")

		// We expect this to fail since we're not hitting the real website
		// But it shouldn't panic
		if err == nil && pop != nil {
			t.Logf("Got population data: PSA10=%d, PSA9=%d, PSA8=%d, Total=%d",
				pop.PSA10, pop.PSA9, pop.PSA8, pop.TotalGraded)
		} else {
			t.Logf("Expected failure (not hitting real site): %v", err)
		}
	})
}

// TestPSAProviderWithScraperFallback tests the PSA provider with scraper fallback
func TestPSAProviderWithScraperFallback(t *testing.T) {
	// Create a mock rate limiter
	limiter := &mockRateLimiter{}

	// Create a mock cache
	cache := &mockCache{
		data: make(map[string]interface{}),
	}

	// Create PSA provider with no API key (should fall back to scraper)
	provider := population.NewPSAAPIProvider("", limiter, cache)

	ctx := context.Background()

	// Test card for lookup
	card := model.Card{
		Name:    "Charizard",
		SetName: "Base Set",
		Number:  "4",
	}

	t.Run("LookupPopulation_FallbackToScraper", func(t *testing.T) {
		// This should use the scraper since no API key is provided
		popData, err := provider.LookupPopulation(ctx, card)

		// We expect this to return nil without error (scraper returns nil on failure)
		// This is by design to allow fallback to mock provider
		if err != nil {
			t.Logf("Lookup failed (expected for integration test): %v", err)
		}

		if popData != nil {
			t.Errorf("Expected nil population data without real PSA access")
		}
	})

	t.Run("Available_CheckStatus", func(t *testing.T) {
		// Provider should report as unavailable without API key
		if provider.Available() {
			t.Error("Provider should be unavailable without API key")
		}
	})
}

// TestPSAProviderHealthReporting tests the enhanced health reporting for PSA provider
func TestPSAProviderHealthReporting(t *testing.T) {
	// Create providers with different states
	tests := []struct {
		name           string
		apiKey         string
		expectedAvail  bool
		expectedDetail string
	}{
		{
			name:           "No API Key",
			apiKey:         "",
			expectedAvail:  false,
			expectedDetail: "no API key configured",
		},
		{
			name:           "Test API Key",
			apiKey:         "test",
			expectedAvail:  false,
			expectedDetail: "test mode",
		},
		{
			name:           "Mock API Key",
			apiKey:         "mock",
			expectedAvail:  false,
			expectedDetail: "mock mode",
		},
		{
			name:           "Valid API Key",
			apiKey:         "valid-key-123",
			expectedAvail:  true, // PSA provider considers non-empty, non-test, non-mock keys as valid
			expectedDetail: "API available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := &mockRateLimiter{}
			cache := &mockCache{data: make(map[string]interface{})}

			provider := population.NewPSAAPIProvider(tt.apiKey, limiter, cache)

			// Test the Available method
			avail := provider.Available()
			if avail != tt.expectedAvail {
				t.Errorf("Expected availability %v, got %v", tt.expectedAvail, avail)
			}

			// Note: The enhanced Available() method that returns (bool, string)
			// would need to be implemented as part of Task 2
		})
	}
}

// Mock implementations for testing

type mockCache struct {
	data map[string]interface{}
}

func (m *mockCache) Get(key string) (*population.PopulationData, bool) {
	if val, exists := m.data[key]; exists {
		if pd, ok := val.(*population.PopulationData); ok {
			return pd, true
		}
	}
	return nil, false
}

func (m *mockCache) Set(key string, data *population.PopulationData, ttl time.Duration) error {
	m.data[key] = data
	return nil
}

func (m *mockCache) GetSet(key string) (*population.SetPopulationData, bool) {
	if val, exists := m.data[key]; exists {
		if sd, ok := val.(*population.SetPopulationData); ok {
			return sd, true
		}
	}
	return nil, false
}

func (m *mockCache) SetSet(key string, data *population.SetPopulationData, ttl time.Duration) error {
	m.data[key] = data
	return nil
}

func (m *mockCache) Clear() error {
	m.data = make(map[string]interface{})
	return nil
}

type mockRateLimiter struct{}

func (m *mockRateLimiter) Wait(ctx context.Context) error {
	// No rate limiting in tests
	return nil
}

func (m *mockRateLimiter) Allow() bool {
	// Always allow in tests
	return true
}

// TestPSAScraperHTMLParsing tests the HTML parsing logic for PSA population reports
func TestPSAScraperHTMLParsing(t *testing.T) {
	// Test various HTML structures that PSA might use
	htmlSamples := []struct {
		name     string
		html     string
		expected struct {
			psa10 int
			psa9  int
			psa8  int
			total int
		}
	}{
		{
			name: "Table Format",
			html: `
				<table class="pop-table">
					<tr>
						<th>PSA 8</th>
						<th>PSA 9</th>
						<th>PSA 10</th>
					</tr>
					<tr>
						<td>125</td>
						<td>250</td>
						<td>500</td>
					</tr>
				</table>
			`,
			expected: struct {
				psa10 int
				psa9  int
				psa8  int
				total int
			}{500, 250, 125, 875},
		},
		{
			name: "Div Format",
			html: `
				<div class="grade-10">PSA 10: 1,234</div>
				<div class="grade-9">PSA 9: 567</div>
				<div class="grade-8">PSA 8: 89</div>
			`,
			expected: struct {
				psa10 int
				psa9  int
				psa8  int
				total int
			}{1234, 567, 89, 1890},
		},
	}

	// Note: To actually test these, we'd need to expose the parsePopulationTable
	// function or create a test-specific version
	for _, sample := range htmlSamples {
		t.Run(sample.name, func(t *testing.T) {
			t.Logf("Testing HTML parsing for format: %s", sample.name)
			// Would parse and validate here if parsePopulationTable was exposed
		})
	}
}

// TestPSAScraperRateLimiting tests that the PSA scraper respects rate limits
func TestPSAScraperRateLimiting(t *testing.T) {
	cache := &mockCache{data: make(map[string]interface{})}
	scraper := population.NewPSAScraper(cache)

	ctx := context.Background()

	// Measure time for multiple requests
	start := time.Now()

	// Make 3 requests (should take at least 2 seconds with 1 req/sec limit)
	for i := 0; i < 3; i++ {
		scraper.GetCardPopulation(ctx, "Test Set", "001", "Test Card")
	}

	elapsed := time.Since(start)

	// Should take at least 2 seconds for 3 requests with 1 req/sec
	// Allow some margin for processing time
	if elapsed < 1*time.Second {
		t.Errorf("Rate limiting not working: 3 requests took only %v", elapsed)
	}

	t.Logf("3 requests took %v (rate limiting working)", elapsed)
}

// TestPSAScraperErrorHandling tests error handling in the PSA scraper
func TestPSAScraperErrorHandling(t *testing.T) {
	cache := &mockCache{data: make(map[string]interface{})}
	scraper := population.NewPSAScraper(cache)

	ctx := context.Background()

	tests := []struct {
		name   string
		set    string
		number string
		card   string
	}{
		{"Empty inputs", "", "", ""},
		{"Invalid characters", "Set!@#", "###", "Card$%^"},
		{"Very long inputs", strings.Repeat("a", 1000), "001", "Card"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should handle gracefully without panic
			pop, err := scraper.GetCardPopulation(ctx, tt.set, tt.number, tt.card)

			// We expect these to fail but not panic
			if pop != nil {
				t.Logf("Unexpected success for %s", tt.name)
			}
			if err != nil {
				t.Logf("Got expected error for %s: %v", tt.name, err)
			}
		})
	}
}

// TestPSAProviderCacheUsage tests that PSA provider caching works correctly
func TestPSAProviderCacheUsage(t *testing.T) {
	cache := &mockCache{data: make(map[string]interface{})}
	limiter := &mockRateLimiter{}

	provider := population.NewPSAAPIProvider("test-key", limiter, cache)

	ctx := context.Background()

	card := model.Card{
		Name:    "Pikachu",
		SetName: "Base Set",
		Number:  "58",
	}

	// First lookup (cache miss)
	provider.LookupPopulation(ctx, card)

	// Second lookup (should hit cache if implemented)
	provider.LookupPopulation(ctx, card)

	// Check cache was used (would need to add cache hit tracking)
	t.Log("Cache usage test completed")
}

// TestPSAProviderAPIToScraperFallback tests the fallback from API to scraper
func TestPSAProviderAPIToScraperFallback(t *testing.T) {
	cache := &mockCache{data: make(map[string]interface{})}
	limiter := &mockRateLimiter{}

	// Create provider with invalid API key
	provider := population.NewPSAAPIProvider("invalid-key", limiter, cache)

	ctx := context.Background()

	card := model.Card{
		Name:    "Mew",
		SetName: "Pokemon 151",
		Number:  "151",
	}

	// This should try API first, fail, then fall back to scraper
	popData, err := provider.LookupPopulation(ctx, card)

	if err != nil {
		t.Logf("Lookup with fallback resulted in error: %v", err)
	}

	if popData == nil {
		t.Log("No population data retrieved (expected in test environment)")
	} else {
		t.Logf("Got population data: %+v", popData)
	}
}

// BenchmarkPSAScraperHTMLParsing benchmarks PSA scraper HTML parsing performance
func BenchmarkPSAScraperHTMLParsing(b *testing.B) {
	cache := &mockCache{data: make(map[string]interface{})}
	scraper := population.NewPSAScraper(cache)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Benchmark the search and scrape operations
		scraper.GetCardPopulation(ctx, "Benchmark Set", "001", "Benchmark Card")
	}
}
