package gamestop

import (
	"regexp"
	"strings"
	"time"
	// "github.com/guarzo/pkmgradegap/internal/fusion" // TODO: Update when fusion package is refactored
)

// ConvertToPriceData converts GameStop listings to fusion.PriceData for the fusion engine
// TODO: Update when fusion package is refactored
func ConvertToPriceData(listingData *ListingData) []interface{} { // []fusion.PriceData {
	if listingData == nil || len(listingData.ActiveList) == 0 {
		return []interface{}{} // []fusion.PriceData{}
	}

	// priceData := make([]fusion.PriceData, 0, len(listingData.ActiveList))
	priceData := make([]interface{}, 0, len(listingData.ActiveList))

	for _, listing := range listingData.ActiveList {
		if !listing.InStock || listing.Price <= 0 {
			continue // Skip out-of-stock or invalid listings
		}

		// Determine if this is raw or graded
		isRaw := isRawCard(listing.Grade, listing.Title)
		grade := normalizeGrade(listing.Grade)

		// Calculate confidence based on listing quality
		confidence := calculateListingConfidence(listing)

		// data := fusion.PriceData{
		// 	Value:    listing.Price,
		// 	Currency: "USD", // GameStop is USD
		// 	Source: fusion.DataSource{
		// 		Name:       "GameStop",
		// 		Type:       fusion.SourceTypeListing,
		// 		Freshness:  calculateFreshness(listingData.LastUpdated),
		// 		Volume:     listingData.ListingCount,
		// 		Confidence: confidence,
		// 		Timestamp:  listingData.LastUpdated,
		// 	},
		// 	Raw:   isRaw,
		// 	Grade: grade,
		// }
		data := map[string]interface{}{
			"value":      listing.Price,
			"currency":   "USD",
			"source":     "GameStop",
			"freshness":  calculateFreshness(listingData.LastUpdated),
			"confidence": confidence,
			"raw":        isRaw,
			"grade":      grade,
		}

		priceData = append(priceData, data)
	}

	return priceData
}

// ConvertToPriceDataByGrade converts GameStop listings grouped by grade
// TODO: Update when fusion package is refactored
func ConvertToPriceDataByGrade(listingData *ListingData) map[string][]interface{} { // map[string][]fusion.PriceData {
	// result := make(map[string][]fusion.PriceData)
	result := make(map[string][]interface{})

	if listingData == nil || len(listingData.ActiveList) == 0 {
		return result
	}

	for _, listing := range listingData.ActiveList {
		if !listing.InStock || listing.Price <= 0 {
			continue
		}

		grade := normalizeGrade(listing.Grade)
		key := getGradeKey(grade, listing.Title)

		confidence := calculateListingConfidence(listing)

		// data := fusion.PriceData{
		// 	Value:    listing.Price,
		// 	Currency: "USD",
		// 	Source: fusion.DataSource{
		// 		Name:       "GameStop",
		// 		Type:       fusion.SourceTypeListing,
		// 		Freshness:  calculateFreshness(listingData.LastUpdated),
		// 		Volume:     listingData.ListingCount,
		// 		Confidence: confidence,
		// 		Timestamp:  listingData.LastUpdated,
		// 	},
		// 	Raw:   isRawCard(grade, listing.Title),
		// 	Grade: grade,
		// }
		data := map[string]interface{}{
			"value":      listing.Price,
			"currency":   "USD",
			"source":     "GameStop",
			"confidence": confidence,
			"raw":        isRawCard(grade, listing.Title),
			"grade":      grade,
		}

		result[key] = append(result[key], data)
	}

	return result
}

// GetLowestPriceByGrade returns the lowest price for each grade
func GetLowestPriceByGrade(listingData *ListingData) map[string]float64 {
	result := make(map[string]float64)

	if listingData == nil || len(listingData.ActiveList) == 0 {
		return result
	}

	for _, listing := range listingData.ActiveList {
		if !listing.InStock || listing.Price <= 0 {
			continue
		}

		grade := normalizeGrade(listing.Grade)
		key := getGradeKey(grade, listing.Title)

		if existing, exists := result[key]; !exists || listing.Price < existing {
			result[key] = listing.Price
		}
	}

	return result
}

func isRawCard(grade, title string) bool {
	gradeLower := strings.ToLower(grade)
	titleLower := strings.ToLower(title)

	// Check for explicit raw card indicators first
	rawIndicators := []string{
		"raw", "ungraded", "nm", "near mint",
		"lp", "light play", "mp", "moderate play",
		"excellent", "very fine", "fine",
		"mint", // plain "mint" without "gem mint" often indicates raw
	}

	for _, indicator := range rawIndicators {
		if strings.Contains(gradeLower, indicator) || strings.Contains(titleLower, indicator) {
			return true
		}
	}

	// Check for graded card indicators
	gradedIndicators := []string{
		"psa", "bgs", "cgc", "sgc",
		"graded", "gem mint",
		"authenticated", "certified",
	}

	for _, indicator := range gradedIndicators {
		if strings.Contains(gradeLower, indicator) || strings.Contains(titleLower, indicator) {
			return false
		}
	}

	// If grade is "Unknown" or empty, check title for grading info
	if gradeLower == "unknown" || gradeLower == "" {
		// Look for PSA/BGS numbers in title
		gradePattern := `(PSA|BGS|CGC|SGC)\s*(\d+(?:\.\d+)?)`
		re := regexp.MustCompile(`(?i)` + gradePattern)
		if re.MatchString(titleLower) {
			return false
		}
		// If no grading info found, assume raw
		return true
	}

	// Default to graded if we have a specific grade
	return false
}

