package sales

import (
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
)

// SalesData represents actual sales data for a card
type SalesData struct {
	Card         model.Card   `json:"card"`
	RecentSales  []SaleRecord `json:"recent_sales"`
	MedianPrice  float64      `json:"median_price"`
	AveragePrice float64      `json:"average_price"`
	SaleCount    int          `json:"sale_count"`
	LastUpdated  time.Time    `json:"last_updated"`
	DataSource   string       `json:"data_source"`
}

// SaleRecord represents a single sale transaction
type SaleRecord struct {
	Price       float64   `json:"price"`
	Grade       string    `json:"grade"` // "PSA 10", "BGS 9.5", "Raw", etc.
	SaleDate    time.Time `json:"sale_date"`
	Title       string    `json:"title"`
	Marketplace string    `json:"marketplace"` // "eBay", "PWCC", etc.
	URL         string    `json:"url,omitempty"`
}

// Provider defines the interface for sales data providers
type Provider interface {
	// Available returns true if the provider is configured and accessible
	Available() bool

	// GetSalesData retrieves sales data for a specific card
	GetSalesData(setName, cardName, number string) (*SalesData, error)

	// GetBulkSalesData retrieves sales data for multiple cards efficiently
	GetBulkSalesData(cards []model.Card) (map[string]*SalesData, error)

	// GetProviderName returns the name of the provider
	GetProviderName() string
}

// Config holds configuration for sales data providers
type Config struct {
	// PokemonPriceTracker API
	PokemonPriceTrackerAPIKey string
	PokemonPriceTrackerURL    string

	// General settings
	CacheEnabled    bool
	CacheTTLMinutes int
	RequestTimeout  time.Duration
	RateLimitPerMin int
	MaxRetries      int

	// Provider preferences (ordered by priority)
	EnabledProviders []string
}

// Factory creates the appropriate sales provider
func NewProvider(config Config) Provider {
	// For now, we'll implement PokemonPriceTracker first
	// Future: return a composite provider that tries multiple sources
	if config.PokemonPriceTrackerAPIKey != "" {
		return NewPokemonPriceTrackerProvider(config)
	}

	// Fallback to mock provider for development/testing
	return NewMockProvider()
}
