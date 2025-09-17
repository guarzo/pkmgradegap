package sales

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
	"github.com/guarzo/pkmgradegap/internal/ratelimit"
)

// PokemonPriceTrackerProvider implements the Provider interface using PokemonPriceTracker API
type PokemonPriceTrackerProvider struct {
	apiKey      string
	baseURL     string
	httpClient  *http.Client
	rateLimiter *ratelimit.Limiter
}

// PokemonPriceTrackerResponse represents the API response structure
type PokemonPriceTrackerResponse struct {
	Card struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Set      string `json:"set"`
		Number   string `json:"number"`
		ImageURL string `json:"image_url"`
	} `json:"card"`
	Prices struct {
		TCGPlayerMarket float64 `json:"tcgplayer_market"`
		EbayAverage     float64 `json:"ebay_average"`
		EbayMedian      float64 `json:"ebay_median"`
		GradedPrices    struct {
			PSA10 float64 `json:"psa_10"`
			PSA9  float64 `json:"psa_9"`
			PSA8  float64 `json:"psa_8"`
			BGS10 float64 `json:"bgs_10"`
			BGS95 float64 `json:"bgs_9_5"`
		} `json:"graded_prices"`
	} `json:"prices"`
	Sales []struct {
		Price       float64   `json:"price"`
		Grade       string    `json:"grade"`
		Date        time.Time `json:"date"`
		Title       string    `json:"title"`
		Marketplace string    `json:"marketplace"`
		URL         string    `json:"url"`
	} `json:"recent_sales"`
	LastUpdated time.Time `json:"last_updated"`
}

// NewPokemonPriceTrackerProvider creates a new PokemonPriceTracker provider
func NewPokemonPriceTrackerProvider(config Config) *PokemonPriceTrackerProvider {
	baseURL := config.PokemonPriceTrackerURL
	if baseURL == "" {
		baseURL = "https://www.pokemonpricetracker.com/api"
	}

	return &PokemonPriceTrackerProvider{
		apiKey:  config.PokemonPriceTrackerAPIKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: config.RequestTimeout,
		},
		rateLimiter: ratelimit.NewLimiter(config.RateLimitPerMin, time.Minute),
	}
}

// Available returns true if the provider is configured
func (p *PokemonPriceTrackerProvider) Available() bool {
	return p.apiKey != "" && p.apiKey != "test" && p.apiKey != "mock"
}

// GetProviderName returns the provider name
func (p *PokemonPriceTrackerProvider) GetProviderName() string {
	return "PokemonPriceTracker"
}

// GetSalesData retrieves sales data for a specific card
func (p *PokemonPriceTrackerProvider) GetSalesData(setName, cardName, number string) (*SalesData, error) {
	if !p.Available() {
		return nil, fmt.Errorf("PokemonPriceTracker provider not available")
	}

	// Wait for rate limit
	p.rateLimiter.Wait()

	// Build the API request
	cardID := p.buildCardID(setName, cardName, number)
	endpoint := fmt.Sprintf("%s/prices", p.baseURL)

	// Create request with query parameters
	params := url.Values{}
	params.Add("id", cardID)
	params.Add("includeSales", "true")
	params.Add("includeGraded", "true")

	req, err := http.NewRequest("GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Add authentication header
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "pkmgradegap/1.0")

	// Make the request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse the response
	var apiResp PokemonPriceTrackerResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	// Convert to our sales data format
	salesData := &SalesData{
		Card: model.Card{
			ID:      apiResp.Card.ID,
			Name:    apiResp.Card.Name,
			SetName: apiResp.Card.Set,
			Number:  apiResp.Card.Number,
		},
		RecentSales: make([]SaleRecord, len(apiResp.Sales)),
		SaleCount:   len(apiResp.Sales),
		LastUpdated: apiResp.LastUpdated,
		DataSource:  "PokemonPriceTracker",
	}

	// Convert sales records
	var prices []float64
	for i, sale := range apiResp.Sales {
		salesData.RecentSales[i] = SaleRecord{
			Price:       sale.Price,
			Grade:       sale.Grade,
			SaleDate:    sale.Date,
			Title:       sale.Title,
			Marketplace: sale.Marketplace,
			URL:         sale.URL,
		}
		if sale.Price > 0 {
			prices = append(prices, sale.Price)
		}
	}

	// Calculate median and average
	if len(prices) > 0 {
		salesData.MedianPrice = calculateMedian(prices)
		salesData.AveragePrice = calculateAverage(prices)
	}

	return salesData, nil
}

// GetBulkSalesData retrieves sales data for multiple cards efficiently
func (p *PokemonPriceTrackerProvider) GetBulkSalesData(cards []model.Card) (map[string]*SalesData, error) {
	if !p.Available() {
		return nil, fmt.Errorf("PokemonPriceTracker provider not available")
	}

	results := make(map[string]*SalesData)

	// Process cards individually for now
	// Future enhancement: Use bulk API endpoint if available
	for _, card := range cards {
		salesData, err := p.GetSalesData(card.SetName, card.Name, card.Number)
		if err != nil {
			// Log error but continue with other cards
			fmt.Printf("Warning: Failed to get sales data for %s #%s: %v\n", card.Name, card.Number, err)
			continue
		}

		cardKey := fmt.Sprintf("%s-%s", card.Number, card.Name)
		results[cardKey] = salesData
	}

	return results, nil
}

// buildCardID creates a card identifier for the API
func (p *PokemonPriceTrackerProvider) buildCardID(setName, cardName, number string) string {
	// PokemonPriceTracker may use different ID formats
	// This is a guess - we'll need to adjust based on their actual API
	setID := p.normalizeSetName(setName)
	return fmt.Sprintf("%s-%s", setID, number)
}

// normalizeSetName converts set names to API format
func (p *PokemonPriceTrackerProvider) normalizeSetName(setName string) string {
	// Convert common set names to API format
	// This mapping will need to be expanded based on actual API requirements
	mapping := map[string]string{
		"Base Set":            "base1",
		"Surging Sparks":      "sv8",
		"Stellar Crown":       "sv7",
		"Twilight Masquerade": "sv6",
	}

	if apiName, exists := mapping[setName]; exists {
		return apiName
	}

	// Fallback: convert to lowercase and replace spaces with dashes
	return strings.ToLower(strings.ReplaceAll(setName, " ", "-"))
}

// Helper functions for calculations
func calculateMedian(prices []float64) float64 {
	if len(prices) == 0 {
		return 0
	}

	// Simple bubble sort for small datasets
	sorted := make([]float64, len(prices))
	copy(sorted, prices)

	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

func calculateAverage(prices []float64) float64 {
	if len(prices) == 0 {
		return 0
	}

	sum := 0.0
	for _, price := range prices {
		sum += price
	}
	return sum / float64(len(prices))
}
