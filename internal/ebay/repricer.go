package ebay

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
	"github.com/guarzo/pkmgradegap/internal/prices"
)

// PriceSuggestion represents pricing recommendation for a listing
type PriceSuggestion struct {
	ListingID          string               `json:"listingId"`
	CurrentPrice       float64              `json:"currentPrice"`
	SuggestedPrice     float64              `json:"suggestedPrice"`
	MarketAverage      float64              `json:"marketAverage"`
	CompetitorLow      float64              `json:"competitorLow"`
	CompetitorHigh     float64              `json:"competitorHigh"`
	TCGPlayerPrice     float64              `json:"tcgPlayerPrice"`
	PriceChartingPrice float64              `json:"priceChartingPrice"`
	RecentSalesAvg     float64              `json:"recentSalesAvg"`
	Confidence         float64              `json:"confidence"`
	Reason             string               `json:"reason"`
	PriceChange        float64              `json:"priceChange"`
	CompetitorCount    int                  `json:"competitorCount"`
	DaysActive         int                  `json:"daysActive"`
	VelocityScore      float64              `json:"velocityScore"`
	Action             string               `json:"action"`
	PopulationData     *model.PSAPopulation `json:"populationData,omitempty"`
	Factors            []PriceFactor        `json:"factors"`
}

// PriceFactor represents a factor affecting price suggestion
type PriceFactor struct {
	Name   string  `json:"name"`
	Impact float64 `json:"impact"` // -1 to 1 scale
	Reason string  `json:"reason"`
}

// MarketData holds market information for price calculation
type MarketData struct {
	TCGPlayerPrice     float64
	PriceChartingPrice float64
	CompetitorPrices   []float64
	RecentSales        []float64
	PopulationData     *model.PSAPopulation
	MarketTrend        string // BULLISH, BEARISH, NEUTRAL
}

// Repricer analyzes listings and suggests optimal prices
type Repricer struct {
	priceProvider *prices.PriceCharting
	findingClient *Client // Use existing Finding API client
}

// NewRepricer creates a new repricer instance
func NewRepricer(priceProvider *prices.PriceCharting, findingClient *Client) *Repricer {
	return &Repricer{
		priceProvider: priceProvider,
		findingClient: findingClient,
	}
}

// AnalyzeListing analyzes a single listing and suggests optimal price
func (r *Repricer) AnalyzeListing(listing UserListing, marketData MarketData) (*PriceSuggestion, error) {
	suggestion := &PriceSuggestion{
		ListingID:          listing.ItemID,
		CurrentPrice:       listing.CurrentPrice,
		TCGPlayerPrice:     marketData.TCGPlayerPrice,
		PriceChartingPrice: marketData.PriceChartingPrice,
		PopulationData:     marketData.PopulationData,
		Factors:            []PriceFactor{},
	}

	// Calculate days active
	suggestion.DaysActive = int(time.Since(listing.StartTime).Hours() / 24)

	// Analyze competitor prices
	if len(marketData.CompetitorPrices) > 0 {
		suggestion.CompetitorCount = len(marketData.CompetitorPrices)
		sort.Float64s(marketData.CompetitorPrices)
		suggestion.CompetitorLow = marketData.CompetitorPrices[0]
		suggestion.CompetitorHigh = marketData.CompetitorPrices[len(marketData.CompetitorPrices)-1]

		// Calculate market average
		var sum float64
		for _, price := range marketData.CompetitorPrices {
			sum += price
		}
		suggestion.MarketAverage = sum / float64(len(marketData.CompetitorPrices))
	}

	// Calculate recent sales average
	if len(marketData.RecentSales) > 0 {
		var sum float64
		for _, sale := range marketData.RecentSales {
			sum += sale
		}
		suggestion.RecentSalesAvg = sum / float64(len(marketData.RecentSales))
	}

	// Calculate velocity score (views to watch ratio)
	if listing.ViewCount > 0 {
		watchRate := float64(listing.WatchCount) / float64(listing.ViewCount)
		suggestion.VelocityScore = math.Min(watchRate*100, 100) // Convert to percentage
	}

	// Apply pricing algorithm
	suggestedPrice, factors := r.calculateOptimalPrice(listing, suggestion, marketData)
	suggestion.SuggestedPrice = suggestedPrice
	suggestion.Factors = factors

	// Calculate price change percentage
	if listing.CurrentPrice > 0 {
		suggestion.PriceChange = ((suggestedPrice - listing.CurrentPrice) / listing.CurrentPrice) * 100
	}

	// Determine action
	if suggestion.PriceChange < -5 {
		suggestion.Action = "DECREASE"
		suggestion.Reason = r.generateReason(factors, "decrease")
	} else if suggestion.PriceChange > 5 {
		suggestion.Action = "INCREASE"
		suggestion.Reason = r.generateReason(factors, "increase")
	} else {
		suggestion.Action = "HOLD"
		suggestion.Reason = "Price is within optimal range"
	}

	// Calculate confidence score
	suggestion.Confidence = r.calculateConfidence(suggestion, marketData)

	return suggestion, nil
}

