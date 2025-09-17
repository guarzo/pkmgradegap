package volatility

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
)

func TestTracker_AddPriceAndCalculateVolatility(t *testing.T) {
	tempDir := t.TempDir()
	trackerPath := filepath.Join(tempDir, "volatility.json")

	tracker := NewTracker(trackerPath)

	// Add some price data with varying prices
	setName := "Base Set"
	cardName := "Charizard"
	cardNumber := "4"
	priceType := "raw"

	prices := []float64{100.0, 105.0, 95.0, 110.0, 90.0, 115.0, 85.0}

	for _, price := range prices {
		tracker.AddPrice(setName, cardName, cardNumber, priceType, price)
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	}

	// Calculate volatility
	volatility := tracker.Calculate30DayVolatility(setName, cardName, cardNumber, priceType)

	// Should have some volatility (not zero)
	if volatility <= 0 {
		t.Errorf("Expected positive volatility, got %f", volatility)
	}

	// Should be reasonable (not extremely high)
	if volatility > 1.0 {
		t.Errorf("Volatility seems too high: %f", volatility)
	}

	t.Logf("Calculated volatility: %.4f", volatility)
}

func TestTracker_InsufficientData(t *testing.T) {
	tempDir := t.TempDir()
	trackerPath := filepath.Join(tempDir, "volatility.json")

	tracker := NewTracker(trackerPath)

	// Test with no data
	volatility := tracker.Calculate30DayVolatility("Set", "Card", "1", "raw")
	if volatility != 0.0 {
		t.Errorf("Expected 0 volatility with no data, got %f", volatility)
	}

	// Test with single data point
	tracker.AddPrice("Set", "Card", "1", "raw", 100.0)
	volatility = tracker.Calculate30DayVolatility("Set", "Card", "1", "raw")
	if volatility != 0.0 {
		t.Errorf("Expected 0 volatility with single data point, got %f", volatility)
	}
}

func TestTracker_StablePrices(t *testing.T) {
	tempDir := t.TempDir()
	trackerPath := filepath.Join(tempDir, "volatility.json")

	tracker := NewTracker(trackerPath)

	// Add stable prices (all the same)
	for i := 0; i < 10; i++ {
		tracker.AddPrice("Set", "Card", "1", "raw", 100.0)
		time.Sleep(1 * time.Millisecond)
	}

	volatility := tracker.Calculate30DayVolatility("Set", "Card", "1", "raw")

	// Volatility should be zero or very close to zero
	if volatility > 0.001 {
		t.Errorf("Expected near-zero volatility for stable prices, got %f", volatility)
	}
}

func TestTracker_HighVolatilityPrices(t *testing.T) {
	tempDir := t.TempDir()
	trackerPath := filepath.Join(tempDir, "volatility.json")

	tracker := NewTracker(trackerPath)

	// Add highly volatile prices
	volatilePrices := []float64{100.0, 200.0, 50.0, 300.0, 25.0, 400.0, 10.0}

	for _, price := range volatilePrices {
		tracker.AddPrice("Set", "Card", "1", "raw", price)
		time.Sleep(1 * time.Millisecond)
	}

	volatility := tracker.Calculate30DayVolatility("Set", "Card", "1", "raw")

	// Should have high volatility
	if volatility < 0.5 {
		t.Errorf("Expected high volatility, got %f", volatility)
	}

	t.Logf("High volatility: %.4f", volatility)
}

func TestTracker_Persistence(t *testing.T) {
	tempDir := t.TempDir()
	trackerPath := filepath.Join(tempDir, "volatility.json")

	// Create tracker and add data
	tracker1 := NewTracker(trackerPath)
	tracker1.AddPrice("Set", "Card", "1", "raw", 100.0)
	tracker1.AddPrice("Set", "Card", "1", "raw", 110.0)

	// Create new tracker instance (should load existing data)
	tracker2 := NewTracker(trackerPath)

	// Should be able to calculate volatility with loaded data
	volatility := tracker2.Calculate30DayVolatility("Set", "Card", "1", "raw")
	if volatility <= 0 {
		t.Errorf("Expected positive volatility from loaded data, got %f", volatility)
	}
}

func TestTracker_AddCardPrices(t *testing.T) {
	tempDir := t.TempDir()
	trackerPath := filepath.Join(tempDir, "volatility.json")

	tracker := NewTracker(trackerPath)

	// Create test card
	card := model.Card{
		Name:   "Pikachu",
		Number: "25",
	}

	grades := map[string]float64{
		"psa10": 150.0,
		"psa9":  120.0,
		"bgs10": 140.0,
	}

	// Add card prices
	tracker.AddCardPrices(card, 50.0, grades, "Base Set")

	// Should have data for all price types
	rawVol := tracker.GetVolatilityForCard("Base Set", "Pikachu", "25", "raw")
	psa10Vol := tracker.GetVolatilityForCard("Base Set", "Pikachu", "25", "psa10")

	// With single data points, volatility should be 0
	if rawVol != 0.0 {
		t.Errorf("Expected 0 volatility for single data point (raw), got %f", rawVol)
	}
	if psa10Vol != 0.0 {
		t.Errorf("Expected 0 volatility for single data point (psa10), got %f", psa10Vol)
	}

	// Add another set of prices
	grades2 := map[string]float64{
		"psa10": 160.0,
		"psa9":  125.0,
		"bgs10": 145.0,
	}
	tracker.AddCardPrices(card, 55.0, grades2, "Base Set")

	// Now should have some volatility
	rawVol = tracker.GetVolatilityForCard("Base Set", "Pikachu", "25", "raw")
	psa10Vol = tracker.GetVolatilityForCard("Base Set", "Pikachu", "25", "psa10")

	if rawVol <= 0 {
		t.Errorf("Expected positive volatility for raw prices, got %f", rawVol)
	}
	if psa10Vol <= 0 {
		t.Errorf("Expected positive volatility for psa10 prices, got %f", psa10Vol)
	}
}

