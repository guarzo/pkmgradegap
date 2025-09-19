package gamestop

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	// "github.com/guarzo/pkmgradegap/internal/fusion" // TODO: Update when fusion package is refactored
	"github.com/guarzo/pkmgradegap/internal/model"
)

// Tests for GameStop web scraper client
// Note: These tests require actual web scraping and may be flaky
// if GameStop's website structure changes

func TestGameStopClientCreation(t *testing.T) {
	config := DefaultConfig()
	client := NewGameStopClient(config)

	if !client.Available() {
		t.Error("GameStop client should be available")
	}

	if client.GetProviderName() != "GameStop" {
		t.Errorf("Expected provider name 'GameStop', got '%s'", client.GetProviderName())
	}
}

func TestConvertToPriceData(t *testing.T) {
	// Create test listing data
	listingData := &ListingData{
		Card: model.Card{Name: "Charizard", Number: "123"},
		ActiveList: []Listing{
			{
				Price:     150.00,
				Grade:     "PSA 10",
				Title:     "Pokemon Charizard PSA 10 Gem Mint",
				InStock:   true,
				SKU:       "TEST001",
				ImageURL:  "test.jpg",
				Condition: "New",
				Seller:    "GameStop",
			},
			{
				Price:   120.00,
				Grade:   "BGS 9.5",
				Title:   "Pokemon Charizard BGS 9.5",
				InStock: true,
				SKU:     "TEST002",
				Seller:  "GameStop",
			},
			{
				Price:   50.00,
				Grade:   "Raw",
				Title:   "Pokemon Charizard Ungraded",
				InStock: false, // Should be filtered out
				Seller:  "GameStop",
			},
		},
		ListingCount: 3,
		LastUpdated:  time.Now(),
		DataSource:   "GameStop",
	}

	priceData := ConvertToPriceData(listingData)

	// Should only have 2 items (filtered out the out-of-stock one)
	if len(priceData) != 2 {
		t.Errorf("Expected 2 price data items, got %d", len(priceData))
	}

	// TODO: Update when fusion package is refactored
	for _, dataInterface := range priceData {
		if data, ok := dataInterface.(map[string]interface{}); ok {
			if value, ok := data["value"].(float64); !ok || value <= 0 {
				t.Error("Price data value should be positive")
			}
			if currency, ok := data["currency"].(string); !ok || currency != "USD" {
				t.Errorf("Expected currency 'USD', got '%v'", data["currency"])
			}
			if source, ok := data["source"].(string); !ok || source != "GameStop" {
				t.Errorf("Expected source 'GameStop', got '%v'", data["source"])
			}
		}
		// if data.Source.Type != fusion.SourceTypeListing {
		// 	t.Errorf("Expected source type 'LISTING', got '%s'", data.Source.Type)
		// }
		// if data.Source.Name != "GameStop" {
		// 	t.Errorf("Expected source name 'GameStop', got '%s'", data.Source.Name)
		// }
	}
}

func TestConvertToPriceDataByGrade(t *testing.T) {
	listingData := &ListingData{
		Card: model.Card{Name: "Charizard", Number: "123"},
		ActiveList: []Listing{
			{Price: 150.00, Grade: "PSA 10", Title: "Pokemon Charizard PSA 10", InStock: true, Seller: "GameStop"},
			{Price: 155.00, Grade: "PSA 10", Title: "Pokemon Charizard PSA 10 Gem", InStock: true, Seller: "GameStop"},
			{Price: 120.00, Grade: "BGS 9.5", Title: "Pokemon Charizard BGS 9.5", InStock: true, Seller: "GameStop"},
			{Price: 45.00, Grade: "Raw", Title: "Pokemon Charizard Ungraded", InStock: true, Seller: "GameStop"},
		},
		ListingCount: 4,
		LastUpdated:  time.Now(),
		DataSource:   "GameStop",
	}

	gradeData := ConvertToPriceDataByGrade(listingData)

	// Should have grouped by grades
	if len(gradeData["psa10"]) != 2 {
		t.Errorf("Expected 2 PSA 10 items, got %d", len(gradeData["psa10"]))
	}
	if len(gradeData["cgc95"]) != 1 {
		t.Errorf("Expected 1 BGS 9.5 item, got %d", len(gradeData["cgc95"]))
	}
	if len(gradeData["raw"]) != 1 {
		t.Errorf("Expected 1 raw item, got %d", len(gradeData["raw"]))
	}
}

func TestGetLowestPriceByGrade(t *testing.T) {
	listingData := &ListingData{
		ActiveList: []Listing{
			{Price: 150.00, Grade: "PSA 10", Title: "Pokemon Charizard PSA 10", InStock: true, Seller: "GameStop"},
			{Price: 155.00, Grade: "PSA 10", Title: "Pokemon Charizard PSA 10 Gem", InStock: true, Seller: "GameStop"},
			{Price: 120.00, Grade: "BGS 9.5", Title: "Pokemon Charizard BGS 9.5", InStock: true, Seller: "GameStop"},
		},
		ListingCount: 3,
		LastUpdated:  time.Now(),
	}

	prices := GetLowestPriceByGrade(listingData)

	if prices["psa10"] != 150.00 {
		t.Errorf("Expected PSA 10 lowest price 150.00, got %.2f", prices["psa10"])
	}
	if prices["cgc95"] != 120.00 {
		t.Errorf("Expected BGS 9.5 lowest price 120.00, got %.2f", prices["cgc95"])
	}
}

