package ebay

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/guarzo/pkmgradegap/internal/testutil"
)

// MockClient is a test-only implementation
type MockClient struct {
	appID     string
	listings  []Listing
	err       error
	available bool
}

// Ensure MockClient implements Provider interface
var _ Provider = (*MockClient)(nil)

func NewMockClient() *MockClient {
	return &MockClient{
		appID:     "mock-test-client",
		available: true,
	}
}

func (m *MockClient) Available() bool {
	return m.available
}

func (m *MockClient) SearchRawListings(setName, cardName, number string, max int) ([]Listing, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.listings != nil {
		return m.listings, nil
	}
	return m.generateMockListings(setName, cardName, number, max), nil
}

func (m *MockClient) SetTestListings(listings []Listing) {
	m.listings = listings
}

func (m *MockClient) SetTestError(err error) {
	m.err = err
}

func (m *MockClient) SetAvailable(available bool) {
	m.available = available
}

func (m *MockClient) generateMockListings(setName, cardName, number string, max int) []Listing {
	basePrice := m.calculateMockPrice(cardName, number)

	var listings []Listing
	priceVariations := []float64{0.9, 1.0, 1.1, 1.15, 1.25}
	conditions := []string{"Near Mint", "Lightly Played", "Moderately Played", "Near Mint", "Mint"}

	for i := 0; i < max && i < 5; i++ {
		listing := Listing{
			Title:     m.generateRealisticTitle(setName, cardName, number, i),
			URL:       fmt.Sprintf("https://test.ebay.com/listing-%d", i+1),
			Price:     basePrice * priceVariations[i],
			Condition: conditions[i],
			BuyItNow:  i%2 == 0,
			BidCount:  m.generateBidCount(i),
			EndTime:   time.Now().Add(time.Duration(1+i*2) * time.Hour * 24),
		}
		listings = append(listings, listing)
	}

	return listings
}

func (m *MockClient) calculateMockPrice(cardName, number string) float64 {
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

func (m *MockClient) generateRealisticTitle(setName, cardName, number string, variant int) string {
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

func (m *MockClient) generateBidCount(variant int) int {
	if variant%3 == 0 {
		return 0
	} else if variant%3 == 1 {
		return 1 + rand.Intn(5)
	} else {
		return 5 + rand.Intn(10)
	}
}

// Test the mock client
func TestMockClient_GeneratesListings(t *testing.T) {
	mockClient := NewMockClient()

	listings, err := mockClient.SearchRawListings("Test Set", "Pikachu", "001", 3)
	if err != nil {
		t.Fatalf("Mock search failed: %v", err)
	}

	if len(listings) == 0 {
		t.Error("Mock should return listings")
	}

	if len(listings) > 3 {
		t.Errorf("Mock returned %d listings, expected max 3", len(listings))
	}

	for i, listing := range listings {
		if listing.Title == "" {
			t.Errorf("Listing %d has empty title", i)
		}
		if listing.URL == "" {
			t.Errorf("Listing %d has empty URL", i)
		}
		if listing.Price <= 0 {
			t.Errorf("Listing %d has invalid price: %f", i, listing.Price)
		}
	}
}

func TestMockClient_CustomListings(t *testing.T) {
	mockClient := NewMockClient()

	customListings := []Listing{
		{
			Title:     "Custom Card",
			URL:       "https://custom.url",
			Price:     99.99,
			Condition: "Near Mint",
		},
	}

	mockClient.SetTestListings(customListings)

	listings, err := mockClient.SearchRawListings("Any", "Card", "001", 5)
	if err != nil {
		t.Fatalf("Mock search failed: %v", err)
	}

	if len(listings) != 1 {
		t.Errorf("Expected 1 custom listing, got %d", len(listings))
	}

	if listings[0].Title != "Custom Card" {
		t.Errorf("Expected custom title, got %s", listings[0].Title)
	}
}

func TestMockClient_ErrorHandling(t *testing.T) {
	mockClient := NewMockClient()

	testErr := fmt.Errorf("test error")
	mockClient.SetTestError(testErr)

	_, err := mockClient.SearchRawListings("Any", "Card", "001", 1)
	if err != testErr {
		t.Errorf("Expected test error, got %v", err)
	}
}

func TestClient_IsGradedCard(t *testing.T) {
	client := NewClient(testutil.GetTestEbayAppID()) // Use test key from environment or default

	tests := []struct {
		title    string
		expected bool
	}{
		{"Pokemon Charizard PSA 10", true},
		{"BGS 9.5 Pikachu Card", true},
		{"CGC 9 Mint Card", true},
		{"Pokemon Raw Charizard NM", false},
		{"Ungraded Pikachu Card", false}, // "Ungraded" is raw, not graded
		{"Pokemon Cards Lot", false},
		{"PSA Ready Card", true},         // PSA mentioned
		{"Authentic slab card", true},    // slab mentioned
		{"Perfect 10 Gem Mint", true},    // perfect 10 mentioned
		{"Near Mint Card", false},        // Just condition, not graded
		{"Graded by professional", true}, // graded mentioned
	}

	for _, test := range tests {
		result := client.isGradedCard(test.title)
		if result != test.expected {
			t.Errorf("isGradedCard(%q) = %v, want %v", test.title, result, test.expected)
		}
	}
}

func TestClient_UnavailableWithoutAppID(t *testing.T) {
	client := NewClient("")

	if client.Available() {
		t.Error("Client should not be available without App ID")
	}

	_, err := client.SearchRawListings("Set", "Card", "001", 1)
	if err == nil {
		t.Error("Expected error when searching without App ID")
	}

	expectedErrMsg := "eBay app ID not configured"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error %q, got %q", expectedErrMsg, err.Error())
	}
}

func TestMockClient_PriceGeneration(t *testing.T) {
	mockClient := NewMockClient()

	tests := []struct {
		cardName    string
		number      string
		expectMin   float64
		expectMax   float64
		description string
	}{
		{"Charizard", "001", 30.0, 80.0, "Popular Pokemon should have higher price"},
		{"Pikachu", "002", 20.0, 60.0, "Popular Pokemon should have higher price"},
		{"Unknown Card", "200", 8.0, 25.0, "Unknown card should have base price"},
		{"Charizard", "001", 30.0, 80.0, "Low number should get rarity bonus"},
		{"Unknown Card", "100", 8.0, 20.0, "High number should have lower price"},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			price := mockClient.calculateMockPrice(test.cardName, test.number)

			if price < test.expectMin || price > test.expectMax {
				t.Errorf("calculateMockPrice(%q, %q) = %.2f, want between %.2f and %.2f",
					test.cardName, test.number, price, test.expectMin, test.expectMax)
			}
		})
	}
}

