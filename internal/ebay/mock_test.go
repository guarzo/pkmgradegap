package ebay

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

// mockClient is a test-only implementation for eBay client
type mockClient struct {
	listings []Listing
	err      error
}

// newMockClient creates a mock client for testing
func newMockClient() *mockClient {
	return &mockClient{}
}

func (m *mockClient) Available() bool {
	return true
}

func (m *mockClient) SearchRawListings(setName, cardName, number string, max int) ([]Listing, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.listings != nil {
		return m.listings, nil
	}
	return m.generateTestListings(setName, cardName, number, max), nil
}

// generateTestListings creates test data for unit tests
func (m *mockClient) generateTestListings(setName, cardName, number string, max int) []Listing {
	basePrice := m.calculateTestPrice(cardName, number)
	
	var listings []Listing
	priceVariations := []float64{0.9, 1.0, 1.1, 1.15, 1.25}
	conditions := []string{"Near Mint", "Lightly Played", "Moderately Played", "Near Mint", "Mint"}
	
	for i := 0; i < max && i < 5; i++ {
		listing := Listing{
			Title:     m.generateTestTitle(setName, cardName, number, i),
			URL:       fmt.Sprintf("https://test.ebay.com/listing-%d", i+1),
			Price:     basePrice * priceVariations[i],
			Condition: conditions[i],
			BuyItNow:  i%2 == 0,
			BidCount:  m.generateTestBidCount(i),
			EndTime:   time.Now().Add(time.Duration(1+i*2) * time.Hour * 24),
		}
		listings = append(listings, listing)
	}
	
	return listings
}

func (m *mockClient) calculateTestPrice(cardName, number string) float64 {
	basePrice := 15.0
	
	popularNames := map[string]float64{
		"charizard": 3.0, "pikachu": 2.0, "mew": 2.5, "lugia": 2.0,
		"rayquaza": 2.0, "arceus": 1.8, "dialga": 1.5, "palkia": 1.5,
	}
	
	cardLower := strings.ToLower(cardName)
	for name, multiplier := range popularNames {
		if strings.Contains(cardLower, name) {
			basePrice *= multiplier
			break
		}
	}
	
	if num, err := strconv.Atoi(number); err == nil {
		if num <= 20 {
			basePrice *= 1.5
		} else if num <= 50 {
			basePrice *= 1.2
		}
	}
	
	variance := 0.8 + (0.4 * rand.Float64())
	return basePrice * variance
}

func (m *mockClient) generateTestTitle(setName, cardName, number string, variant int) string {
	templates := []string{
		"Pokemon %s %s #%s NM Raw Card",
		"%s %s Card #%s Ungraded Near Mint",
		"Pokemon Trading Card %s %s #%s Mint Condition",
		"%s Set %s Pokemon Card #%s Raw Ungraded",
		"Pokemon %s %s #%s Near Mint Trading Card",
	}
	
	template := templates[variant%len(templates)]
	return fmt.Sprintf(template, setName, cardName, number)
}

func (m *mockClient) generateTestBidCount(variant int) int {
	if variant%3 == 0 {
		return 0
	} else if variant%3 == 1 {
		return 1 + rand.Intn(5)
	} else {
		return 5 + rand.Intn(10)
	}
}

func (m *mockClient) setTestListings(listings []Listing) {
	m.listings = listings
}

func (m *mockClient) setTestError(err error) {
	m.err = err
}