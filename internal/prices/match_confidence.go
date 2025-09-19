package prices

import (
	"math"
	"strings"
)

// MatchMethod represents how a match was found
type MatchMethod string

const (
	MatchMethodUPC    MatchMethod = "upc"
	MatchMethodID     MatchMethod = "id"
	MatchMethodSearch MatchMethod = "search"
	MatchMethodFuzzy  MatchMethod = "fuzzy"
	MatchMethodManual MatchMethod = "manual"
)

// MatchConfidence calculates confidence score for a product match
type MatchConfidenceScorer struct {
	// Weights for different factors
	methodWeight    float64
	nameWeight      float64
	setWeight       float64
	numberWeight    float64
	priceWeight     float64
	attributeWeight float64
}

// NewMatchConfidenceScorer creates a new confidence scorer
func NewMatchConfidenceScorer() *MatchConfidenceScorer {
	return &MatchConfidenceScorer{
		methodWeight:    0.3,  // How the match was found
		nameWeight:      0.25, // Card name similarity
		setWeight:       0.15, // Set name match
		numberWeight:    0.15, // Card number match
		priceWeight:     0.1,  // Price data completeness
		attributeWeight: 0.05, // Additional attributes match
	}
}

// CalculateConfidence computes overall match confidence
func (mcs *MatchConfidenceScorer) CalculateConfidence(
	method MatchMethod,
	query string,
	match *PCMatch,
	expectedSet string,
	expectedNumber string,
) float64 {
	score := 0.0

	// Method score
	score += mcs.methodWeight * mcs.scoreMethod(method)

	// Name similarity score
	if match != nil && match.ProductName != "" {
		nameSim := mcs.calculateStringSimilarity(query, match.ProductName)
		score += mcs.nameWeight * nameSim
	}

	// Set match score
	if expectedSet != "" && match != nil {
		setSim := mcs.calculateSetMatch(expectedSet, match.ProductName)
		score += mcs.setWeight * setSim
	}

	// Number match score
	if expectedNumber != "" && match != nil {
		numMatch := mcs.calculateNumberMatch(expectedNumber, match.ProductName)
		score += mcs.numberWeight * numMatch
	}

	// Price data completeness
	if match != nil {
		priceComplete := mcs.calculatePriceCompleteness(match)
		score += mcs.priceWeight * priceComplete
	}

	// Additional attributes
	if match != nil {
		attrScore := mcs.calculateAttributeScore(match)
		score += mcs.attributeWeight * attrScore
	}

	// Ensure score is between 0 and 1
	return math.Min(1.0, math.Max(0.0, score))
}

// scoreMethod returns confidence based on match method
func (mcs *MatchConfidenceScorer) scoreMethod(method MatchMethod) float64 {
	switch method {
	case MatchMethodUPC:
		return 1.0 // Highest confidence
	case MatchMethodID:
		return 0.95 // Very high confidence
	case MatchMethodSearch:
		return 0.7 // Moderate confidence
	case MatchMethodFuzzy:
		return 0.5 // Lower confidence
	case MatchMethodManual:
		return 0.9 // High confidence (human verified)
	default:
		return 0.3 // Unknown method
	}
}

// calculateStringSimilarity computes similarity between two strings
func (mcs *MatchConfidenceScorer) calculateStringSimilarity(s1, s2 string) float64 {
	// Normalize strings
	s1 = strings.ToLower(strings.TrimSpace(s1))
	s2 = strings.ToLower(strings.TrimSpace(s2))

	if s1 == s2 {
		return 1.0
	}

	// Handle empty strings
	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	// Check for substring match - if s1 is contained in s2 or vice versa
	// This gives high score for "Pikachu" matching "Pikachu ex"
	if strings.Contains(s2, s1) || strings.Contains(s1, s2) {
		// Calculate score based on length ratio
		minLen := math.Min(float64(len(s1)), float64(len(s2)))
		maxLen := math.Max(float64(len(s1)), float64(len(s2)))
		return 0.85 + (0.15 * (minLen / maxLen)) // Score between 0.85 and 1.0
	}

	// Levenshtein distance for other cases
	distance := levenshteinDistance(s1, s2)
	maxLen := math.Max(float64(len(s1)), float64(len(s2)))

	// Convert distance to similarity
	similarity := 1.0 - (float64(distance) / maxLen)
	return math.Max(0, similarity)
}

