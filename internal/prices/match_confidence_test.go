package prices

import (
	"math"
	"strings"
	"testing"
)

func TestMatchConfidenceScorer_ScoreMethod(t *testing.T) {
	scorer := NewMatchConfidenceScorer()

	tests := []struct {
		method   MatchMethod
		expected float64
	}{
		{MatchMethodUPC, 1.0},
		{MatchMethodID, 0.95},
		{MatchMethodManual, 0.9},
		{MatchMethodSearch, 0.7},
		{MatchMethodFuzzy, 0.5},
		{MatchMethod("unknown"), 0.3},
	}

	for _, tt := range tests {
		t.Run(string(tt.method), func(t *testing.T) {
			score := scorer.scoreMethod(tt.method)
			if math.Abs(score-tt.expected) > 0.01 {
				t.Errorf("Expected score %.2f for method %s, got %.2f", tt.expected, tt.method, score)
			}
		})
	}
}

func TestMatchConfidenceScorer_CalculateStringSimilarity(t *testing.T) {
	scorer := NewMatchConfidenceScorer()

	tests := []struct {
		s1       string
		s2       string
		expected float64
	}{
		{"exact match", "exact match", 1.0},
		{"EXACT MATCH", "exact match", 1.0}, // Case insensitive
		{"  spaces  ", "spaces", 1.0},       // Trimmed
		{"similar", "similiar", 0.875},
		{"different", "totally", 0.0},
		{"", "", 1.0}, // Empty strings are identical
		{"something", "", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.s1+"_vs_"+tt.s2, func(t *testing.T) {
			similarity := scorer.calculateStringSimilarity(tt.s1, tt.s2)
			if math.Abs(similarity-tt.expected) > 0.1 {
				t.Errorf("Expected similarity ~%.2f, got %.2f", tt.expected, similarity)
			}
		})
	}
}

func TestMatchConfidenceScorer_CalculateSetMatch(t *testing.T) {
	scorer := NewMatchConfidenceScorer()

	tests := []struct {
		expectedSet string
		productName string
		expected    float64
	}{
		{"Surging Sparks", "Pokemon Surging Sparks Pikachu #250", 1.0},
		{"Base Set", "Pokemon Base Set Charizard", 1.0},
		{"Sword Shield", "Pokemon SWSH Base Set", 0.8}, // Abbreviation match
		{"Sun Moon", "Pokemon SM Guardians Rising", 0.8},
		{"Brilliant Stars", "Pokemon BRS Card", 0.8},
		{"Random Set", "Completely Different Product", 0.0},
		{"Crown Zenith", "Crown Card Zenith Pack", 1.0}, // Both words found
	}

	for _, tt := range tests {
		t.Run(tt.expectedSet, func(t *testing.T) {
			score := scorer.calculateSetMatch(tt.expectedSet, tt.productName)
			if math.Abs(score-tt.expected) > 0.1 {
				t.Errorf("Expected score ~%.2f for set '%s' in '%s', got %.2f",
					tt.expected, tt.expectedSet, tt.productName, score)
			}
		})
	}
}

func TestMatchConfidenceScorer_CalculateNumberMatch(t *testing.T) {
	scorer := NewMatchConfidenceScorer()

	tests := []struct {
		expectedNumber string
		productName    string
		expected       float64
	}{
		{"250", "Pikachu ex #250", 1.0},
		{"004", "Charizard 004/102", 1.0},
		{"1", "Card Name 1 Special", 1.0},
		{"123", "Product-123", 1.0},
		{"99", "Card #98", 0.0},
		{"50", "Card 50", 0.5}, // Number appears but not in expected format
	}

	for _, tt := range tests {
		t.Run(tt.expectedNumber+"_in_"+tt.productName, func(t *testing.T) {
			score := scorer.calculateNumberMatch(tt.expectedNumber, tt.productName)
			if math.Abs(score-tt.expected) > 0.1 {
				t.Errorf("Expected score ~%.2f for number '%s' in '%s', got %.2f",
					tt.expected, tt.expectedNumber, tt.productName, score)
			}
		})
	}
}

func TestMatchConfidenceScorer_CalculatePriceCompleteness(t *testing.T) {
	tests := []struct {
		name     string
		match    *PCMatch
		expected float64
	}{
		{
			name: "all prices",
			match: &PCMatch{
				LooseCents:   1000,
				Grade9Cents:  2000,
				Grade95Cents: 2500,
				PSA10Cents:   3000,
				BGS10Cents:   3500,
			},
			expected: 1.0,
		},
		{
			name: "some prices",
			match: &PCMatch{
				LooseCents: 1000,
				PSA10Cents: 3000,
				BGS10Cents: 3500,
			},
			expected: 0.6,
		},
		{
			name:     "no prices",
			match:    &PCMatch{},
			expected: 0.0,
		},
	}

	scorer := NewMatchConfidenceScorer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scorer.calculatePriceCompleteness(tt.match)
			if math.Abs(score-tt.expected) > 0.01 {
				t.Errorf("Expected completeness %.2f, got %.2f", tt.expected, score)
			}
		})
	}
}

