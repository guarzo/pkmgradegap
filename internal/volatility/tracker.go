package volatility

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
)

// PricePoint represents a price observation at a specific time
type PricePoint struct {
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
}

// CardHistory holds price history for a specific card
type CardHistory struct {
	CardID     string       `json:"card_id"`
	SetName    string       `json:"set_name"`
	CardName   string       `json:"card_name"`
	CardNumber string       `json:"card_number"`
	PriceType  string       `json:"price_type"` // "raw", "psa10", "psa9", etc.
	History    []PricePoint `json:"history"`
}

// Tracker manages price history and volatility calculations
type Tracker struct {
	filePath string
	data     map[string]*CardHistory // key: cardID_priceType
}

// NewTracker creates a new volatility tracker
func NewTracker(filePath string) *Tracker {
	tracker := &Tracker{
		filePath: filePath,
		data:     make(map[string]*CardHistory),
	}

	// Load existing data if file exists
	tracker.loadFromFile()

	return tracker
}

// AddPrice records a new price observation
func (t *Tracker) AddPrice(setName, cardName, cardNumber, priceType string, price float64) {
	cardID := buildCardID(setName, cardName, cardNumber)
	key := buildKey(cardID, priceType)

	history, exists := t.data[key]
	if !exists {
		history = &CardHistory{
			CardID:     cardID,
			SetName:    setName,
			CardName:   cardName,
			CardNumber: cardNumber,
			PriceType:  priceType,
			History:    make([]PricePoint, 0),
		}
		t.data[key] = history
	}

	// Add new price point
	point := PricePoint{
		Price:     price,
		Timestamp: time.Now(),
	}
	history.History = append(history.History, point)

	// Keep only last 30 days of data
	t.pruneOld(history, 30*24*time.Hour)

	// Save updated data
	t.saveToFile()
}

// Calculate30DayVolatility calculates the 30-day price volatility for a card
func (t *Tracker) Calculate30DayVolatility(setName, cardName, cardNumber, priceType string) float64 {
	cardID := buildCardID(setName, cardName, cardNumber)
	key := buildKey(cardID, priceType)

	history, exists := t.data[key]
	if !exists || len(history.History) < 2 {
		return 0.0 // No data or insufficient data
	}

	// Filter to last 30 days
	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	var prices []float64

	for _, point := range history.History {
		if point.Timestamp.After(cutoff) {
			prices = append(prices, point.Price)
		}
	}

	if len(prices) < 2 {
		return 0.0
	}

	return calculateCoeffientOfVariation(prices)
}

// AddCardPrices adds multiple price observations for a card from analysis.Row
func (t *Tracker) AddCardPrices(row model.Card, rawPrice float64, grades map[string]float64, setName string) {
	if rawPrice > 0 {
		t.AddPrice(setName, row.Name, row.Number, "raw", rawPrice)
	}

	for gradeType, price := range grades {
		if price > 0 {
			t.AddPrice(setName, row.Name, row.Number, gradeType, price)
		}
	}
}

// GetVolatilityForCard is a convenience method to get volatility for a specific price type
func (t *Tracker) GetVolatilityForCard(setName, cardName, cardNumber, priceType string) float64 {
	return t.Calculate30DayVolatility(setName, cardName, cardNumber, priceType)
}

// pruneOld removes price points older than the specified duration
func (t *Tracker) pruneOld(history *CardHistory, maxAge time.Duration) {
	cutoff := time.Now().Add(-maxAge)
	var filtered []PricePoint

	for _, point := range history.History {
		if point.Timestamp.After(cutoff) {
			filtered = append(filtered, point)
		}
	}

	history.History = filtered
}

// calculateCoeffientOfVariation calculates the coefficient of variation (CV)
// CV = standard deviation / mean
func calculateCoeffientOfVariation(prices []float64) float64 {
	if len(prices) < 2 {
		return 0.0
	}

	// Calculate mean
	var sum float64
	for _, price := range prices {
		sum += price
	}
	mean := sum / float64(len(prices))

	if mean == 0 {
		return 0.0 // Avoid division by zero
	}

	// Calculate variance
	var varianceSum float64
	for _, price := range prices {
		diff := price - mean
		varianceSum += diff * diff
	}
	variance := varianceSum / float64(len(prices)-1) // Sample variance

	// Calculate standard deviation
	stdDev := math.Sqrt(variance)

	// Return coefficient of variation
	return stdDev / mean
}

// loadFromFile loads historical data from JSON file
func (t *Tracker) loadFromFile() {
	if _, err := os.Stat(t.filePath); os.IsNotExist(err) {
		return // File doesn't exist, start fresh
	}

	data, err := os.ReadFile(t.filePath)
	if err != nil {
		return // Silently fail and start fresh
	}

	var histories []CardHistory
	if err := json.Unmarshal(data, &histories); err != nil {
		return // Silently fail and start fresh
	}

	// Convert to map
	for i := range histories {
		history := &histories[i]
		key := buildKey(history.CardID, history.PriceType)
		t.data[key] = history
	}
}

// saveToFile saves current data to JSON file
func (t *Tracker) saveToFile() {
	// Create directory if it doesn't exist
	if dir := filepath.Dir(t.filePath); dir != "" && dir != "." {
		os.MkdirAll(dir, 0755)
	}

	// Convert map to slice for JSON serialization
	var histories []CardHistory
	for _, history := range t.data {
		histories = append(histories, *history)
	}

	data, err := json.MarshalIndent(histories, "", "  ")
	if err != nil {
		return // Silently fail
	}

	os.WriteFile(t.filePath, data, 0644)
}

// buildCardID creates a unique identifier for a card
func buildCardID(setName, cardName, cardNumber string) string {
	return fmt.Sprintf("%s|%s|%s", setName, cardName, cardNumber)
}

// buildKey creates a key for the data map
func buildKey(cardID, priceType string) string {
	return fmt.Sprintf("%s_%s", cardID, priceType)
}

// GetHistoryStats returns statistics about the tracked data
func (t *Tracker) GetHistoryStats() map[string]int {
	stats := make(map[string]int)
	stats["total_cards"] = len(t.data)

	var totalPoints int
	for _, history := range t.data {
		totalPoints += len(history.History)
	}
	stats["total_price_points"] = totalPoints

	return stats
}

// CleanupOldData removes all data older than the specified duration
func (t *Tracker) CleanupOldData(maxAge time.Duration) {
	for _, history := range t.data {
		t.pruneOld(history, maxAge)
	}

	// Remove histories with no data
	for key, history := range t.data {
		if len(history.History) == 0 {
			delete(t.data, key)
		}
	}

	t.saveToFile()
}