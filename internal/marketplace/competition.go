package marketplace

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// CompetitionAnalyzer provides market competition analysis
type CompetitionAnalyzer struct {
	provider MarketplaceProvider
}

// NewCompetitionAnalyzer creates a new competition analyzer
func NewCompetitionAnalyzer(provider MarketplaceProvider) *CompetitionAnalyzer {
	return &CompetitionAnalyzer{
		provider: provider,
	}
}

// AnalyzeMarket performs comprehensive market analysis for a product
func (ca *CompetitionAnalyzer) AnalyzeMarket(productID string) (*MarketAnalysis, error) {
	if ca.provider == nil || !ca.provider.Available() {
		return nil, fmt.Errorf("marketplace provider not available")
	}

	// Get all necessary data
	listings, err := ca.provider.GetActiveListings(productID)
	if err != nil {
		return nil, fmt.Errorf("getting listings: %w", err)
	}

	stats, err := ca.provider.GetPriceDistribution(productID)
	if err != nil {
		return nil, fmt.Errorf("getting price distribution: %w", err)
	}

	analysis := &MarketAnalysis{
		ProductID:      productID,
		PriceAnomalies: []Anomaly{},
		LastUpdated:    time.Now(),
	}

	// Calculate listing velocity (new listings per day)
	analysis.ListingVelocity = ca.calculateListingVelocity(listings)

	// Calculate sales velocity (estimate based on listing turnover)
	analysis.SalesVelocity = ca.estimateSalesVelocity(listings)

	// Calculate average days on market
	analysis.DaysOnMarket = ca.calculateDaysOnMarket(listings)

	// Calculate price volatility (coefficient of variation)
	if stats.Mean > 0 {
		analysis.PriceVolatility = stats.StandardDeviation / float64(stats.Mean)
	}

	// Calculate supply/demand ratio
	if analysis.SalesVelocity > 0 {
		analysis.SupplyDemandRatio = float64(listings.TotalListings) / analysis.SalesVelocity
	}

	// Calculate optimal listing price
	analysis.OptimalListingPrice = ca.calculateOptimalPrice(listings, stats)

	// Detect price anomalies
	analysis.PriceAnomalies = ca.detectAnomalies(listings, stats)

	return analysis, nil
}

// calculateListingVelocity calculates new listings per day
func (ca *CompetitionAnalyzer) calculateListingVelocity(listings *MarketListings) float64 {
	if len(listings.Listings) == 0 {
		return 0
	}

	now := time.Now()
	var totalAge float64
	var count int

	for _, listing := range listings.Listings {
		if !listing.ListedDate.IsZero() {
			age := now.Sub(listing.ListedDate).Hours() / 24 // days
			if age > 0 {
				totalAge += age
				count++
			}
		}
	}

	if count == 0 || totalAge == 0 {
		return 0
	}

	// Average listings per day
	return float64(count) / (totalAge / float64(count))
}

// estimateSalesVelocity estimates sales per day based on listing patterns
func (ca *CompetitionAnalyzer) estimateSalesVelocity(listings *MarketListings) float64 {
	// This is an estimate based on typical turnover rates
	// In a real implementation, this would use actual sales data
	if listings.TotalListings == 0 {
		return 0
	}

	// Estimate based on listing count and typical turnover
	// Lower prices tend to sell faster
	if listings.MedianPriceCents > 0 {
		// Base turnover rate adjusted by price
		baseTurnover := 0.1                                             // 10% daily turnover base
		priceAdjustment := 10000.0 / float64(listings.MedianPriceCents) // Higher price = lower turnover
		return float64(listings.TotalListings) * baseTurnover * math.Min(priceAdjustment, 2.0)
	}

	return float64(listings.TotalListings) * 0.1 // Default 10% daily turnover
}

// calculateDaysOnMarket calculates average days listings stay on market
func (ca *CompetitionAnalyzer) calculateDaysOnMarket(listings *MarketListings) float64 {
	if len(listings.Listings) == 0 {
		return 0
	}

	now := time.Now()
	var totalDays float64
	var count int

	for _, listing := range listings.Listings {
		if !listing.ListedDate.IsZero() {
			days := now.Sub(listing.ListedDate).Hours() / 24
			if days > 0 {
				totalDays += days
				count++
			}
		}
	}

	if count == 0 {
		return 0
	}

	return totalDays / float64(count)
}