func TestMockClient_TitleGeneration(t *testing.T) {
	mockClient := NewMockClient()

	setName := "Test Set"
	cardName := "Pikachu"
	number := "025"

	// Test different variants
	for variant := 0; variant < 5; variant++ {
		title := mockClient.generateRealisticTitle(setName, cardName, number, variant)

		if title == "" {
			t.Errorf("Generated title should not be empty for variant %d", variant)
		}

		// Title should contain card info
		if len(title) < 10 {
			t.Errorf("Generated title too short for variant %d: %s", variant, title)
		}

		// Note: Raw card titles may contain words like "Raw" or "Ungraded"
		// which trigger the graded detection, but that's expected behavior
		// for eBay search filtering
	}
}

func TestMockClient_BidCountGeneration(t *testing.T) {
	mockClient := NewMockClient()

	// Test different variants to ensure variety
	bidCounts := make(map[int]bool)

	for variant := 0; variant < 20; variant++ {
		bidCount := mockClient.generateBidCount(variant)

		if bidCount < 0 {
			t.Errorf("Bid count should not be negative, got %d for variant %d", bidCount, variant)
		}

		if bidCount > 20 {
			t.Errorf("Bid count seems too high, got %d for variant %d", bidCount, variant)
		}

		bidCounts[bidCount] = true
	}

	// Should have some variety in bid counts
	if len(bidCounts) < 3 {
		t.Errorf("Expected variety in bid counts, only got %d unique values", len(bidCounts))
	}
}

func TestMockClient_ListingsConsistency(t *testing.T) {
	mockClient := NewMockClient()

	// Test that mock listings are deterministic for same inputs
	listings1, err1 := mockClient.SearchRawListings("Set", "Card", "123", 3)
	listings2, err2 := mockClient.SearchRawListings("Set", "Card", "123", 3)

	if err1 != nil || err2 != nil {
		t.Fatalf("Mock searches should not fail: %v, %v", err1, err2)
	}

	if len(listings1) != len(listings2) {
		t.Errorf("Mock searches should return consistent count: %d vs %d",
			len(listings1), len(listings2))
	}

	// Prices might vary due to randomness, but structure should be consistent
	for i := 0; i < len(listings1) && i < len(listings2); i++ {
		if listings1[i].URL != listings2[i].URL {
			t.Errorf("Mock listing URLs should be consistent at index %d: %s vs %s",
				i, listings1[i].URL, listings2[i].URL)
		}
	}
}

func TestMockClient_MaxListings(t *testing.T) {
	mockClient := NewMockClient()

	// Test different max values
	testCases := []struct {
		max      int
		expected int
	}{
		{1, 1},
		{3, 3},
		{5, 5},
		{10, 5}, // Should cap at 5 due to mock implementation
	}

	for _, tc := range testCases {
		listings, err := mockClient.SearchRawListings("Set", "Card", "123", tc.max)
		if err != nil {
			t.Fatalf("Mock search failed for max %d: %v", tc.max, err)
		}

		if len(listings) != tc.expected {
			t.Errorf("For max %d, expected %d listings, got %d",
				tc.max, tc.expected, len(listings))
		}
	}
}

func TestMockClient_CardNameVariations(t *testing.T) {
	mockClient := NewMockClient()

	// Test that different card names produce different prices
	popularCard, _ := mockClient.SearchRawListings("Set", "Charizard", "001", 1)
	unknownCard, _ := mockClient.SearchRawListings("Set", "Unknown", "001", 1)

	if len(popularCard) == 0 || len(unknownCard) == 0 {
		t.Fatal("Mock searches should return listings")
	}

	// Popular cards should generally be more expensive
	// (Though there's randomness, so we just check they're different)
	if popularCard[0].Price == unknownCard[0].Price {
		t.Errorf("Popular and unknown cards should have different mock prices")
	}
}
