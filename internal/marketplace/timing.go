package marketplace

import (
	"fmt"
	"math"
	"time"
)

// MarketTimingAnalyzer provides market timing recommendations
type MarketTimingAnalyzer struct {
	provider MarketplaceProvider
}

// NewMarketTimingAnalyzer creates a new market timing analyzer
func NewMarketTimingAnalyzer(provider MarketplaceProvider) *MarketTimingAnalyzer {
	return &MarketTimingAnalyzer{
		provider: provider,
	}
}

// GetTimingRecommendations generates timing recommendations for a product
func (mta *MarketTimingAnalyzer) GetTimingRecommendations(productID string, historicalData []PricePoint) (*MarketTiming, error) {
	if mta.provider == nil || !mta.provider.Available() {
		return nil, fmt.Errorf("marketplace provider not available")
	}

	// Get current market data
	listings, err := mta.provider.GetActiveListings(productID)
	if err != nil {
		return nil, fmt.Errorf("getting listings: %w", err)
	}

	stats, err := mta.provider.GetPriceDistribution(productID)
	if err != nil {
		return nil, fmt.Errorf("getting price stats: %w", err)
	}

	timing := &MarketTiming{
		ProductID:          productID,
		SeasonalFactors:    mta.analyzeSeasonalFactors(historicalData),
		EventDrivenFactors: mta.analyzeEventFactors(),
		LastUpdated:        time.Now(),
	}

	// Determine current trend
	timing.CurrentTrend = mta.determineTrend(historicalData)

	// Determine best buy/sell times
	timing.BestBuyTime, timing.BestSellTime = mta.determineBestTimes(listings, stats, historicalData)

	// Generate recommendation
	timing.Recommendation = mta.generateRecommendation(timing, listings, stats)

	// Calculate confidence
	timing.Confidence = mta.calculateConfidence(historicalData, listings)

	return timing, nil
}

// PricePoint represents a historical price point
type PricePoint struct {
	Date       time.Time `json:"date"`
	PriceCents int       `json:"price_cents"`
	Volume     int       `json:"volume"`
}

// determineTrend analyzes price trend from historical data
func (mta *MarketTimingAnalyzer) determineTrend(historicalData []PricePoint) string {
	if len(historicalData) < 2 {
		return "NEUTRAL"
	}

	// Calculate moving averages
	shortTermMA := mta.calculateMovingAverage(historicalData, 7) // 7-day MA
	longTermMA := mta.calculateMovingAverage(historicalData, 30) // 30-day MA

	// Compare recent prices to moving averages
	if len(historicalData) > 0 {
		currentPrice := float64(historicalData[len(historicalData)-1].PriceCents)

		if shortTermMA > longTermMA && currentPrice > shortTermMA {
			return "BULLISH"
		} else if shortTermMA < longTermMA && currentPrice < shortTermMA {
			return "BEARISH"
		}
	}

	// Calculate price change for available data
	if len(historicalData) >= 2 {
		// Use first and last data points for trend
		oldPrice := float64(historicalData[0].PriceCents)
		newPrice := float64(historicalData[len(historicalData)-1].PriceCents)

		if oldPrice > 0 {
			change := (newPrice - oldPrice) / oldPrice * 100

			if change > 10 {
				return "BULLISH"
			} else if change < -10 {
				return "BEARISH"
			}
		}
	}

	return "NEUTRAL"
}

// calculateMovingAverage calculates simple moving average
func (mta *MarketTimingAnalyzer) calculateMovingAverage(data []PricePoint, period int) float64 {
	if len(data) < period {
		period = len(data)
	}

	if period == 0 {
		return 0
	}

	sum := 0.0
	start := len(data) - period
	for i := start; i < len(data); i++ {
		sum += float64(data[i].PriceCents)
	}

	return sum / float64(period)
}