// calculateOptimalPrice determines the optimal price based on multiple factors
func (r *Repricer) calculateOptimalPrice(listing UserListing, suggestion *PriceSuggestion, marketData MarketData) (float64, []PriceFactor) {
	factors := []PriceFactor{}
	basePrice := listing.CurrentPrice

	// Start with market average as baseline
	if suggestion.MarketAverage > 0 {
		basePrice = suggestion.MarketAverage
		factors = append(factors, PriceFactor{
			Name:   "Market Average",
			Impact: 0,
			Reason: fmt.Sprintf("Starting from market average of $%.2f", suggestion.MarketAverage),
		})
	}

	// Factor 1: Competition level
	if suggestion.CompetitorCount > 10 {
		// High competition - price competitively
		if listing.CurrentPrice > suggestion.CompetitorLow*1.1 {
			adjustment := -0.05 // 5% reduction
			basePrice *= (1 + adjustment)
			factors = append(factors, PriceFactor{
				Name:   "High Competition",
				Impact: adjustment,
				Reason: fmt.Sprintf("%d competitors found, suggesting competitive pricing", suggestion.CompetitorCount),
			})
		}
	} else if suggestion.CompetitorCount < 3 {
		// Low competition - can price higher
		adjustment := 0.05 // 5% increase
		basePrice *= (1 + adjustment)
		factors = append(factors, PriceFactor{
			Name:   "Low Competition",
			Impact: adjustment,
			Reason: "Limited competition allows premium pricing",
		})
	}

	// Factor 2: Days on market (staleness)
	if suggestion.DaysActive > 30 {
		adjustment := -0.10 // 10% reduction for stale listings
		basePrice *= (1 + adjustment)
		factors = append(factors, PriceFactor{
			Name:   "Stale Listing",
			Impact: adjustment,
			Reason: fmt.Sprintf("Listing active for %d days without sale", suggestion.DaysActive),
		})
	} else if suggestion.DaysActive < 7 && suggestion.VelocityScore > 10 {
		// New listing with good engagement
		adjustment := 0.03 // 3% increase
		basePrice *= (1 + adjustment)
		factors = append(factors, PriceFactor{
			Name:   "High Engagement",
			Impact: adjustment,
			Reason: "New listing with strong viewer interest",
		})
	}

	// Factor 3: View/Watch ratio (demand indicator)
	if suggestion.VelocityScore < 5 && listing.ViewCount > 50 {
		// Low watch rate despite views
		adjustment := -0.07
		basePrice *= (1 + adjustment)
		factors = append(factors, PriceFactor{
			Name:   "Low Demand",
			Impact: adjustment,
			Reason: fmt.Sprintf("Only %.1f%% watch rate from %d views", suggestion.VelocityScore, listing.ViewCount),
		})
	} else if suggestion.VelocityScore > 15 {
		// High demand
		adjustment := 0.05
		basePrice *= (1 + adjustment)
		factors = append(factors, PriceFactor{
			Name:   "High Demand",
			Impact: adjustment,
			Reason: fmt.Sprintf("Strong %.1f%% watch rate indicates high interest", suggestion.VelocityScore),
		})
	}

	// Factor 4: Recent sales comparison
	if suggestion.RecentSalesAvg > 0 {
		if listing.CurrentPrice > suggestion.RecentSalesAvg*1.15 {
			// Priced too high vs recent sales
			adjustment := -0.08
			basePrice = suggestion.RecentSalesAvg * 1.05 // Price slightly above recent sales
			factors = append(factors, PriceFactor{
				Name:   "Above Recent Sales",
				Impact: adjustment,
				Reason: fmt.Sprintf("Current price exceeds recent sales avg of $%.2f", suggestion.RecentSalesAvg),
			})
		}
	}

	// Factor 5: Population/Rarity (if available)
	if marketData.PopulationData != nil {
		pop10 := marketData.PopulationData.PSA10
		if pop10 > 0 && pop10 < 100 {
			// Rare card - premium pricing
			adjustment := 0.10
			basePrice *= (1 + adjustment)
			factors = append(factors, PriceFactor{
				Name:   "Low Population",
				Impact: adjustment,
				Reason: fmt.Sprintf("Only %d PSA 10s in existence", pop10),
			})
		} else if pop10 > 1000 {
			// Common card - competitive pricing
			adjustment := -0.05
			basePrice *= (1 + adjustment)
			factors = append(factors, PriceFactor{
				Name:   "High Population",
				Impact: adjustment,
				Reason: fmt.Sprintf("%d PSA 10s available", pop10),
			})
		}
	}

	// Factor 6: Market trend
	if marketData.MarketTrend == "BULLISH" {
		adjustment := 0.03
		basePrice *= (1 + adjustment)
		factors = append(factors, PriceFactor{
			Name:   "Bullish Market",
			Impact: adjustment,
			Reason: "Market trending upward for this card",
		})
	} else if marketData.MarketTrend == "BEARISH" {
		adjustment := -0.05
		basePrice *= (1 + adjustment)
		factors = append(factors, PriceFactor{
			Name:   "Bearish Market",
			Impact: adjustment,
			Reason: "Market trending downward, aggressive pricing recommended",
		})
	}

	// Round to nearest sensible price point
	if basePrice < 10 {
		basePrice = math.Round(basePrice*100) / 100 // Round to cents
	} else if basePrice < 100 {
		basePrice = math.Round(basePrice*2) / 2 // Round to nearest 0.50
	} else {
		basePrice = math.Round(basePrice) // Round to nearest dollar
	}

	return basePrice, factors
}