// calculateOptimalPrice calculates the optimal listing price
func (ca *CompetitionAnalyzer) calculateOptimalPrice(listings *MarketListings, stats *PriceStats) int {
	if listings.TotalListings == 0 {
		return 0
	}

	// Strategy: Price slightly below median for faster sales
	// but above 25th percentile to maintain profitability
	optimalPrice := stats.Median

	// If high competition (many listings), price more aggressively
	if listings.TotalListings > 10 {
		// Price between 25th percentile and median
		optimalPrice = (stats.Percentile25 + stats.Median) / 2
	} else if listings.TotalListings < 3 {
		// Low competition, can price higher
		optimalPrice = (stats.Median + stats.Percentile75) / 2
	}

	// Ensure price is not below lowest current listing by more than 10%
	minAcceptable := int(float64(listings.LowestPriceCents) * 0.9)
	if optimalPrice < minAcceptable {
		optimalPrice = minAcceptable
	}

	return optimalPrice
}

// detectAnomalies identifies pricing anomalies
func (ca *CompetitionAnalyzer) detectAnomalies(listings *MarketListings, stats *PriceStats) []Anomaly {
	anomalies := []Anomaly{}

	if stats.Mean == 0 {
		return anomalies
	}

	// Calculate IQR for outlier detection
	iqr := float64(stats.Percentile75 - stats.Percentile25)
	if iqr == 0 {
		// Use standard deviation if IQR is 0
		iqr = stats.StandardDeviation * 2
	}

	// Calculate bounds - use median as center if available, otherwise use mean
	center := float64(stats.Median)
	if center == 0 {
		center = float64(stats.Mean)
	}

	lowerBound := center - 1.5*iqr
	upperBound := center + 1.5*iqr

	// Make sure bounds are reasonable
	if lowerBound < 0 {
		lowerBound = float64(stats.Mean) * 0.2 // At least 80% below mean
	}

	for _, listing := range listings.Listings {
		price := float64(listing.PriceCents)
		mean := float64(stats.Mean)

		// Calculate deviation percentage
		deviation := (price - mean) / mean * 100

		var anomalyType string
		var description string

		// Check for outliers or significant deviations
		if price < lowerBound || deviation < -50 {
			anomalyType = "UNDERPRICED"
			description = fmt.Sprintf("Price is %.1f%% below market average, potential arbitrage opportunity", math.Abs(deviation))
		} else if price > upperBound || deviation > 100 {
			anomalyType = "OVERPRICED"
			description = fmt.Sprintf("Price is %.1f%% above market average, unlikely to sell quickly", deviation)
		} else if math.Abs(deviation) > 50 {
			anomalyType = "OUTLIER"
			description = fmt.Sprintf("Significant price deviation of %.1f%% from market average", deviation)
		}

		if anomalyType != "" {
			anomalies = append(anomalies, Anomaly{
				Type:        anomalyType,
				PriceCents:  listing.PriceCents,
				Deviation:   deviation,
				SellerID:    listing.SellerID,
				DetectedAt:  time.Now(),
				Description: description,
			})
		}
	}

	// Sort anomalies by absolute deviation
	sort.Slice(anomalies, func(i, j int) bool {
		return math.Abs(anomalies[i].Deviation) > math.Abs(anomalies[j].Deviation)
	})

	// Return top 10 anomalies
	if len(anomalies) > 10 {
		anomalies = anomalies[:10]
	}

	return anomalies
}

// IdentifyOpportunities identifies market opportunities
func (ca *CompetitionAnalyzer) IdentifyOpportunities(analysis *MarketAnalysis) []string {
	opportunities := []string{}

	// Check for arbitrage opportunities
	if len(analysis.PriceAnomalies) > 0 {
		for _, anomaly := range analysis.PriceAnomalies {
			if anomaly.Type == "UNDERPRICED" && anomaly.Deviation < -30 {
				opportunities = append(opportunities,
					fmt.Sprintf("Arbitrage opportunity: listing %.1f%% below market average", math.Abs(anomaly.Deviation)))
			}
		}
	}

	// Check supply/demand imbalance
	if analysis.SupplyDemandRatio < 5 && analysis.SalesVelocity > 1 {
		opportunities = append(opportunities, "High demand with limited supply - good time to list")
	} else if analysis.SupplyDemandRatio > 20 {
		opportunities = append(opportunities, "Oversupplied market - consider waiting or competitive pricing")
	}

	// Check price volatility
	if analysis.PriceVolatility > 0.3 {
		opportunities = append(opportunities, "High price volatility - monitor for buying dips")
	} else if analysis.PriceVolatility < 0.1 {
		opportunities = append(opportunities, "Stable pricing - predictable market conditions")
	}

	// Check market velocity
	if analysis.DaysOnMarket < 7 && analysis.SalesVelocity > 2 {
		opportunities = append(opportunities, "Fast-moving market - quick turnover expected")
	} else if analysis.DaysOnMarket > 30 {
		opportunities = append(opportunities, "Slow-moving market - patience required for sales")
	}

	return opportunities
}
