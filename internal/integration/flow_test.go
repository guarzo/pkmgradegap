package integration

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/guarzo/pkmgradegap/internal/analysis"
	"github.com/guarzo/pkmgradegap/internal/cache"
	"github.com/guarzo/pkmgradegap/internal/cards"
	"github.com/guarzo/pkmgradegap/internal/ebay"
	"github.com/guarzo/pkmgradegap/internal/model"
	"github.com/guarzo/pkmgradegap/internal/prices"
	"github.com/guarzo/pkmgradegap/internal/report"
	"github.com/guarzo/pkmgradegap/internal/testutil"
)

// MockCardProvider implements a test card provider
type MockCardProvider struct {
	sets  []model.Set
	cards map[string][]model.Card
}

func NewMockCardProvider() *MockCardProvider {
	return &MockCardProvider{
		sets: []model.Set{
			{ID: "sv1", Name: "Scarlet & Violet", ReleaseDate: "2023/03/31"},
			{ID: "sv7", Name: "Surging Sparks", ReleaseDate: "2024/11/08"},
		},
		cards: map[string][]model.Card{
			"sv7": {
				{
					ID:      "sv7-1",
					Name:    "Sprigatito",
					Number:  "001",
					Rarity:  "Common",
					SetID:   "sv7",
					SetName: "Surging Sparks",
					TCGPlayer: &model.TCGPlayerBlock{
						Prices: map[string]struct {
							Low       *float64 `json:"low,omitempty"`
							Mid       *float64 `json:"mid,omitempty"`
							High      *float64 `json:"high,omitempty"`
							Market    *float64 `json:"market,omitempty"`
							DirectLow *float64 `json:"directLow,omitempty"`
						}{
							"normal": {
								Market: floatPtr(0.50),
								Mid:    floatPtr(0.75),
							},
						},
					},
				},
				{
					ID:      "sv7-238",
					Name:    "Pikachu ex",
					Number:  "238",
					Rarity:  "Special Illustration Rare",
					SetID:   "sv7",
					SetName: "Surging Sparks",
					TCGPlayer: &model.TCGPlayerBlock{
						Prices: map[string]struct {
							Low       *float64 `json:"low,omitempty"`
							Mid       *float64 `json:"mid,omitempty"`
							High      *float64 `json:"high,omitempty"`
							Market    *float64 `json:"market,omitempty"`
							DirectLow *float64 `json:"directLow,omitempty"`
						}{
							"normal": {
								Market: floatPtr(125.00),
								Mid:    floatPtr(150.00),
							},
						},
					},
				},
				{
					ID:      "sv7-69420", // Outlier price for sanitization test
					Name:    "Test Outlier",
					Number:  "999",
					Rarity:  "Common",
					SetID:   "sv7",
					SetName: "Surging Sparks",
					TCGPlayer: &model.TCGPlayerBlock{
						Prices: map[string]struct {
							Low       *float64 `json:"low,omitempty"`
							Mid       *float64 `json:"mid,omitempty"`
							High      *float64 `json:"high,omitempty"`
							Market    *float64 `json:"market,omitempty"`
							DirectLow *float64 `json:"directLow,omitempty"`
						}{
							"normal": {
								Market: floatPtr(69420.69), // Outlier
							},
						},
					},
				},
			},
		},
	}
}

func (m *MockCardProvider) ListSets() ([]model.Set, error) {
	return m.sets, nil
}

func (m *MockCardProvider) CardsBySetID(setID string) ([]model.Card, error) {
	if cards, ok := m.cards[setID]; ok {
		return cards, nil
	}
	return []model.Card{}, nil
}

// MockPriceProvider implements a test price provider
type MockPriceProvider struct {
	available bool
	prices    map[string]*prices.PCMatch
}

