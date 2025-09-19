package sales

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
)

// SalesData represents sales transaction data for a card
type SalesData struct {
	CardName     string       `json:"cardName"`
	SetName      string       `json:"setName"`
	CardNumber   string       `json:"cardNumber"`
	LastUpdated  time.Time    `json:"lastUpdated"`
	SaleCount    int          `json:"saleCount"`
	AveragePrice float64      `json:"averagePrice"`
	MedianPrice  float64      `json:"medianPrice"`
	RecentSales  []SaleRecord `json:"recentSales"`
	DataSource   string       `json:"dataSource"`
}

// SaleRecord represents a single sale transaction
type SaleRecord struct {
	Date     time.Time `json:"date"`
	Price    float64   `json:"price"`
	Grade    string    `json:"grade,omitempty"`
	Platform string    `json:"platform,omitempty"`
	Title    string    `json:"title,omitempty"`
}

// Provider interface for sales data providers
type Provider interface {
	Available() bool
	GetProviderName() string
	IsMockMode() bool
	GetSalesData(setName, cardName, number string) (*SalesData, error)
}

// Config holds configuration for sales providers
type Config struct {
	PokemonPriceTrackerAPIKey string
	PokemonPriceTrackerURL    string
	CacheEnabled              bool
	CacheTTLMinutes           int
	RequestTimeout            time.Duration
	MaxRetries                int
	RateLimitPerMin           int
}

// MockProvider implements a mock sales data provider for testing
type MockProvider struct {
	// Can be configured to simulate different scenarios
	FailureRate float32
	DelayMS     int
}

// NewMockProvider creates a new mock provider
func NewMockProvider() *MockProvider {
	return &MockProvider{
		FailureRate: 0.0,
		DelayMS:     0,
	}
}

// Available always returns true for mock provider
func (m *MockProvider) Available() bool {
	return true
}

// GetProviderName returns the provider name
func (m *MockProvider) GetProviderName() string {
	return "MockSalesProvider"
}

// IsMockMode returns true since this is a mock provider
func (m *MockProvider) IsMockMode() bool {
	return true
}

// NewProvider creates a new sales provider (returns mock for now)
func NewProvider(config Config) Provider {
	return NewMockProvider()
}

// calculateAverage calculates the average of a slice of prices
func calculateAverage(prices []float64) float64 {
	if len(prices) == 0 {
		return 0
	}
	sum := 0.0
	for _, p := range prices {
		sum += p
	}
	return sum / float64(len(prices))
}

// calculateMedian calculates the median of a slice of prices
func calculateMedian(prices []float64) float64 {
	if len(prices) == 0 {
		return 0
	}
	if len(prices) == 1 {
		return prices[0]
	}

	// Simple median calculation (would need sorting in real implementation)
	return prices[len(prices)/2]
}

// GetSalesData returns mock sales data for testing
func (m *MockProvider) GetSalesData(setName, cardName, number string) (*SalesData, error) {
	// Simulate delay if configured
	if m.DelayMS > 0 {
		time.Sleep(time.Duration(m.DelayMS) * time.Millisecond)
	}

	// Simulate random failures if configured
	if m.FailureRate > 0 && rand.Float32() < m.FailureRate {
		return nil, fmt.Errorf("mock provider simulated failure")
	}

	// Generate deterministic mock data based on card name
	basePrice := float64(len(cardName) * 10) // Price based on name length for consistency

	// Create mock sales records
	sales := []SaleRecord{
		{
			Price:    basePrice * 0.9,
			Grade:    "Raw",
			Date:     time.Now().AddDate(0, 0, -7),
			Title:    fmt.Sprintf("%s #%s Near Mint", cardName, number),
			Platform: "eBay",
		},
		{
			Price:    basePrice * 4.5,
			Grade:    "PSA 10",
			Date:     time.Now().AddDate(0, 0, -5),
			Title:    fmt.Sprintf("%s #%s PSA 10 GEM MINT", cardName, number),
			Platform: "eBay",
		},
		{
			Price:    basePrice * 3.2,
			Grade:    "PSA 9",
			Date:     time.Now().AddDate(0, 0, -3),
			Title:    fmt.Sprintf("%s #%s PSA 9 MINT", cardName, number),
			Platform: "eBay",
		},
		{
			Price:    basePrice * 1.1,
			Grade:    "Raw",
			Date:     time.Now().AddDate(0, 0, -2),
			Title:    fmt.Sprintf("%s #%s LP/NM", cardName, number),
			Platform: "eBay",
		},
		{
			Price:    basePrice * 4.8,
			Grade:    "PSA 10",
			Date:     time.Now().AddDate(0, 0, -1),
			Title:    fmt.Sprintf("%s #%s PSA 10", cardName, number),
			Platform: "PWCC",
		},
	}

	// Calculate median and average for raw cards
	var rawPrices []float64
	for _, sale := range sales {
		if sale.Grade == "Raw" {
			rawPrices = append(rawPrices, sale.Price)
		}
	}

	medianPrice := basePrice // Default
	avgPrice := basePrice
	if len(rawPrices) > 0 {
		medianPrice = calculateMedian(rawPrices)
		avgPrice = calculateAverage(rawPrices)
	}

	return &SalesData{
		CardName:     cardName,
		SetName:      setName,
		CardNumber:   number,
		RecentSales:  sales,
		MedianPrice:  medianPrice,
		AveragePrice: avgPrice,
		SaleCount:    len(sales),
		LastUpdated:  time.Now(),
		DataSource:   "Mock",
	}, nil
}

// GetBulkSalesData returns mock sales data for multiple cards
func (m *MockProvider) GetBulkSalesData(cards []model.Card) (map[string]*SalesData, error) {
	results := make(map[string]*SalesData)

	for _, card := range cards {
		salesData, err := m.GetSalesData(card.SetName, card.Name, card.Number)
		if err != nil {
			// Skip failed cards in bulk mode
			continue
		}

		cardKey := fmt.Sprintf("%s-%s", card.Number, card.Name)
		results[cardKey] = salesData
	}

	return results, nil
}
