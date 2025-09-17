package monitoring

import (
	"fmt"
	"sort"
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
)

// MarketTrend represents price movement direction
type MarketTrend string

const (
	TrendUp   MarketTrend = "UPWARD"
	TrendDown MarketTrend = "DOWNWARD"
	TrendFlat MarketTrend = "FLAT"
)

// TimingRecommendation provides buy/sell guidance
type TimingRecommendation struct {
	Card         model.Card
	Action       string  // "BUY", "SELL", "HOLD", "SUBMIT"
	Confidence   float64 // 0-100%
	Trend        MarketTrend
	Reasoning    string
	OptimalPrice float64
	CurrentPrice float64
	Timestamp    time.Time
}

// MarketAnalyzer provides timing recommendations based on historical data
type MarketAnalyzer struct {
	snapshots []*Snapshot // Historical snapshots in chronological order
}

// NewMarketAnalyzer creates a new market analyzer
func NewMarketAnalyzer(snapshots []*Snapshot) *MarketAnalyzer {
	// Sort snapshots by timestamp
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Timestamp.Before(snapshots[j].Timestamp)
	})
	return &MarketAnalyzer{snapshots: snapshots}
}

// AnalyzeCard provides timing recommendations for a specific card
func (ma *MarketAnalyzer) AnalyzeCard(cardKey string) *TimingRecommendation {
	if len(ma.snapshots) < 2 {
		return nil // Need at least 2 snapshots for analysis
	}

	// Extract price history for this card
	var rawPrices []float64
	var psa10Prices []float64
	var timestamps []time.Time

	for _, snapshot := range ma.snapshots {
		if card, exists := snapshot.Cards[cardKey]; exists {
			rawPrices = append(rawPrices, card.RawPriceUSD)
			psa10Prices = append(psa10Prices, card.PSA10Price)
			timestamps = append(timestamps, snapshot.Timestamp)
		}
	}

	if len(rawPrices) < 2 {
		return nil // Not enough data points
	}

	latest := ma.snapshots[len(ma.snapshots)-1].Cards[cardKey]
	if latest == nil {
		return nil
	}

	// Calculate trends
	rawTrend := calculateTrend(rawPrices)
	psa10Trend := calculateTrend(psa10Prices)

	// Generate recommendation
	rec := &TimingRecommendation{
		Card:         latest.Card,
		CurrentPrice: latest.RawPriceUSD,
		Timestamp:    time.Now(),
	}

	// Determine action based on trends
	if rawTrend == TrendDown && psa10Trend != TrendDown {
		rec.Action = "BUY"
		rec.Confidence = calculateConfidence(rawPrices, psa10Prices)
		rec.Trend = rawTrend
		rec.OptimalPrice = calculateOptimalBuyPrice(rawPrices)
		rec.Reasoning = fmt.Sprintf("Raw prices trending down (%.1f%% from peak) while PSA10 stable. Good entry point.",
			calculateDropFromPeak(rawPrices))
	} else if psa10Trend == TrendUp && rawTrend != TrendUp {
		rec.Action = "SUBMIT"
		rec.Confidence = calculateConfidence(rawPrices, psa10Prices)
		rec.Trend = psa10Trend
		rec.Reasoning = "PSA10 prices rising faster than raw. Submit existing inventory for grading."
	} else if psa10Trend == TrendUp && calculateRecentGrowth(psa10Prices) > 20 {
		rec.Action = "SELL"
		rec.Confidence = calculateConfidence(rawPrices, psa10Prices)
		rec.Trend = psa10Trend
		rec.OptimalPrice = latest.PSA10Price
		rec.Reasoning = fmt.Sprintf("PSA10 up %.1f%% recently. Consider taking profits.",
			calculateRecentGrowth(psa10Prices))
	} else {
		rec.Action = "HOLD"
		rec.Confidence = 50
		rec.Trend = TrendFlat
		rec.Reasoning = "No clear market signal. Continue monitoring."
	}

	return rec
}

