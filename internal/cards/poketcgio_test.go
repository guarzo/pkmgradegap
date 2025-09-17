package cards

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

// Simple comprehensive tests that focus on achieving high coverage
// by testing the actual public methods through dependency injection

// TestPokeTCGIO_ComprehensiveListSets tests the ListSets method comprehensively
func TestPokeTCGIO_ComprehensiveListSets(t *testing.T) {
	// Test with cache hit
	t.Run("CacheHit", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test_cache.json")
		testCache, err := cache.New(cachePath)
		if err != nil {
			t.Fatalf("failed to create cache: %v", err)
		}

		// Pre-populate cache
		testSets := []model.Set{
			{ID: "cached1", Name: "Cached Set 1", ReleaseDate: "2024/01/01"},
		}
		err = testCache.Put(cache.SetsKey(), testSets, 1*time.Hour)
		if err != nil {
			t.Fatalf("failed to populate cache: %v", err)
		}

		p := &PokeTCGIO{
			cache:  testCache,
			client: &http.Client{Timeout: 5 * time.Second},
		}

		sets, err := p.ListSets()
		if err != nil {
			t.Fatalf("ListSets with cache failed: %v", err)
		}

		if len(sets) != 1 {
			t.Errorf("expected 1 cached set, got %d", len(sets))
		}
	})

	// Test with cache miss (will fail API call but we can test the cache miss path)
	t.Run("CacheMiss", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test_cache.json")
		testCache, err := cache.New(cachePath)
		if err != nil {
			t.Fatalf("failed to create cache: %v", err)
		}

		p := &PokeTCGIO{
			cache:  testCache,
			client: &http.Client{Timeout: 1 * time.Millisecond}, // Very short timeout to fail quickly
		}

		// This will fail due to network error, but it tests the cache miss path
		_, err = p.ListSets()
		if err == nil {
			t.Error("expected error due to network failure")
		}
	})

	// Test without cache
	t.Run("NoCache", func(t *testing.T) {
		p := &PokeTCGIO{
			cache:  nil,                                         // No cache
			client: &http.Client{Timeout: 1 * time.Millisecond}, // Very short timeout
		}

		// This will fail but tests the no-cache path
		_, err := p.ListSets()
		if err == nil {
			t.Error("expected error due to network failure")
		}
	})
}

// TestPokeTCGIO_ComprehensiveCardsBySetID tests the CardsBySetID method comprehensively
func TestPokeTCGIO_ComprehensiveCardsBySetID(t *testing.T) {
	// Test with cache hit
	t.Run("CacheHit", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test_cache.json")
		testCache, err := cache.New(cachePath)
		if err != nil {
			t.Fatalf("failed to create cache: %v", err)
		}

		// Pre-populate cache
		testCards := []model.Card{
			{ID: "cached-1", Name: "Cached Card 1", SetID: "sv1", SetName: "Test Set"},
		}
		err = testCache.Put(cache.CardsKey("sv1"), testCards, 1*time.Hour)
		if err != nil {
			t.Fatalf("failed to populate cache: %v", err)
		}

		p := &PokeTCGIO{
			cache:  testCache,
			client: &http.Client{Timeout: 5 * time.Second},
		}

		cards, err := p.CardsBySetID("sv1")
		if err != nil {
			t.Fatalf("CardsBySetID with cache failed: %v", err)
		}

		if len(cards) != 1 {
			t.Errorf("expected 1 cached card, got %d", len(cards))
		}
	})

	// Test with cache miss
	t.Run("CacheMiss", func(t *testing.T) {
		tmpDir := t.TempDir()
		cachePath := filepath.Join(tmpDir, "test_cache.json")
		testCache, err := cache.New(cachePath)
		if err != nil {
			t.Fatalf("failed to create cache: %v", err)
		}

		p := &PokeTCGIO{
			cache:  testCache,
			client: &http.Client{Timeout: 1 * time.Millisecond}, // Very short timeout
		}

		// This will fail due to network error, but it tests the cache miss path
		_, err = p.CardsBySetID("sv1")
		if err == nil {
			t.Error("expected error due to network failure")
		}
	})

	// Test without cache
	t.Run("NoCache", func(t *testing.T) {
		p := &PokeTCGIO{
			cache:  nil,                                         // No cache
			client: &http.Client{Timeout: 1 * time.Millisecond}, // Very short timeout
		}

		// This will fail but tests the no-cache path
		_, err := p.CardsBySetID("sv1")
		if err == nil {
			t.Error("expected error due to network failure")
		}
	})
}

// Test the constructor function
func TestNewPokeTCGIO_Comprehensive(t *testing.T) {
	tests := []struct {
		name         string
		apiKey       string
		cache        *cache.Cache
		expectNilKey bool
	}{
		{
			name:         "with API key and cache",
			apiKey:       "test-key",
			cache:        createTestCache(t),
			expectNilKey: false,
		},
		{
			name:         "with API key no cache",
			apiKey:       "test-key",
			cache:        nil,
			expectNilKey: false,
		},
		{
			name:         "no API key with cache",
			apiKey:       "",
			cache:        createTestCache(t),
			expectNilKey: true,
		},
		{
			name:         "no API key no cache",
			apiKey:       "",
			cache:        nil,
			expectNilKey: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPokeTCGIO(tt.apiKey, tt.cache)

			if p == nil {
				t.Fatal("NewPokeTCGIO returned nil")
			}

			if tt.expectNilKey && p.apiKey != "" {
				t.Errorf("expected empty API key, got %s", p.apiKey)
			} else if !tt.expectNilKey && p.apiKey != tt.apiKey {
				t.Errorf("expected API key %s, got %s", tt.apiKey, p.apiKey)
			}

			if (tt.cache == nil) != (p.cache == nil) {
				t.Errorf("cache mismatch: expected nil=%v, got nil=%v", tt.cache == nil, p.cache == nil)
			}

			if p.client == nil {
				t.Error("expected non-nil HTTP client")
			}

			if p.client.Timeout != 30*time.Second {
				t.Errorf("expected 30s timeout, got %v", p.client.Timeout)
			}
		})
	}
}

