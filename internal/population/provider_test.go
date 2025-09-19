package population

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
)

// Mock implementations for testing

type mockHTTPClient struct {
	response *HTTPResponse
	err      error
}

func (m *mockHTTPClient) Get(url string) (*HTTPResponse, error) {
	return m.response, m.err
}

func (m *mockHTTPClient) Post(url string, data []byte) (*HTTPResponse, error) {
	return m.response, m.err
}

type mockCache struct {
	data map[string]*PopulationData
	sets map[string]*SetPopulationData
}

func (m *mockCache) Get(key string) (*PopulationData, bool) {
	data, found := m.data[key]
	return data, found
}

func (m *mockCache) Set(key string, data *PopulationData, ttl time.Duration) error {
	if m.data == nil {
		m.data = make(map[string]*PopulationData)
	}
	m.data[key] = data
	return nil
}

func (m *mockCache) GetSet(key string) (*SetPopulationData, bool) {
	data, found := m.sets[key]
	return data, found
}

func (m *mockCache) SetSet(key string, data *SetPopulationData, ttl time.Duration) error {
	if m.sets == nil {
		m.sets = make(map[string]*SetPopulationData)
	}
	m.sets[key] = data
	return nil
}

func (m *mockCache) Clear() error {
	m.data = make(map[string]*PopulationData)
	m.sets = make(map[string]*SetPopulationData)
	return nil
}

type mockRateLimiter struct{}

func (m *mockRateLimiter) Wait(ctx context.Context) error {
	return nil
}

func (m *mockRateLimiter) Allow() bool {
	return true
}

// Test data - realistic PSA API response samples

const samplePSAAPIResponse = `{
	"success": true,
	"data": {
		"specId": 123456,
		"description": "2022 Pokemon Japanese VMAX Climax Charizard VMAX #003/184",
		"brand": "Pokemon",
		"category": "Pokemon TCG",
		"sport": "Non-Sport",
		"year": "2022",
		"setName": "VMAX Climax",
		"population": {
			"total": 8769,
			"auth": 45,
			"grades": {
				"1": 12,
				"2": 23,
				"3": 67,
				"4": 128,
				"5": 234,
				"6": 445,
				"7": 890,
				"8": 1920,
				"9": 2800,
				"10": 1250
			},
			"lastUpdate": "2024-01-15"
		}
	}
}`

const samplePSASetSearchResponse = `{
	"success": true,
	"results": [
		{
			"specId": 123456,
			"description": "2022 Pokemon Japanese VMAX Climax Charizard VMAX #003/184",
			"brand": "Pokemon",
			"category": "Pokemon TCG",
			"year": "2022",
			"setName": "VMAX Climax",
			"population": {
				"total": 8769,
				"auth": 45,
				"grades": {
					"8": 1920,
					"9": 2800,
					"10": 1250
				},
				"lastUpdate": "2024-01-15"
			}
		},
		{
			"specId": 123457,
			"description": "2022 Pokemon Japanese VMAX Climax Blastoise VMAX #002/184",
			"brand": "Pokemon",
			"category": "Pokemon TCG",
			"year": "2022",
			"setName": "VMAX Climax",
			"population": {
				"total": 4521,
				"auth": 22,
				"grades": {
					"8": 980,
					"9": 1850,
					"10": 691
				},
				"lastUpdate": "2024-01-15"
			}
		}
	]
}`

const sampleErrorResponse = `{
	"success": false,
	"error": "Card not found in PSA database"
}`

