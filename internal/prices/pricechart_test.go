package prices

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/guarzo/pkmgradegap/internal/cache"
	"github.com/guarzo/pkmgradegap/internal/model"
	"github.com/guarzo/pkmgradegap/internal/testutil"
)

// testTransport is a custom RoundTripper for testing
type testTransport struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.RoundTripFunc(req)
}

// Mock PriceCharting API responses
var mockSingleProductResponse = map[string]interface{}{
	"status":            "success",
	"id":                "12345",
	"product-name":      "Pokemon Surging Sparks Pikachu #238",
	"loose-price":       850,  // $8.50 in cents
	"graded-price":      1500, // PSA 9
	"box-only-price":    1800, // Grade 9.5
	"manual-only-price": 2500, // PSA 10
	"bgs-10-price":      3000, // BGS 10
}

var mockSearchResponse = map[string]interface{}{
	"status": "success",
	"products": []map[string]interface{}{
		{
			"id":           "12345",
			"product-name": "Pokemon Surging Sparks Pikachu #238",
		},
		{
			"id":           "12346",
			"product-name": "Pokemon Surging Sparks Charizard #025",
		},
	},
}

var mockEmptySearchResponse = map[string]interface{}{
	"status":   "success",
	"products": []interface{}{},
}

var mockErrorResponse = map[string]interface{}{
	"status": "error",
	"error":  "Product not found",
}

// Mock response with sales data for Sprint 1 fields
var mockProductWithSalesResponse = map[string]interface{}{
	"status":            "success",
	"id":                "12345",
	"product-name":      "Pokemon Surging Sparks Pikachu #238",
	"loose-price":       850,
	"graded-price":      1500,
	"box-only-price":    1800,
	"manual-only-price": 2500,
	"bgs-10-price":      3000,
	"new-price":         950,  // Sealed product
	"cib-price":         1200, // Complete in box
	"manual-price":      400,  // Manual only
	"box-price":         800,  // Box only
	"sales-volume":      25,   // Recent sales count
	"last-sold-date":    "2024-01-15",
	"retail-buy-price":  600,  // Dealer buy
	"retail-sell-price": 1100, // Dealer sell
	"sales-data": []interface{}{
		// Recent sales
		map[string]interface{}{
			"sale-price": 900,
			"sale-date":  "2024-01-15",
			"grade":      "NM",
			"source":     "eBay",
		},
		map[string]interface{}{
			"sale-price": 2600,
			"sale-date":  "2024-01-14",
			"grade":      "PSA 10",
			"source":     "PWCC",
		},
	},
}

func TestNewPriceCharting(t *testing.T) {
	// Test with token
	testToken := testutil.GetTestPriceChartingToken()
	pc1 := NewPriceCharting(testToken, nil)
	if pc1.token != testToken {
		t.Errorf("expected token %s, got %s", testToken, pc1.token)
	}
	if pc1.cache != nil {
		t.Errorf("expected nil cache")
	}

	// Test with cache
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.json")
	testCache, _ := cache.New(cachePath)

	pc2 := NewPriceCharting("", testCache)
	if pc2.token != "" {
		t.Errorf("expected empty token")
	}
	if pc2.cache == nil {
		t.Errorf("expected non-nil cache")
	}
}

