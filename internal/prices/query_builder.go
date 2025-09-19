package prices

import (
	"fmt"
	"strings"
)

// QueryBuilder creates optimized search queries for PriceCharting API
type QueryBuilder struct {
	baseQuery string
	filters   []string
	variant   string
	console   string
	region    string
	language  string
	condition string
	grader    string
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		filters: make([]string, 0),
	}
}

// SetBase sets the base query (card name, set, number)
func (qb *QueryBuilder) SetBase(setName, cardName, number string) *QueryBuilder {
	// Clean and normalize inputs
	setName = qb.normalizeSetName(setName)
	cardName = qb.normalizeCardName(cardName)

	qb.baseQuery = fmt.Sprintf("pokemon %s %s", setName, cardName)

	// Add number if provided
	if number != "" {
		qb.baseQuery += fmt.Sprintf(" #%s", number)
	}

	return qb
}

// WithVariant adds variant filter (1st Edition, Shadowless, etc.)
func (qb *QueryBuilder) WithVariant(variant string) *QueryBuilder {
	if variant != "" {
		qb.variant = variant
		// Map common variants to PriceCharting format
		switch strings.ToLower(variant) {
		case "1st edition", "first edition":
			qb.filters = append(qb.filters, "1st edition")
		case "shadowless":
			qb.filters = append(qb.filters, "shadowless")
		case "unlimited":
			qb.filters = append(qb.filters, "unlimited")
		case "reverse holo", "reverse":
			qb.filters = append(qb.filters, "reverse holo")
		case "holo":
			qb.filters = append(qb.filters, "holo")
		case "staff", "staff promo":
			qb.filters = append(qb.filters, "staff")
		case "prerelease":
			qb.filters = append(qb.filters, "prerelease")
		default:
			qb.filters = append(qb.filters, variant)
		}
	}
	return qb
}

// WithConsole adds console/platform filter
func (qb *QueryBuilder) WithConsole(console string) *QueryBuilder {
	if console != "" {
		qb.console = console
		// Pokemon TCG specific - typically for different game versions
		qb.filters = append(qb.filters, console)
	}
	return qb
}

// WithRegion adds region filter (USA, Japan, Europe)
func (qb *QueryBuilder) WithRegion(region string) *QueryBuilder {
	if region != "" {
		qb.region = region
		switch strings.ToLower(region) {
		case "japan", "japanese", "jp":
			qb.filters = append(qb.filters, "japanese")
			qb.language = "Japanese"
		case "usa", "us", "english", "en":
			// English is default, no filter needed unless explicitly excluding others
			qb.language = "English"
		case "europe", "eu", "european":
			qb.filters = append(qb.filters, "european")
		case "korea", "korean", "kr":
			qb.filters = append(qb.filters, "korean")
			qb.language = "Korean"
		default:
			qb.filters = append(qb.filters, region)
		}
	}
	return qb
}

// WithLanguage adds language filter
func (qb *QueryBuilder) WithLanguage(language string) *QueryBuilder {
	if language != "" && qb.language == "" {
		qb.language = language
		switch strings.ToLower(language) {
		case "japanese", "jp":
			qb.filters = append(qb.filters, "japanese")
		case "korean", "kr":
			qb.filters = append(qb.filters, "korean")
		case "french", "fr":
			qb.filters = append(qb.filters, "french")
		case "german", "de":
			qb.filters = append(qb.filters, "german")
		case "spanish", "es":
			qb.filters = append(qb.filters, "spanish")
		case "italian", "it":
			qb.filters = append(qb.filters, "italian")
			// English is default, no filter needed
		}
	}
	return qb
}

// WithCondition adds condition filter (mint, near mint, etc.)
func (qb *QueryBuilder) WithCondition(condition string) *QueryBuilder {
	if condition != "" {
		qb.condition = condition
		// Map to PriceCharting conditions
		switch strings.ToLower(condition) {
		case "mint", "m":
			qb.filters = append(qb.filters, "mint")
		case "near mint", "nm":
			qb.filters = append(qb.filters, "near mint")
		case "excellent", "ex":
			qb.filters = append(qb.filters, "excellent")
		case "good", "gd":
			qb.filters = append(qb.filters, "good")
		case "poor", "pr":
			qb.filters = append(qb.filters, "poor")
		case "graded":
			qb.filters = append(qb.filters, "graded")
		}
	}
	return qb
}

