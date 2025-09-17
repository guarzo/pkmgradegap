package gamestop

import (
	"testing"
	"time"

	"github.com/guarzo/pkmgradegap/internal/fusion"
	"github.com/guarzo/pkmgradegap/internal/model"
)

func TestMockProvider(t *testing.T) {
	provider := NewMockProvider()

	if !provider.Available() {
		t.Error("Mock provider should always be available")
	}

	if provider.GetProviderName() != "GameStopMock" {
		t.Errorf("Expected provider name 'GameStopMock', got '%s'", provider.GetProviderName())
	}
}

func TestMockProviderGetListings(t *testing.T) {
	provider := NewMockProvider()

	// Test with popular card
	data, err := provider.GetListings("Surging Sparks", "Charizard", "123")
	if err != nil {
		t.Fatalf("GetListings failed: %v", err)
	}

	if data.DataSource != "GameStopMock" {
		t.Errorf("Expected data source 'GameStopMock', got '%s'", data.DataSource)
	}

	if len(data.ActiveList) == 0 {
		t.Error("Expected some listings for Charizard")
	}

	// Verify listing structure
	for _, listing := range data.ActiveList {
		if listing.Price <= 0 {
			t.Error("Listing price should be positive")
		}
		if listing.Grade == "" {
			t.Error("Listing should have a grade")
		}
		if listing.Seller != "GameStop" {
			t.Errorf("Expected seller 'GameStop', got '%s'", listing.Seller)
		}
	}

	// Test with unknown card
	data2, err := provider.GetListings("Unknown Set", "UnknownCard", "999")
	if err != nil {
		t.Fatalf("GetListings failed for unknown card: %v", err)
	}

	if len(data2.ActiveList) != 1 {
		t.Errorf("Expected 1 listing for unknown card, got %d", len(data2.ActiveList))
	}
}

func TestMockProviderSearchCards(t *testing.T) {
	provider := NewMockProvider()

	listings, err := provider.SearchCards("Charizard PSA 10")
	if err != nil {
		t.Fatalf("SearchCards failed: %v", err)
	}

	if len(listings) != 2 {
		t.Errorf("Expected 2 search results, got %d", len(listings))
	}

	// Verify search results structure
	for _, listing := range listings {
		if listing.Price <= 0 {
			t.Error("Search result price should be positive")
		}
		if listing.Title == "" {
			t.Error("Search result should have a title")
		}
	}
}

func TestMockProviderGetBulkListings(t *testing.T) {
	provider := NewMockProvider()

	cards := []model.Card{
		{Name: "Charizard", Number: "123", SetName: "Surging Sparks"},
		{Name: "Pikachu", Number: "456", SetName: "Surging Sparks"},
	}

	results, err := provider.GetBulkListings(cards)
	if err != nil {
		t.Fatalf("GetBulkListings failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	for key, data := range results {
		if data == nil {
			t.Errorf("Listing data for %s should not be nil", key)
		}
		if len(data.ActiveList) == 0 {
			t.Errorf("Expected listings for %s", key)
		}
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
				Price:     120.00,
				Grade:     "BGS 9.5",
				Title:     "Pokemon Charizard BGS 9.5",
				InStock:   true,
				SKU:       "TEST002",
				Seller:    "GameStop",
			},
			{
				Price:     50.00,
				Grade:     "Raw",
				Title:     "Pokemon Charizard Ungraded",
				InStock:   false, // Should be filtered out
				Seller:    "GameStop",
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

	for _, data := range priceData {
		if data.Value <= 0 {
			t.Error("Price data value should be positive")
		}
		if data.Currency != "USD" {
			t.Errorf("Expected currency 'USD', got '%s'", data.Currency)
		}
		if data.Source.Type != fusion.SourceTypeListing {
			t.Errorf("Expected source type 'LISTING', got '%s'", data.Source.Type)
		}
		if data.Source.Name != "GameStop" {
			t.Errorf("Expected source name 'GameStop', got '%s'", data.Source.Name)
		}
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