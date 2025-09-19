package monitoring

import (
	"fmt"
	"sort"
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
)

// PSAServiceLevel represents different PSA grading service tiers
type PSAServiceLevel struct {
	Name             string
	MaxDeclaredValue float64
	CostPerCard      float64
	TurnaroundDays   int
	MinCards         int
}

var PSAServiceLevels = []PSAServiceLevel{
	{"Value", 199, 19, 65, 20},
	{"Value Plus", 499, 25, 45, 20},
	{"Regular", 999, 39, 30, 20},
	{"Express", 2499, 75, 15, 10},
	{"Super Express", 4999, 150, 10, 5},
	{"Walk Through", 9999, 300, 5, 2},
}

// SubmissionBatch represents a group of cards for a specific service level
type SubmissionBatch struct {
	ServiceLevel    PSAServiceLevel
	Cards           []SubmissionCard
	TotalValue      float64
	TotalCost       float64
	EstimatedProfit float64
	EstimatedROI    float64
}

// SubmissionCard represents a card to be submitted for grading
type SubmissionCard struct {
	Card          model.Card
	RawUSD        float64
	PSA10Price    float64
	PSA9Price     float64
	ExpectedGrade float64
	ExpectedValue float64
}

// BulkOptimizer optimizes card submissions across PSA service levels
type BulkOptimizer struct {
	feePct               float64
	shippingCostPerBatch float64
}

// NewBulkOptimizer creates a new bulk submission optimizer
func NewBulkOptimizer(feePct, shippingCostPerBatch float64) *BulkOptimizer {
	return &BulkOptimizer{
		feePct:               feePct,
		shippingCostPerBatch: shippingCostPerBatch,
	}
}

// OptimizeSubmission groups cards into optimal batches by service level
func (bo *BulkOptimizer) OptimizeSubmission(cards []SubmissionCard) []SubmissionBatch {
	// Sort cards by PSA10 value (highest first)
	sort.Slice(cards, func(i, j int) bool {
		return cards[i].PSA10Price > cards[j].PSA10Price
	})

	batches := make(map[string]*SubmissionBatch)

	for _, card := range cards {
		// Find appropriate service level
		serviceLevel := bo.findServiceLevel(card.PSA10Price)

		// Get or create batch for this service level
		batchKey := serviceLevel.Name
		batch, exists := batches[batchKey]
		if !exists {
			batch = &SubmissionBatch{
				ServiceLevel: serviceLevel,
				Cards:        []SubmissionCard{},
			}
			batches[batchKey] = batch
		}

		// Add card to batch
		batch.Cards = append(batch.Cards, card)
		batch.TotalValue += card.PSA10Price
		batch.TotalCost += card.RawUSD + serviceLevel.CostPerCard
	}

	// Calculate profitability for each batch
	var result []SubmissionBatch
	for serviceName, batch := range batches {
		// Skip batches that don't meet minimum card requirements
		if len(batch.Cards) < batch.ServiceLevel.MinCards {
			// Try to combine with next service level down
			// fmt.Printf("Skipping %s: %d cards < %d minimum\n", serviceName, len(batch.Cards), batch.ServiceLevel.MinCards)
			_ = serviceName // avoid unused variable warning
			continue
		}

		// Calculate expected profit
		batch.EstimatedProfit = bo.calculateBatchProfit(batch)
		batch.EstimatedROI = (batch.EstimatedProfit / batch.TotalCost) * 100

		result = append(result, *batch)
	}

	// Sort batches by ROI
	sort.Slice(result, func(i, j int) bool {
		return result[i].EstimatedROI > result[j].EstimatedROI
	})

	return result
}