// WithGrader adds grading company filter (PSA, BGS, CGC)
func (qb *QueryBuilder) WithGrader(grader string) *QueryBuilder {
	if grader != "" {
		qb.grader = grader
		switch strings.ToUpper(grader) {
		case "PSA":
			qb.filters = append(qb.filters, "PSA")
		case "BGS", "BECKETT":
			qb.filters = append(qb.filters, "BGS")
		case "CGC":
			qb.filters = append(qb.filters, "CGC")
		case "SGC":
			qb.filters = append(qb.filters, "SGC")
		}
	}
	return qb
}

// Build creates the final query string
func (qb *QueryBuilder) Build() string {
	if qb.baseQuery == "" {
		return ""
	}

	query := qb.baseQuery

	// Add all filters
	for _, filter := range qb.filters {
		query += " " + filter
	}

	return strings.TrimSpace(query)
}

// BuildWithConfidence returns query with confidence score
func (qb *QueryBuilder) BuildWithConfidence() (string, float64) {
	query := qb.Build()
	confidence := qb.calculateConfidence()
	return query, confidence
}

// calculateConfidence estimates match confidence based on query specificity
func (qb *QueryBuilder) calculateConfidence() float64 {
	confidence := 0.5 // Base confidence

	// More specific queries have higher confidence
	if qb.baseQuery != "" {
		confidence += 0.2
	}

	// Each filter adds confidence
	confidence += float64(len(qb.filters)) * 0.05

	// Variant specificity adds confidence
	if qb.variant != "" {
		confidence += 0.1
	}

	// Language/region specificity
	if qb.language != "" || qb.region != "" {
		confidence += 0.05
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// normalizeSetName cleans and normalizes set names
func (qb *QueryBuilder) normalizeSetName(setName string) string {
	// Remove special characters that cause issues
	setName = strings.ReplaceAll(setName, ":", "")
	setName = strings.ReplaceAll(setName, "-", " ")
	// Keep & in set names to preserve original formatting

	// Handle common abbreviations
	replacements := map[string]string{
		"swsh": "Sword Shield",
		"sm":   "Sun Moon",
		"xy":   "XY",
		"bw":   "Black White",
		"sv":   "Scarlet Violet",
	}

	lower := strings.ToLower(setName)
	for abbr, full := range replacements {
		if strings.HasPrefix(lower, abbr) {
			setName = full + setName[len(abbr):]
			break
		}
	}

	return strings.TrimSpace(setName)
}

// normalizeCardName cleans and normalizes card names
func (qb *QueryBuilder) normalizeCardName(cardName string) string {
	// Handle "Reverse Holo" in card names - extract and add as variant
	if strings.Contains(cardName, "Reverse Holo") {
		cardName = strings.ReplaceAll(cardName, " Reverse Holo", " Reverse")
		qb.WithVariant("reverse holo")
	}

	// Remove card type suffixes for better matching
	suffixes := []string{
		" ex", " gx", " v", " vmax", " vstar",
		" EX", " GX", " V", " VMAX", " VSTAR",
		" Prime", " LEGEND", " BREAK", " Tag Team",
	}

	for _, suffix := range suffixes {
		if strings.HasSuffix(cardName, suffix) {
			// Keep original case for lowercase suffixes, uppercase for already uppercase
			baseName := strings.TrimSuffix(cardName, suffix)
			if strings.HasSuffix(cardName, strings.ToLower(suffix)) {
				// Keep lowercase suffixes as is
				return baseName + suffix
			} else {
				// Keep uppercase suffixes as is
				return baseName + suffix
			}
		}
	}

	return cardName
}

// QueryOptions represents advanced search options
type QueryOptions struct {
	Variant       string
	Region        string
	Language      string
	Condition     string
	Grader        string
	ExactMatch    bool
	IncludePromos bool
}

// BuildAdvancedQuery creates an optimized query with options
func BuildAdvancedQuery(setName, cardName, number string, options QueryOptions) string {
	qb := NewQueryBuilder().SetBase(setName, cardName, number)

	if options.Variant != "" {
		qb.WithVariant(options.Variant)
	}

	if options.Region != "" {
		qb.WithRegion(options.Region)
	}

	if options.Language != "" {
		qb.WithLanguage(options.Language)
	}

	if options.Condition != "" {
		qb.WithCondition(options.Condition)
	}

	if options.Grader != "" {
		qb.WithGrader(options.Grader)
	}

	query := qb.Build()

	// Add exact match operator if requested
	if options.ExactMatch && query != "" {
		query = "\"" + query + "\""
	}

	return query
}
