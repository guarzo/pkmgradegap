package prices

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/guarzo/pkmgradegap/internal/cache"
	"github.com/guarzo/pkmgradegap/internal/model"
)

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

func TestNewPriceCharting(t *testing.T) {
	// Test with token
	pc1 := NewPriceCharting("test-token", nil)
	if pc1.token != "test-token" {
		t.Errorf("expected token test-token, got %s", pc1.token)
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

	pc := NewPriceCharting("test-token", testCache)

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
			name: "complete data",
			data: map[string]interface{}{
				"id":                "12345",
				"product-name":      "Pokemon Card",
				"loose-price":       850,
				"graded-price":      1500,
				"box-only-price":    1800,
				"manual-only-price": 2500,
				"bgs-10-price":      3000,
			},
			expected: &PCMatch{
				ID:           "12345",
				ProductName:  "Pokemon Card",
				LooseCents:   850,
				Grade9Cents:  1500,
				Grade95Cents: 1800,
				PSA10Cents:   2500,
				BGS10Cents:   3000,
			},
		},
		{
			name: "partial data with float64",
			data: map[string]interface{}{
				"id":                "67890",
				"product-name":      "Another Card",
				"loose-price":       12.5, // float64
				"manual-only-price": 25.0, // float64
			},
			expected: &PCMatch{
				ID:           "67890",
				ProductName:  "Another Card",
				LooseCents:   12,
				Grade9Cents:  0,
				Grade95Cents: 0,
				PSA10Cents:   25,
				BGS10Cents:   0,
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
				"id":           "invalid",
				"product-name": "Invalid Prices",
				"loose-price":  "not-a-number",
				"graded-price": true,
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

			if result.ID != tt.expected.ID {
				t.Errorf("ID: expected %s, got %s", tt.expected.ID, result.ID)
			}
			if result.ProductName != tt.expected.ProductName {
				t.Errorf("ProductName: expected %s, got %s", tt.expected.ProductName, result.ProductName)
			}
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
		pc := NewPriceCharting("test-token", nil)
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
		pc := NewPriceCharting("test-token", nil)
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
		pc := NewPriceCharting("test-token", nil)
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

		pc := NewPriceCharting("test-token", testCache)
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
			pc := NewPriceCharting("test-token", nil)
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

			pc := NewPriceCharting("test-token", nil)
			card := model.Card{Name: tt.cardName, Number: tt.number}

			// Call will fail, but we can check the query was formatted correctly
			_, _ = pc.LookupCard(tt.setName, card)

			if capturedQuery != tt.expected {
				t.Errorf("expected query %q, got %q", tt.expected, capturedQuery)
			}
		})
	}
}