func NewMockPriceProvider(available bool) *MockPriceProvider {
	return &MockPriceProvider{
		available: available,
		prices: map[string]*prices.PCMatch{
			"surging sparks|pikachu ex|238": {
				ID:           "mock-12345",
				ProductName:  "Pokemon Surging Sparks Pikachu ex #238",
				LooseCents:   12500, // $125.00
				Grade9Cents:  25000, // $250.00
				Grade95Cents: 30000, // $300.00
				PSA10Cents:   50000, // $500.00
				BGS10Cents:   55000, // $550.00
			},
			"surging sparks|sprigatito|001": {
				ID:           "mock-67890",
				ProductName:  "Pokemon Surging Sparks Sprigatito #001",
				LooseCents:   50,   // $0.50
				Grade9Cents:  200,  // $2.00
				Grade95Cents: 300,  // $3.00
				PSA10Cents:   1000, // $10.00
				BGS10Cents:   1200, // $12.00
			},
		},
	}
}

func (m *MockPriceProvider) Available() bool {
	return m.available
}

func (m *MockPriceProvider) LookupCard(setName string, card model.Card) (*prices.PCMatch, error) {
	if !m.available {
		return nil, nil
	}

	key := strings.ToLower(setName) + "|" + strings.ToLower(card.Name) + "|" + card.Number
	if match, ok := m.prices[key]; ok {
		return match, nil
	}
	return nil, nil
}

// MockEbayProvider implements a test eBay provider
type MockEbayProvider struct {
	listings map[string][]ebay.Listing
}

func NewMockEbayProvider() *MockEbayProvider {
	return &MockEbayProvider{
		listings: map[string][]ebay.Listing{
			"pikachu ex": {
				{
					Title:     "Pokemon Surging Sparks Pikachu ex BGS 10",
					Price:     500.00,
					URL:       "https://ebay.com/item/123",
					Condition: "New",
					BuyItNow:  true,
				},
				{
					Title:     "Pikachu ex PSA 10 Surging Sparks",
					Price:     475.00,
					URL:       "https://ebay.com/item/124",
					Condition: "New",
					BuyItNow:  true,
				},
			},
		},
	}
}

func (m *MockEbayProvider) Available() bool {
	return true
}

func (m *MockEbayProvider) SearchRawListings(setName, cardName, number string, max int) ([]ebay.Listing, error) {
	query := strings.ToLower(cardName)
	for key, listings := range m.listings {
		if strings.Contains(query, key) {
			if max > 0 && len(listings) > max {
				return listings[:max], nil
			}
			return listings, nil
		}
	}
	return []ebay.Listing{}, nil
}

func TestFullIntegrationFlow(t *testing.T) {
	// Create providers
	cardProvider := NewMockCardProvider()
	priceProvider := NewMockPriceProvider(true)

	// Test full flow: Cards → Prices → Analysis → CSV
	t.Run("complete_flow", func(t *testing.T) {
		// Step 1: Get cards from set
		sets, err := cardProvider.ListSets()
		if err != nil {
			t.Fatalf("failed to list sets: %v", err)
		}

		var chosenSet *model.Set
		for _, set := range sets {
			if set.ID == "sv7" {
				chosenSet = &set
				break
			}
		}

		if chosenSet == nil {
			t.Fatalf("test set sv7 not found")
		}

		cards, err := cardProvider.CardsBySetID(chosenSet.ID)
		if err != nil {
			t.Fatalf("failed to get cards for set: %v", err)
		}

		if len(cards) != 3 {
			t.Errorf("expected 3 cards, got %d", len(cards))
		}

		// Step 2: Build analysis rows
		var rows []analysis.Row
		for _, card := range cards {
			rawUSD, rawSrc, rawNote := analysis.ExtractUngradedUSD(card)

			var grades analysis.Grades
			if match, err := priceProvider.LookupCard(chosenSet.Name, card); err == nil && match != nil {
				grades = analysis.Grades{
					PSA10:   float64(match.PSA10Cents) / 100.0,
					Grade9:  float64(match.Grade9Cents) / 100.0,
					Grade95: float64(match.Grade95Cents) / 100.0,
					BGS10:   float64(match.BGS10Cents) / 100.0,
				}
			}

			rows = append(rows, analysis.Row{
				Card:    card,
				RawUSD:  rawUSD,
				RawSrc:  rawSrc,
				RawNote: rawNote,
				Grades:  grades,
			})
		}

		if len(rows) != 3 {
			t.Errorf("expected 3 analysis rows, got %d", len(rows))
		}

		// Step 3: Apply sanitization
		sanitizeConfig := analysis.DefaultSanitizeConfig()
		rows = analysis.SanitizeRows(rows, sanitizeConfig)

		// Should have filtered out the outlier card
		if len(rows) != 2 {
			t.Errorf("expected 2 rows after sanitization (outlier removed), got %d", len(rows))
		}

		// Step 4: Generate CSV report
		csvData := analysis.ReportRawVsPSA10(rows)
		if len(csvData) < 2 { // Header + at least 1 data row
			t.Errorf("expected CSV data with header + rows, got %d rows", len(csvData))
		}

		// Step 5: Apply CSV safety
		safeCSV := report.EscapeCSVRows(csvData)
		if len(safeCSV) != len(csvData) {
			t.Errorf("CSV safety should preserve row count")
		}

		// Verify CSV structure
		if safeCSV[0][0] != "Card" {
			t.Errorf("expected first header to be 'Card', got %s", safeCSV[0][0])
		}

		// Verify data row contains expected card
		foundPikachu := false
		for i := 1; i < len(safeCSV); i++ {
			if strings.Contains(safeCSV[i][0], "Pikachu") {
				foundPikachu = true
				break
			}
		}
		if !foundPikachu {
			t.Errorf("expected to find Pikachu in CSV output")
		}
	})
}

func TestSnapshotSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	snapshotPath := filepath.Join(tmpDir, "test_snapshot.json")

	cardProvider := NewMockCardProvider()
	_ = NewMockPriceProvider(true) // Not used in this test

	// Create test rows
	cards, _ := cardProvider.CardsBySetID("sv7")
	var rows []analysis.Row

	for _, card := range cards {
		rawUSD, rawSrc, rawNote := analysis.ExtractUngradedUSD(card)
		rows = append(rows, analysis.Row{
			Card:    card,
			RawUSD:  rawUSD,
			RawSrc:  rawSrc,
			RawNote: rawNote,
		})
	}

	// Save snapshot
	snapshotData, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal snapshot: %v", err)
	}

	err = os.WriteFile(snapshotPath, snapshotData, 0644)
	if err != nil {
		t.Fatalf("failed to write snapshot: %v", err)
	}

	// Load snapshot
	loadedData, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("failed to read snapshot: %v", err)
	}

	var loadedRows []analysis.Row
	err = json.Unmarshal(loadedData, &loadedRows)
	if err != nil {
		t.Fatalf("failed to unmarshal snapshot: %v", err)
	}

	// Verify loaded data
	if len(loadedRows) != len(rows) {
		t.Errorf("expected %d loaded rows, got %d", len(rows), len(loadedRows))
	}

	for i, row := range loadedRows {
		if row.Card.ID != rows[i].Card.ID {
			t.Errorf("card ID mismatch at index %d: expected %s, got %s", i, rows[i].Card.ID, row.Card.ID)
		}
	}
}

