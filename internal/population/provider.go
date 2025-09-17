package population

import (
	"context"
	"fmt"
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

// LookupPopulation retrieves PSA population data for a specific card
func (p *PSAProvider) LookupPopulation(ctx context.Context, card model.Card) (*PopulationData, error) {
	if !p.Available() {
		return nil, fmt.Errorf("PSA provider not available")
	}

	// Create cache key
	cacheKey := fmt.Sprintf("psa_pop_%s_%s_%s", card.SetName, card.Name, card.Number)

	// Check cache first
	if cached, found := p.cache.Get(cacheKey); found {
		return cached, nil
	}

	// Rate limit the request
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	// Build API request URL
	url := fmt.Sprintf("%s/lookup?set=%s&card=%s&number=%s",
		p.baseURL, card.SetName, card.Name, card.Number)

	// Make the API request
	resp, err := p.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse the response (this would be implemented based on actual PSA API format)
	popData, err := p.parsePopulationResponse(resp.Body, card)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Cache the result
	cacheTTL := 24 * time.Hour // Population data doesn't change frequently
	if err := p.cache.Set(cacheKey, popData, cacheTTL); err != nil {
		// Log warning but don't fail the request
		fmt.Printf("Warning: failed to cache population data: %v\n", err)
	}

	return popData, nil
}

// BatchLookupPopulation retrieves population data for multiple cards efficiently
func (p *PSAProvider) BatchLookupPopulation(ctx context.Context, cards []model.Card) (map[string]*PopulationData, error) {
	if !p.Available() {
		return nil, fmt.Errorf("PSA provider not available")
	}

	results := make(map[string]*PopulationData)
	var uncachedCards []model.Card

	// Check cache for each card
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
	// This is a placeholder implementation
	// In a real implementation, this would parse the actual PSA API response format

	// For now, return mock data to demonstrate the structure
	gradePopulation := map[string]int{
		"PSA 10": 1250,
		"PSA 9":  2800,
		"PSA 8":  1920,
		"PSA 7":  890,
		"PSA 6":  445,
		"PSA 5":  234,
		"PSA 4":  128,
		"PSA 3":  67,
		"PSA 2":  23,
		"PSA 1":  12,
	}

	totalGraded := 0
	for _, count := range gradePopulation {
		totalGraded += count
	}

	// Determine scarcity level based on PSA 10 population
	psa10Count := gradePopulation["PSA 10"]
	var scarcityLevel string
	switch {
	case psa10Count > 5000:
		scarcityLevel = "COMMON"
	case psa10Count > 1000:
		scarcityLevel = "UNCOMMON"
	case psa10Count > 100:
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
		TotalGraded:     totalGraded,
		PSA10Population: psa10Count,
		PSA9Population:  gradePopulation["PSA 9"],
		PSA8Population:  gradePopulation["PSA 8"],
		QualifierCounts: map[string]int{
			"OC": 45, // Off-center
			"MC": 12, // Miscut
			"ST": 8,  // Staining
		},
		ScarcityLevel:   scarcityLevel,
		PopulationTrend: "INCREASING", // Would be calculated from historical data
	}, nil
}

func (p *PSAProvider) parseSetPopulationResponse(body []byte, setName string) (*SetPopulationData, error) {
	// Placeholder implementation
	return &SetPopulationData{
		SetName:     setName,
		LastUpdated: time.Now(),
		TotalCards:  200, // Example
		CardsGraded: 175, // Example
		CardData:    make(map[string]*PopulationData),
		SetStatistics: &SetStatistics{
			AveragePopulation: 1500.0,
			MedianPopulation:  1200,
			MostGradedCard:    "Charizard",
			LeastGradedCard:   "Energy Card",
			GradeDistribution: map[string]int{
				"PSA 10": 125000,
				"PSA 9":  280000,
				"PSA 8":  192000,
			},
			ScarcityBreakdown: map[string]int{
				"COMMON":     120,
				"UNCOMMON":   45,
				"RARE":       30,
				"ULTRA_RARE": 5,
			},
		},
	}, nil
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