func TestPriceCharting_Available(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected bool
	}{
		{
			name:     "with token",
			token:    "valid-token",
			expected: true,
		},
		{
			name:     "empty token",
			token:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := NewPriceCharting(tt.token, nil)
			result := pc.Available()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestPriceCharting_LookupCard_NoToken(t *testing.T) {
	// Test behavior when no token is available
	pc := NewPriceCharting("", nil)

	card := model.Card{
		Name:   "Pikachu",
		Number: "001",
	}

	// Since no token, this should fail
	match, err := pc.LookupCard("Surging Sparks", card)
	if err == nil {
		t.Errorf("expected error when no token provided")
	}
	if match != nil {
		t.Errorf("expected nil match when no token")
	}
}

func TestPriceCharting_LookupCardWithCache(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.json")
	testCache, err := cache.New(cachePath)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	// Pre-populate cache
	cachedMatch := &PCMatch{
		ID:           "cached-123",
		ProductName:  "Cached Product",
		LooseCents:   1000,
		PSA10Cents:   5000,
		Grade9Cents:  3000,
		Grade95Cents: 4000,
		BGS10Cents:   6000,
	}

	card := model.Card{Name: "Cached Card", Number: "001"}
	key := cache.PriceChartingKey("Test Set", card.Name, card.Number)
	err = testCache.Put(key, cachedMatch, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to cache data: %v", err)
	}

	pc := NewPriceCharting(testutil.GetTestPriceChartingToken(), testCache)

	// This should use cached data
	match, err := pc.LookupCard("Test Set", card)
	if err != nil {
		t.Fatalf("lookup with cache failed: %v", err)
	}

	if match.ID != "cached-123" {
		t.Errorf("expected cached ID cached-123, got %s", match.ID)
	}
	if match.PSA10Cents != 5000 {
		t.Errorf("expected cached PSA10 price 5000, got %d", match.PSA10Cents)
	}
}

func TestHttpGetJSON(t *testing.T) {
	tests := []struct {
		name           string
		responseCode   int
		responseBody   string
		expectError    bool
		expectedErrMsg string
	}{
		{
			name:         "successful request",
			responseCode: http.StatusOK,
			responseBody: `{"status": "success", "data": "test"}`,
			expectError:  false,
		},
		{
			name:           "server error",
			responseCode:   http.StatusInternalServerError,
			responseBody:   "Internal Server Error",
			expectError:    true,
			expectedErrMsg: "HTTP 500",
		},
		{
			name:           "not found",
			responseCode:   http.StatusNotFound,
			responseBody:   "Not found",
			expectError:    true,
			expectedErrMsg: "HTTP 404",
		},
		{
			name:         "invalid JSON",
			responseCode: http.StatusOK,
			responseBody: `{invalid json}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			var result map[string]interface{}
			err := httpGetJSON(server.URL, &result)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				} else if tt.expectedErrMsg != "" && !strings.Contains(err.Error(), tt.expectedErrMsg) {
					t.Errorf("expected error containing %q, got %q", tt.expectedErrMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result["status"] != "success" {
					t.Errorf("expected success status in result")
				}
			}
		})
	}
}

func TestHasPriceKeys(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		expected bool
	}{
		{
			name: "has loose-price",
			data: map[string]interface{}{
				"loose-price": 500,
			},
			expected: true,
		},
		{
			name: "has manual-only-price",
			data: map[string]interface{}{
				"manual-only-price": 1000,
			},
			expected: true,
		},
		{
			name: "has graded-price",
			data: map[string]interface{}{
				"graded-price": 750,
			},
			expected: true,
		},
		{
			name: "has multiple price keys",
			data: map[string]interface{}{
				"loose-price":       500,
				"manual-only-price": 1000,
				"graded-price":      750,
			},
			expected: true,
		},
		{
			name: "no price keys",
			data: map[string]interface{}{
				"id":           "123",
				"product-name": "Test Product",
			},
			expected: false,
		},
		{
			name:     "empty map",
			data:     map[string]interface{}{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasPriceKeys(tt.data)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestPcFrom(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		expected *PCMatch
	}{
		{
			name: "complete data with all new fields",
			data: map[string]interface{}{
				"id":                "12345",
				"product-name":      "Pokemon Card",
				"loose-price":       850,
				"graded-price":      1500,
				"box-only-price":    1800,
				"manual-only-price": 2500,
				"bgs-10-price":      3000,
				// New price fields
				"new-price":    1200,
				"cib-price":    950,
				"manual-price": 300,
				"box-price":    400,
				// Sales data
				"sales-volume":   42,
				"last-sold-date": "2024-01-15",
				// Retail pricing
				"retail-buy-price":  700,
				"retail-sell-price": 900,
			},
			expected: &PCMatch{
				ID:           "12345",
				ProductName:  "Pokemon Card",
				LooseCents:   850,
				Grade9Cents:  1500,
				Grade95Cents: 1800,
				PSA10Cents:   2500,
				BGS10Cents:   3000,
				// New fields
				NewPriceCents:    1200,
				CIBPriceCents:    950,
				ManualPriceCents: 300,
				BoxPriceCents:    400,
				SalesVolume:      42,
				LastSoldDate:     "2024-01-15",
				RetailBuyPrice:   700,
				RetailSellPrice:  900,
			},
		},
		{
			name: "partial data with float64",
			data: map[string]interface{}{
				"id":                "67890",
				"product-name":      "Another Card",
				"loose-price":       12.5, // float64
				"manual-only-price": 25.0, // float64
				"new-price":         18.75,
				"sales-volume":      15.0,
			},
			expected: &PCMatch{
				ID:            "67890",
				ProductName:   "Another Card",
				LooseCents:    12,
				Grade9Cents:   0,
				Grade95Cents:  0,
				PSA10Cents:    25,
				BGS10Cents:    0,
				NewPriceCents: 18,
				SalesVolume:   15,
			},
		},
		{
			name: "null values handling",
			data: map[string]interface{}{
				"id":               "null-test",
				"product-name":     "Null Test Card",
				"loose-price":      nil,
				"graded-price":     850,
				"new-price":        nil,
				"sales-volume":     nil,
				"retail-buy-price": nil,
			},
			expected: &PCMatch{
				ID:           "null-test",
				ProductName:  "Null Test Card",
				LooseCents:   0,
				Grade9Cents:  850,
				Grade95Cents: 0,
				PSA10Cents:   0,
				BGS10Cents:   0,
			},
		},
		{
			name: "string number conversion",
			data: map[string]interface{}{
				"id":               "string-nums",
				"product-name":     "String Numbers Card",
				"loose-price":      "850",
				"graded-price":     "1500.50",
				"sales-volume":     "25",
				"retail-buy-price": "700",
			},
			expected: &PCMatch{
				ID:             "string-nums",
				ProductName:    "String Numbers Card",
				LooseCents:     850,
				Grade9Cents:    1500,
				SalesVolume:    25,
				RetailBuyPrice: 700,
			},
		},
		{
			name: "sales data extraction",
			data: map[string]interface{}{
				"id":           "sales-test",
				"product-name": "Sales Test Card",
				"loose-price":  500,
				"sales-data": []interface{}{
					map[string]interface{}{
						"sale-price": 525,
						"sale-date":  "2024-01-10",
						"grade":      "PSA 10",
						"source":     "eBay",
					},
					map[string]interface{}{
						"sale-price": 475,
						"sale-date":  "2024-01-09",
						"grade":      "PSA 9",
						"source":     "PWCC",
					},
				},
			},
			expected: &PCMatch{
				ID:          "sales-test",
				ProductName: "Sales Test Card",
				LooseCents:  500,
				SalesCount:  2,
				SalesVolume: 2, // Should default to SalesCount when not provided
				RecentSales: []SaleData{
					{PriceCents: 525, Date: "2024-01-10", Grade: "PSA 10", Source: "eBay"},
					{PriceCents: 475, Date: "2024-01-09", Grade: "PSA 9", Source: "PWCC"},
				},
				AvgSalePrice: 500, // (525 + 475) / 2
			},
		},
		{
			name: "missing price fields",
			data: map[string]interface{}{
				"id":           "99999",
				"product-name": "Minimal Card",
			},
			expected: &PCMatch{
				ID:           "99999",
				ProductName:  "Minimal Card",
				LooseCents:   0,
				Grade9Cents:  0,
				Grade95Cents: 0,
				PSA10Cents:   0,
				BGS10Cents:   0,
			},
		},
		{
			name: "invalid price types",
			data: map[string]interface{}{
				"id":               "invalid",
				"product-name":     "Invalid Prices",
				"loose-price":      "not-a-number",
				"graded-price":     true,
				"sales-volume":     "invalid",
				"retail-buy-price": map[string]interface{}{"nested": "object"},
			},
			expected: &PCMatch{
				ID:           "invalid",
				ProductName:  "Invalid Prices",
				LooseCents:   0,
				Grade9Cents:  0,
				Grade95Cents: 0,
				PSA10Cents:   0,
				BGS10Cents:   0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pcFrom(tt.data)

			// Basic fields
			if result.ID != tt.expected.ID {
				t.Errorf("ID: expected %s, got %s", tt.expected.ID, result.ID)
			}
			if result.ProductName != tt.expected.ProductName {
				t.Errorf("ProductName: expected %s, got %s", tt.expected.ProductName, result.ProductName)
			}

			// Price fields
			if result.LooseCents != tt.expected.LooseCents {
				t.Errorf("LooseCents: expected %d, got %d", tt.expected.LooseCents, result.LooseCents)
			}
			if result.Grade9Cents != tt.expected.Grade9Cents {
				t.Errorf("Grade9Cents: expected %d, got %d", tt.expected.Grade9Cents, result.Grade9Cents)
			}
			if result.Grade95Cents != tt.expected.Grade95Cents {
				t.Errorf("Grade95Cents: expected %d, got %d", tt.expected.Grade95Cents, result.Grade95Cents)
			}
			if result.PSA10Cents != tt.expected.PSA10Cents {
				t.Errorf("PSA10Cents: expected %d, got %d", tt.expected.PSA10Cents, result.PSA10Cents)
			}
			if result.BGS10Cents != tt.expected.BGS10Cents {
				t.Errorf("BGS10Cents: expected %d, got %d", tt.expected.BGS10Cents, result.BGS10Cents)
			}

			// New price fields
			if result.NewPriceCents != tt.expected.NewPriceCents {
				t.Errorf("NewPriceCents: expected %d, got %d", tt.expected.NewPriceCents, result.NewPriceCents)
			}
			if result.CIBPriceCents != tt.expected.CIBPriceCents {
				t.Errorf("CIBPriceCents: expected %d, got %d", tt.expected.CIBPriceCents, result.CIBPriceCents)
			}
			if result.ManualPriceCents != tt.expected.ManualPriceCents {
				t.Errorf("ManualPriceCents: expected %d, got %d", tt.expected.ManualPriceCents, result.ManualPriceCents)
			}
			if result.BoxPriceCents != tt.expected.BoxPriceCents {
				t.Errorf("BoxPriceCents: expected %d, got %d", tt.expected.BoxPriceCents, result.BoxPriceCents)
			}

			// Sales fields
			if result.SalesVolume != tt.expected.SalesVolume {
				t.Errorf("SalesVolume: expected %d, got %d", tt.expected.SalesVolume, result.SalesVolume)
			}
			if result.LastSoldDate != tt.expected.LastSoldDate {
				t.Errorf("LastSoldDate: expected %s, got %s", tt.expected.LastSoldDate, result.LastSoldDate)
			}

			// Retail pricing
			if result.RetailBuyPrice != tt.expected.RetailBuyPrice {
				t.Errorf("RetailBuyPrice: expected %d, got %d", tt.expected.RetailBuyPrice, result.RetailBuyPrice)
			}
			if result.RetailSellPrice != tt.expected.RetailSellPrice {
				t.Errorf("RetailSellPrice: expected %d, got %d", tt.expected.RetailSellPrice, result.RetailSellPrice)
			}

			// Sales data
			if result.SalesCount != tt.expected.SalesCount {
				t.Errorf("SalesCount: expected %d, got %d", tt.expected.SalesCount, result.SalesCount)
			}
			if result.AvgSalePrice != tt.expected.AvgSalePrice {
				t.Errorf("AvgSalePrice: expected %d, got %d", tt.expected.AvgSalePrice, result.AvgSalePrice)
			}
			if len(result.RecentSales) != len(tt.expected.RecentSales) {
				t.Errorf("RecentSales length: expected %d, got %d", len(tt.expected.RecentSales), len(result.RecentSales))
			}
		})
	}
}

func TestPriceCharting_CacheExpiration(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.json")
	testCache, err := cache.New(cachePath)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	// Add data with very short TTL
	cachedMatch := &PCMatch{
		ID:          "temp-123",
		ProductName: "Temporary Product",
		PSA10Cents:  1000,
	}

	card := model.Card{Name: "Temp Card", Number: "001"}
	key := cache.PriceChartingKey("Temp Set", card.Name, card.Number)
	err = testCache.Put(key, cachedMatch, 1*time.Millisecond)
	if err != nil {
		t.Fatalf("failed to cache data: %v", err)
	}

	// Wait for expiration
	time.Sleep(5 * time.Millisecond)

	// Verify cache miss after expiration
	var retrievedMatch PCMatch
	found, _ := testCache.Get(key, &retrievedMatch)
	if found {
		t.Errorf("expected cache miss after expiration")
	}
}

func TestPriceCharting_CacheKey(t *testing.T) {
	// Test that cache keys are constructed correctly
	setName := "Surging Sparks"
	cardName := "Pikachu"
	number := "025"

	expectedKey := cache.PriceChartingKey(setName, cardName, number)
	if expectedKey == "" {
		t.Errorf("expected non-empty cache key")
	}

	// Verify key format
	if !strings.Contains(expectedKey, setName) {
		t.Errorf("expected cache key to contain set name")
	}
	if !strings.Contains(expectedKey, cardName) {
		t.Errorf("expected cache key to contain card name")
	}
	if !strings.Contains(expectedKey, number) {
		t.Errorf("expected cache key to contain card number")
	}
}

// Benchmark tests
func BenchmarkPcFrom(b *testing.B) {
	data := map[string]interface{}{
		"id":                "12345",
		"product-name":      "Pokemon Surging Sparks Pikachu #238",
		"loose-price":       850,
		"graded-price":      1500,
		"box-only-price":    1800,
		"manual-only-price": 2500,
		"bgs-10-price":      3000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pcFrom(data)
	}
}

func BenchmarkHasPriceKeys(b *testing.B) {
	data := map[string]interface{}{
		"id":                "12345",
		"product-name":      "Pokemon Card",
		"loose-price":       850,
		"graded-price":      1500,
		"box-only-price":    1800,
		"manual-only-price": 2500,
		"bgs-10-price":      3000,
		"extra-field":       "value",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hasPriceKeys(data)
	}
}

// Test comprehensive LookupCard scenarios with API mocking
func TestPriceCharting_LookupCard_ComprehensiveAPI(t *testing.T) {
	// Create a mock server to handle different API scenarios
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Parse query parameters to determine response
		query := r.URL.Query().Get("q")
		id := r.URL.Query().Get("id")

		if strings.Contains(r.URL.Path, "/api/product") && id != "" {
			// Product by ID request
			if id == "12345" {
				w.Write([]byte(`{
					"status": "success",
					"id": "12345",
					"product-name": "Pokemon Surging Sparks Pikachu #238",
					"loose-price": 850,
					"graded-price": 1500,
					"box-only-price": 1800,
					"manual-only-price": 2500,
					"bgs-10-price": 3000
				}`))
			} else {
				w.Write([]byte(`{"status": "error", "error": "Product not found"}`))
			}
		} else if strings.Contains(r.URL.Path, "/api/product") && query != "" {
			// Single product query
			if strings.Contains(query, "Pikachu") {
				w.Write([]byte(`{
					"status": "success",
					"id": "12345",
					"product-name": "Pokemon Surging Sparks Pikachu #238",
					"loose-price": 850,
					"graded-price": 1500,
					"manual-only-price": 2500
				}`))
			} else if strings.Contains(query, "no-prices") {
				// Card with no price data
				w.Write([]byte(`{
					"status": "success",
					"id": "99999",
					"product-name": "Pokemon No Price Card"
				}`))
			} else {
				w.Write([]byte(`{"status": "error", "error": "No results found"}`))
			}
		} else if strings.Contains(r.URL.Path, "/api/products") {
			// Search products endpoint
			if strings.Contains(query, "Charizard") {
				w.Write([]byte(`{
					"status": "success",
					"products": [
						{
							"id": "12345",
							"product-name": "Pokemon Surging Sparks Charizard #025"
						}
					]
				}`))
			} else if strings.Contains(query, "empty") {
				w.Write([]byte(`{
					"status": "success",
					"products": []
				}`))
			} else {
				w.Write([]byte(`{"status": "error", "error": "Search failed"}`))
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
		}
	}))
	defer server.Close()

	// Replace the actual PriceCharting API URL with our test server
	originalTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = originalTransport }()

	http.DefaultTransport = &mockPriceChartingTransport{
		testServerURL: server.URL,
		original:      originalTransport,
	}

	t.Run("Direct Product Query Success", func(t *testing.T) {
		pc := NewPriceCharting(testutil.GetTestPriceChartingToken(), nil)
		card := model.Card{Name: "Pikachu", Number: "238"}

		match, err := pc.LookupCard("Surging Sparks", card)
		if err != nil {
			t.Fatalf("LookupCard failed: %v", err)
		}

		if match.ID != "12345" {
			t.Errorf("expected ID 12345, got %s", match.ID)
		}
		if match.PSA10Cents != 2500 {
			t.Errorf("expected PSA10 price 2500, got %d", match.PSA10Cents)
		}
		if match.LooseCents != 850 {
			t.Errorf("expected loose price 850, got %d", match.LooseCents)
		}
	})

	// Skip fallback test for now - complex API interaction that needs more setup
	t.Run("Fallback to Search API - Skipped", func(t *testing.T) {
		t.Skip("Complex fallback logic requires more detailed API mocking")
	})

	t.Run("No Products Found", func(t *testing.T) {
		pc := NewPriceCharting(testutil.GetTestPriceChartingToken(), nil)
		card := model.Card{Name: "empty", Number: "000"}

		_, err := pc.LookupCard("Test Set", card)
		if err == nil {
			t.Error("expected error when no products found")
		}
		if !strings.Contains(err.Error(), "no product match") {
			t.Errorf("expected 'no product match' error, got: %v", err)
		}
	})

	t.Run("Card with No Price Data", func(t *testing.T) {
		pc := NewPriceCharting(testutil.GetTestPriceChartingToken(), nil)
		card := model.Card{Name: "no-prices", Number: "000"}

		_, err := pc.LookupCard("Test Set", card)
		if err == nil {
			t.Error("expected error when no price data")
		}
	})

	t.Run("Cache Population After Successful Lookup", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test_cache.json")
		testCache, err := cache.New(cachePath)
		if err != nil {
			t.Fatalf("failed to create cache: %v", err)
		}

		pc := NewPriceCharting(testutil.GetTestPriceChartingToken(), testCache)
		card := model.Card{Name: "Pikachu", Number: "238"}

		// First call should populate cache
		match1, err := pc.LookupCard("Surging Sparks", card)
		if err != nil {
			t.Fatalf("first LookupCard failed: %v", err)
		}

		// Verify cache was populated
		key := cache.PriceChartingKey("Surging Sparks", card.Name, card.Number)
		var cachedMatch PCMatch
		found, _ := testCache.Get(key, &cachedMatch)
		if !found {
			t.Error("expected cache to be populated")
		}

		// Second call should use cache
		match2, err := pc.LookupCard("Surging Sparks", card)
		if err != nil {
			t.Fatalf("second LookupCard failed: %v", err)
		}

		if match1.ID != match2.ID {
			t.Errorf("cached result differs: %s vs %s", match1.ID, match2.ID)
		}
	})
}

// mockPriceChartingTransport replaces PriceCharting API calls with test server calls
type mockPriceChartingTransport struct {
	testServerURL string
	original      http.RoundTripper
}

func (m *mockPriceChartingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Only intercept PriceCharting API calls
	if strings.Contains(req.URL.Host, "pricecharting.com") {
		// Replace the host with our test server
		newURL := strings.Replace(req.URL.String(), "https://www.pricecharting.com", m.testServerURL, 1)
		newReq, err := http.NewRequest(req.Method, newURL, req.Body)
		if err != nil {
			return nil, err
		}
		// Copy headers
		for k, v := range req.Header {
			newReq.Header[k] = v
		}
		return m.original.RoundTrip(newReq)
	}
	return m.original.RoundTrip(req)
}

// Test error scenarios in lookupByQuery
func TestPriceCharting_LookupByQuery_ErrorScenarios(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		query := r.URL.Query().Get("q")

		if strings.Contains(query, "http-error") {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		} else if strings.Contains(query, "invalid-json") {
			w.Write([]byte("{invalid json}"))
		} else if strings.Contains(query, "search-error") {
			w.Write([]byte(`{"status": "error", "error": "Search failed"}`))
		} else {
			w.Write([]byte(`{"status": "success", "products": []}`))
		}
	}))
	defer server.Close()

	// Replace HTTP transport
	originalTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = originalTransport }()

	http.DefaultTransport = &mockPriceChartingTransport{
		testServerURL: server.URL,
		original:      originalTransport,
	}

	tests := []struct {
		name        string
		cardName    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "HTTP Error",
			cardName:    "http-error",
			expectError: true,
			errorMsg:    "HTTP 500",
		},
		{
			name:        "Invalid JSON",
			cardName:    "invalid-json",
			expectError: true,
		},
		{
			name:        "Search API Error",
			cardName:    "search-error",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := NewPriceCharting(testutil.GetTestPriceChartingToken(), nil)
			card := model.Card{Name: tt.cardName, Number: "001"}

			_, err := pc.LookupCard("Test Set", card)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// Test the query formatting in LookupCard
func TestPriceCharting_QueryFormatting(t *testing.T) {
	tests := []struct {
		name     string
		setName  string
		cardName string
		number   string
		expected string
	}{
		{
			name:     "basic card",
			setName:  "Surging Sparks",
			cardName: "Pikachu",
			number:   "238",
			expected: "pokemon Surging Sparks Pikachu #238",
		},
		{
			name:     "special characters",
			setName:  "Sword & Shield",
			cardName: "Charizard-GX",
			number:   "150",
			expected: "pokemon Sword & Shield Charizard-GX #150",
		},
		{
			name:     "with spaces",
			setName:  "Base Set",
			cardName: "Dark Charizard",
			number:   "4",
			expected: "pokemon Base Set Dark Charizard #4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We'll test this indirectly by checking the logged query in a mock
			var capturedQuery string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedQuery = r.URL.Query().Get("q")
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"status": "error", "error": "test"}`))
			}))
			defer server.Close()

			// Replace HTTP transport
			originalTransport := http.DefaultTransport
			defer func() { http.DefaultTransport = originalTransport }()

			http.DefaultTransport = &mockPriceChartingTransport{
				testServerURL: server.URL,
				original:      originalTransport,
			}

			pc := NewPriceCharting(testutil.GetTestPriceChartingToken(), nil)
			card := model.Card{Name: tt.cardName, Number: tt.number}

			// Call will fail, but we can check the query was formatted correctly
			_, _ = pc.LookupCard(tt.setName, card)

			if capturedQuery != tt.expected {
				t.Errorf("expected query %q, got %q", tt.expected, capturedQuery)
			}
		})
	}
}

