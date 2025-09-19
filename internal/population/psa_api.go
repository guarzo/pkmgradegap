package population

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
)

// PSAAPIProvider implements population data access through PSA's API
// Falls back to web scraping when API is not available
type PSAAPIProvider struct {
	apiKey      string
	baseURL     string
	client      *http.Client
	rateLimiter RateLimiter
	cache       Cache
	scraper     *PSAScraper // Fallback web scraper
}

// PSAAPIResponse represents the structure of PSA API responses
type PSAAPIResponse struct {
	Success bool        `json:"success"`
	Data    PSASpecData `json:"data"`
	Error   string      `json:"error,omitempty"`
}

// PSASpecData represents PSA spec population data
type PSASpecData struct {
	SpecID      int                `json:"specId"`
	Description string             `json:"description"`
	Brand       string             `json:"brand"`
	Category    string             `json:"category"`
	Sport       string             `json:"sport"`
	Year        string             `json:"year"`
	SetName     string             `json:"setName"`
	Population  PSAGradePopulation `json:"population"`
}

// PSAGradePopulation contains the grade breakdown
type PSAGradePopulation struct {
	Total      int            `json:"total"`
	Auth       int            `json:"auth"`
	Grades     map[string]int `json:"grades"`
	LastUpdate string         `json:"lastUpdate"`
}

// PSASearchResponse represents search results for finding spec IDs
type PSASearchResponse struct {
	Success bool      `json:"success"`
	Results []PSASpec `json:"results"`
	Error   string    `json:"error,omitempty"`
}

// PSASpec represents a PSA spec search result
type PSASpec struct {
	SpecID      int    `json:"specId"`
	Description string `json:"description"`
	Brand       string `json:"brand"`
	Category    string `json:"category"`
	Year        string `json:"year"`
	SetName     string `json:"setName"`
}

// NewPSAAPIProvider creates a new PSA API provider with web scraper fallback
func NewPSAAPIProvider(apiKey string, rateLimiter RateLimiter, cache Cache) *PSAAPIProvider {
	return &PSAAPIProvider{
		apiKey:      apiKey,
		baseURL:     "https://api.psacard.com/publicapi",
		client:      &http.Client{Timeout: 30 * time.Second},
		rateLimiter: rateLimiter,
		cache:       cache,
		scraper:     NewPSAScraper(cache), // Pass cache directly to scraper
	}
}

// Available returns true if the API key is configured
func (p *PSAAPIProvider) Available() bool {
	return p.apiKey != "" && p.apiKey != "test" && p.apiKey != "mock"
}

// GetProviderName returns the name of the provider
func (p *PSAAPIProvider) GetProviderName() string {
	return "PSA API"
}

// IsMockMode returns false since this is a real provider
func (p *PSAAPIProvider) IsMockMode() bool {
	return false
}

// LookupPopulation retrieves population data for a specific card
func (p *PSAAPIProvider) LookupPopulation(ctx context.Context, card model.Card) (*PopulationData, error) {
	// Try API first if available
	if p.Available() {
		popData, err := p.lookupViaAPI(ctx, card)
		if err == nil && popData != nil {
			return popData, nil
		}
		// Log API failure but continue to scraper
		fmt.Printf("PSA API lookup failed, falling back to scraper: %v\n", err)
	}

	// Fall back to web scraping
	if p.scraper != nil {
		return p.lookupViaScraper(ctx, card)
	}

	return nil, fmt.Errorf("PSA data not available (no API key and scraper disabled)")
}

// lookupViaAPI uses the PSA API to get population data
func (p *PSAAPIProvider) lookupViaAPI(ctx context.Context, card model.Card) (*PopulationData, error) {

	// Check cache first
	cacheKey := fmt.Sprintf("psa_api_%s_%s_%s",
		normalizeSetName(card.SetName),
		strings.ReplaceAll(card.Name, " ", "_"),
		card.Number)

	if cached, found := p.cache.Get(cacheKey); found {
		return cached, nil
	}

	// Rate limit the request
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// First, find the spec ID for this card
	specID, err := p.findSpecID(ctx, card)
	if err != nil {
		return nil, fmt.Errorf("failed to find PSA spec ID: %w", err)
	}

	// Get population data for the spec ID
	popData, err := p.getPopulationBySpecID(ctx, specID)
	if err != nil {
		return nil, fmt.Errorf("failed to get population data: %w", err)
	}

	// Cache the result for 24 hours
	if err := p.cache.Set(cacheKey, popData, 24*time.Hour); err != nil {
		// Log warning but don't fail
		fmt.Printf("Warning: failed to cache PSA population data: %v\n", err)
	}

	return popData, nil
}