func TestPSAProvider_parsePopulationResponse_Success(t *testing.T) {
	provider := &PSAProvider{}
	card := model.Card{
		Name:    "Charizard VMAX",
		SetName: "VMAX Climax",
		Number:  "003",
	}

	popData, err := provider.parsePopulationResponse([]byte(samplePSAAPIResponse), card)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if popData == nil {
		t.Fatal("Expected population data, got nil")
	}

	// Verify basic structure
	if popData.Card.Name != card.Name {
		t.Errorf("Expected card name %s, got %s", card.Name, popData.Card.Name)
	}

	if popData.TotalGraded != 8769 {
		t.Errorf("Expected total graded 8769, got %d", popData.TotalGraded)
	}

	if popData.PSA10Population != 1250 {
		t.Errorf("Expected PSA 10 population 1250, got %d", popData.PSA10Population)
	}

	if popData.PSA9Population != 2800 {
		t.Errorf("Expected PSA 9 population 2800, got %d", popData.PSA9Population)
	}

	if popData.PSA8Population != 1920 {
		t.Errorf("Expected PSA 8 population 1920, got %d", popData.PSA8Population)
	}

	// Verify grade population map
	expectedGrades := map[string]int{
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

	for grade, expectedCount := range expectedGrades {
		if actualCount, exists := popData.GradePopulation[grade]; !exists {
			t.Errorf("Expected grade %s to exist in population map", grade)
		} else if actualCount != expectedCount {
			t.Errorf("Expected grade %s count %d, got %d", grade, expectedCount, actualCount)
		}
	}

	// Verify scarcity level calculation
	if popData.ScarcityLevel != "COMMON" { // 1250 PSA 10s should be COMMON
		t.Errorf("Expected scarcity level COMMON, got %s", popData.ScarcityLevel)
	}

	// Verify last updated parsing
	expectedDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	if !popData.LastUpdated.Equal(expectedDate) {
		t.Errorf("Expected last updated %v, got %v", expectedDate, popData.LastUpdated)
	}
}

func TestPSAProvider_parsePopulationResponse_Error(t *testing.T) {
	provider := &PSAProvider{}
	card := model.Card{
		Name:    "Nonexistent Card",
		SetName: "Fake Set",
		Number:  "999",
	}

	popData, err := provider.parsePopulationResponse([]byte(sampleErrorResponse), card)
	if err == nil {
		t.Fatal("Expected error for unsuccessful API response")
	}

	if popData != nil {
		t.Error("Expected nil population data for error response")
	}

	if !strings.Contains(err.Error(), "Card not found in PSA database") {
		t.Errorf("Expected error message to contain API error, got: %v", err)
	}
}

func TestPSAProvider_parsePopulationResponse_InvalidJSON(t *testing.T) {
	provider := &PSAProvider{}
	card := model.Card{Name: "Test", SetName: "Test", Number: "001"}

	popData, err := provider.parsePopulationResponse([]byte("invalid json"), card)
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}

	if popData != nil {
		t.Error("Expected nil population data for invalid JSON")
	}

	if !strings.Contains(err.Error(), "failed to parse JSON response") {
		t.Errorf("Expected JSON parsing error, got: %v", err)
	}
}

func TestPSAProvider_parsePopulationResponse_EmptyBody(t *testing.T) {
	provider := &PSAProvider{}
	card := model.Card{Name: "Test", SetName: "Test", Number: "001"}

	popData, err := provider.parsePopulationResponse([]byte(""), card)
	if err == nil {
		t.Fatal("Expected error for empty body")
	}

	if popData != nil {
		t.Error("Expected nil population data for empty body")
	}

	if !strings.Contains(err.Error(), "empty response body") {
		t.Errorf("Expected empty body error, got: %v", err)
	}
}

func TestPSAProvider_parseSetPopulationResponse_Success(t *testing.T) {
	provider := &PSAProvider{}
	setName := "VMAX Climax"

	setData, err := provider.parseSetPopulationResponse([]byte(samplePSASetSearchResponse), setName)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if setData == nil {
		t.Fatal("Expected set population data, got nil")
	}

	// Verify basic structure
	if setData.SetName != setName {
		t.Errorf("Expected set name %s, got %s", setName, setData.SetName)
	}

	if setData.TotalCards != 2 {
		t.Errorf("Expected total cards 2, got %d", setData.TotalCards)
	}

	if setData.CardsGraded != 2 {
		t.Errorf("Expected cards graded 2, got %d", setData.CardsGraded)
	}

	// Verify card data
	if len(setData.CardData) != 2 {
		t.Errorf("Expected 2 cards in data, got %d", len(setData.CardData))
	}

	// Verify statistics
	if setData.SetStatistics == nil {
		t.Fatal("Expected set statistics, got nil")
	}

	expectedAverage := float64(8769+4521) / 2.0 // Average of the two cards
	if setData.SetStatistics.AveragePopulation != expectedAverage {
		t.Errorf("Expected average population %.1f, got %.1f",
			expectedAverage, setData.SetStatistics.AveragePopulation)
	}

	// Verify grade distribution aggregation
	expectedPSA10Total := 1250 + 691 // Sum of PSA 10s from both cards
	if setData.SetStatistics.GradeDistribution["PSA 10"] != expectedPSA10Total {
		t.Errorf("Expected PSA 10 total %d, got %d",
			expectedPSA10Total, setData.SetStatistics.GradeDistribution["PSA 10"])
	}
}

