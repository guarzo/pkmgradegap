package population

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
)

// Provider defines the interface for PSA population data providers
type Provider interface {
	// Available returns true if the provider is configured and ready to use
	Available() bool

	// LookupPopulation retrieves PSA population data for a specific card
	LookupPopulation(ctx context.Context, card model.Card) (*PopulationData, error)

	// BatchLookupPopulation retrieves population data for multiple cards efficiently
	BatchLookupPopulation(ctx context.Context, cards []model.Card) (map[string]*PopulationData, error)

	// GetSetPopulation retrieves population summary for an entire set
	GetSetPopulation(ctx context.Context, setName string) (*SetPopulationData, error)

	// GetProviderName returns the name of the provider
	GetProviderName() string

	// IsMockMode returns true if the provider is running in mock/test mode
	IsMockMode() bool
}

// PopulationData represents PSA grading population data for a single card
type PopulationData struct {
	Card            model.Card     `json:"card"`
	SetName         string         `json:"set_name"`
	CardNumber      string         `json:"card_number"`
	LastUpdated     time.Time      `json:"last_updated"`
	GradePopulation map[string]int `json:"grade_population"` // Grade → Population count
	TotalGraded     int            `json:"total_graded"`
	PSA10Population int            `json:"psa10_population"`
	PSA9Population  int            `json:"psa9_population"`
	PSA8Population  int            `json:"psa8_population"`
	QualifierCounts map[string]int `json:"qualifier_counts"` // OC, MC, etc. → count
	ScarcityLevel   string         `json:"scarcity_level"`   // "COMMON", "UNCOMMON", "RARE", "ULTRA_RARE"
	PopulationTrend string         `json:"population_trend"` // "INCREASING", "STABLE", "DECREASING"
}

// SetPopulationData represents population data for an entire set
type SetPopulationData struct {
	SetName       string                     `json:"set_name"`
	LastUpdated   time.Time                  `json:"last_updated"`
	TotalCards    int                        `json:"total_cards"`
	CardsGraded   int                        `json:"cards_graded"`
	CardData      map[string]*PopulationData `json:"card_data"` // CardKey → PopulationData
	SetStatistics *SetStatistics             `json:"set_statistics"`
}

// SetStatistics contains aggregate statistics for a set
type SetStatistics struct {
	AveragePopulation float64        `json:"average_population"`
	MedianPopulation  int            `json:"median_population"`
	MostGradedCard    string         `json:"most_graded_card"`
	LeastGradedCard   string         `json:"least_graded_card"`
	GradeDistribution map[string]int `json:"grade_distribution"` // Grade → Total count across set
	ScarcityBreakdown map[string]int `json:"scarcity_breakdown"` // Scarcity level → count
}

// PSAProvider implements the Provider interface for PSA population reports
type PSAProvider struct {
	apiKey      string
	baseURL     string
	httpClient  HTTPClient
	cache       Cache
	rateLimiter RateLimiter
}

// HTTPClient interface for making HTTP requests
type HTTPClient interface {
	Get(url string) (*HTTPResponse, error)
	Post(url string, data []byte) (*HTTPResponse, error)
}

// HTTPResponse represents an HTTP response
type HTTPResponse struct {
	StatusCode int
	Body       []byte
	Headers    map[string]string
}

// Cache interface for caching population data
type Cache interface {
	Get(key string) (*PopulationData, bool)
	Set(key string, data *PopulationData, ttl time.Duration) error
	GetSet(key string) (*SetPopulationData, bool)
	SetSet(key string, data *SetPopulationData, ttl time.Duration) error
	Clear() error
}

// RateLimiter interface for controlling request rates
type RateLimiter interface {
	Wait(ctx context.Context) error
	Allow() bool
}

// PSASetSearchResponse represents the response when searching for all cards in a set
type PSASetSearchResponse struct {
	Success bool             `json:"success"`
	Results []PSASpecWithPop `json:"results"`
	Error   string           `json:"error,omitempty"`
}

// PSASpecWithPop represents a PSA spec with population data (for set searches)
type PSASpecWithPop struct {
	SpecID      int                `json:"specId"`
	Description string             `json:"description"`
	Brand       string             `json:"brand"`
	Category    string             `json:"category"`
	Year        string             `json:"year"`
	SetName     string             `json:"setName"`
	Population  PSAGradePopulation `json:"population"`
}