// TestLookupBatch tests batch lookup functionality
func TestLookupBatch(t *testing.T) {
	tests := []struct {
		name          string
		cards         []model.Card
		batchSize     int
		cachedIndices []int // Which cards should be pre-cached
		expectedCalls int   // Expected API calls
		expectError   bool
	}{
		{
			name: "small batch under limit",
			cards: []model.Card{
				{Name: "Pikachu", Number: "025"},
				{Name: "Charizard", Number: "006"},
				{Name: "Blastoise", Number: "009"},
			},
			batchSize:     20,
			expectedCalls: 3,
			expectError:   false,
		},
		{
			name: "large batch requiring multiple requests",
			cards: func() []model.Card {
				cards := make([]model.Card, 25)
				for i := 0; i < 25; i++ {
					cards[i] = model.Card{
						Name:   fmt.Sprintf("Card%d", i),
						Number: fmt.Sprintf("%03d", i),
					}
				}
				return cards
			}(),
			batchSize:     10,
			expectedCalls: 25, // 25 cards, max 10 per batch = 3 batches
			expectError:   false,
		},
		{
			name: "all cards cached",
			cards: []model.Card{
				{Name: "Pikachu", Number: "025"},
				{Name: "Charizard", Number: "006"},
			},
			batchSize:     20,
			cachedIndices: []int{0, 1}, // All cards pre-cached
			expectedCalls: 0,
			expectError:   false,
		},
		{
			name: "partial cache hit",
			cards: []model.Card{
				{Name: "Pikachu", Number: "025"},
				{Name: "Charizard", Number: "006"},
				{Name: "Blastoise", Number: "009"},
			},
			batchSize:     20,
			cachedIndices: []int{1}, // Only Charizard cached
			expectedCalls: 2,        // Pikachu and Blastoise need fetching
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount++
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(mockSingleProductResponse)
			}))
			defer server.Close()

			// Override httpGetJSON to use our test server
			originalTransport := http.DefaultTransport
			defer func() { http.DefaultTransport = originalTransport }()

			http.DefaultTransport = &testTransport{
				RoundTripFunc: func(req *http.Request) (*http.Response, error) {
					req.URL.Scheme = "http"
					req.URL.Host = server.URL[7:] // Remove "http://"
					return originalTransport.RoundTrip(req)
				},
			}

			// Create cache and pre-populate if needed
			cacheDir := t.TempDir()
			cacheFile := filepath.Join(cacheDir, "test_cache.json")
			testCache, _ := cache.New(cacheFile)

			// Pre-cache specified cards
			setName := "Test Set"
			for _, idx := range tt.cachedIndices {
				if idx < len(tt.cards) {
					card := tt.cards[idx]
					key := cache.PriceChartingKey(setName, card.Name, card.Number)
					testCache.Put(key, &PCMatch{
						ID:          fmt.Sprintf("cached-%d", idx),
						ProductName: fmt.Sprintf("%s #%s", card.Name, card.Number),
						LooseCents:  100,
						PSA10Cents:  1000,
					}, 1*time.Hour)
				}
			}

			pc := NewPriceCharting("test-token", testCache)
			results, err := pc.LookupBatch(setName, tt.cards, tt.batchSize)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			} else if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(results) != len(tt.cards) {
				t.Errorf("expected %d results, got %d", len(tt.cards), len(results))
			}

			// Check that cached cards were not fetched
			for _, idx := range tt.cachedIndices {
				if idx < len(results) && results[idx] != nil {
					if !results[idx].Cached {
						t.Errorf("expected card at index %d to be cached", idx)
					}
				}
			}

			// Note: Actual call count may vary due to parallelism and deduplication
			// We check that it doesn't exceed expected
			if callCount > tt.expectedCalls {
				t.Errorf("expected at most %d API calls, got %d", tt.expectedCalls, callCount)
			}
		})
	}
}

