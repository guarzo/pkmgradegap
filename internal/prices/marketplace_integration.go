package prices

import (
	"github.com/guarzo/pkmgradegap/internal/cache"
	"github.com/guarzo/pkmgradegap/internal/marketplace"
)

// MarketplaceEnricher enriches price data with marketplace information
type MarketplaceEnricher struct {
	marketProvider      marketplace.MarketplaceProvider
	competitionAnalyzer *marketplace.CompetitionAnalyzer
	timingAnalyzer      *marketplace.MarketTimingAnalyzer
}

// NewMarketplaceEnricher creates a new marketplace enricher
func NewMarketplaceEnricher(apiKey string, c *cache.Cache) *MarketplaceEnricher {
	provider := marketplace.NewPriceChartingMarketplace(apiKey, c)

	if provider == nil || !provider.Available() {
		return nil
	}

	return &MarketplaceEnricher{
		marketProvider:      provider,
		competitionAnalyzer: marketplace.NewCompetitionAnalyzer(provider),
		timingAnalyzer:      marketplace.NewMarketTimingAnalyzer(provider),
	}
}

// EnrichPCMatch adds marketplace data to a PCMatch
func (e *MarketplaceEnricher) EnrichPCMatch(match *PCMatch) error {
	if e == nil || match == nil || match.ID == "" {
		return nil // Silent failure for missing enricher or invalid match
	}

	// Get marketplace analysis
	analysis, err := e.competitionAnalyzer.AnalyzeMarket(match.ID)
	if err != nil {
		// Silently skip on error - marketplace data is supplemental
		return nil
	}

	// Populate marketplace fields
	if analysis != nil {
		match.ListingVelocity = analysis.ListingVelocity
		match.OptimalListingPrice = analysis.OptimalListingPrice
		match.SupplyDemandRatio = analysis.SupplyDemandRatio
		match.PriceVolatility = analysis.PriceVolatility
	}

	// Get basic listing data for additional fields
	listings, _ := e.marketProvider.GetActiveListings(match.ID)
	if listings != nil {
		match.ActiveListings = listings.TotalListings
		match.LowestListing = listings.LowestPriceCents
		match.AverageListingPrice = listings.AveragePriceCents
	}

	// Get seller metrics for competition level
	sellerData, _ := e.marketProvider.GetSellerMetrics(match.ID)
	if sellerData != nil {
		match.CompetitionLevel = sellerData.CompetitionLevel
	}

	// Get market timing for trend and confidence
	timing, _ := e.timingAnalyzer.GetTimingRecommendations(match.ID, nil)
	if timing != nil {
		match.MarketTrend = timing.CurrentTrend
		match.MarketConfidence = timing.Confidence
	}

	return nil
}

// Available returns true if marketplace enrichment is available
func (e *MarketplaceEnricher) Available() bool {
	return e != nil && e.marketProvider != nil && e.marketProvider.Available()
}