// lookupViaScraper uses web scraping to get population data
func (p *PSAAPIProvider) lookupViaScraper(ctx context.Context, card model.Card) (*PopulationData, error) {
	if p.scraper == nil {
		return nil, fmt.Errorf("scraper not initialized")
	}

	// Use scraper to get population data
	psaPop, err := p.scraper.GetCardPopulation(ctx, card.SetName, card.Number, card.Name)
	if err != nil {
		return nil, fmt.Errorf("scraper failed: %w", err)
	}

	if psaPop == nil {
		return nil, fmt.Errorf("no population data found")
	}

	// Convert model.PSAPopulation to PopulationData
	scarcity := calculateScarcity(psaPop.PSA10)

	// Build grade population map
	gradePopulation := make(map[string]int)
	gradePopulation["PSA 10"] = psaPop.PSA10
	gradePopulation["PSA 9"] = psaPop.PSA9
	gradePopulation["PSA 8"] = psaPop.PSA8

	return &PopulationData{
		Card:            card,
		SetName:         card.SetName,
		CardNumber:      card.Number,
		LastUpdated:     psaPop.LastUpdated,
		GradePopulation: gradePopulation,
		TotalGraded:     psaPop.TotalGraded,
		PSA10Population: psaPop.PSA10,
		PSA9Population:  psaPop.PSA9,
		PSA8Population:  psaPop.PSA8,
		ScarcityLevel:   scarcity,
		PopulationTrend: "STABLE", // Would need historical data
	}, nil
}

// BatchLookupPopulation retrieves population data for multiple cards
func (p *PSAAPIProvider) BatchLookupPopulation(ctx context.Context, cards []model.Card) (map[string]*PopulationData, error) {
	results := make(map[string]*PopulationData)

	// For now, use individual lookups since PSA doesn't have a batch endpoint
	// In the future, this could be optimized with goroutines and better batching
	for _, card := range cards {
		cardKey := fmt.Sprintf("%s-%s", card.Number, card.Name)

		popData, err := p.LookupPopulation(ctx, card)
		if err != nil {
			// Log error but continue with other cards
			fmt.Printf("Warning: failed to lookup PSA population for %s: %v\n", cardKey, err)
			continue
		}

		if popData != nil {
			results[cardKey] = popData
		}
	}

	return results, nil
}

// GetSetPopulation retrieves population summary for an entire set
func (p *PSAAPIProvider) GetSetPopulation(ctx context.Context, setName string) (*SetPopulationData, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("psa_set_%s", normalizeSetName(setName))
	if cached, found := p.cache.GetSet(cacheKey); found {
		return cached, nil
	}

	// Search for all cards in the set
	specs, err := p.searchSetSpecs(ctx, setName)
	if err != nil {
		return nil, fmt.Errorf("failed to search set specs: %w", err)
	}

	// Get population data for each spec
	setData := &SetPopulationData{
		SetName:     setName,
		LastUpdated: time.Now(),
		CardData:    make(map[string]*PopulationData),
	}

	totalGraded := 0
	gradeDistribution := make(map[string]int)
	scarcityBreakdown := make(map[string]int)

	for _, spec := range specs {
		popData, err := p.getPopulationBySpecID(ctx, spec.SpecID)
		if err != nil {
			fmt.Printf("Warning: failed to get population for spec %d: %v\n", spec.SpecID, err)
			continue
		}

		cardKey := fmt.Sprintf("%s-%s", extractCardNumber(spec.Description), extractCardName(spec.Description))
		setData.CardData[cardKey] = popData

		totalGraded += popData.TotalGraded
		gradeDistribution["PSA 10"] += popData.PSA10Population
		gradeDistribution["PSA 9"] += popData.PSA9Population
		gradeDistribution["PSA 8"] += popData.PSA8Population
		scarcityBreakdown[popData.ScarcityLevel]++
	}

	setData.TotalCards = len(specs)
	setData.CardsGraded = len(setData.CardData)
	setData.SetStatistics = &SetStatistics{
		AveragePopulation: float64(totalGraded) / float64(len(setData.CardData)),
		GradeDistribution: gradeDistribution,
		ScarcityBreakdown: scarcityBreakdown,
	}

	// Cache for 12 hours
	if err := p.cache.SetSet(cacheKey, setData, 12*time.Hour); err != nil {
		fmt.Printf("Warning: failed to cache set population data: %v\n", err)
	}

	return setData, nil
}

// findSpecID searches for the PSA spec ID of a card
func (p *PSAAPIProvider) findSpecID(ctx context.Context, card model.Card) (int, error) {
	// Build search query
	query := fmt.Sprintf("%s %s %s pokemon", card.SetName, card.Name, card.Number)

	// URL encode the query
	encodedQuery := url.QueryEscape(query)

	// Make search request
	searchURL := fmt.Sprintf("%s/search?q=%s&category=pokemon&limit=10", p.baseURL, encodedQuery)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("search API returned status %d", resp.StatusCode)
	}

	var searchResp PSASearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return 0, fmt.Errorf("failed to decode search response: %w", err)
	}

	if !searchResp.Success {
		return 0, fmt.Errorf("search failed: %s", searchResp.Error)
	}

	// Find the best match
	for _, spec := range searchResp.Results {
		if p.isCardMatch(card, spec) {
			return spec.SpecID, nil
		}
	}

	return 0, fmt.Errorf("no matching spec found for card %s #%s", card.Name, card.Number)
}