func TestPSAProvider_calculateScarcityLevel(t *testing.T) {
	provider := &PSAProvider{}

	tests := []struct {
		psa10Count int
		expected   string
	}{
		{5, "ULTRA_RARE"},
		{25, "RARE"},
		{250, "UNCOMMON"},
		{1000, "COMMON"},
	}

	for _, test := range tests {
		result := provider.calculateScarcityLevel(test.psa10Count)
		if result != test.expected {
			t.Errorf("PSA 10 count %d: expected %s, got %s",
				test.psa10Count, test.expected, result)
		}
	}
}

func TestPSAProvider_calculatePopulationTrend(t *testing.T) {
	provider := &PSAProvider{}

	// Recent high population should be INCREASING
	recent := time.Now().AddDate(0, 0, -3) // 3 days ago
	trend := provider.calculatePopulationTrend(2000, recent)
	if trend != "INCREASING" {
		t.Errorf("Recent high population should be INCREASING, got %s", trend)
	}

	// Old data should be STABLE
	old := time.Now().AddDate(0, 0, -45) // 45 days ago
	trend = provider.calculatePopulationTrend(1000, old)
	if trend != "STABLE" {
		t.Errorf("Old data should be STABLE, got %s", trend)
	}

	// Low population should be STABLE
	trend = provider.calculatePopulationTrend(50, recent)
	if trend != "STABLE" {
		t.Errorf("Low population should be STABLE, got %s", trend)
	}
}

func TestPSAProvider_LookupPopulation_Integration(t *testing.T) {
	// Mock HTTP client with successful response
	httpClient := &mockHTTPClient{
		response: &HTTPResponse{
			StatusCode: 200,
			Body:       []byte(samplePSAAPIResponse),
		},
	}

	cache := &mockCache{}
	rateLimiter := &mockRateLimiter{}

	provider := NewPSAProvider("test-api-key", httpClient, cache, rateLimiter)

	card := model.Card{
		Name:    "Charizard VMAX",
		SetName: "VMAX Climax",
		Number:  "003",
	}

	ctx := context.Background()
	popData, err := provider.LookupPopulation(ctx, card)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if popData == nil {
		t.Fatal("Expected population data, got nil")
	}

	// Verify the data was cached
	cacheKey := "psa_pop_VMAX_Climax_Charizard_VMAX_003"
	if cached, found := cache.Get(cacheKey); !found {
		t.Error("Expected data to be cached")
	} else if cached.PSA10Population != 1250 {
		t.Errorf("Expected cached PSA 10 population 1250, got %d", cached.PSA10Population)
	}
}

func TestPSAProvider_LookupPopulation_CacheHit(t *testing.T) {
	// Mock HTTP client that should not be called
	httpClient := &mockHTTPClient{
		err: testError{Message: "HTTP client should not be called"},
	}

	cache := &mockCache{
		data: map[string]*PopulationData{
			"psa_pop_Test_Set_Test_Card_001": {
				Card:            model.Card{Name: "Test Card", SetName: "Test Set", Number: "001"},
				PSA10Population: 500,
				TotalGraded:     2000,
				ScarcityLevel:   "UNCOMMON",
			},
		},
	}

	rateLimiter := &mockRateLimiter{}
	provider := NewPSAProvider("test-api-key", httpClient, cache, rateLimiter)

	card := model.Card{
		Name:    "Test Card",
		SetName: "Test Set",
		Number:  "001",
	}

	ctx := context.Background()
	popData, err := provider.LookupPopulation(ctx, card)

	if err != nil {
		t.Fatalf("Expected no error for cache hit, got: %v", err)
	}

	if popData == nil {
		t.Fatal("Expected population data from cache, got nil")
	}

	if popData.PSA10Population != 500 {
		t.Errorf("Expected PSA 10 population 500 from cache, got %d", popData.PSA10Population)
	}
}

// testError implements the error interface for testing
type testError struct {
	Message string
}

func (e testError) Error() string {
	return e.Message
}

// Helper to create test error
func newTestError(message string) error {
	return testError{Message: message}
}