// GenerateSubmissionForm creates a submission summary for PSA
func (bo *BulkOptimizer) GenerateSubmissionForm(batch SubmissionBatch) string {
	output := fmt.Sprintf("PSA SUBMISSION FORM\n")
	output += fmt.Sprintf("==================\n\n")
	output += fmt.Sprintf("Service Level: %s\n", batch.ServiceLevel.Name)
	output += fmt.Sprintf("Turnaround: %d business days\n", batch.ServiceLevel.TurnaroundDays)
	output += fmt.Sprintf("Cost per Card: $%.2f\n", batch.ServiceLevel.CostPerCard)
	output += fmt.Sprintf("Total Cards: %d\n\n", len(batch.Cards))

	output += fmt.Sprintf("CARD LIST:\n")
	output += fmt.Sprintf("----------\n")

	for i, card := range batch.Cards {
		output += fmt.Sprintf("%d. %s - %s #%s\n",
			i+1, card.Card.Name, card.Card.SetName, card.Card.Number)
		output += fmt.Sprintf("   Declared Value: $%.2f\n", card.PSA10Price)
		output += fmt.Sprintf("   Expected Grade: %.1f\n\n", card.ExpectedGrade)
	}

	output += fmt.Sprintf("COST BREAKDOWN:\n")
	output += fmt.Sprintf("--------------\n")
	output += fmt.Sprintf("Grading Fees: $%.2f\n", batch.ServiceLevel.CostPerCard*float64(len(batch.Cards)))
	output += fmt.Sprintf("Shipping: $%.2f\n", bo.shippingCostPerBatch)
	output += fmt.Sprintf("Total Cost: $%.2f\n\n", batch.TotalCost+bo.shippingCostPerBatch)

	output += fmt.Sprintf("PROFIT PROJECTION:\n")
	output += fmt.Sprintf("-----------------\n")
	output += fmt.Sprintf("Total Expected Value: $%.2f\n", batch.TotalValue)
	output += fmt.Sprintf("Estimated Profit: $%.2f\n", batch.EstimatedProfit)
	output += fmt.Sprintf("Estimated ROI: %.1f%%\n", batch.EstimatedROI)

	return output
}

// RecommendSubmissionTiming suggests when to submit based on PSA backlog
func (bo *BulkOptimizer) RecommendSubmissionTiming() string {
	now := time.Now()
	month := now.Month()

	// Based on historical patterns
	goodMonths := []time.Month{time.February, time.March, time.September, time.October}
	badMonths := []time.Month{time.December, time.January, time.July, time.August}

	for _, m := range goodMonths {
		if month == m {
			return "GOOD - Lower volume period, faster turnaround expected"
		}
	}

	for _, m := range badMonths {
		if month == m {
			return "POOR - High volume period, expect delays"
		}
	}

	return "NEUTRAL - Standard turnaround times expected"
}

// SuggestBulkDiscounts provides bulk submission strategies
func (bo *BulkOptimizer) SuggestBulkDiscounts(totalCards int) string {
	if totalCards >= 100 {
		return "Eligible for PSA bulk submission (100+ cards). Consider negotiating custom pricing."
	} else if totalCards >= 50 {
		return fmt.Sprintf("Add %d more cards to reach 100-card bulk threshold for better pricing.", 100-totalCards)
	} else if totalCards >= 20 {
		return "Meets minimum for Value service (20 cards). Good for batch submission."
	}
	return fmt.Sprintf("Need %d more cards to meet Value service minimum (20 cards).", 20-totalCards)
}

func (bo *BulkOptimizer) findServiceLevel(declaredValue float64) PSAServiceLevel {
	for _, level := range PSAServiceLevels {
		if declaredValue <= level.MaxDeclaredValue {
			return level
		}
	}
	// Return highest service level if value exceeds all
	return PSAServiceLevels[len(PSAServiceLevels)-1]
}

func (bo *BulkOptimizer) calculateBatchProfit(batch *SubmissionBatch) float64 {
	totalRevenue := 0.0
	totalCost := batch.TotalCost + bo.shippingCostPerBatch

	for _, card := range batch.Cards {
		// Use expected value based on likely grade
		expectedRevenue := card.ExpectedValue * (1 - bo.feePct)
		totalRevenue += expectedRevenue
	}

	return totalRevenue - totalCost
}

// EstimateExpectedGrade calculates likely PSA grade based on historical data
func EstimateExpectedGrade(psa10Rate, psa9Rate float64) float64 {
	// Weighted average based on historical rates
	// This is simplified - real implementation would use more data
	if psa10Rate > 0.3 {
		return 9.7 // Likely 10 with some 9s
	} else if psa10Rate > 0.2 {
		return 9.5 // Mix of 10s and 9s
	} else if psa10Rate > 0.1 {
		return 9.3 // Mostly 9s with some 10s
	}
	return 9.0 // Conservative estimate
}