// calculateConfidence determines how confident we are in the suggestion
func (r *Repricer) calculateConfidence(suggestion *PriceSuggestion, marketData MarketData) float64 {
	confidence := 50.0 // Start at 50%

	// More data points increase confidence
	if suggestion.CompetitorCount >= 5 {
		confidence += 10
	}
	if suggestion.CompetitorCount >= 10 {
		confidence += 10
	}

	if suggestion.RecentSalesAvg > 0 {
		confidence += 15
	}

	if marketData.TCGPlayerPrice > 0 {
		confidence += 10
	}

	if marketData.PriceChartingPrice > 0 {
		confidence += 10
	}

	if marketData.PopulationData != nil {
		confidence += 5
	}

	// High variance reduces confidence
	if suggestion.CompetitorCount > 0 {
		priceRange := suggestion.CompetitorHigh - suggestion.CompetitorLow
		avgPrice := (suggestion.CompetitorHigh + suggestion.CompetitorLow) / 2
		variance := priceRange / avgPrice

		if variance > 0.5 {
			confidence -= 10
		}
		if variance > 1.0 {
			confidence -= 10
		}
	}

	// Cap confidence at 95%
	return math.Min(confidence, 95)
}

// generateReason creates a human-readable reason for the price suggestion
func (r *Repricer) generateReason(factors []PriceFactor, action string) string {
	if len(factors) == 0 {
		return "Price adjustment based on market analysis"
	}

	// Find the most impactful factor
	var primaryFactor PriceFactor
	maxImpact := 0.0
	for _, factor := range factors {
		if math.Abs(factor.Impact) > maxImpact {
			maxImpact = math.Abs(factor.Impact)
			primaryFactor = factor
		}
	}

	// Build reason string
	reasons := []string{primaryFactor.Reason}

	// Add secondary factors if significant
	for _, factor := range factors {
		if factor.Name != primaryFactor.Name && math.Abs(factor.Impact) >= 0.05 {
			reasons = append(reasons, strings.ToLower(factor.Reason))
		}
	}

	if len(reasons) > 3 {
		reasons = reasons[:3] // Limit to 3 reasons
	}

	return strings.Join(reasons, "; ")
}

