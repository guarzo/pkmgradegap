package gamestop

import (
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
)

// ListingData represents GameStop graded card listings
type ListingData struct {
	Card         model.Card `json:"card"`
	ActiveList   []Listing  `json:"active_listings"`
	LowestPrice  float64    `json:"lowest_price"`
	AveragePrice float64    `json:"average_price"`
	ListingCount int        `json:"listing_count"`
	LastUpdated  time.Time  `json:"last_updated"`
	DataSource   string     `json:"data_source"`
}

// Listing represents a single GameStop graded card listing
type Listing struct {
	Price       float64 `json:"price"`
	Grade       string  `json:"grade"`       // "PSA 10", "BGS 9.5", etc.
	Title       string  `json:"title"`       // Full product title
	URL         string  `json:"url"`         // Product URL
	SKU         string  `json:"sku"`         // GameStop SKU
	InStock     bool    `json:"in_stock"`    // Availability status
	Condition   string  `json:"condition"`   // "New", "Pre-owned", etc.
	Seller      string  `json:"seller"`      // Always "GameStop" for this provider
	ImageURL    string  `json:"image_url"`   // Product image
	Description string  `json:"description"` // Product description
}

// Provider defines the interface for GameStop listings data
type Provider interface {
	// Available returns true if the provider is configured and accessible
	Available() bool

	// GetListings retrieves listing data for a specific card
	GetListings(setName, cardName, number string) (*ListingData, error)

	// SearchCards searches for cards matching query terms
	SearchCards(query string) ([]Listing, error)

	// GetBulkListings retrieves listing data for multiple cards efficiently
	GetBulkListings(cards []model.Card) (map[string]*ListingData, error)

	// GetProviderName returns the name of the provider
	GetProviderName() string

	// IsMockMode returns true if the provider is running in mock/test mode
	IsMockMode() bool
}

// Config holds configuration for GameStop provider
type Config struct {
	// Request settings
	RequestTimeout     time.Duration
	MaxRetries         int
	RateLimitPerMin    int
	MaxListingsPerCard int

	// Cache settings
	CacheEnabled    bool
	CacheTTLMinutes int

	// Search settings
	MaxSearchResults int
	SearchTimeout    time.Duration

	// Headers and user agent rotation
	UserAgents  []string
	UseRandomUA bool

	// Rate limiting
	RequestDelay time.Duration
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() Config {
	return Config{
		RequestTimeout:     30 * time.Second,
		MaxRetries:         3,
		RateLimitPerMin:    10, // Conservative to avoid blocking
		MaxListingsPerCard: 5,
		CacheEnabled:       true,
		CacheTTLMinutes:    60, // 1 hour cache
		MaxSearchResults:   25,
		SearchTimeout:      45 * time.Second,
		UserAgents: []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
		},
		UseRandomUA:  true,
		RequestDelay: 2 * time.Second,
	}
}

// NewProvider creates a GameStop web scraper client
// Note: This uses web scraping, not an official API
func NewProvider(config Config) Provider {
	return NewGameStopClient(config)
}