func createTestCache(t *testing.T) *cache.Cache {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "test_cache.json")
	testCache, err := cache.New(cachePath)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	return testCache
}

// Test error scenarios and edge cases thoroughly
func TestPokeTCGIO_ErrorScenarios(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectError    bool
		errorContains  string
	}{
		{
			name: "404 Not Found",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("not found"))
			},
			expectError:   true,
			errorContains: "404",
		},
		{
			name: "500 Internal Server Error",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("server error"))
			},
			expectError:   true,
			errorContains: "500",
		},
		{
			name: "502 Bad Gateway with retry",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadGateway)
				w.Write([]byte("bad gateway"))
			},
			expectError:   true,
			errorContains: "502",
		},
		{
			name: "503 Service Unavailable with retry",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("service unavailable"))
			},
			expectError:   true,
			errorContains: "503",
		},
		{
			name: "504 Gateway Timeout with retry",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusGatewayTimeout)
				w.Write([]byte("gateway timeout"))
			},
			expectError:   true,
			errorContains: "504",
		},
		{
			name: "429 Rate Limited",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("rate limited"))
			},
			expectError:   true,
			errorContains: "429",
		},
		{
			name: "Invalid JSON response",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("{invalid json}"))
			},
			expectError:   true,
			errorContains: "invalid",
		},
		{
			name: "Empty response body",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(""))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			p := &PokeTCGIO{
				client: &http.Client{Timeout: 5 * time.Second},
			}

			var result map[string]interface{}
			err := p.get(server.URL, &result)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// Test successful retry scenarios
func TestPokeTCGIO_RetrySuccess(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": ["success"]}`))
	}))
	defer server.Close()

	p := &PokeTCGIO{
		client: &http.Client{Timeout: 10 * time.Second},
	}

	var result map[string]interface{}
	err := p.get(server.URL, &result)
	if err != nil {
		t.Fatalf("unexpected error after retries: %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}

	if data, ok := result["data"].([]interface{}); !ok || len(data) != 1 {
		t.Errorf("unexpected result: %v", result)
	}
}

// Test rate limit handling
func TestPokeTCGIO_RateLimitHandling(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": []}`))
	}))
	defer server.Close()

	p := &PokeTCGIO{
		client: &http.Client{Timeout: 10 * time.Second},
	}

	var result map[string]interface{}
	err := p.get(server.URL, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if attempts != 2 {
		t.Errorf("expected 2 attempts after rate limit, got %d", attempts)
	}
}

// Test request headers
func TestPokeTCGIO_RequestHeaders(t *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": []}`))
	}))
	defer server.Close()

	tests := []struct {
		name     string
		apiKey   string
		expected map[string]string
	}{
		{
			name:   "with API key",
			apiKey: "test-key-123",
			expected: map[string]string{
				"X-Api-Key":  "test-key-123",
				"Accept":     "application/json",
				"User-Agent": "pkmgradegap/1.0",
			},
		},
		{
			name:   "without API key",
			apiKey: "",
			expected: map[string]string{
				"Accept":     "application/json",
				"User-Agent": "pkmgradegap/1.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PokeTCGIO{
				apiKey: tt.apiKey,
				client: &http.Client{Timeout: 5 * time.Second},
			}

			var result map[string]interface{}
			err := p.get(server.URL, &result)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}

			for header, expectedValue := range tt.expected {
				actualValue := receivedHeaders.Get(header)
				if actualValue != expectedValue {
					t.Errorf("header %s: expected %q, got %q", header, expectedValue, actualValue)
				}
			}

			// If no API key, ensure X-Api-Key header is not set
			if tt.apiKey == "" {
				if receivedHeaders.Get("X-Api-Key") != "" {
					t.Errorf("expected no X-Api-Key header, but got %s", receivedHeaders.Get("X-Api-Key"))
				}
			}
		})
	}
}

// Test timeout scenarios
func TestPokeTCGIO_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Sleep longer than client timeout
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": []}`))
	}))
	defer server.Close()

	p := &PokeTCGIO{
		client: &http.Client{Timeout: 500 * time.Millisecond}, // Short timeout
	}

	var result map[string]interface{}
	err := p.get(server.URL, &result)
	if err == nil {
		t.Error("expected timeout error but got nil")
	}

	// Check that it's a timeout-related error
	if !strings.Contains(err.Error(), "timeout") &&
		!strings.Contains(err.Error(), "context deadline exceeded") &&
		!strings.Contains(err.Error(), "Client.Timeout") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

// Test cache key functions
func TestCacheKeys(t *testing.T) {
	tests := []struct {
		name     string
		keyFunc  func() string
		expected string
	}{
		{
			name:     "sets key",
			keyFunc:  cache.SetsKey,
			expected: "sets:v2",
		},
		{
			name:     "cards key for sv7",
			keyFunc:  func() string { return cache.CardsKey("sv7") },
			expected: "cards|set|sv7",
		},
		{
			name:     "cards key for base-set",
			keyFunc:  func() string { return cache.CardsKey("base-set") },
			expected: "cards|set|base-set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.keyFunc()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
