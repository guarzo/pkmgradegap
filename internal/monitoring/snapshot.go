package monitoring

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/guarzo/pkmgradegap/internal/analysis"
	"github.com/guarzo/pkmgradegap/internal/model"
)

// Snapshot represents a point-in-time capture of card prices
type Snapshot struct {
	Timestamp time.Time                    `json:"timestamp"`
	SetName   string                       `json:"set_name"`
	Cards     map[string]*SnapshotCardData `json:"cards"`
}

// SnapshotCardData contains price data for a card at a point in time
type SnapshotCardData struct {
	Card         model.Card `json:"card"`
	RawPriceUSD  float64    `json:"raw_price_usd"`
	PSA10Price   float64    `json:"psa10_price"`
	PSA9Price    float64    `json:"psa9_price"`
	Grade95Price float64    `json:"grade95_price"`
	BGS10Price   float64    `json:"bgs10_price"`
}

// LoadSnapshot loads a snapshot from a JSON file
func LoadSnapshot(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading snapshot: %w", err)
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("parsing snapshot: %w", err)
	}

	return &snapshot, nil
}

// SaveSnapshot saves a snapshot to a JSON file
func SaveSnapshot(path string, snapshot *Snapshot) error {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling snapshot: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing snapshot: %w", err)
	}

	return nil
}

// CreateSnapshotFromRows creates a snapshot from analysis rows
func CreateSnapshotFromRows(setName string, rows []analysis.Row) *Snapshot {
	snapshot := &Snapshot{
		Timestamp: time.Now(),
		SetName:   setName,
		Cards:     make(map[string]*SnapshotCardData),
	}

	for _, row := range rows {
		key := fmt.Sprintf("%s-%s", row.Card.Number, row.Card.Name)
		snapshot.Cards[key] = &SnapshotCardData{
			Card:         row.Card,
			RawPriceUSD:  row.RawUSD,
			PSA10Price:   row.Grades.PSA10,
			PSA9Price:    row.Grades.Grade9,
			Grade95Price: row.Grades.Grade95,
			BGS10Price:   row.Grades.BGS10,
		}
	}

	return snapshot
}

// PriceDelta represents a price change between snapshots
type PriceDelta struct {
	Card        model.Card
	Field       string
	OldPrice    float64
	NewPrice    float64
	DeltaUSD    float64
	DeltaPct    float64
	OldSnapshot time.Time
	NewSnapshot time.Time
}

// CompareSnapshots compares two snapshots and returns significant price changes
func CompareSnapshots(old, new *Snapshot, thresholdPct, thresholdUSD float64) []PriceDelta {
	var deltas []PriceDelta

	for key, newCard := range new.Cards {
		oldCard, exists := old.Cards[key]
		if !exists {
			continue // Card not in old snapshot
		}

		// Check raw price changes
		checkPriceChange(&deltas, oldCard.Card, "Raw",
			oldCard.RawPriceUSD, newCard.RawPriceUSD,
			old.Timestamp, new.Timestamp, thresholdPct, thresholdUSD)

		// Check PSA10 price changes
		checkPriceChange(&deltas, oldCard.Card, "PSA10",
			oldCard.PSA10Price, newCard.PSA10Price,
			old.Timestamp, new.Timestamp, thresholdPct, thresholdUSD)

		// Check PSA9 price changes
		checkPriceChange(&deltas, oldCard.Card, "PSA9",
			oldCard.PSA9Price, newCard.PSA9Price,
			old.Timestamp, new.Timestamp, thresholdPct, thresholdUSD)
	}

	return deltas
}

func checkPriceChange(deltas *[]PriceDelta, card model.Card, field string,
	oldPrice, newPrice float64, oldTime, newTime time.Time,
	thresholdPct, thresholdUSD float64) {

	if oldPrice <= 0 || newPrice <= 0 {
		return // Skip if either price is invalid
	}

	deltaUSD := newPrice - oldPrice
	deltaPct := (deltaUSD / oldPrice) * 100

	// Check if change exceeds thresholds
	if abs(deltaPct) >= thresholdPct || abs(deltaUSD) >= thresholdUSD {
		*deltas = append(*deltas, PriceDelta{
			Card:        card,
			Field:       field,
			OldPrice:    oldPrice,
			NewPrice:    newPrice,
			DeltaUSD:    deltaUSD,
			DeltaPct:    deltaPct,
			OldSnapshot: oldTime,
			NewSnapshot: newTime,
		})
	}
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}