// AnalyzeMarket provides overall market timing recommendations
func (ma *MarketAnalyzer) AnalyzeMarket(gradingCost, shippingCost, feePct float64) []TimingRecommendation {
	if len(ma.snapshots) < 2 {
		return nil
	}

	latest := ma.snapshots[len(ma.snapshots)-1]
	var recommendations []TimingRecommendation

	for cardKey := range latest.Cards {
		rec := ma.AnalyzeCard(cardKey)
		if rec != nil && rec.Action != "HOLD" {
			// Calculate ROI for context
			card := latest.Cards[cardKey]
			roi := calculateROI(card.RawPriceUSD, card.PSA10Price, gradingCost, shippingCost, feePct)

			// Only include if profitable
			if roi > 20 && rec.Action == "BUY" {
				recommendations = append(recommendations, *rec)
			} else if rec.Action == "SELL" || rec.Action == "SUBMIT" {
				recommendations = append(recommendations, *rec)
			}
		}
	}

	// Sort by confidence
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Confidence > recommendations[j].Confidence
	})

	return recommendations
}

// SeasonalAnalysis identifies seasonal patterns
func (ma *MarketAnalyzer) SeasonalAnalysis() map[string]string {
	patterns := make(map[string]string)

	// Analyze monthly patterns
	monthlyPrices := make(map[time.Month][]float64)
	for _, snapshot := range ma.snapshots {
		month := snapshot.Timestamp.Month()
		var avgPrice float64
		count := 0
		for _, card := range snapshot.Cards {
			avgPrice += card.PSA10Price
			count++
		}
		if count > 0 {
			monthlyPrices[month] = append(monthlyPrices[month], avgPrice/float64(count))
		}
	}

	// Identify best/worst months
	var bestMonth time.Month
	var worstMonth time.Month
	var highestAvg, lowestAvg float64 = 0, 999999

	for month, prices := range monthlyPrices {
		avg := average(prices)
		if avg > highestAvg {
			highestAvg = avg
			bestMonth = month
		}
		if avg < lowestAvg {
			lowestAvg = avg
			worstMonth = month
		}
	}

	patterns["best_month"] = bestMonth.String()
	patterns["worst_month"] = worstMonth.String()
	patterns["seasonal_delta"] = fmt.Sprintf("%.1f%%", ((highestAvg-lowestAvg)/lowestAvg)*100)

	return patterns
}

func calculateTrend(prices []float64) MarketTrend {
	if len(prices) < 2 {
		return TrendFlat
	}

	// Simple linear regression slope
	n := float64(len(prices))
	var sumX, sumY, sumXY, sumX2 float64
	for i, price := range prices {
		x := float64(i)
		sumX += x
		sumY += price
		sumXY += x * price
		sumX2 += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	// Determine trend based on slope relative to average price
	avgPrice := sumY / n
	slopePercent := (slope / avgPrice) * 100

	if slopePercent > 2 {
		return TrendUp
	} else if slopePercent < -2 {
		return TrendDown
	}
	return TrendFlat
}

func calculateConfidence(rawPrices, psa10Prices []float64) float64 {
	// Base confidence on consistency of trend
	rawTrend := calculateTrend(rawPrices)
	psa10Trend := calculateTrend(psa10Prices)

	confidence := 50.0

	// Strong signal: opposite trends
	if (rawTrend == TrendDown && psa10Trend == TrendUp) ||
		(rawTrend == TrendUp && psa10Trend == TrendDown) {
		confidence += 30
	}

	// Check volatility
	rawVol := calculateVolatility(rawPrices)
	if rawVol < 10 {
		confidence += 10 // Low volatility increases confidence
	} else if rawVol > 30 {
		confidence -= 10 // High volatility decreases confidence
	}

	// Ensure confidence stays in valid range
	if confidence > 100 {
		confidence = 100
	} else if confidence < 0 {
		confidence = 0
	}

	return confidence
}

func calculateOptimalBuyPrice(prices []float64) float64 {
	if len(prices) == 0 {
		return 0
	}

	// Find recent low (last 30% of data points)
	recentStart := len(prices) * 7 / 10
	recentPrices := prices[recentStart:]

	min := recentPrices[0]
	for _, p := range recentPrices {
		if p < min {
			min = p
		}
	}

	// Add small margin above recent low
	return min * 1.02
}

func calculateDropFromPeak(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}

	max := prices[0]
	for _, p := range prices {
		if p > max {
			max = p
		}
	}

	current := prices[len(prices)-1]
	return ((max - current) / max) * 100
}

func calculateRecentGrowth(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}

	// Compare last value to value from 30% back
	lookback := len(prices) * 7 / 10
	if lookback >= len(prices)-1 {
		lookback = len(prices) - 2
	}

	old := prices[lookback]
	current := prices[len(prices)-1]

	if old <= 0 {
		return 0
	}

	return ((current - old) / old) * 100
}

func calculateVolatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}

	avg := average(prices)
	var sumSquaredDiff float64

	for _, p := range prices {
		diff := p - avg
		sumSquaredDiff += diff * diff
	}

	variance := sumSquaredDiff / float64(len(prices))
	stdDev := variance // Simplified, should be sqrt(variance)

	return (stdDev / avg) * 100
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// FormatTimingReport creates a human-readable timing analysis report
func FormatTimingReport(recommendations []TimingRecommendation, setName string, seasonal map[string]string) string {
	output := fmt.Sprintf("MARKET TIMING ANALYSIS - %s\n", setName)
	output += "===============================\n\n"

	if len(recommendations) == 0 {
		output += "No timing recommendations available (insufficient data or no profitable opportunities)\n"
		return output
	}

	// Group recommendations by action
	buyRecs := make([]TimingRecommendation, 0)
	sellRecs := make([]TimingRecommendation, 0)
	submitRecs := make([]TimingRecommendation, 0)

	for _, rec := range recommendations {
		switch rec.Action {
		case "BUY":
			buyRecs = append(buyRecs, rec)
		case "SELL":
			sellRecs = append(sellRecs, rec)
		case "SUBMIT":
			submitRecs = append(submitRecs, rec)
		}
	}

	// Seasonal analysis section
	if len(seasonal) > 0 {
		output += "SEASONAL PATTERNS:\n"
		output += "==================\n"
		if bestMonth, ok := seasonal["best_month"]; ok {
			output += fmt.Sprintf("Best Month for Selling: %s\n", bestMonth)
		}
		if worstMonth, ok := seasonal["worst_month"]; ok {
			output += fmt.Sprintf("Best Month for Buying: %s\n", worstMonth)
		}
		if delta, ok := seasonal["seasonal_delta"]; ok {
			output += fmt.Sprintf("Seasonal Price Variation: %s\n", delta)
		}
		output += "\n"
	}

	// Buy recommendations
	if len(buyRecs) > 0 {
		output += "BUY RECOMMENDATIONS:\n"
		output += "====================\n"
		for i, rec := range buyRecs {
			if i >= 10 { // Limit to top 10
				break
			}
			output += formatSingleRecommendation(rec, i+1)
		}
		output += "\n"
	}

	// Sell recommendations
	if len(sellRecs) > 0 {
		output += "SELL RECOMMENDATIONS:\n"
		output += "=====================\n"
		for i, rec := range sellRecs {
			if i >= 10 { // Limit to top 10
				break
			}
			output += formatSingleRecommendation(rec, i+1)
		}
		output += "\n"
	}

	// Submit recommendations
	if len(submitRecs) > 0 {
		output += "GRADING SUBMISSION RECOMMENDATIONS:\n"
		output += "===================================\n"
		for i, rec := range submitRecs {
			if i >= 10 { // Limit to top 10
				break
			}
			output += formatSingleRecommendation(rec, i+1)
		}
		output += "\n"
	}

	// Summary
	output += "SUMMARY:\n"
	output += "========\n"
	output += fmt.Sprintf("Total Recommendations: %d\n", len(recommendations))
	output += fmt.Sprintf("Buy Opportunities: %d\n", len(buyRecs))
	output += fmt.Sprintf("Sell Opportunities: %d\n", len(sellRecs))
	output += fmt.Sprintf("Submit Opportunities: %d\n", len(submitRecs))

	// Calculate average confidence
	avgConfidence := 0.0
	for _, rec := range recommendations {
		avgConfidence += rec.Confidence
	}
	if len(recommendations) > 0 {
		avgConfidence /= float64(len(recommendations))
	}
	output += fmt.Sprintf("Average Confidence: %.1f%%\n", avgConfidence)

	return output
}

func formatSingleRecommendation(rec TimingRecommendation, rank int) string {
	output := fmt.Sprintf("%d. %s - %s (#%s)\n", rank, rec.Card.Name, rec.Card.SetName, rec.Card.Number)
	output += fmt.Sprintf("   Action: %s (Confidence: %.1f%%)\n", rec.Action, rec.Confidence)
	output += fmt.Sprintf("   Trend: %s\n", rec.Trend)
	output += fmt.Sprintf("   Current Price: $%.2f\n", rec.CurrentPrice)
	if rec.OptimalPrice > 0 {
		output += fmt.Sprintf("   Target Price: $%.2f\n", rec.OptimalPrice)
	}
	output += fmt.Sprintf("   Reasoning: %s\n", rec.Reasoning)
	output += "\n"
	return output
}
