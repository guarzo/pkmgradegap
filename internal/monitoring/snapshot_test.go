package monitoring

import (
	"testing"
	"time"

	"github.com/guarzo/pkmgradegap/internal/analysis"
	"github.com/guarzo/pkmgradegap/internal/model"
)

func TestCreateSnapshotFromRows(t *testing.T) {
	rows := []analysis.Row{
		{
			Card: model.Card{
				Name:   "Test Card 1",
				Number: "001",
			},
			RawUSD: 10.50,
			Grades: analysis.Grades{
				PSA10:   50.00,
				Grade9:  35.00,
				Grade95: 42.00,
				BGS10:   48.00,
			},
		},
		{
			Card: model.Card{
				Name:   "Test Card 2",
				Number: "002",
			},
			RawUSD: 5.25,
			Grades: analysis.Grades{
				PSA10:   25.00,
				Grade9:  18.00,
				Grade95: 21.00,
				BGS10:   23.00,
			},
		},
	}

	snapshot := CreateSnapshotFromRows("Test Set", rows)

	if snapshot.SetName != "Test Set" {
		t.Errorf("Expected set name 'Test Set', got '%s'", snapshot.SetName)
	}

	if len(snapshot.Cards) != 2 {
		t.Errorf("Expected 2 cards, got %d", len(snapshot.Cards))
	}

	card1Key := "001-Test Card 1"
	if card, exists := snapshot.Cards[card1Key]; exists {
		if card.RawUSD != 10.50 {
			t.Errorf("Expected raw price 10.50, got %.2f", card.RawUSD)
		}
		if card.PSA10Price != 50.00 {
			t.Errorf("Expected PSA10 price 50.00, got %.2f", card.PSA10Price)
		}
	} else {
		t.Errorf("Card %s not found in snapshot", card1Key)
	}
}

func TestCompareSnapshots(t *testing.T) {
	old := &Snapshot{
		Timestamp: time.Now().Add(-24 * time.Hour),
		SetName:   "Test Set",
		Cards: map[string]*SnapshotCardData{
			"001-Test Card": {
				Card:        model.Card{Name: "Test Card", Number: "001"},
				RawUSD: 10.00,
				PSA10Price:  50.00,
			},
		},
	}

	new := &Snapshot{
		Timestamp: time.Now(),
		SetName:   "Test Set",
		Cards: map[string]*SnapshotCardData{
			"001-Test Card": {
				Card:        model.Card{Name: "Test Card", Number: "001"},
				RawUSD: 8.00,  // 20% drop
				PSA10Price:  55.00, // 10% increase
			},
		},
	}

	deltas := CompareSnapshots(old, new, 10.0, 1.0)

	if len(deltas) != 2 {
		t.Errorf("Expected 2 deltas (raw and PSA10), got %d", len(deltas))
	}

	// Check raw price drop
	found := false
	for _, delta := range deltas {
		if delta.Field == "Raw" && delta.DeltaPct < -15 {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find raw price drop delta")
	}

	// Check PSA10 price increase
	found = false
	for _, delta := range deltas {
		if delta.Field == "PSA10" && delta.DeltaPct > 5 {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find PSA10 price increase delta")
	}
}
