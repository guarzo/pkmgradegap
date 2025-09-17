package fusion

import (
	"math"
	"time"
)

type ConfidenceCalculator struct {
	dataAgeWeights      map[time.Duration]float64
	sourceReliability   map[string]float64
	volumeWeights       map[int]float64
	volatilityThreshold float64
}

func NewConfidenceCalculator() *ConfidenceCalculator {
	return &ConfidenceCalculator{
		dataAgeWeights: map[time.Duration]float64{
			24 * time.Hour:      1.0,
			7 * 24 * time.Hour:  0.9,
			30 * 24 * time.Hour: 0.7,
			90 * 24 * time.Hour: 0.5,
		},
		sourceReliability: map[string]float64{
			"PSA_API":             1.0,
			"PriceCharting":       0.95,
			"TCGPlayer":           0.9,
			"eBay_Sales":          0.85,
			"eBay_Listings":       0.7,
			"Cardmarket":          0.8,
			"PokemonPriceTracker": 0.85,
		},
		volumeWeights: map[int]float64{
			1:  0.3,
			5:  0.6,
			10: 0.8,
			25: 0.9,
			50: 1.0,
		},
		volatilityThreshold: 0.2,
	}
}

func (c *ConfidenceCalculator) CalculateConfidence(data FusedData) ConfidenceScore {
	factors := make(map[string]float64)
	warnings := []string{}

	factors["data_freshness"] = c.calculateFreshnessScore(data)
	factors["source_reliability"] = c.calculateSourceReliabilityScore(data)
	factors["data_volume"] = c.calculateVolumeScore(data)
	factors["price_variance"] = c.varianceScoreForFusedData(data)
	factors["market_volatility"] = c.calculateVolatilityScore(data)
	factors["completeness"] = c.calculateCompletenessScore(data)

	freshness := factors["data_freshness"]
	if freshness < 0.5 {
		warnings = append(warnings, "Data may be stale (>7 days old)")
	}

	variance := factors["price_variance"]
	if variance < 0.6 {
		warnings = append(warnings, "High price variance between sources")
	}

	volume := factors["data_volume"]
	if volume < 0.5 {
		warnings = append(warnings, "Limited data points available")
	}

	overall := c.calculateWeightedConfidence(factors)

	return ConfidenceScore{
		Overall:  overall,
		Factors:  factors,
		Warnings: warnings,
	}
}

func (c *ConfidenceCalculator) calculateFreshnessScore(data FusedData) float64 {
	prices := []FusedPrice{data.RawPrice, data.PSA10Price, data.PSA9Price, data.CGC95Price, data.BGS10Price}

	var totalAge time.Duration
	var count int

	for _, price := range prices {
		if price.Value > 0 && len(price.Sources) > 0 {
			for _, source := range price.Sources {
				age := time.Since(source.Timestamp)
				totalAge += age
				count++
			}
		}
	}

	if count == 0 {
		return 0
	}

	avgAge := totalAge / time.Duration(count)

	switch {
	case avgAge <= 24*time.Hour:
		return 1.0
	case avgAge <= 7*24*time.Hour:
		return 0.9
	case avgAge <= 30*24*time.Hour:
		return 0.7
	case avgAge <= 90*24*time.Hour:
		return 0.5
	default:
		return 0.3
	}
}

func (c *ConfidenceCalculator) calculateSourceReliabilityScore(data FusedData) float64 {
	var totalReliability float64
	var count int

	prices := []FusedPrice{data.RawPrice, data.PSA10Price, data.PSA9Price, data.CGC95Price, data.BGS10Price}

	for _, price := range prices {
		if price.Value > 0 {
			for _, source := range price.Sources {
				if reliability, exists := c.sourceReliability[source.Name]; exists {
					totalReliability += reliability
					count++
				} else {
					totalReliability += 0.5
					count++
				}
			}
		}
	}

	if count == 0 {
		return 0
	}

	return totalReliability / float64(count)
}

func (c *ConfidenceCalculator) calculateVolumeScore(data FusedData) float64 {
	prices := []FusedPrice{data.RawPrice, data.PSA10Price, data.PSA9Price, data.CGC95Price, data.BGS10Price}

	var totalVolume int
	var sourceCount int

	for _, price := range prices {
		if price.Value > 0 {
			for _, source := range price.Sources {
				totalVolume += source.Volume
				sourceCount++
			}
		}
	}

	if sourceCount == 0 {
		return 0
	}

	avgVolume := totalVolume / sourceCount

	switch {
	case avgVolume >= 50:
		return 1.0
	case avgVolume >= 25:
		return 0.9
	case avgVolume >= 10:
		return 0.8
	case avgVolume >= 5:
		return 0.6
	case avgVolume >= 1:
		return 0.3
	default:
		return 0.1
	}
}

