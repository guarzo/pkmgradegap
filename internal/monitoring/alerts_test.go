package monitoring

import (
	"strings"
	"testing"
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
)

func TestAlertGeneration(t *testing.T) {
	config := AlertConfig{
		PriceDropThresholdPct:   15.0,
		PriceDropThresholdUSD:   2.0,
		OpportunityThresholdROI: 20.0,
	}

	alertEngine := NewAlertEngine(config)

	deltas := []PriceDelta{
		{
			Card:        model.Card{Name: "Test Card", Number: "001"},
			Field:       "Raw",
			OldPrice:    10.00,
			NewPrice:    8.00,
			DeltaUSD:    -2.00,
			DeltaPct:    -20.0,
			OldSnapshot: time.Now().Add(-24 * time.Hour),
			NewSnapshot: time.Now(),
		},
		{
			Card:        model.Card{Name: "Test Card", Number: "001"},
			Field:       "PSA10",
			OldPrice:    50.00,
			NewPrice:    60.00,
			DeltaUSD:    10.00,
			DeltaPct:    20.0,
			OldSnapshot: time.Now().Add(-24 * time.Hour),
			NewSnapshot: time.Now(),
		},
	}

	alerts := alertEngine.GenerateAlerts(deltas)

	if len(alerts) != 2 {
		t.Errorf("Expected 2 alerts, got %d", len(alerts))
	}

	// Check for price drop alert
	foundDrop := false
	foundIncrease := false

	for _, alert := range alerts {
		if alert.Type == AlertPriceDrop {
			foundDrop = true
			if alert.Severity != "MEDIUM" {
				t.Errorf("Expected MEDIUM severity for 20%% drop, got %s", alert.Severity)
			}
		}
		if alert.Type == AlertPriceIncrease {
			foundIncrease = true
		}
	}

	if !foundDrop {
		t.Error("Expected to find price drop alert")
	}
	if !foundIncrease {
		t.Error("Expected to find price increase alert")
	}
}

func TestFormatAlert(t *testing.T) {
	alert := Alert{
		Type:     AlertPriceDrop,
		Severity: "HIGH",
		Card:     model.Card{Name: "Test Card", SetName: "Test Set", Number: "001"},
		Message:  "Raw price dropped 20.0% ($2.00)",
		ActionItems: []string{
			"Consider buying at new price of $8.00",
			"Check eBay for current listings",
		},
	}

	formatted := FormatAlert(alert)

	if !strings.Contains(formatted, "HIGH") {
		t.Error("Expected formatted alert to contain severity")
	}
	if !strings.Contains(formatted, "Test Card") {
		t.Error("Expected formatted alert to contain card name")
	}
	if !strings.Contains(formatted, "Recommended Actions") {
		t.Error("Expected formatted alert to contain action items")
	}
}