// getPopulationBySpecID retrieves population data for a specific PSA spec ID
func (p *PSAAPIProvider) getPopulationBySpecID(ctx context.Context, specID int) (*PopulationData, error) {
	popURL := fmt.Sprintf("%s/population/%d", p.baseURL, specID)

	req, err := http.NewRequestWithContext(ctx, "GET", popURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("population API returned status %d", resp.StatusCode)
	}

	var apiResp PSAAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode population response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("population lookup failed: %s", apiResp.Error)
	}

	// Convert PSA API response to our PopulationData format
	return p.convertToPopulationData(apiResp.Data), nil
}

// searchSetSpecs searches for all specs in a given set
func (p *PSAAPIProvider) searchSetSpecs(ctx context.Context, setName string) ([]PSASpec, error) {
	query := fmt.Sprintf("set:\"%s\" pokemon", setName)
	encodedQuery := url.QueryEscape(query)

	searchURL := fmt.Sprintf("%s/search?q=%s&category=pokemon&limit=1000", p.baseURL, encodedQuery)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("set search API returned status %d", resp.StatusCode)
	}

	var searchResp PSASearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode set search response: %w", err)
	}

	if !searchResp.Success {
		return nil, fmt.Errorf("set search failed: %s", searchResp.Error)
	}

	return searchResp.Results, nil
}

// convertToPopulationData converts PSA API data to our internal format
func (p *PSAAPIProvider) convertToPopulationData(spec PSASpecData) *PopulationData {
	// Extract card information from description
	cardName := extractCardName(spec.Description)
	cardNumber := extractCardNumber(spec.Description)

	// Build grade population map
	gradePopulation := make(map[string]int)
	for grade, count := range spec.Population.Grades {
		gradePopulation[fmt.Sprintf("PSA %s", grade)] = count
	}

	// Get specific grade counts
	psa10 := spec.Population.Grades["10"]
	psa9 := spec.Population.Grades["9"]
	psa8 := spec.Population.Grades["8"]

	// Parse last update time
	lastUpdated := time.Now()
	if spec.Population.LastUpdate != "" {
		if parsed, err := time.Parse("2006-01-02", spec.Population.LastUpdate); err == nil {
			lastUpdated = parsed
		}
	}

	// Calculate scarcity level
	scarcity := calculateScarcity(psa10)

	return &PopulationData{
		Card: model.Card{
			Name:    cardName,
			SetName: spec.SetName,
			Number:  cardNumber,
		},
		SetName:         spec.SetName,
		CardNumber:      cardNumber,
		LastUpdated:     lastUpdated,
		GradePopulation: gradePopulation,
		TotalGraded:     spec.Population.Total,
		PSA10Population: psa10,
		PSA9Population:  psa9,
		PSA8Population:  psa8,
		ScarcityLevel:   scarcity,
		PopulationTrend: "STABLE", // Would need historical data to determine
	}
}

// isCardMatch checks if a PSA spec matches our card
func (p *PSAAPIProvider) isCardMatch(card model.Card, spec PSASpec) bool {
	// Normalize names for comparison
	specName := strings.ToLower(spec.Description)
	cardName := strings.ToLower(card.Name)
	cardNumber := strings.ToLower(card.Number)
	setName := strings.ToLower(normalizeSetName(card.SetName))
	specSetName := strings.ToLower(normalizeSetName(spec.SetName))

	// Check if set names match
	if !strings.Contains(specSetName, setName) && !strings.Contains(setName, specSetName) {
		return false
	}

	// Check if card name is in the description
	if !strings.Contains(specName, cardName) {
		return false
	}

	// Check if card number is in the description
	if cardNumber != "" && !strings.Contains(specName, cardNumber) {
		return false
	}

	return true
}

// Helper functions for parsing PSA descriptions
func extractCardName(description string) string {
	// PSA descriptions typically follow patterns like:
	// "2022 Pokemon Japanese VMAX Climax Charizard VMAX #003/184"
	// "1998 Pokemon Japanese Base Set Charizard #006"

	parts := strings.Fields(description)

	// Look for the card name (usually after "Pokemon" and before "#")
	var nameStart, nameEnd int

	for i, part := range parts {
		if strings.ToLower(part) == "pokemon" {
			nameStart = i + 1
			// Skip language/set descriptors
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

	// Find where the name ends (usually at "#" or end of string)
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

func extractCardNumber(description string) string {
	// Look for patterns like "#123", "#123/456", etc.
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

func calculateScarcity(psa10Count int) string {
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