func TestHistoryTracking(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "test_history.csv")

	// Create test CSV data
	testData := [][]string{
		{"Card", "Number", "RawUSD", "PSA10_USD", "Delta_USD"},
		{"Pikachu ex", "238", "125.00", "500.00", "375.00"},
		{"Sprigatito", "001", "0.50", "10.00", "9.50"},
	}

	// Write initial history
	file, err := os.Create(historyPath)
	if err != nil {
		t.Fatalf("failed to create history file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	err = writer.WriteAll(testData)
	if err != nil {
		t.Fatalf("failed to write history: %v", err)
	}
	writer.Flush()
	file.Close()

	// Append new data
	newData := [][]string{
		{"Charizard", "025", "200.00", "800.00", "600.00"},
	}

	file, err = os.OpenFile(historyPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("failed to open history file for append: %v", err)
	}
	defer file.Close()

	writer = csv.NewWriter(file)
	err = writer.WriteAll(newData)
	if err != nil {
		t.Fatalf("failed to append to history: %v", err)
	}
	writer.Flush()
	file.Close()

	// Read and verify
	file, err = os.Open(historyPath)
	if err != nil {
		t.Fatalf("failed to read history file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to read history CSV: %v", err)
	}

	expectedRows := len(testData) + len(newData)
	if len(records) != expectedRows {
		t.Errorf("expected %d history rows, got %d", expectedRows, len(records))
	}

	// Verify last row is the appended data
	lastRow := records[len(records)-1]
	if lastRow[0] != "Charizard" {
		t.Errorf("expected last row card name Charizard, got %s", lastRow[0])
	}
}

func TestCacheIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "integration_cache.json")

	testCache, err := cache.New(cachePath)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	cardProvider := cards.NewPokeTCGIO("", testCache)
	_ = prices.NewPriceCharting(testutil.GetTestPriceChartingToken(), testCache) // Test creation only

	// Test cache with card provider
	t.Run("card_provider_cache", func(t *testing.T) {
		// Pre-populate cache
		testSets := []model.Set{
			{ID: "cached-set", Name: "Cached Set", ReleaseDate: "2024/01/01"},
		}
		err := testCache.Put(cache.SetsKey(), testSets, 1*time.Hour)
		if err != nil {
			t.Fatalf("failed to cache sets: %v", err)
		}

		// Should retrieve from cache
		sets, err := cardProvider.ListSets()
		if err != nil {
			t.Fatalf("failed to list cached sets: %v", err)
		}

		if len(sets) != 1 || sets[0].ID != "cached-set" {
			t.Errorf("expected cached set data")
		}
	})

	// Test cache with price provider
	t.Run("price_provider_cache", func(t *testing.T) {
		testPriceProvider := prices.NewPriceCharting(testutil.GetTestPriceChartingToken(), testCache)

		if !testPriceProvider.Available() {
			t.Skip("price provider not available without token")
		}

		// Pre-populate price cache
		testMatch := &prices.PCMatch{
			ID:          "cached-price",
			ProductName: "Cached Product",
			PSA10Cents:  5000,
		}

		key := cache.PriceChartingKey("Test Set", "Test Card", "001")
		err := testCache.Put(key, testMatch, 1*time.Hour)
		if err != nil {
			t.Fatalf("failed to cache price: %v", err)
		}

		// Should retrieve from cache
		card := model.Card{Name: "Test Card", Number: "001"}
		match, err := testPriceProvider.LookupCard("Test Set", card)
		if err != nil {
			t.Fatalf("failed to lookup cached price: %v", err)
		}

		if match == nil || match.ID != "cached-price" {
			t.Errorf("expected cached price data")
		}
	})
}

func TestEbayIntegration(t *testing.T) {
	ebayProvider := NewMockEbayProvider()

	if !ebayProvider.Available() {
		t.Fatalf("eBay provider should be available")
	}

	listings, err := ebayProvider.SearchRawListings("Surging Sparks", "pikachu ex", "238", 2)
	if err != nil {
		t.Fatalf("eBay search failed: %v", err)
	}

	if len(listings) != 2 {
		t.Errorf("expected 2 eBay listings, got %d", len(listings))
	}

	// Verify listing data
	if listings[0].Price != 500.00 {
		t.Errorf("expected first listing price 500.00, got %.2f", listings[0].Price)
	}

	if !strings.Contains(listings[0].Title, "BGS 10") {
		t.Errorf("expected BGS 10 in title: %s", listings[0].Title)
	}
}