func TestMatchConfidenceScorer_CalculateConfidence(t *testing.T) {
	scorer := NewMatchConfidenceScorer()

	tests := []struct {
		name           string
		method         MatchMethod
		query          string
		match          *PCMatch
		expectedSet    string
		expectedNumber string
		minConfidence  float64
		maxConfidence  float64
	}{
		{
			name:   "UPC match",
			method: MatchMethodUPC,
			query:  "pokemon surging sparks pikachu #250",
			match: &PCMatch{
				ID:          "123",
				ProductName: "Pokemon Surging Sparks Pikachu ex #250",
				PSA10Cents:  5000,
			},
			expectedSet:    "Surging Sparks",
			expectedNumber: "250",
			minConfidence:  0.8,
			maxConfidence:  1.0,
		},
		{
			name:   "fuzzy match",
			method: MatchMethodFuzzy,
			query:  "pokemon base charizard",
			match: &PCMatch{
				ProductName: "Pokemon Base Set Charizard #4",
				PSA10Cents:  100000,
			},
			expectedSet:    "Base Set",
			expectedNumber: "4",
			minConfidence:  0.3,
			maxConfidence:  0.7,
		},
		{
			name:           "poor match",
			method:         MatchMethodSearch,
			query:          "pokemon random card",
			match:          &PCMatch{ProductName: "Different Product"},
			expectedSet:    "Random Set",
			expectedNumber: "999",
			minConfidence:  0.1,
			maxConfidence:  0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := scorer.CalculateConfidence(
				tt.method,
				tt.query,
				tt.match,
				tt.expectedSet,
				tt.expectedNumber,
			)

			if confidence < tt.minConfidence || confidence > tt.maxConfidence {
				t.Errorf("Expected confidence between %.2f and %.2f, got %.2f",
					tt.minConfidence, tt.maxConfidence, confidence)
			}
		})
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		expected int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "a", 1},
		{"abc", "abc", 0},
		{"abc", "ab", 1},
		{"abc", "adc", 1},
		{"abc", "xyz", 3},
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
	}

	for _, tt := range tests {
		t.Run(tt.s1+"_"+tt.s2, func(t *testing.T) {
			distance := levenshteinDistance(tt.s1, tt.s2)
			if distance != tt.expected {
				t.Errorf("Expected distance %d between '%s' and '%s', got %d",
					tt.expected, tt.s1, tt.s2, distance)
			}
		})
	}
}

func TestFuzzyMatcher_Match(t *testing.T) {
	matcher := NewFuzzyMatcher(0.7)

	tests := []struct {
		name         string
		query        string
		candidates   []string
		expectedBest string
		minScore     float64
	}{
		{
			name:         "exact match",
			query:        "Pikachu",
			candidates:   []string{"Pikachu", "Raichu", "Pichu"},
			expectedBest: "Pikachu",
			minScore:     1.0,
		},
		{
			name:         "similar match",
			query:        "Charizard",
			candidates:   []string{"Charmeleon", "Charizard EX", "Blastoise"},
			expectedBest: "Charizard EX",
			minScore:     0.7,
		},
		{
			name:         "no good match",
			query:        "Mewtwo",
			candidates:   []string{"Pikachu", "Charizard", "Blastoise"},
			expectedBest: "",
			minScore:     0.0,
		},
		{
			name:         "empty candidates",
			query:        "Anything",
			candidates:   []string{},
			expectedBest: "",
			minScore:     0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			best, score := matcher.Match(tt.query, tt.candidates)
			if best != tt.expectedBest {
				t.Errorf("Expected best match '%s', got '%s'", tt.expectedBest, best)
			}
			if score < tt.minScore {
				t.Errorf("Expected score >= %.2f, got %.2f", tt.minScore, score)
			}
		})
	}
}

func TestFuzzyMatcher_MatchWithDetails(t *testing.T) {
	matcher := NewFuzzyMatcher(0.6)

	candidates := []string{
		"Pikachu ex",
		"Pikachu VMAX",
		"Raichu",
		"Pichu",
	}

	results := matcher.MatchWithDetails("Pikachu", candidates)

	if len(results) < 2 {
		t.Errorf("Expected at least 2 results, got %d", len(results))
	}

	// Check that results are sorted by similarity
	for i := 0; i < len(results)-1; i++ {
		if results[i].Similarity < results[i+1].Similarity {
			t.Error("Results not sorted by similarity (highest first)")
		}
	}

	// Check that first result is best match
	if len(results) > 0 && !strings.Contains(results[0].Candidate, "Pikachu") {
		t.Errorf("Expected first result to contain 'Pikachu', got '%s'", results[0].Candidate)
	}
}

func TestMatchConfidenceScorer_CalculateAttributeScore(t *testing.T) {
	scorer := NewMatchConfidenceScorer()

	tests := []struct {
		name     string
		match    *PCMatch
		minScore float64
		maxScore float64
	}{
		{
			name: "full attributes",
			match: &PCMatch{
				ID:             "123",
				RecentSales:    []SaleData{{PriceCents: 1000}},
				RetailBuyPrice: 800,
				ActiveListings: 5,
			},
			minScore: 0.8,
			maxScore: 1.0,
		},
		{
			name: "partial attributes",
			match: &PCMatch{
				ID:             "123",
				ActiveListings: 3,
			},
			minScore: 0.4,
			maxScore: 0.6,
		},
		{
			name:     "no attributes",
			match:    &PCMatch{},
			minScore: 0.4,
			maxScore: 0.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scorer.calculateAttributeScore(tt.match)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("Expected score between %.2f and %.2f, got %.2f",
					tt.minScore, tt.maxScore, score)
			}
		})
	}
}
