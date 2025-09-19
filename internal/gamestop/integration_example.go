package gamestop

import (
	"fmt"
	"log"
	"time"

	// "github.com/guarzo/pkmgradegap/internal/fusion" // TODO: Update when fusion package is refactored
	"github.com/guarzo/pkmgradegap/internal/model"
)

// Example demonstrates how to integrate GameStop provider into the main analysis pipeline
func ExampleIntegration() {
	// Initialize GameStop provider
	config := DefaultConfig()
	// Reduce rate limiting for real usage
	config.RateLimitPerMin = 6 // 6 requests per minute to be respectful
	config.RequestDelay = 10   // 10 seconds between requests

	// Create GameStop web scraper client
	provider := NewGameStopClient(config)

	// Initialize fusion engine
	// fusionEngine := fusion.NewFusionEngine() // TODO: Update when fusion package is refactored
	var fusionEngine interface{} // placeholder

	// Example card to analyze
	card := model.Card{
		Name:    "Charizard",
		Number:  "123",
		SetName: "Surging Sparks",
	}

	// Get GameStop listings
	listingData, err := provider.GetListings(card.SetName, card.Name, card.Number)
	if err != nil {
		log.Printf("Failed to get GameStop listings: %v", err)
		return
	}

	fmt.Printf("Found %d GameStop listings for %s #%s\n",
		listingData.ListingCount, card.Name, card.Number)

	// Example: Combine with other price sources (PriceCharting, sales data, etc.)
	// This would normally come from your existing price providers
	// otherPrices := make(map[string][]fusion.PriceData) // TODO: Update when fusion package is refactored
	otherPrices := make(map[string][]interface{})

	// Merge and fuse prices
	fusedPrices := MergeWithFusionEngine(fusionEngine, listingData, otherPrices)

	// Display results
	// TODO: Update when fusion package is refactored
	for grade, fusedPrice := range fusedPrices {
		// fmt.Printf("%s: $%.2f (confidence: %.2f)\n",
		// 	grade, fusedPrice.Value, fusedPrice.Confidence)
		//
		// if len(fusedPrice.Warnings) > 0 {
		// 	fmt.Printf("  Warnings: %v\n", fusedPrice.Warnings)
		// }
		fmt.Printf("%s: %v\n", grade, fusedPrice)
	}

	// Get lowest prices by grade for quick comparison
	lowestPrices := GetLowestPriceByGrade(listingData)
	fmt.Printf("\nLowest GameStop prices:\n")
	for grade, price := range lowestPrices {
		fmt.Printf("  %s: $%.2f\n", grade, price)
	}
}

// IntegrateWithExistingAnalysis shows how to add GameStop data to your existing analysis
// TODO: Update when fusion package is refactored
func IntegrateWithExistingAnalysis(
	card model.Card,
	existingPrices map[string][]interface{}, // map[string][]fusion.PriceData,
	fusionEngine interface{}, // *fusion.FusionEngine,
) map[string]interface{} { // map[string]fusion.FusedPrice {

	// Initialize GameStop web scraper
	config := DefaultConfig()
	provider := NewGameStopClient(config)

	// Get GameStop data
	listingData, err := provider.GetListings(card.SetName, card.Name, card.Number)
	if err != nil {
		log.Printf("GameStop lookup failed: %v", err)
		// Continue analysis without GameStop data
		return fusePricesWithoutGameStop(existingPrices, fusionEngine)
	}

	// If no GameStop listings found, continue without them
	if listingData.ListingCount == 0 {
		log.Printf("No GameStop listings found for %s #%s", card.Name, card.Number)
		return fusePricesWithoutGameStop(existingPrices, fusionEngine)
	}

	// Merge GameStop data with existing prices
	return MergeWithFusionEngine(fusionEngine, listingData, existingPrices)
}

// TODO: Update when fusion package is refactored
func fusePricesWithoutGameStop(
	prices map[string][]interface{}, // map[string][]fusion.PriceData,
	engine interface{}, // *fusion.FusionEngine,
) map[string]interface{} { // map[string]fusion.FusedPrice {
	// result := make(map[string]fusion.FusedPrice)
	result := make(map[string]interface{})

	for grade, priceData := range prices {
		if len(priceData) > 0 {
			// result[grade] = engine.FusePrice(priceData)
			result[grade] = priceData // TODO: Implement fusion logic
		}
	}

	return result
}

// CLI flag integration example
type CLIFlags struct {
	WithGameStop     bool
	GameStopMaxItems int
	GameStopTimeout  int
}

func CreateGameStopProvider(flags CLIFlags) Provider {
	config := DefaultConfig()
	if flags.GameStopMaxItems > 0 {
		config.MaxListingsPerCard = flags.GameStopMaxItems
	}
	if flags.GameStopTimeout > 0 {
		config.RequestTimeout = time.Duration(flags.GameStopTimeout) * time.Second
	}

	return NewGameStopClient(config)
}

// BulkAnalysis demonstrates processing multiple cards with GameStop data
func BulkAnalysis(cards []model.Card, provider Provider) map[string]*ListingData {
	fmt.Printf("Analyzing %d cards with GameStop provider...\n", len(cards))

	// Use bulk method for efficiency
	results, err := provider.GetBulkListings(cards)
	if err != nil {
		log.Printf("Bulk GameStop analysis failed: %v", err)
		return nil
	}

	// Summary stats
	totalListings := 0
	cardsWithListings := 0

	for cardKey, data := range results {
		if data.ListingCount > 0 {
			cardsWithListings++
			totalListings += data.ListingCount
			fmt.Printf("%s: %d listings (lowest: $%.2f)\n",
				cardKey, data.ListingCount, data.LowestPrice)
		}
	}

	fmt.Printf("\nSummary: %d/%d cards have GameStop listings (%d total listings)\n",
		cardsWithListings, len(cards), totalListings)

	return results
}
