package sales

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
)

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
			Price:       basePrice * 0.9,
			Grade:       "Raw",
			SaleDate:    time.Now().AddDate(0, 0, -7),
			Title:       fmt.Sprintf("%s #%s Near Mint", cardName, number),
			Marketplace: "eBay",
			URL:         fmt.Sprintf("https://ebay.com/mock/%s", number),
		},
		{
			Price:       basePrice * 4.5,
			Grade:       "PSA 10",
			SaleDate:    time.Now().AddDate(0, 0, -5),
			Title:       fmt.Sprintf("%s #%s PSA 10 GEM MINT", cardName, number),
			Marketplace: "eBay",
			URL:         fmt.Sprintf("https://ebay.com/mock/psa10-%s", number),
		},
		{
			Price:       basePrice * 3.2,
			Grade:       "PSA 9",
			SaleDate:    time.Now().AddDate(0, 0, -3),
			Title:       fmt.Sprintf("%s #%s PSA 9 MINT", cardName, number),
			Marketplace: "eBay",
			URL:         fmt.Sprintf("https://ebay.com/mock/psa9-%s", number),
		},
		{
			Price:       basePrice * 1.1,
			Grade:       "Raw",
			SaleDate:    time.Now().AddDate(0, 0, -2),
			Title:       fmt.Sprintf("%s #%s LP/NM", cardName, number),
			Marketplace: "eBay",
			URL:         fmt.Sprintf("https://ebay.com/mock/%s-2", number),
		},
		{
			Price:       basePrice * 4.8,
			Grade:       "PSA 10",
			SaleDate:    time.Now().AddDate(0, 0, -1),
			Title:       fmt.Sprintf("%s #%s PSA 10", cardName, number),
			Marketplace: "PWCC",
			URL:         fmt.Sprintf("https://pwcc.com/mock/%s", number),
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
		Card: model.Card{
			ID:      fmt.Sprintf("mock-%s", number),
			Name:    cardName,
			SetName: setName,
			Number:  number,
		},
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
