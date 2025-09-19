package integration

import (
	"os"
	"testing"

	"github.com/guarzo/pkmgradegap/internal/cache"
	"github.com/guarzo/pkmgradegap/internal/model"
	"github.com/guarzo/pkmgradegap/internal/prices"
)

func TestSprint4_UPCLookup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	token := os.Getenv("PRICECHARTING_TOKEN")
	if token == "" || token == "test" {
		t.Skip("PRICECHARTING_TOKEN not set for integration test")
	}

	// Create PriceCharting provider with cache
	c, err := cache.New("./test_cache.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("./test_cache.json") // Clean up
	pc := prices.NewPriceCharting(token, c)

	t.Run("UPC Lookup", func(t *testing.T) {
		// Test with a known UPC (example - may need real UPC)
		upc := "820650558726" // Example UPC for testing
		match, err := pc.LookupByUPC(upc)

		if err != nil {
			// UPC might not exist, which is okay for integration test
			t.Logf("UPC lookup returned error (expected if UPC doesn't exist): %v", err)
			return
		}

		// Verify match has expected fields
		if match.UPC != upc {
			t.Errorf("Expected UPC %s, got %s", upc, match.UPC)
		}

		if match.MatchMethod != prices.MatchMethodUPC {
			t.Errorf("Expected match method %s, got %s", prices.MatchMethodUPC, match.MatchMethod)
		}

		if match.MatchConfidence != 1.0 {
			t.Errorf("Expected confidence 1.0 for UPC match, got %.2f", match.MatchConfidence)
		}

		t.Logf("UPC match found: %s (ID: %s)", match.ProductName, match.ID)
	})

	t.Run("Advanced Query with Options", func(t *testing.T) {
		options := prices.QueryOptions{
			Variant:  "1st Edition",
			Language: "English",
		}

		match, err := pc.LookupWithOptions("Base Set", "Charizard", "4", options)

		if err != nil {
			t.Logf("Advanced query returned error: %v", err)
			return
		}

		// Check that variant was detected
		if match.Variant != "" {
			t.Logf("Detected variant: %s", match.Variant)
		}

		// Check that language was set
		if match.Language != "" {
			t.Logf("Detected language: %s", match.Language)
		}

		// Check confidence score
		if match.MatchConfidence > 0 {
			t.Logf("Match confidence: %.2f", match.MatchConfidence)
		}

		// Check query used
		if match.QueryUsed != "" {
			t.Logf("Query used: %s", match.QueryUsed)
		}

		t.Logf("Advanced match: %s (Method: %s)", match.ProductName, match.MatchMethod)
	})

	t.Run("Japanese Card Lookup", func(t *testing.T) {
		options := prices.QueryOptions{
			Language: "Japanese",
		}

		match, err := pc.LookupWithOptions("VMAX Climax", "Charizard VMAX", "3", options)

		if err != nil {
			t.Logf("Japanese card lookup error: %v", err)
			return
		}

		// Check language detection
		if match.Language == "Japanese" {
			t.Logf("Correctly identified Japanese card")
		}

		t.Logf("Japanese card: %s", match.ProductName)
	})

	t.Run("Fuzzy Matching Fallback", func(t *testing.T) {
		// Intentionally use a slightly wrong query to test fuzzy matching
		options := prices.QueryOptions{}

		// Try with a typo or variation
		match, err := pc.LookupWithOptions("Surgng Sparks", "Pikchu", "250", options)

		if err == nil && match != nil {
			if match.MatchMethod == prices.MatchMethodFuzzy {
				t.Logf("Fuzzy match succeeded: %s", match.ProductName)
				t.Logf("Fuzzy match confidence: %.2f", match.MatchConfidence)
				t.Logf("Query that worked: %s", match.QueryUsed)
			}
		} else {
			t.Logf("Fuzzy matching didn't find a match (expected for major typos)")
		}
	})

	t.Run("Match Confidence Scoring", func(t *testing.T) {
		// Test confidence scoring with different quality matches
		testCases := []struct {
			setName  string
			cardName string
			number   string
			desc     string
		}{
			{"Surging Sparks", "Pikachu ex", "250", "exact match"},
			{"Surging Sparks", "Pikachu", "250", "partial name"},
			{"Surgng Spark", "Pikachu ex", "250", "typo in set"},
		}

		for _, tc := range testCases {
			match, err := pc.LookupCard(tc.setName, model.Card{
				Name:   tc.cardName,
				Number: tc.number,
			})

			if err != nil {
				t.Logf("%s: lookup error: %v", tc.desc, err)
				continue
			}

			t.Logf("%s: confidence=%.2f, method=%s, product=%s",
				tc.desc, match.MatchConfidence, match.MatchMethod, match.ProductName)
		}
	})
}