// determineBestTimes determines optimal buy and sell times
func (mta *MarketTimingAnalyzer) determineBestTimes(listings *MarketListings, stats *PriceStats, historicalData []PricePoint) (string, string) {
	bestBuyTime := "Weekend mornings (lower competition)"
	bestSellTime := "Weekday evenings (higher traffic)"

	// Analyze day-of-week patterns if we have enough data
	if len(historicalData) >= 14 {
		weekdayPrices := mta.analyzeWeekdayPatterns(historicalData)

		// Find best buy day (lowest average price)
		minDay := 0
		minPrice := math.MaxFloat64
		for day, price := range weekdayPrices {
			if price < minPrice && price > 0 {
				minPrice = price
				minDay = day
			}
		}

		// Find best sell day (highest average price)
		maxDay := 0
		maxPrice := 0.0
		for day, price := range weekdayPrices {
			if price > maxPrice {
				maxPrice = price
				maxDay = day
			}
		}

		dayNames := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
		if minDay >= 0 && minDay < 7 {
			bestBuyTime = fmt.Sprintf("%s (historically lower prices)", dayNames[minDay])
		}
		if maxDay >= 0 && maxDay < 7 {
			bestSellTime = fmt.Sprintf("%s (historically higher prices)", dayNames[maxDay])
		}
	}

	// Adjust based on current market conditions
	if listings.TotalListings < 5 {
		bestSellTime = "Immediately (low supply)"
	} else if listings.TotalListings > 20 {
		bestBuyTime = "Now (high supply, competitive prices)"
	}

	return bestBuyTime, bestSellTime
}

// analyzeWeekdayPatterns analyzes price patterns by day of week
func (mta *MarketTimingAnalyzer) analyzeWeekdayPatterns(historicalData []PricePoint) map[int]float64 {
	weekdayPrices := make(map[int][]float64)

	for _, point := range historicalData {
		weekday := int(point.Date.Weekday())
		weekdayPrices[weekday] = append(weekdayPrices[weekday], float64(point.PriceCents))
	}

	averages := make(map[int]float64)
	for day, prices := range weekdayPrices {
		if len(prices) > 0 {
			sum := 0.0
			for _, price := range prices {
				sum += price
			}
			averages[day] = sum / float64(len(prices))
		}
	}

	return averages
}

// analyzeSeasonalFactors identifies seasonal price influences
func (mta *MarketTimingAnalyzer) analyzeSeasonalFactors(historicalData []PricePoint) []SeasonalFactor {
	factors := []SeasonalFactor{}

	// Common Pokemon TCG seasonal patterns
	factors = append(factors, SeasonalFactor{
		Period:      "December",
		Impact:      15.0,
		Description: "Holiday season typically increases demand by 15-20%",
	})

	factors = append(factors, SeasonalFactor{
		Period:      "August-September",
		Impact:      10.0,
		Description: "Back-to-school season increases trading activity",
	})

	factors = append(factors, SeasonalFactor{
		Period:      "January",
		Impact:      -10.0,
		Description: "Post-holiday slowdown typically reduces prices",
	})

	factors = append(factors, SeasonalFactor{
		Period:      "Summer (June-July)",
		Impact:      5.0,
		Description: "Tournament season increases competitive card demand",
	})

	// Analyze historical data for custom patterns if available
	if len(historicalData) >= 365 {
		monthlyAverages := mta.calculateMonthlyAverages(historicalData)
		yearAverage := mta.calculateYearAverage(monthlyAverages)

		for month, avg := range monthlyAverages {
			if avg > 0 && yearAverage > 0 {
				impact := (avg - yearAverage) / yearAverage * 100
				if math.Abs(impact) > 5 {
					factors = append(factors, SeasonalFactor{
						Period:      time.Month(month).String(),
						Impact:      impact,
						Description: fmt.Sprintf("Historical data shows %.1f%% price variation", impact),
					})
				}
			}
		}
	}

	return factors
}