// AnalyzeBatch analyzes multiple listings for repricing
func (r *Repricer) AnalyzeBatch(listings []UserListing) ([]*PriceSuggestion, error) {
	suggestions := make([]*PriceSuggestion, 0, len(listings))

	for _, listing := range listings {
		// Get market data for this listing
		marketData, err := r.fetchMarketData(listing)
		if err != nil {
			// Skip listings we can't analyze
			continue
		}

		suggestion, err := r.AnalyzeListing(listing, *marketData)
		if err != nil {
			continue
		}

		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// fetchMarketData retrieves market data for a listing
func (r *Repricer) fetchMarketData(listing UserListing) (*MarketData, error) {
	marketData := &MarketData{
		CompetitorPrices: []float64{},
		RecentSales:      []float64{},
	}

	// Get competitor prices using Finding API
	if r.findingClient != nil && r.findingClient.Available() {
		competitors, err := r.findingClient.SearchRawListings(
			listing.SetName,
			listing.CardName,
			listing.CardNumber,
			20, // Get top 20 competitors
		)
		if err == nil {
			for _, comp := range competitors {
				if comp.Price > 0 {
					marketData.CompetitorPrices = append(marketData.CompetitorPrices, comp.Price)
				}
			}
		}
	}

	// Get prices from PriceCharting if available
	if r.priceProvider != nil && r.priceProvider.Available() {
		// Create a model.Card for lookup
		card := model.Card{
			Name:   listing.CardName,
			Number: listing.CardNumber,
		}

		if match, err := r.priceProvider.LookupCard(listing.SetName, card); err == nil && match != nil {
			// Extract relevant prices based on condition (convert cents to dollars)
			if strings.Contains(strings.ToLower(listing.Condition), "mint") ||
				strings.Contains(strings.ToLower(listing.Condition), "near mint") {
				marketData.PriceChartingPrice = float64(match.LooseCents) / 100.0
			}
		}
	}

	// Determine market trend (simplified - would use historical data in production)
	if len(marketData.CompetitorPrices) > 10 {
		// Simple trend detection based on price distribution
		avgPrice := 0.0
		for _, p := range marketData.CompetitorPrices {
			avgPrice += p
		}
		avgPrice /= float64(len(marketData.CompetitorPrices))

		highCount := 0
		for _, p := range marketData.CompetitorPrices {
			if p > avgPrice*1.1 {
				highCount++
			}
		}

		if float64(highCount)/float64(len(marketData.CompetitorPrices)) > 0.3 {
			marketData.MarketTrend = "BULLISH"
		} else if float64(highCount)/float64(len(marketData.CompetitorPrices)) < 0.1 {
			marketData.MarketTrend = "BEARISH"
		} else {
			marketData.MarketTrend = "NEUTRAL"
		}
	} else {
		marketData.MarketTrend = "NEUTRAL"
	}

	return marketData, nil
}