func TestIsRawCard(t *testing.T) {
	tests := []struct {
		grade    string
		title    string
		expected bool
	}{
		{"PSA 10", "Pokemon Charizard PSA 10", false},
		{"BGS 9.5", "Pokemon Charizard BGS 9.5", false},
		{"Raw", "Pokemon Charizard Raw", true},
		{"Unknown", "Pokemon Charizard", true},
		{"", "Pokemon Charizard PSA 10", false},
		{"", "Pokemon Charizard Ungraded", true},
		{"NM", "Pokemon Charizard Near Mint", true},
		{"Graded", "Pokemon Charizard Graded", false},
	}

	for _, test := range tests {
		result := isRawCard(test.grade, test.title)
		if result != test.expected {
			t.Errorf("isRawCard(%q, %q) = %v, expected %v", test.grade, test.title, result, test.expected)
		}
	}
}

func TestNormalizeGrade(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"PSA 10", "PSA 10"},
		{"psa 10", "PSA 10"},
		{"PSA10", "PSA 10"},
		{"BGS 9.5", "BGS 9.5"},
		{"bgs 9.5", "BGS 9.5"},
		{"CGC 9", "CGC 9"},
		{"Unknown", "Unknown"},
		{"", "Unknown"},
		{"RAW", "RAW"},
	}

	for _, test := range tests {
		result := normalizeGrade(test.input)
		if result != test.expected {
			t.Errorf("normalizeGrade(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestGetGradeKey(t *testing.T) {
	tests := []struct {
		grade    string
		title    string
		expected string
	}{
		{"PSA 10", "Pokemon Charizard PSA 10", "psa10"},
		{"BGS 10", "Pokemon Charizard BGS 10", "bgs10"},
		{"PSA 9", "Pokemon Charizard PSA 9", "psa9"},
		{"BGS 9.5", "Pokemon Charizard BGS 9.5", "cgc95"},
		{"CGC 9.5", "Pokemon Charizard CGC 9.5", "cgc95"},
		{"Raw", "Pokemon Charizard Raw", "raw"},
		{"Unknown", "Pokemon Charizard", "raw"},
		{"PSA 8", "Pokemon Charizard PSA 8", "other"},
	}

	for _, test := range tests {
		result := getGradeKey(test.grade, test.title)
		if result != test.expected {
			t.Errorf("getGradeKey(%q, %q) = %q, expected %q", test.grade, test.title, result, test.expected)
		}
	}
}

func TestCalculateListingConfidence(t *testing.T) {
	// High quality listing
	highQuality := Listing{
		SKU:         "TEST001",
		ImageURL:    "test.jpg",
		Description: "High quality card",
		Grade:       "PSA 10",
		Title:       "Pokemon Charizard PSA 10 Gem Mint Condition",
		InStock:     true,
	}

	confidence := calculateListingConfidence(highQuality)
	if confidence <= 0.5 {
		t.Errorf("High quality listing should have confidence > 0.5, got %.2f", confidence)
	}

	// Low quality listing
	lowQuality := Listing{
		Grade:   "Unknown",
		Title:   "Card",
		InStock: false,
	}

	confidence = calculateListingConfidence(lowQuality)
	if confidence >= 0.5 {
		t.Errorf("Low quality listing should have confidence < 0.5, got %.2f", confidence)
	}
}

func TestGetReader(t *testing.T) {
	client := NewGameStopClient(DefaultConfig())

	testData := "Hello, World! This is test data for compression testing."

	tests := []struct {
		name        string
		encoding    string
		setupBody   func() io.ReadCloser
		expectError bool
	}{
		{
			name:     "no compression",
			encoding: "",
			setupBody: func() io.ReadCloser {
				return io.NopCloser(strings.NewReader(testData))
			},
			expectError: false,
		},
		{
			name:     "gzip compression",
			encoding: "gzip",
			setupBody: func() io.ReadCloser {
				var buf bytes.Buffer
				gzipWriter := gzip.NewWriter(&buf)
				_, _ = gzipWriter.Write([]byte(testData))
				_ = gzipWriter.Close()
				return io.NopCloser(&buf)
			},
			expectError: false,
		},
		{
			name:     "deflate compression",
			encoding: "deflate",
			setupBody: func() io.ReadCloser {
				var buf bytes.Buffer
				deflateWriter, _ := flate.NewWriter(&buf, flate.DefaultCompression)
				_, _ = deflateWriter.Write([]byte(testData))
				_ = deflateWriter.Close()
				return io.NopCloser(&buf)
			},
			expectError: false,
		},
		{
			name:     "brotli compression",
			encoding: "br",
			setupBody: func() io.ReadCloser {
				var buf bytes.Buffer
				brotliWriter := brotli.NewWriter(&buf)
				_, _ = brotliWriter.Write([]byte(testData))
				_ = brotliWriter.Close()
				return io.NopCloser(&buf)
			},
			expectError: false,
		},
		{
			name:     "unknown compression",
			encoding: "unknown",
			setupBody: func() io.ReadCloser {
				return io.NopCloser(strings.NewReader(testData))
			},
			expectError: false, // Should fallback to raw body
		},
		{
			name:     "malformed gzip",
			encoding: "gzip",
			setupBody: func() io.ReadCloser {
				return io.NopCloser(strings.NewReader("invalid gzip data"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				Header: make(http.Header),
				Body:   tt.setupBody(),
			}

			if tt.encoding != "" {
				resp.Header.Set("Content-Encoding", tt.encoding)
			}

			reader, err := client.getReader(resp)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
				return
			}

			if reader == nil {
				t.Errorf("Expected reader for %s, but got nil", tt.name)
				return
			}

			// For non-error cases, try to read the data
			if !tt.expectError && tt.encoding != "unknown" {
				data, readErr := io.ReadAll(reader)
				if readErr != nil {
					t.Errorf("Failed to read from reader for %s: %v", tt.name, readErr)
					return
				}

				if string(data) != testData {
					t.Errorf("Data mismatch for %s: expected %q, got %q", tt.name, testData, string(data))
				}
			}
		})
	}
}
