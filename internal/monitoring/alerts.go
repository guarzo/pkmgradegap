package monitoring

import (
	"fmt"
	"sort"
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
)

// AlertType represents different types of price alerts
type AlertType string

const (
	AlertPriceDrop     AlertType = "PRICE_DROP"
	AlertPriceIncrease AlertType = "PRICE_INCREASE"
	AlertNewOpportunity AlertType = "NEW_OPPORTUNITY"
	AlertPopulationSpike AlertType = "POPULATION_SPIKE"
)

// Alert represents a significant market event
type Alert struct {
	Type        AlertType
	Severity    string // "HIGH", "MEDIUM", "LOW"
	Card        model.Card
	Message     string
	Details     map[string]interface{}
	Timestamp   time.Time
	ActionItems []string // Suggested actions
}

// AlertConfig contains alert generation parameters
type AlertConfig struct {
	PriceDropThresholdPct    float64 // Trigger alert if price drops by this %
	PriceDropThresholdUSD    float64 // Trigger alert if price drops by this $
	OpportunityThresholdROI  float64 // Min ROI to trigger opportunity alert
	PopulationIncreaseThreshold float64 // % increase in population to trigger alert
}

// AlertEngine processes snapshots and generates alerts
type AlertEngine struct {
	config AlertConfig
}

// NewAlertEngine creates a new alert engine with the given config
func NewAlertEngine(config AlertConfig) *AlertEngine {
	return &AlertEngine{config: config}
}

// GenerateAlerts analyzes price deltas and creates relevant alerts
func (ae *AlertEngine) GenerateAlerts(deltas []PriceDelta) []Alert {
	var alerts []Alert

	for _, delta := range deltas {
		// Price drop alerts for raw cards (buying opportunity)
		if delta.Field == "Raw" && delta.DeltaPct <= -ae.config.PriceDropThresholdPct {
			alert := Alert{
				Type:      AlertPriceDrop,
				Severity:  ae.getSeverity(delta.DeltaPct),
				Card:      delta.Card,
				Message:   fmt.Sprintf("Raw price dropped %.1f%% ($%.2f)", -delta.DeltaPct, -delta.DeltaUSD),
				Timestamp: time.Now(),
				Details: map[string]interface{}{
					"old_price": delta.OldPrice,
					"new_price": delta.NewPrice,
					"delta_pct": delta.DeltaPct,
					"delta_usd": delta.DeltaUSD,
				},
				ActionItems: []string{
					fmt.Sprintf("Consider buying at new price of $%.2f", delta.NewPrice),
					"Check eBay for current listings",
					"Verify condition requirements for grading",
				},
			}
			alerts = append(alerts, alert)
		}

		// Price increase alerts for PSA10 (selling opportunity)
		if delta.Field == "PSA10" && delta.DeltaPct >= ae.config.PriceDropThresholdPct {
			alert := Alert{
				Type:      AlertPriceIncrease,
				Severity:  ae.getSeverity(delta.DeltaPct),
				Card:      delta.Card,
				Message:   fmt.Sprintf("PSA10 price increased %.1f%% ($%.2f)", delta.DeltaPct, delta.DeltaUSD),
				Timestamp: time.Now(),
				Details: map[string]interface{}{
					"old_price": delta.OldPrice,
					"new_price": delta.NewPrice,
					"delta_pct": delta.DeltaPct,
					"delta_usd": delta.DeltaUSD,
				},
				ActionItems: []string{
					"Consider selling if you own this card graded",
					fmt.Sprintf("New PSA10 price: $%.2f", delta.NewPrice),
					"Monitor for sustained price level",
				},
			}
			alerts = append(alerts, alert)
		}
	}

	// Sort alerts by severity and timestamp
	sort.Slice(alerts, func(i, j int) bool {
		if alerts[i].Severity != alerts[j].Severity {
			return severityRank(alerts[i].Severity) > severityRank(alerts[j].Severity)
		}
		return alerts[i].Timestamp.After(alerts[j].Timestamp)
	})

	return alerts
}

// CheckNewOpportunities identifies cards that have entered profitable grading range
func (ae *AlertEngine) CheckNewOpportunities(old, new *Snapshot, gradingCost, shippingCost, feePct float64) []Alert {
	var alerts []Alert

	for key, newCard := range new.Cards {
		oldCard, exists := old.Cards[key]
		if !exists {
			continue
		}

		// Calculate old and new ROI
		oldROI := calculateROI(oldCard.RawPriceUSD, oldCard.PSA10Price, gradingCost, shippingCost, feePct)
		newROI := calculateROI(newCard.RawPriceUSD, newCard.PSA10Price, gradingCost, shippingCost, feePct)

		// Check if card crossed into profitable territory
		if oldROI < ae.config.OpportunityThresholdROI && newROI >= ae.config.OpportunityThresholdROI {
			alert := Alert{
				Type:      AlertNewOpportunity,
				Severity:  "MEDIUM",
				Card:      newCard.Card,
				Message:   fmt.Sprintf("Card now profitable to grade! ROI: %.1f%%", newROI),
				Timestamp: time.Now(),
				Details: map[string]interface{}{
					"old_roi":      oldROI,
					"new_roi":      newROI,
					"raw_price":    newCard.RawPriceUSD,
					"psa10_price":  newCard.PSA10Price,
					"profit_est":   newCard.PSA10Price - newCard.RawPriceUSD - gradingCost - shippingCost - (newCard.PSA10Price * feePct),
				},
				ActionItems: []string{
					fmt.Sprintf("Buy raw at $%.2f", newCard.RawPriceUSD),
					fmt.Sprintf("Expected profit: $%.2f", newCard.PSA10Price - newCard.RawPriceUSD - gradingCost - shippingCost - (newCard.PSA10Price * feePct)),
					"Submit for grading with next batch",
				},
			}
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

func calculateROI(rawPrice, psa10Price, gradingCost, shippingCost, feePct float64) float64 {
	totalCost := rawPrice + gradingCost + shippingCost
	netRevenue := psa10Price * (1 - feePct)
	profit := netRevenue - totalCost
	return (profit / totalCost) * 100
}

func (ae *AlertEngine) getSeverity(deltaPct float64) string {
	absDelta := abs(deltaPct)
	if absDelta >= 30 {
		return "HIGH"
	} else if absDelta >= 15 {
		return "MEDIUM"
	}
	return "LOW"
}

func severityRank(severity string) int {
	switch severity {
	case "HIGH":
		return 3
	case "MEDIUM":
		return 2
	case "LOW":
		return 1
	default:
		return 0
	}
}

// FormatAlert creates a human-readable string representation of an alert
func FormatAlert(alert Alert) string {
	output := fmt.Sprintf("\n[%s] %s\n", alert.Severity, string(alert.Type))
	output += fmt.Sprintf("Card: %s - %s (#%s)\n", alert.Card.Name, alert.Card.SetName, alert.Card.Number)
	output += fmt.Sprintf("Message: %s\n", alert.Message)

	if len(alert.ActionItems) > 0 {
		output += "Recommended Actions:\n"
		for i, action := range alert.ActionItems {
			output += fmt.Sprintf("  %d. %s\n", i+1, action)
		}
	}

	return output
}