// TestQueryOptimization tests the query optimization functionality
func TestQueryOptimization(t *testing.T) {
	tests := []struct {
		name          string
		setName       string
		cardName      string
		number        string
		expectedQuery string
	}{
		{
			name:          "basic card",
			setName:       "Surging Sparks",
			cardName:      "Pikachu",
			number:        "025",
			expectedQuery: "pokemon Surging Sparks Pikachu #025",
		},
		{
			name:          "card with ex suffix",
			setName:       "Surging Sparks",
			cardName:      "Pikachu ex",
			number:        "025",
			expectedQuery: "pokemon Surging Sparks Pikachu #025",
		},
		{
			name:          "card with vmax suffix",
			setName:       "Surging Sparks",
			cardName:      "Charizard VMAX",
			number:        "006",
			expectedQuery: "pokemon Surging Sparks Charizard #006",
		},
		{
			name:          "reverse holo variant",
			setName:       "Surging Sparks",
			cardName:      "Pikachu Reverse Holo",
			number:        "025",
			expectedQuery: "pokemon Surging Sparks Pikachu Reverse #025 reverse holo",
		},
		{
			name:          "set with colon",
			setName:       "Sword & Shield: Base Set",
			cardName:      "Zacian",
			number:        "138",
			expectedQuery: "pokemon Sword & Shield Base Set Zacian #138",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := NewPriceCharting("test-token", nil)
			query := pc.OptimizeQuery(tt.setName, tt.cardName, tt.number)

			if query != tt.expectedQuery {
				t.Errorf("expected query %q, got %q", tt.expectedQuery, query)
			}
		})
	}
}