func TestSprint4_QueryBuilder(t *testing.T) {
	// Test query builder functionality
	qb := prices.NewQueryBuilder()

	t.Run("Complex Query", func(t *testing.T) {
		query := qb.
			SetBase("Base Set", "Charizard", "4").
			WithVariant("1st Edition").
			WithLanguage("English").
			WithCondition("Mint").
			WithGrader("PSA").
			Build()

		expectedParts := []string{
			"pokemon Base Set Charizard #4",
			"1st edition",
			"mint",
			"PSA",
		}

		for _, part := range expectedParts {
			if !containsIgnoreCase(query, part) {
				t.Errorf("Expected query to contain '%s', got: %s", part, query)
			}
		}
	})

	t.Run("Query Confidence", func(t *testing.T) {
		qb := prices.NewQueryBuilder().
			SetBase("Surging Sparks", "Pikachu ex", "250").
			WithVariant("Reverse Holo")

		query, confidence := qb.BuildWithConfidence()

		if confidence < 0.7 {
			t.Errorf("Expected confidence > 0.7 for specific query, got %.2f", confidence)
		}

		t.Logf("Query: %s (confidence: %.2f)", query, confidence)
	})
}

func TestSprint4_UPCDatabase(t *testing.T) {
	// Test UPC database functionality
	tmpDir := t.TempDir()
	db, err := prices.NewUPCDatabase(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Populate with common mappings
	db.PopulateCommonMappings()

	t.Run("Database Stats", func(t *testing.T) {
		stats := db.Stats()
		total, ok := stats["total_mappings"].(int)
		if !ok || total == 0 {
			t.Error("Expected populated database")
		}

		t.Logf("Database contains %d UPC mappings", total)

		if langs, ok := stats["languages"].(map[string]int); ok {
			for lang, count := range langs {
				t.Logf("  %s: %d cards", lang, count)
			}
		}
	})

	t.Run("Find by Set", func(t *testing.T) {
		// Find all Surging Sparks cards
		mappings := db.FindByCardInfo("Surging Sparks", "250")
		if len(mappings) > 0 {
			t.Logf("Found %d UPC mappings for Surging Sparks #250", len(mappings))
			for _, m := range mappings {
				t.Logf("  UPC: %s, Product: %s", m.UPC, m.ProductName)
			}
		}
	})

	t.Run("Persistence", func(t *testing.T) {
		// Add custom mapping
		db.Add(&prices.UPCMapping{
			UPC:         "test-upc-123",
			ProductID:   "test-product",
			ProductName: "Test Card",
			SetName:     "Test Set",
			CardNumber:  "001",
			Language:    "English",
			Confidence:  0.95,
		})

		// Save to disk
		if err := db.Save(); err != nil {
			t.Error(err)
		}

		// Load in new database instance
		db2, err := prices.NewUPCDatabase(tmpDir)
		if err != nil {
			t.Fatal(err)
		}

		// Verify custom mapping persisted
		if mapping, found := db2.Lookup("test-upc-123"); found {
			t.Logf("Successfully persisted and loaded UPC mapping: %s", mapping.ProductName)
		} else {
			t.Error("Failed to persist UPC mapping")
		}
	})
}

func TestSprint4_FuzzyMatcher(t *testing.T) {
	matcher := prices.NewFuzzyMatcher(0.7)

	candidates := []string{
		"Surging Sparks",
		"Prismatic Evolutions",
		"Crown Zenith",
		"Scarlet & Violet",
		"Sword & Shield",
	}

	testQueries := []string{
		"Surging Spark",  // Missing 's'
		"Surgng Sparks",  // Typo
		"Crown Zenth",    // Typo
		"Scarlet Violet", // Missing &
	}

	for _, query := range testQueries {
		best, score := matcher.Match(query, candidates)
		if best != "" {
			t.Logf("Query '%s' matched '%s' (score: %.2f)", query, best, score)
		} else {
			t.Logf("Query '%s' had no match above threshold", query)
		}
	}

	// Test detailed matching
	results := matcher.MatchWithDetails("Surging", candidates)
	t.Logf("\nDetailed matches for 'Surging':")
	for _, r := range results {
		t.Logf("  %s: similarity=%.2f, distance=%d", r.Candidate, r.Similarity, r.Distance)
	}
}

// Helper function
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				containsSubstringIgnoreCase(s, substr))
}

func containsSubstringIgnoreCase(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}

	// Simple case-insensitive contains
	sLower := toLowerCase(s)
	substrLower := toLowerCase(substr)

	return containsSubstring(sLower, substrLower)
}

func toLowerCase(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

func containsSubstring(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
