package gamestop

import (
	"fmt"
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
)

// MockProvider implements the Provider interface for testing
type MockProvider struct{}

func (m *MockProvider) Available() bool {
	return true
}

func (m *MockProvider) GetProviderName() string {
	return "GameStopMock"
}

func (m *MockProvider) GetListings(setName, cardName, number string) (*ListingData, error) {
	// Generate mock listings based on card info
	listings := m.generateMockListings(setName, cardName, number)

	if len(listings) == 0 {
		return &ListingData{
			Card: model.Card{
				Name:   cardName,
				Number: number,
			},
			ActiveList:   []Listing{},
			ListingCount: 0,
			LastUpdated:  time.Now(),
			DataSource:   "GameStopMock",
		}, nil
	}

	// Calculate stats
	var total float64
	var lowest float64 = listings[0].Price

	for _, listing := range listings {
		total += listing.Price
		if listing.Price < lowest {
			lowest = listing.Price
		}
	}

	return &ListingData{
		Card: model.Card{
			Name:   cardName,
			Number: number,
		},
		ActiveList:   listings,
		LowestPrice:  lowest,
		AveragePrice: total / float64(len(listings)),
		ListingCount: len(listings),
		LastUpdated:  time.Now(),
		DataSource:   "GameStopMock",
	}, nil
}

func (m *MockProvider) SearchCards(query string) ([]Listing, error) {
	// Return some mock search results
	return []Listing{
		{
			Price:       89.99,
			Grade:       "PSA 10",
			Title:       "Pokemon Charizard PSA 10 Gem Mint",
			URL:         "https://www.gamestop.com/mock/charizard-psa10",
			SKU:         "MOCK001",
			InStock:     true,
			Condition:   "New",
			Seller:      "GameStop",
			ImageURL:    "https://example.com/charizard.jpg",
			Description: "Mock Charizard PSA 10",
		},
		{
			Price:       149.99,
			Grade:       "BGS 9.5",
			Title:       "Pokemon Pikachu BGS 9.5 Gem Mint",
			URL:         "https://www.gamestop.com/mock/pikachu-bgs95",
			SKU:         "MOCK002",
			InStock:     true,
			Condition:   "New",
			Seller:      "GameStop",
			ImageURL:    "https://example.com/pikachu.jpg",
			Description: "Mock Pikachu BGS 9.5",
		},
	}, nil
}

func (m *MockProvider) GetBulkListings(cards []model.Card) (map[string]*ListingData, error) {
	results := make(map[string]*ListingData)

	for _, card := range cards {
		key := fmt.Sprintf("%s-%s", card.Name, card.Number)
		data, err := m.GetListings(card.SetName, card.Name, card.Number)
		if err != nil {
			continue
		}
		results[key] = data
	}

	return results, nil
}

func (m *MockProvider) generateMockListings(setName, cardName, number string) []Listing {
	// Generate different numbers of listings based on card popularity
	listingCount := m.getListingCount(cardName)
	if listingCount == 0 {
		return []Listing{}
	}

	listings := make([]Listing, listingCount)
	basePrice := m.getBasePrice(cardName)

	for i := 0; i < listingCount; i++ {
		grade := m.getGrade(i)
		priceMultiplier := m.getPriceMultiplier(grade)
		price := basePrice * priceMultiplier

		listings[i] = Listing{
			Price:       price,
			Grade:       grade,
			Title:       fmt.Sprintf("Pokemon %s %s %s #%s", setName, cardName, grade, number),
			URL:         fmt.Sprintf("https://www.gamestop.com/mock/%s-%s-%d", cardName, number, i),
			SKU:         fmt.Sprintf("MOCK%s%s%d", cardName[:min(3, len(cardName))], number, i),
			InStock:     i < listingCount-1, // Last one might be out of stock
			Condition:   "New",
			Seller:      "GameStop",
			ImageURL:    fmt.Sprintf("https://example.com/%s.jpg", cardName),
			Description: fmt.Sprintf("Mock %s %s", cardName, grade),
		}
	}

	return listings
}

func (m *MockProvider) getListingCount(cardName string) int {
	// Popular cards have more listings
	switch cardName {
	case "Charizard", "Pikachu":
		return 3
	case "Mewtwo", "Mew", "Lugia":
		return 2
	default:
		return 1
	}
}

func (m *MockProvider) getBasePrice(cardName string) float64 {
	// Different base prices for different cards
	switch cardName {
	case "Charizard":
		return 120.0
	case "Pikachu":
		return 85.0
	case "Mewtwo":
		return 95.0
	case "Mew":
		return 75.0
	default:
		return 45.0
	}
}

func (m *MockProvider) getGrade(index int) string {
	grades := []string{"PSA 10", "BGS 9.5", "PSA 9"}
	return grades[index%len(grades)]
}

func (m *MockProvider) getPriceMultiplier(grade string) float64 {
	switch grade {
	case "PSA 10":
		return 1.0
	case "BGS 9.5":
		return 0.85
	case "PSA 9":
		return 0.65
	default:
		return 0.5
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}