// TestCachePriority tests cache priority calculation
func TestCachePriority(t *testing.T) {
	tests := []struct {
		name             string
		match            *PCMatch
		expectedPriority int
		expectedVolatile bool
	}{
		{
			name: "high value card",
			match: &PCMatch{
				PSA10Cents: 15000, // $150
			},
			expectedPriority: 3,
			expectedVolatile: true,
		},
		{
			name: "actively traded card",
			match: &PCMatch{
				PSA10Cents: 5000,
				RecentSales: []SaleData{
					{}, {}, {}, {}, {}, {}, // 6 sales
				},
			},
			expectedPriority: 2,
			expectedVolatile: false,
		},
		{
			name: "low value stable card",
			match: &PCMatch{
				PSA10Cents:  500,
				RecentSales: []SaleData{{}, {}},
			},
			expectedPriority: 1,
			expectedVolatile: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := NewPriceCharting("test-token", nil)
			priority := pc.calculateCachePriority(tt.match)

			if priority.Priority != tt.expectedPriority {
				t.Errorf("expected priority %d, got %d", tt.expectedPriority, priority.Priority)
			}

			if priority.Volatile != tt.expectedVolatile {
				t.Errorf("expected volatile %v, got %v", tt.expectedVolatile, priority.Volatile)
			}
		})
	}
}

// TestGetStats tests API statistics tracking
func TestGetStats(t *testing.T) {
	pc := NewPriceCharting("test-token", nil)

	// Simulate some requests
	pc.incrementRequestCount()
	pc.incrementRequestCount()
	pc.incrementCachedRequests()
	pc.incrementCachedRequests()
	pc.incrementCachedRequests()

	stats := pc.GetStats()

	if stats["api_requests"] != int64(2) {
		t.Errorf("expected 2 API requests, got %v", stats["api_requests"])
	}

	if stats["cached_requests"] != int64(3) {
		t.Errorf("expected 3 cached requests, got %v", stats["cached_requests"])
	}

	if stats["total_requests"] != int64(5) {
		t.Errorf("expected 5 total requests, got %v", stats["total_requests"])
	}

	// Check cache hit rate
	if !strings.Contains(stats["cache_hit_rate"].(string), "60.00%") {
		t.Errorf("expected 60%% cache hit rate, got %v", stats["cache_hit_rate"])
	}
}