func (c *ConfidenceCalculator) calculateVarianceScore(data FusedPrice) float64 {
	if len(data.Sources) < 2 {
		return 0.8
	}

	variance := (data.ConfidenceRange.High - data.ConfidenceRange.Low) / data.Value

	if variance <= 0.1 {
		return 1.0
	} else if variance <= 0.2 {
		return 0.8
	} else if variance <= 0.3 {
		return 0.6
	} else if variance <= 0.5 {
		return 0.4
	} else {
		return 0.2
	}
}

func (c *ConfidenceCalculator) varianceScoreForFusedData(data FusedData) float64 {
	prices := []FusedPrice{data.RawPrice, data.PSA10Price, data.PSA9Price, data.CGC95Price, data.BGS10Price}

	var totalVariance float64
	var count int

	for _, price := range prices {
		if price.Value > 0 {
			variance := c.calculateVarianceScore(price)
			totalVariance += variance
			count++
		}
	}

	if count == 0 {
		return 0.5
	}

	return totalVariance / float64(count)
}

func (c *ConfidenceCalculator) calculateVolatilityScore(data FusedData) float64 {
	if len(data.Sales) < 5 {
		return 0.5
	}

	var prices []float64
	for _, sale := range data.Sales {
		prices = append(prices, sale.Price)
	}

	volatility := c.calculatePriceVolatility(prices)

	if volatility <= c.volatilityThreshold {
		return 1.0
	} else if volatility <= c.volatilityThreshold*2 {
		return 0.7
	} else if volatility <= c.volatilityThreshold*3 {
		return 0.5
	} else {
		return 0.3
	}
}

func (c *ConfidenceCalculator) calculateCompletenessScore(data FusedData) float64 {
	score := 0.0
	maxScore := 5.0

	if data.RawPrice.Value > 0 {
		score += 1.0
	}

	if data.PSA10Price.Value > 0 {
		score += 1.5
	}

	if data.Population != nil && data.Population.TotalGraded > 0 {
		score += 1.0
	}

	if len(data.Sales) > 0 {
		score += 1.0
	}

	if data.PSA9Price.Value > 0 || data.CGC95Price.Value > 0 || data.BGS10Price.Value > 0 {
		score += 0.5
	}

	return score / maxScore
}

func (c *ConfidenceCalculator) calculatePriceVolatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}

	var sum float64
	for _, price := range prices {
		sum += price
	}
	mean := sum / float64(len(prices))

	var variance float64
	for _, price := range prices {
		diff := price - mean
		variance += diff * diff
	}
	variance /= float64(len(prices))

	stdDev := math.Sqrt(variance)
	return stdDev / mean
}

func (c *ConfidenceCalculator) calculateWeightedConfidence(factors map[string]float64) float64 {
	weights := map[string]float64{
		"data_freshness":     0.20,
		"source_reliability": 0.25,
		"data_volume":        0.15,
		"price_variance":     0.20,
		"market_volatility":  0.10,
		"completeness":       0.10,
	}

	var weightedSum float64
	var totalWeight float64

	for factor, value := range factors {
		if weight, exists := weights[factor]; exists {
			weightedSum += value * weight
			totalWeight += weight
		}
	}

	if totalWeight == 0 {
		return 0
	}

	return math.Min(1.0, math.Max(0.0, weightedSum/totalWeight))
}

func (c *ConfidenceCalculator) GetConfidenceLevel(score float64) string {
	switch {
	case score >= 0.8:
		return "HIGH"
	case score >= 0.6:
		return "MEDIUM"
	case score >= 0.4:
		return "LOW"
	default:
		return "VERY_LOW"
	}
}

func (c *ConfidenceCalculator) GetRecommendation(confidence ConfidenceScore) string {
	switch {
	case confidence.Overall >= 0.8:
		return "High confidence - suitable for investment decisions"
	case confidence.Overall >= 0.6:
		return "Medium confidence - consider additional research"
	case confidence.Overall >= 0.4:
		return "Low confidence - use with caution"
	default:
		return "Very low confidence - insufficient data for decisions"
	}
}