// NewPSAProvider creates a new PSA population provider
func NewPSAProvider(apiKey string, httpClient HTTPClient, cache Cache, rateLimiter RateLimiter) *PSAProvider {
	return &PSAProvider{
		apiKey:      apiKey,
		baseURL:     "https://api.psacard.com/publicapi/population", // Hypothetical API endpoint
		httpClient:  httpClient,
		cache:       cache,
		rateLimiter: rateLimiter,
	}
}

// Available returns true if the PSA provider is configured and ready
func (p *PSAProvider) Available() bool {
	return p.apiKey != "" && p.httpClient != nil
}

// GetProviderName returns the name of the provider
func (p *PSAProvider) GetProviderName() string {
	return "PSA Population"
}

// IsMockMode returns false since this is a real provider
func (p *PSAProvider) IsMockMode() bool {
	return false
}

// LookupPopulation retrieves PSA population data for a specific card
func (p *PSAProvider) LookupPopulation(ctx context.Context, card model.Card) (*PopulationData, error) {
	if !p.Available() {
		log.Printf("PSA provider not available - no API key configured")
		return nil, fmt.Errorf("PSA provider not available")
	}

	// Validate input parameters
	if card.SetName == "" || card.Name == "" {
		log.Printf("Invalid card parameters: SetName='%s', Name='%s', Number='%s'",
			card.SetName, card.Name, card.Number)
		return nil, fmt.Errorf("invalid card parameters: SetName and Name are required")
	}

	// Create cache key
	cacheKey := fmt.Sprintf("psa_pop_%s_%s_%s",
		strings.ReplaceAll(card.SetName, " ", "_"),
		strings.ReplaceAll(card.Name, " ", "_"),
		card.Number)

	log.Printf("Looking up PSA population for %s #%s from set '%s'",
		card.Name, card.Number, card.SetName)

	// Check cache first
	if cached, found := p.cache.Get(cacheKey); found {
		log.Printf("Found cached PSA population data for %s #%s", card.Name, card.Number)
		return cached, nil
	}

	// Rate limit the request
	if err := p.rateLimiter.Wait(ctx); err != nil {
		log.Printf("Rate limiting failed for PSA lookup: %v", err)
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	// Build API request URL with proper encoding
	url := fmt.Sprintf("%s/lookup?set=%s&card=%s&number=%s",
		p.baseURL,
		strings.ReplaceAll(card.SetName, " ", "%20"),
		strings.ReplaceAll(card.Name, " ", "%20"),
		card.Number)

	log.Printf("Making PSA API request to: %s", url)

	// Make the API request
	resp, err := p.httpClient.Get(url)
	if err != nil {
		log.Printf("PSA API request failed: %v", err)
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	if resp.StatusCode != 200 {
		log.Printf("PSA API returned non-200 status: %d", resp.StatusCode)

		// Handle common HTTP error codes
		switch resp.StatusCode {
		case 401:
			return nil, fmt.Errorf("PSA API authentication failed - check API key")
		case 403:
			return nil, fmt.Errorf("PSA API access forbidden - insufficient permissions")
		case 404:
			return nil, fmt.Errorf("PSA API endpoint not found")
		case 429:
			return nil, fmt.Errorf("PSA API rate limit exceeded")
		case 500, 502, 503, 504:
			return nil, fmt.Errorf("PSA API server error (status %d)", resp.StatusCode)
		default:
			return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
		}
	}

	// Validate response body
	if len(resp.Body) == 0 {
		log.Printf("PSA API returned empty response body")
		return nil, fmt.Errorf("received empty response from PSA API")
	}

	log.Printf("PSA API response received, parsing %d bytes", len(resp.Body))

	// Parse the response
	popData, err := p.parsePopulationResponse(resp.Body, card)
	if err != nil {
		log.Printf("Failed to parse PSA API response: %v", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if popData == nil {
		log.Printf("Parsed PSA response but got nil data for %s #%s", card.Name, card.Number)
		return nil, fmt.Errorf("no population data found for card")
	}

	log.Printf("Successfully parsed PSA population data: Total=%d, PSA10=%d, PSA9=%d, PSA8=%d",
		popData.TotalGraded, popData.PSA10Population, popData.PSA9Population, popData.PSA8Population)

	// Cache the result
	cacheTTL := 24 * time.Hour // Population data doesn't change frequently
	if err := p.cache.Set(cacheKey, popData, cacheTTL); err != nil {
		// Log warning but don't fail the request
		log.Printf("Warning: failed to cache PSA population data for %s #%s: %v",
			card.Name, card.Number, err)
	} else {
		log.Printf("Cached PSA population data for %s #%s", card.Name, card.Number)
	}

	return popData, nil
}

// BatchLookupPopulation retrieves population data for multiple cards efficiently.
// This method implements a cache-first strategy to minimize API calls:
// 1. Checks cache for each card using set/name/number as key
// 2. Collects uncached cards for batch API lookup
// 3. Processes API results and updates cache with TTL
// 4. Returns combined cached and fresh results
func (p *PSAProvider) BatchLookupPopulation(ctx context.Context, cards []model.Card) (map[string]*PopulationData, error) {
	if !p.Available() {
		return nil, fmt.Errorf("PSA provider not available")
	}

	results := make(map[string]*PopulationData)
	var uncachedCards []model.Card

	// Check cache for each card using standardized cache key format
	for _, card := range cards {
		cacheKey := fmt.Sprintf("psa_pop_%s_%s_%s", card.SetName, card.Name, card.Number)
		cardKey := fmt.Sprintf("%s-%s", card.Number, card.Name)

		if cached, found := p.cache.Get(cacheKey); found {
			results[cardKey] = cached
		} else {
			uncachedCards = append(uncachedCards, card)
		}
	}

	// Batch request for uncached cards
	if len(uncachedCards) > 0 {
		batchResults, err := p.performBatchLookup(ctx, uncachedCards)
		if err != nil {
			return nil, fmt.Errorf("batch lookup failed: %w", err)
		}

		// Merge batch results
		for key, data := range batchResults {
			results[key] = data
		}
	}

	return results, nil
}

// GetSetPopulation retrieves population summary for an entire set
func (p *PSAProvider) GetSetPopulation(ctx context.Context, setName string) (*SetPopulationData, error) {
	if !p.Available() {
		return nil, fmt.Errorf("PSA provider not available")
	}

	// Check cache first
	cacheKey := fmt.Sprintf("psa_set_%s", setName)
	if cached, found := p.cache.GetSet(cacheKey); found {
		return cached, nil
	}

	// Rate limit the request
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	// Build API request URL for set data
	url := fmt.Sprintf("%s/set?name=%s", p.baseURL, setName)

	// Make the API request
	resp, err := p.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("set API request failed: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("set API returned status %d", resp.StatusCode)
	}

	// Parse the response
	setData, err := p.parseSetPopulationResponse(resp.Body, setName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse set response: %w", err)
	}

	// Cache the result
	cacheTTL := 12 * time.Hour // Set data changes less frequently
	if err := p.cache.SetSet(cacheKey, setData, cacheTTL); err != nil {
		fmt.Printf("Warning: failed to cache set population data: %v\n", err)
	}

	return setData, nil
}

// Helper methods for parsing responses (these would be implemented based on actual PSA API format)

func (p *PSAProvider) parsePopulationResponse(body []byte, card model.Card) (*PopulationData, error) {
	log.Printf("Parsing PSA API response for %s #%s", card.Name, card.Number)

	if len(body) == 0 {
		log.Printf("Empty response body received")
		return nil, fmt.Errorf("empty response body")
	}

	// Log first 200 characters of response for debugging (avoid logging sensitive data)
	if len(body) > 200 {
		log.Printf("PSA API response preview: %s...", string(body[:200]))
	} else {
		log.Printf("PSA API response: %s", string(body))
	}

	var apiResp PSAAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		log.Printf("JSON unmarshaling failed: %v", err)
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if !apiResp.Success {
		if apiResp.Error != "" {
			log.Printf("PSA API returned error: %s", apiResp.Error)
			return nil, fmt.Errorf("PSA API error: %s", apiResp.Error)
		}
		log.Printf("PSA API request unsuccessful with no error message")
		return nil, fmt.Errorf("PSA API request unsuccessful")
	}

	// Validate that we have population data
	if apiResp.Data.Population.Grades == nil {
		log.Printf("PSA API response missing grade population data")
		return nil, fmt.Errorf("response missing grade population data")
	}

	if apiResp.Data.Population.Total == 0 && len(apiResp.Data.Population.Grades) == 0 {
		log.Printf("PSA API response contains no population data")
		return nil, fmt.Errorf("no population data found in response")
	}

	log.Printf("PSA API response valid, converting to internal format")
	return p.convertSpecDataToPopulationData(apiResp.Data, card)
}

// convertSpecDataToPopulationData converts PSA API spec data to PopulationData
func (p *PSAProvider) convertSpecDataToPopulationData(spec PSASpecData, card model.Card) (*PopulationData, error) {
	log.Printf("Converting PSA spec data for SpecID=%d, Description='%s'",
		spec.SpecID, spec.Description)

	// Validate spec data
	if spec.Population.Grades == nil {
		log.Printf("Spec data missing grade population map")
		return nil, fmt.Errorf("spec data missing grade population information")
	}

	// Build grade population map from API response
	gradePopulation := make(map[string]int)
	totalCalculated := 0

	for grade, count := range spec.Population.Grades {
		if count < 0 {
			log.Printf("Warning: negative population count for grade %s: %d", grade, count)
			count = 0 // Sanitize negative values
		}
		gradeKey := fmt.Sprintf("PSA %s", grade)
		gradePopulation[gradeKey] = count
		totalCalculated += count
		log.Printf("Grade %s: %d cards", gradeKey, count)
	}

	// Validate population consistency
	if spec.Population.Total > 0 && totalCalculated > 0 {
		difference := spec.Population.Total - totalCalculated
		if difference > 10 || difference < -10 { // Allow small discrepancies
			log.Printf("Warning: population total mismatch - API reports %d, calculated %d (diff: %d)",
				spec.Population.Total, totalCalculated, difference)
		}
	}

	// Extract specific grade counts with defaults
	psa10Count := spec.Population.Grades["10"]
	psa9Count := spec.Population.Grades["9"]
	psa8Count := spec.Population.Grades["8"]

	log.Printf("Key grades - PSA 10: %d, PSA 9: %d, PSA 8: %d",
		psa10Count, psa9Count, psa8Count)

	// Parse last update time with multiple format attempts
	lastUpdated := time.Now()
	if spec.Population.LastUpdate != "" {
		formats := []string{
			"2006-01-02",
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05-07:00",
			"2006-01-02 15:04:05",
		}

		var parseErr error
		for _, format := range formats {
			if parsed, err := time.Parse(format, spec.Population.LastUpdate); err == nil {
				lastUpdated = parsed
				log.Printf("Parsed last update time: %v", lastUpdated)
				parseErr = nil
				break
			} else {
				parseErr = err
			}
		}

		if parseErr != nil {
			log.Printf("Warning: failed to parse last update time '%s': %v",
				spec.Population.LastUpdate, parseErr)
		}
	}

	// Calculate scarcity level based on PSA 10 population
	scarcityLevel := p.calculateScarcityLevel(psa10Count)
	log.Printf("Calculated scarcity level: %s", scarcityLevel)

	// Calculate population trend (simplified - would need historical data for accurate trends)
	trend := p.calculatePopulationTrend(spec.Population.Total, lastUpdated)
	log.Printf("Calculated population trend: %s", trend)

	// Extract qualifier counts if available (PSA sometimes provides separate data for qualified grades)
	qualifierCounts := make(map[string]int)
	for grade, count := range spec.Population.Grades {
		if strings.Contains(grade, "Q") { // Qualified grades like "10Q", "9Q"
			baseGrade := strings.TrimSuffix(grade, "Q")
			qualifierKey := fmt.Sprintf("PSA %s OC", baseGrade)
			qualifierCounts[qualifierKey] = count
			log.Printf("Found qualifier grade: %s = %d", qualifierKey, count)
		}
	}

	totalGraded := spec.Population.Total
	if totalGraded == 0 {
		totalGraded = totalCalculated // Use calculated total if API doesn't provide one
	}

	log.Printf("Final population data - Total: %d, Scarcity: %s, Trend: %s",
		totalGraded, scarcityLevel, trend)

	return &PopulationData{
		Card:            card,
		SetName:         card.SetName,
		CardNumber:      card.Number,
		LastUpdated:     lastUpdated,
		GradePopulation: gradePopulation,
		TotalGraded:     totalGraded,
		PSA10Population: psa10Count,
		PSA9Population:  psa9Count,
		PSA8Population:  psa8Count,
		QualifierCounts: qualifierCounts,
		ScarcityLevel:   scarcityLevel,
		PopulationTrend: trend,
	}, nil
}

func (p *PSAProvider) parseSetPopulationResponse(body []byte, setName string) (*SetPopulationData, error) {
	var searchResp PSASetSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse set search JSON response: %w", err)
	}

	if !searchResp.Success {
		if searchResp.Error != "" {
			return nil, fmt.Errorf("PSA set search API error: %s", searchResp.Error)
		}
		return nil, fmt.Errorf("PSA set search API request unsuccessful")
	}

	// Initialize set population data
	setData := &SetPopulationData{
		SetName:     setName,
		LastUpdated: time.Now(),
		TotalCards:  len(searchResp.Results),
		CardData:    make(map[string]*PopulationData),
		SetStatistics: &SetStatistics{
			GradeDistribution: make(map[string]int),
			ScarcityBreakdown: make(map[string]int),
		},
	}

	// Aggregate statistics
	var totalGraded int
	var populations []int
	var mostGradedCount, leastGradedCount int
	var mostGradedCard, leastGradedCard string

	// Process each card spec in the set
	for _, spec := range searchResp.Results {
		// Convert spec to PopulationData
		cardPopData := p.convertSpecToPopulationData(spec, setName)

		// Create card key
		cardKey := fmt.Sprintf("%s-%s", cardPopData.CardNumber, cardPopData.Card.Name)
		setData.CardData[cardKey] = cardPopData

		// Update aggregate statistics
		totalGraded += cardPopData.TotalGraded
		populations = append(populations, cardPopData.TotalGraded)

		// Track most/least graded cards
		if mostGradedCard == "" || cardPopData.TotalGraded > mostGradedCount {
			mostGradedCard = cardPopData.Card.Name
			mostGradedCount = cardPopData.TotalGraded
		}
		if leastGradedCard == "" || cardPopData.TotalGraded < leastGradedCount {
			leastGradedCard = cardPopData.Card.Name
			leastGradedCount = cardPopData.TotalGraded
		}

		// Accumulate grade distribution
		for grade, count := range cardPopData.GradePopulation {
			setData.SetStatistics.GradeDistribution[grade] += count
		}

		// Accumulate scarcity breakdown
		setData.SetStatistics.ScarcityBreakdown[cardPopData.ScarcityLevel]++
	}

	// Calculate final statistics
	setData.CardsGraded = len(setData.CardData)

	if len(populations) > 0 {
		setData.SetStatistics.AveragePopulation = float64(totalGraded) / float64(len(populations))
		setData.SetStatistics.MedianPopulation = p.calculateMedian(populations)
	}

	setData.SetStatistics.MostGradedCard = mostGradedCard
	setData.SetStatistics.LeastGradedCard = leastGradedCard

	return setData, nil
}

func (p *PSAProvider) performBatchLookup(ctx context.Context, cards []model.Card) (map[string]*PopulationData, error) {
	// For now, fall back to individual lookups
	// In a real implementation, this would use a batch API endpoint if available
	results := make(map[string]*PopulationData)

	for _, card := range cards {
		cardKey := fmt.Sprintf("%s-%s", card.Number, card.Name)

		popData, err := p.LookupPopulation(ctx, card)
		if err != nil {
			// Log error but continue with other cards
			fmt.Printf("Warning: failed to lookup population for %s: %v\n", cardKey, err)
			continue
		}

		results[cardKey] = popData
	}

	return results, nil
}

// MockProvider provides a mock implementation for testing and development
type MockProvider struct {
	enabled bool
}

// NewMockProvider creates a new mock population provider
func NewMockProvider() *MockProvider {
	return &MockProvider{enabled: true}
}

// Available returns true if the mock provider is enabled
func (m *MockProvider) Available() bool {
	return m.enabled
}

// GetProviderName returns the name of the provider
func (m *MockProvider) GetProviderName() string {
	return "Population Mock"
}

// IsMockMode returns true since this is a mock provider
func (m *MockProvider) IsMockMode() bool {
	return true
}

// LookupPopulation returns mock population data
func (m *MockProvider) LookupPopulation(ctx context.Context, card model.Card) (*PopulationData, error) {
	if !m.enabled {
		return nil, fmt.Errorf("mock provider disabled")
	}

	// Generate deterministic but varied mock data based on card
	hash := simpleHash(card.Name + card.Number)
	psa10Pop := 500 + (hash % 2000) // 500-2500 range

	gradePopulation := map[string]int{
		"PSA 10": psa10Pop,
		"PSA 9":  psa10Pop * 2,
		"PSA 8":  psa10Pop * 3 / 2,
		"PSA 7":  psa10Pop / 2,
		"PSA 6":  psa10Pop / 4,
	}

	var scarcityLevel string
	switch {
	case psa10Pop > 2000:
		scarcityLevel = "COMMON"
	case psa10Pop > 1000:
		scarcityLevel = "UNCOMMON"
	case psa10Pop > 500:
		scarcityLevel = "RARE"
	default:
		scarcityLevel = "ULTRA_RARE"
	}

	return &PopulationData{
		Card:            card,
		SetName:         card.SetName,
		CardNumber:      card.Number,
		LastUpdated:     time.Now(),
		GradePopulation: gradePopulation,
		TotalGraded:     psa10Pop * 6, // Rough estimate
		PSA10Population: psa10Pop,
		PSA9Population:  gradePopulation["PSA 9"],
		PSA8Population:  gradePopulation["PSA 8"],
		ScarcityLevel:   scarcityLevel,
		PopulationTrend: "STABLE",
	}, nil
}

// BatchLookupPopulation returns mock data for multiple cards
func (m *MockProvider) BatchLookupPopulation(ctx context.Context, cards []model.Card) (map[string]*PopulationData, error) {
	results := make(map[string]*PopulationData)

	for _, card := range cards {
		cardKey := fmt.Sprintf("%s-%s", card.Number, card.Name)
		popData, err := m.LookupPopulation(ctx, card)
		if err != nil {
			return nil, err
		}
		results[cardKey] = popData
	}

	return results, nil
}

// GetSetPopulation returns mock set population data
func (m *MockProvider) GetSetPopulation(ctx context.Context, setName string) (*SetPopulationData, error) {
	return &SetPopulationData{
		SetName:     setName,
		LastUpdated: time.Now(),
		TotalCards:  200,
		CardsGraded: 180,
		CardData:    make(map[string]*PopulationData),
		SetStatistics: &SetStatistics{
			AveragePopulation: 1200.0,
			MedianPopulation:  900,
			MostGradedCard:    "Popular Card",
			LeastGradedCard:   "Rare Card",
			GradeDistribution: map[string]int{
				"PSA 10": 180000,
				"PSA 9":  360000,
				"PSA 8":  270000,
			},
			ScarcityBreakdown: map[string]int{
				"COMMON":     100,
				"UNCOMMON":   60,
				"RARE":       35,
				"ULTRA_RARE": 5,
			},
		},
	}, nil
}

// calculateScarcityLevel determines scarcity based on PSA 10 population
func (p *PSAProvider) calculateScarcityLevel(psa10Count int) string {
	switch {
	case psa10Count <= 10:
		return "ULTRA_RARE"
	case psa10Count <= 50:
		return "RARE"
	case psa10Count <= 500:
		return "UNCOMMON"
	default:
		return "COMMON"
	}
}

// calculatePopulationTrend calculates population trend based on total and age
func (p *PSAProvider) calculatePopulationTrend(totalGraded int, lastUpdated time.Time) string {
	// Simplified trend calculation - in real implementation would use historical data
	daysSinceUpdate := time.Since(lastUpdated).Hours() / 24

	// If data is very recent (< 7 days) and population is high, likely increasing
	if daysSinceUpdate < 7 && totalGraded > 1000 {
		return "INCREASING"
	}

	// If data is old (> 30 days) or population is low, likely stable
	if daysSinceUpdate > 30 || totalGraded < 100 {
		return "STABLE"
	}

	// Default to stable for intermediate cases
	return "STABLE"
}

// convertSpecToPopulationData converts a PSASpecWithPop to PopulationData
func (p *PSAProvider) convertSpecToPopulationData(spec PSASpecWithPop, setName string) *PopulationData {
	// Extract card info from description
	cardName := p.extractCardNameFromDescription(spec.Description)
	cardNumber := p.extractCardNumberFromDescription(spec.Description)

	// Build grade population map
	gradePopulation := make(map[string]int)
	for grade, count := range spec.Population.Grades {
		gradePopulation[fmt.Sprintf("PSA %s", grade)] = count
	}

	// Get specific grade counts
	psa10Count := spec.Population.Grades["10"]
	psa9Count := spec.Population.Grades["9"]
	psa8Count := spec.Population.Grades["8"]

	// Parse last update time
	lastUpdated := time.Now()
	if spec.Population.LastUpdate != "" {
		if parsed, err := time.Parse("2006-01-02", spec.Population.LastUpdate); err == nil {
			lastUpdated = parsed
		} else if parsed, err := time.Parse("2006-01-02T15:04:05Z", spec.Population.LastUpdate); err == nil {
			lastUpdated = parsed
		}
	}

	return &PopulationData{
		Card: model.Card{
			Name:    cardName,
			SetName: setName,
			Number:  cardNumber,
		},
		SetName:         setName,
		CardNumber:      cardNumber,
		LastUpdated:     lastUpdated,
		GradePopulation: gradePopulation,
		TotalGraded:     spec.Population.Total,
		PSA10Population: psa10Count,
		PSA9Population:  psa9Count,
		PSA8Population:  psa8Count,
		QualifierCounts: make(map[string]int), // Could be enhanced to parse from grades
		ScarcityLevel:   p.calculateScarcityLevel(psa10Count),
		PopulationTrend: p.calculatePopulationTrend(spec.Population.Total, lastUpdated),
	}
}

// calculateMedian calculates the median of a slice of integers
func (p *PSAProvider) calculateMedian(numbers []int) int {
	if len(numbers) == 0 {
		return 0
	}

	// Sort the numbers (create a copy to avoid modifying original)
	sorted := make([]int, len(numbers))
	copy(sorted, numbers)

	// Simple sorting (for better performance, could use sort.Ints)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

// extractCardNameFromDescription extracts card name from PSA description
func (p *PSAProvider) extractCardNameFromDescription(description string) string {
	// PSA descriptions typically follow patterns like:
	// "2022 Pokemon Japanese VMAX Climax Charizard VMAX #003/184"
	// Extract the part before the "#"
	parts := strings.Fields(description)

	var nameStart, nameEnd int
	for i, part := range parts {
		if strings.ToLower(part) == "pokemon" {
			nameStart = i + 1
			// Skip descriptors like "Japanese", "English", set names
			for nameStart < len(parts) &&
				(strings.ToLower(parts[nameStart]) == "japanese" ||
					strings.ToLower(parts[nameStart]) == "english" ||
					strings.Contains(strings.ToLower(parts[nameStart]), "set") ||
					strings.Contains(strings.ToLower(parts[nameStart]), "promo")) {
				nameStart++
			}
			break
		}
	}

	// Find where name ends (at "#" or end)
	nameEnd = len(parts)
	for i := nameStart; i < len(parts); i++ {
		if strings.HasPrefix(parts[i], "#") {
			nameEnd = i
			break
		}
	}

	if nameStart >= nameEnd {
		return "Unknown"
	}

	return strings.Join(parts[nameStart:nameEnd], " ")
}

// extractCardNumberFromDescription extracts card number from PSA description
func (p *PSAProvider) extractCardNumberFromDescription(description string) string {
	// Look for patterns like "#123", "#123/456"
	parts := strings.Fields(description)

	for _, part := range parts {
		if strings.HasPrefix(part, "#") {
			number := strings.TrimPrefix(part, "#")
			// Handle cases like "#123/456" - we want just "123"
			if slashIndex := strings.Index(number, "/"); slashIndex != -1 {
				number = number[:slashIndex]
			}
			return number
		}
	}

	return ""
}

// Simple hash function for generating deterministic mock data
func simpleHash(s string) int {
	hash := 0
	for _, c := range s {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}