func normalizeGrade(grade string) string {
	if grade == "" || strings.ToLower(grade) == "unknown" {
		return "Unknown"
	}

	// Normalize common grading formats
	grade = strings.TrimSpace(grade)

	// Handle PSA grades
	psaPattern := `(?i)PSA\s*(\d+(?:\.\d+)?)`
	re := regexp.MustCompile(psaPattern)
	if matches := re.FindStringSubmatch(grade); len(matches) > 1 {
		return "PSA " + matches[1]
	}

	// Handle BGS grades
	bgsPattern := `(?i)BGS\s*(\d+(?:\.\d+)?)`
	re = regexp.MustCompile(bgsPattern)
	if matches := re.FindStringSubmatch(grade); len(matches) > 1 {
		return "BGS " + matches[1]
	}

	// Handle CGC grades
	cgcPattern := `(?i)CGC\s*(\d+(?:\.\d+)?)`
	re = regexp.MustCompile(cgcPattern)
	if matches := re.FindStringSubmatch(grade); len(matches) > 1 {
		return "CGC " + matches[1]
	}

	// Handle SGC grades
	sgcPattern := `(?i)SGC\s*(\d+(?:\.\d+)?)`
	re = regexp.MustCompile(sgcPattern)
	if matches := re.FindStringSubmatch(grade); len(matches) > 1 {
		return "SGC " + matches[1]
	}

	return strings.ToUpper(grade)
}

func getGradeKey(grade, title string) string {
	// Map grades to standard keys used by the analysis system
	gradeLower := strings.ToLower(grade)
	titleLower := strings.ToLower(title)

	// Check specific grades first (most specific to least specific)
	if strings.Contains(gradeLower, "psa 10") || strings.Contains(titleLower, "psa 10") {
		return "psa10"
	}
	if strings.Contains(gradeLower, "bgs 10") || strings.Contains(titleLower, "bgs 10") {
		return "bgs10"
	}

	// Check for 9.5 grades (BGS 9.5, CGC 9.5) before checking for 9 grades
	if strings.Contains(gradeLower, "9.5") || strings.Contains(titleLower, "9.5") ||
		strings.Contains(gradeLower, "cgc") || strings.Contains(titleLower, "cgc") {
		return "cgc95"
	}

	// Check for 9 grades (PSA 9, BGS 9)
	if (strings.Contains(gradeLower, "psa 9") && !strings.Contains(gradeLower, "9.5")) ||
		(strings.Contains(gradeLower, "bgs 9") && !strings.Contains(gradeLower, "9.5")) ||
		(strings.Contains(titleLower, "psa 9") && !strings.Contains(titleLower, "9.5")) ||
		(strings.Contains(titleLower, "bgs 9") && !strings.Contains(titleLower, "9.5")) {
		return "psa9"
	}

	// Check if raw
	if isRawCard(grade, title) {
		return "raw"
	}

	// Default grouping for other grades
	return "other"
}

func calculateListingConfidence(listing Listing) float64 {
	confidence := 0.5 // Base confidence for listings

	// Increase confidence based on listing quality
	if listing.SKU != "" {
		confidence += 0.1
	}
	if listing.ImageURL != "" {
		confidence += 0.1
	}
	if listing.Description != "" {
		confidence += 0.1
	}

	// Grade clarity increases confidence
	if listing.Grade != "Unknown" && listing.Grade != "" {
		confidence += 0.15
	}

	// Title quality (length and detail)
	if len(listing.Title) > 30 {
		confidence += 0.05
	}

	// Stock status
	if listing.InStock {
		confidence += 0.1
	} else {
		confidence -= 0.2
	}

	// Ensure confidence is between 0 and 1
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	}

	return confidence
}

func calculateFreshness(lastUpdated time.Time) time.Duration {
	return time.Since(lastUpdated)
}

// Helper to merge GameStop data with other sources in the fusion engine
// TODO: Update when fusion package is refactored
func MergeWithFusionEngine(engine interface{}, gameStopData *ListingData,
	otherPrices map[string][]interface{}) map[string]interface{} { // map[string]fusion.FusedPrice {

	// result := make(map[string]fusion.FusedPrice)
	result := make(map[string]interface{})

	// Convert GameStop data by grade
	gameStopPrices := ConvertToPriceDataByGrade(gameStopData)

	// Merge with other sources for each grade
	allGrades := []string{"raw", "psa9", "cgc95", "psa10", "bgs10"}

	for _, grade := range allGrades {
		// var combinedPrices []fusion.PriceData
		var combinedPrices []interface{}

		// Add GameStop prices for this grade
		if gsPrice, exists := gameStopPrices[grade]; exists {
			combinedPrices = append(combinedPrices, gsPrice...)
		}

		// Add other source prices for this grade
		if otherPrice, exists := otherPrices[grade]; exists {
			combinedPrices = append(combinedPrices, otherPrice...)
		}

		// Fuse the prices
		if len(combinedPrices) > 0 {
			// result[grade] = engine.FusePrice(combinedPrices)
			result[grade] = combinedPrices // TODO: Implement fusion logic
		}
	}

	return result
}