// calculateMonthlyAverages calculates average prices by month
func (mta *MarketTimingAnalyzer) calculateMonthlyAverages(historicalData []PricePoint) map[int]float64 {
	monthlyPrices := make(map[int][]float64)

	for _, point := range historicalData {
		month := int(point.Date.Month())
		monthlyPrices[month] = append(monthlyPrices[month], float64(point.PriceCents))
	}

	averages := make(map[int]float64)
	for month, prices := range monthlyPrices {
		if len(prices) > 0 {
			sum := 0.0
			for _, price := range prices {
				sum += price
			}
			averages[month] = sum / float64(len(prices))
		}
	}

	return averages
}

// calculateYearAverage calculates overall year average
func (mta *MarketTimingAnalyzer) calculateYearAverage(monthlyAverages map[int]float64) float64 {
	if len(monthlyAverages) == 0 {
		return 0
	}

	sum := 0.0
	for _, avg := range monthlyAverages {
		sum += avg
	}

	return sum / float64(len(monthlyAverages))
}

// analyzeEventFactors identifies event-driven price factors
func (mta *MarketTimingAnalyzer) analyzeEventFactors() []EventDrivenFactor {
	factors := []EventDrivenFactor{}
	now := time.Now()

	// Check for upcoming Pokemon TCG events
	factors = append(factors, EventDrivenFactor{
		EventType:   "TOURNAMENT",
		EventDate:   now.AddDate(0, 1, 0), // Next month placeholder
		Impact:      8.0,
		Description: "Regional tournaments typically increase meta card prices by 5-10%",
	})

	// Check for set releases
	factors = append(factors, EventDrivenFactor{
		EventType:   "RELEASE",
		EventDate:   now.AddDate(0, 2, 0), // 2 months placeholder
		Impact:      -5.0,
		Description: "New set releases may reduce demand for older cards temporarily",
	})

	// Anniversary events
	nextFeb := time.Date(now.Year(), 2, 27, 0, 0, 0, 0, time.UTC)
	if nextFeb.Before(now) {
		nextFeb = nextFeb.AddDate(1, 0, 0)
	}
	factors = append(factors, EventDrivenFactor{
		EventType:   "ANNIVERSARY",
		EventDate:   nextFeb,
		Impact:      12.0,
		Description: "Pokemon Day (Feb 27) increases nostalgic card demand",
	})

	return factors
}

// generateRecommendation creates actionable recommendation
func (mta *MarketTimingAnalyzer) generateRecommendation(timing *MarketTiming, listings *MarketListings, stats *PriceStats) string {
	switch timing.CurrentTrend {
	case "BULLISH":
		if listings.TotalListings < 5 {
			return "SELL NOW: Rising prices with limited supply creates optimal selling conditions"
		}
		return "HOLD/SELL: Market trending upward, consider selling at peak"

	case "BEARISH":
		if stats.Mean > 0 && listings.LowestPriceCents < int(float64(stats.Mean)*0.8) {
			return "BUY: Prices declining, opportunities below market average available"
		}
		return "WAIT: Declining market, avoid buying unless finding exceptional deals"

	default: // NEUTRAL
		if listings.TotalListings > 15 {
			return "BUY SELECTIVELY: Stable market with good supply, look for underpriced listings"
		}
		return "MONITOR: Stable market conditions, wait for clear opportunities"
	}
}

// calculateConfidence calculates confidence score for recommendations
func (mta *MarketTimingAnalyzer) calculateConfidence(historicalData []PricePoint, listings *MarketListings) float64 {
	confidence := 0.5 // Base confidence

	// More historical data increases confidence
	if len(historicalData) >= 90 {
		confidence += 0.2
	} else if len(historicalData) >= 30 {
		confidence += 0.1
	}

	// More active listings increase confidence
	if listings.TotalListings >= 10 {
		confidence += 0.15
	} else if listings.TotalListings >= 5 {
		confidence += 0.05
	}

	// Recent data increases confidence
	if len(historicalData) > 0 {
		lastDataPoint := historicalData[len(historicalData)-1].Date
		daysSinceLastData := time.Since(lastDataPoint).Hours() / 24
		if daysSinceLastData < 1 {
			confidence += 0.15
		} else if daysSinceLastData < 7 {
			confidence += 0.05
		}
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}