// calculateSetMatch checks if set name appears in product name
func (mcs *MatchConfidenceScorer) calculateSetMatch(expectedSet, productName string) float64 {
	expectedSet = strings.ToLower(expectedSet)
	productName = strings.ToLower(productName)

	// Direct contains check
	if strings.Contains(productName, expectedSet) {
		return 1.0
	}

	// Check for common abbreviations
	setAbbreviations := map[string][]string{
		"base set":             {"base", "bs"},
		"jungle":               {"jgl", "jun"},
		"fossil":               {"fos", "fsl"},
		"team rocket":          {"tr", "rocket"},
		"gym heroes":           {"gh", "heroes"},
		"gym challenge":        {"gc", "challenge"},
		"neo genesis":          {"ng", "genesis"},
		"sword shield":         {"swsh", "ss"},
		"sun moon":             {"sm"},
		"scarlet violet":       {"sv"},
		"brilliant stars":      {"brs"},
		"astral radiance":      {"asr"},
		"crown zenith":         {"cwz", "cz"},
		"surging sparks":       {"ssp"},
		"prismatic evolutions": {"pe", "pev"},
	}

	for fullName, abbrevs := range setAbbreviations {
		if strings.Contains(expectedSet, fullName) {
			for _, abbrev := range abbrevs {
				if strings.Contains(productName, abbrev) {
					return 0.8 // High confidence for abbreviation match
				}
			}
		}
	}

	// Partial match check
	words := strings.Fields(expectedSet)
	matchCount := 0
	for _, word := range words {
		if len(word) > 3 && strings.Contains(productName, word) {
			matchCount++
		}
	}

	if len(words) > 0 {
		return float64(matchCount) / float64(len(words))
	}

	return 0
}

// calculateNumberMatch checks if card number matches
func (mcs *MatchConfidenceScorer) calculateNumberMatch(expectedNumber, productName string) float64 {
	// Look for number patterns in product name
	patterns := []string{
		"#" + expectedNumber,
		expectedNumber + "/",
		" " + expectedNumber + " ",
		"-" + expectedNumber,
	}

	productLower := strings.ToLower(productName)
	for _, pattern := range patterns {
		if strings.Contains(productLower, strings.ToLower(pattern)) {
			return 1.0
		}
	}

	// Check if number appears at all
	if strings.Contains(productName, expectedNumber) {
		return 0.5
	}

	return 0
}

// calculatePriceCompleteness scores based on available price data
func (mcs *MatchConfidenceScorer) calculatePriceCompleteness(match *PCMatch) float64 {
	fieldCount := 0
	totalFields := 5

	if match.LooseCents > 0 {
		fieldCount++
	}
	if match.Grade9Cents > 0 {
		fieldCount++
	}
	if match.Grade95Cents > 0 {
		fieldCount++
	}
	if match.PSA10Cents > 0 {
		fieldCount++
	}
	if match.BGS10Cents > 0 {
		fieldCount++
	}

	return float64(fieldCount) / float64(totalFields)
}

// calculateAttributeScore scores based on additional attributes
func (mcs *MatchConfidenceScorer) calculateAttributeScore(match *PCMatch) float64 {
	score := 0.0
	hasAnyAttribute := false

	// Has sales data
	if len(match.RecentSales) > 0 {
		score += 0.25
		hasAnyAttribute = true
	}

	// Has retail pricing
	if match.RetailBuyPrice > 0 || match.RetailSellPrice > 0 {
		score += 0.25
		hasAnyAttribute = true
	}

	// Has marketplace data
	if match.ActiveListings > 0 {
		score += 0.25
		hasAnyAttribute = true
	}

	// Has ID (means it's a verified product)
	if match.ID != "" {
		score += 0.25
		hasAnyAttribute = true
	}

	// Return neutral score if no attributes, otherwise return total score
	if !hasAnyAttribute {
		return 0.5 // Neutral score when no attributes present
	}

	return score
}

// levenshteinDistance calculates edit distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	// Initialize first column and row
	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// min returns the minimum of three integers
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// FuzzyMatcher provides fuzzy matching capabilities
type FuzzyMatcher struct {
	threshold float64 // Minimum similarity threshold
}

// NewFuzzyMatcher creates a new fuzzy matcher
func NewFuzzyMatcher(threshold float64) *FuzzyMatcher {
	return &FuzzyMatcher{
		threshold: threshold,
	}
}

// Match performs fuzzy matching and returns best match
func (fm *FuzzyMatcher) Match(query string, candidates []string) (string, float64) {
	if len(candidates) == 0 {
		return "", 0
	}

	bestMatch := ""
	bestScore := 0.0

	scorer := NewMatchConfidenceScorer()

	for _, candidate := range candidates {
		similarity := scorer.calculateStringSimilarity(query, candidate)
		if similarity > bestScore && similarity >= fm.threshold {
			bestScore = similarity
			bestMatch = candidate
		}
	}

	return bestMatch, bestScore
}

// MatchWithDetails performs fuzzy matching with detailed results
func (fm *FuzzyMatcher) MatchWithDetails(query string, candidates []string) []FuzzyMatchResult {
	results := make([]FuzzyMatchResult, 0)
	scorer := NewMatchConfidenceScorer()

	for _, candidate := range candidates {
		similarity := scorer.calculateStringSimilarity(query, candidate)
		if similarity >= fm.threshold {
			results = append(results, FuzzyMatchResult{
				Candidate:  candidate,
				Similarity: similarity,
				Distance:   levenshteinDistance(strings.ToLower(query), strings.ToLower(candidate)),
			})
		}
	}

	// Sort by similarity (highest first)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Similarity > results[i].Similarity {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}

// FuzzyMatchResult represents a fuzzy match result
type FuzzyMatchResult struct {
	Candidate  string
	Similarity float64
	Distance   int
}