func TestTracker_DataPruning(t *testing.T) {
	tempDir := t.TempDir()
	trackerPath := filepath.Join(tempDir, "volatility.json")

	tracker := NewTracker(trackerPath)

	// Add old data by manually setting timestamps
	setName := "Set"
	cardName := "Card"
	cardNumber := "1"
	priceType := "raw"

	tracker.AddPrice(setName, cardName, cardNumber, priceType, 100.0)

	// Manually add old data
	cardID := buildCardID(setName, cardName, cardNumber)
	key := buildKey(cardID, priceType)
	history := tracker.data[key]

	// Add point that's 40 days old
	oldPoint := PricePoint{
		Price:     90.0,
		Timestamp: time.Now().Add(-40 * 24 * time.Hour),
	}
	history.History = append(history.History, oldPoint)

	// Add point that's 20 days old (should be kept)
	recentPoint := PricePoint{
		Price:     95.0,
		Timestamp: time.Now().Add(-20 * 24 * time.Hour),
	}
	history.History = append(history.History, recentPoint)

	// Add another recent price
	tracker.AddPrice(setName, cardName, cardNumber, priceType, 105.0)

	// Check that old data was pruned
	if len(history.History) != 3 {
		t.Errorf("Expected 3 price points after pruning, got %d", len(history.History))
	}

	// The 40-day old point should be gone
	for _, point := range history.History {
		age := time.Since(point.Timestamp)
		if age > 35*24*time.Hour {
			t.Errorf("Found price point older than 35 days: %v", age)
		}
	}
}

func TestTracker_GetHistoryStats(t *testing.T) {
	tempDir := t.TempDir()
	trackerPath := filepath.Join(tempDir, "volatility.json")

	tracker := NewTracker(trackerPath)

	// Add some data
	tracker.AddPrice("Set1", "Card1", "1", "raw", 100.0)
	tracker.AddPrice("Set1", "Card1", "1", "psa10", 200.0)
	tracker.AddPrice("Set1", "Card2", "2", "raw", 50.0)

	stats := tracker.GetHistoryStats()

	if stats["total_cards"] != 3 {
		t.Errorf("Expected 3 cards, got %d", stats["total_cards"])
	}

	if stats["total_price_points"] != 3 {
		t.Errorf("Expected 3 price points, got %d", stats["total_price_points"])
	}
}

func TestTracker_CleanupOldData(t *testing.T) {
	tempDir := t.TempDir()
	trackerPath := filepath.Join(tempDir, "volatility.json")

	tracker := NewTracker(trackerPath)

	// Add recent data
	tracker.AddPrice("Set", "Card1", "1", "raw", 100.0)

	// Manually add very old data
	cardID := buildCardID("Set", "Card2", "2")
	key := buildKey(cardID, "raw")

	tracker.data[key] = &CardHistory{
		CardID:     cardID,
		SetName:    "Set",
		CardName:   "Card2",
		CardNumber: "2",
		PriceType:  "raw",
		History: []PricePoint{
			{
				Price:     50.0,
				Timestamp: time.Now().Add(-60 * 24 * time.Hour), // 60 days old
			},
		},
	}

	// Stats before cleanup
	statsBefore := tracker.GetHistoryStats()

	// Cleanup data older than 30 days
	tracker.CleanupOldData(30 * 24 * time.Hour)

	// Stats after cleanup
	statsAfter := tracker.GetHistoryStats()

	// Should have one less card (the old one should be removed)
	if statsAfter["total_cards"] != statsBefore["total_cards"]-1 {
		t.Errorf("Expected cleanup to remove 1 card, before: %d, after: %d",
			statsBefore["total_cards"], statsAfter["total_cards"])
	}
}

func TestCoefficientOfVariation(t *testing.T) {
	// Test with known values
	prices := []float64{100.0, 110.0, 90.0, 105.0, 95.0}

	cv := calculateCoeffientOfVariation(prices)

	// Should be positive
	if cv <= 0 {
		t.Errorf("Expected positive coefficient of variation, got %f", cv)
	}

	// Test with identical values (should be 0)
	identicalPrices := []float64{100.0, 100.0, 100.0, 100.0}
	cv = calculateCoeffientOfVariation(identicalPrices)

	if cv != 0.0 {
		t.Errorf("Expected 0 coefficient of variation for identical prices, got %f", cv)
	}

	// Test with empty slice
	cv = calculateCoeffientOfVariation([]float64{})
	if cv != 0.0 {
		t.Errorf("Expected 0 coefficient of variation for empty slice, got %f", cv)
	}

	// Test with single value
	cv = calculateCoeffientOfVariation([]float64{100.0})
	if cv != 0.0 {
		t.Errorf("Expected 0 coefficient of variation for single value, got %f", cv)
	}
}