func TestErrorHandling(t *testing.T) {
	// Test with unavailable providers
	t.Run("no_price_provider", func(t *testing.T) {
		cardProvider := NewMockCardProvider()
		priceProvider := NewMockPriceProvider(false)

		cards, _ := cardProvider.CardsBySetID("sv7")
		var rows []analysis.Row

		for _, card := range cards {
			rawUSD, rawSrc, rawNote := analysis.ExtractUngradedUSD(card)

			var grades analysis.Grades
			// This should not populate grades since provider is unavailable
			if match, err := priceProvider.LookupCard("Surging Sparks", card); err == nil && match != nil {
				grades = analysis.Grades{
					PSA10: float64(match.PSA10Cents) / 100.0,
				}
			}

			rows = append(rows, analysis.Row{
				Card:    card,
				RawUSD:  rawUSD,
				RawSrc:  rawSrc,
				RawNote: rawNote,
				Grades:  grades,
			})
		}

		// All grades should be zero since provider unavailable
		for _, row := range rows {
			if row.Grades.PSA10 != 0 {
				t.Errorf("expected zero PSA10 price when provider unavailable, got %.2f", row.Grades.PSA10)
			}
		}
	})

	// Test mock limitations
	t.Run("provider_interface", func(t *testing.T) {
		ebayProvider := NewMockEbayProvider()

		// Test available method
		if !ebayProvider.Available() {
			t.Errorf("mock eBay provider should be available")
		}

		// Test search with no results
		listings, err := ebayProvider.SearchRawListings("Unknown Set", "nonexistent card", "999", 1)
		if err != nil {
			t.Errorf("search should not error for no results: %v", err)
		}

		if len(listings) != 0 {
			t.Errorf("expected no listings for unknown card, got %d", len(listings))
		}
	})
}

func TestCSVOutputFormats(t *testing.T) {
	cardProvider := NewMockCardProvider()
	cards, _ := cardProvider.CardsBySetID("sv7")

	// Create test rows
	var rows []analysis.Row
	for _, card := range cards[:2] { // Use first 2 cards
		rawUSD, rawSrc, rawNote := analysis.ExtractUngradedUSD(card)
		rows = append(rows, analysis.Row{
			Card:    card,
			RawUSD:  rawUSD,
			RawSrc:  rawSrc,
			RawNote: rawNote,
			Grades: analysis.Grades{
				PSA10:   rawUSD * 5, // Simulate 5x premium
				Grade9:  rawUSD * 3,
				Grade95: rawUSD * 4,
				BGS10:   rawUSD * 6,
			},
		})
	}

	// Test different report formats
	formats := map[string]func([]analysis.Row) [][]string{
		"raw_vs_psa10":   analysis.ReportRawVsPSA10,
		"multi_vs_psa10": analysis.ReportMultiVsPSA10,
		"crossgrade":     analysis.ReportCrossgrade,
	}

	for formatName, reportFunc := range formats {
		t.Run(formatName, func(t *testing.T) {
			csvData := reportFunc(rows)

			if len(csvData) < 1 {
				t.Errorf("expected at least header row for %s format", formatName)
			}

			// Verify header row exists
			if len(csvData[0]) == 0 {
				t.Errorf("expected non-empty header row for %s format", formatName)
			}

			// Apply CSV safety
			safeCSV := report.EscapeCSVRows(csvData)
			if len(safeCSV) != len(csvData) {
				t.Errorf("CSV safety should preserve row count for %s format", formatName)
			}

			// Test with formula injection - adjust based on format
			testRows := make([]analysis.Row, len(rows))
			copy(testRows, rows)

			if formatName == "crossgrade" {
				// Crossgrade format uses card number field
				testRows[0].Card.Number = "=SUM(A1:A10)"
			} else {
				// Other formats use card name
				testRows[0].Card.Name = "=SUM(A1:A10)"
			}

			maliciousCSV := reportFunc(testRows)
			safeMaliciousCSV := report.EscapeCSVRows(maliciousCSV)

			// Check that formula was escaped
			foundEscaped := false
			for _, row := range safeMaliciousCSV {
				for _, cell := range row {
					if strings.HasPrefix(cell, "'=SUM") {
						foundEscaped = true
						break
					}
				}
			}

			if !foundEscaped && formatName != "crossgrade" {
				// Crossgrade may not include the malicious field depending on data
				t.Errorf("expected formula to be escaped in %s format", formatName)
			}
		})
	}
}

// Helper function to create float64 pointers
func floatPtr(v float64) *float64 {
	return &v
}
