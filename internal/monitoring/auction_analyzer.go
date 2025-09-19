package monitoring

import (
	"fmt"
	"strings"
	"time"

	"github.com/guarzo/pkmgradegap/internal/ebay"
)

// AuctionOpportunity represents a profitable auction opportunity
type AuctionOpportunity struct {
	Auction        ebay.Auction
	EstimatedValue float64
	ProfitScore    float64
	Risk           string    // "LOW", "MEDIUM", "HIGH"
	LastUpdated    time.Time
}

// AuctionAnalyzer provides on-demand auction analysis for specific cards
type AuctionAnalyzer struct {
	ebayClient *ebay.Client
	config     AuctionAnalyzerConfig
}

// AuctionAnalyzerConfig contains configuration for auction analysis
type AuctionAnalyzerConfig struct {
	EndingWithinMinutes   int     // Look for auctions ending within this timeframe (default: 60)
	MinProfitThresholdPct float64 // Minimum profit percentage to consider (default: 30)
	MaxBidThresholdUSD    float64 // Maximum current bid to consider (default: 500)
	GradingCostUSD        float64 // Cost to grade a card (default: 20)
	ShippingCostUSD       float64 // Estimated shipping cost (default: 5)
	EbayFeePct            float64 // eBay fee percentage (default: 0.1295)
}

// NewAuctionAnalyzer creates a new auction analyzer
func NewAuctionAnalyzer(ebayClient *ebay.Client, config AuctionAnalyzerConfig) *AuctionAnalyzer {
	// Set defaults
	if config.EndingWithinMinutes == 0 {
		config.EndingWithinMinutes = 60
	}
	if config.MinProfitThresholdPct == 0 {
		config.MinProfitThresholdPct = 30
	}
	if config.MaxBidThresholdUSD == 0 {
		config.MaxBidThresholdUSD = 500
	}
	if config.GradingCostUSD == 0 {
		config.GradingCostUSD = 20
	}
	if config.ShippingCostUSD == 0 {
		config.ShippingCostUSD = 5
	}
	if config.EbayFeePct == 0 {
		config.EbayFeePct = 0.1295 // eBay + PayPal fees
	}

	return &AuctionAnalyzer{
		ebayClient: ebayClient,
		config:     config,
	}
}

// GetAuctionOpportunities finds profitable auction opportunities for a specific card
func (aa *AuctionAnalyzer) GetAuctionOpportunities(setName, cardName, number string) ([]*AuctionOpportunity, error) {
	if !aa.ebayClient.Available() {
		return nil, fmt.Errorf("eBay client is not available")
	}

	// Search for ending auctions matching the card
	auctions, err := aa.findEndingAuctionsForCard(setName, cardName, number)
	if err != nil {
		return nil, fmt.Errorf("failed to find auctions: %w", err)
	}

	// Evaluate each auction for profit potential
	var opportunities []*AuctionOpportunity
	for _, auction := range auctions {
		// Skip if current bid is too high
		if auction.CurrentBid > aa.config.MaxBidThresholdUSD {
			continue
		}

		// Skip if ending too soon (less than 10 minutes)
		timeToEnd := time.Until(auction.EndTime)
		if timeToEnd < 10*time.Minute {
			continue
		}

		opportunity := aa.scoreAuction(auction)
		if opportunity != nil && opportunity.ProfitScore >= aa.config.MinProfitThresholdPct {
			opportunities = append(opportunities, opportunity)
		}
	}

	return opportunities, nil
}

// findEndingAuctionsForCard searches for auctions ending soon for a specific card
func (aa *AuctionAnalyzer) findEndingAuctionsForCard(setName, cardName, number string) ([]ebay.Auction, error) {
	// Get general ending Pokemon auctions
	allAuctions, err := aa.ebayClient.GetEndingAuctions(aa.config.EndingWithinMinutes, "pokemon")
	if err != nil {
		return nil, err
	}

	// Filter for the specific card
	var matchingAuctions []ebay.Auction
	for _, auction := range allAuctions {
		if aa.isCardMatch(auction.Title, setName, cardName, number) {
			matchingAuctions = append(matchingAuctions, auction)
		}
	}

	return matchingAuctions, nil
}

// isCardMatch checks if an auction title matches the specified card
func (aa *AuctionAnalyzer) isCardMatch(title, setName, cardName, number string) bool {
	titleLower := strings.ToLower(title)

	// Must contain the card name
	if !strings.Contains(titleLower, strings.ToLower(cardName)) {
		return false
	}

	// Check for set name (optional but helpful)
	if setName != "" {
		setWords := strings.Fields(strings.ToLower(setName))
		setMatches := 0
		for _, word := range setWords {
			if len(word) > 2 && strings.Contains(titleLower, word) {
				setMatches++
			}
		}
		// If we have set words, require at least one match
		if len(setWords) > 0 && setMatches == 0 {
			return false
		}
	}

	// Check for card number (optional)
	if number != "" {
		numberPatterns := []string{
			"#" + number,
			"# " + number,
			number + "/",
			"/" + number,
		}
		hasNumber := false
		for _, pattern := range numberPatterns {
			if strings.Contains(titleLower, strings.ToLower(pattern)) {
				hasNumber = true
				break
			}
		}
		// For numbered cards, prefer matches with numbers but don't require
		if !hasNumber {
			// Slightly lower confidence for unnumbered matches
			return strings.Contains(titleLower, "pokemon")
		}
	}

	return true
}

// scoreAuction calculates the profit potential of an auction
func (aa *AuctionAnalyzer) scoreAuction(auction ebay.Auction) *AuctionOpportunity {
	estimatedValue := aa.estimateCardValue(auction)
	if estimatedValue <= 0 {
		return nil
	}

	// Calculate total costs
	totalCosts := auction.CurrentBid + auction.ShippingCost + aa.config.GradingCostUSD + aa.config.ShippingCostUSD

	// Calculate net revenue after eBay fees (when selling graded card)
	netRevenue := estimatedValue * (1 - aa.config.EbayFeePct)

	// Calculate profit and score
	profit := netRevenue - totalCosts
	profitScore := (profit / totalCosts) * 100

	risk := aa.assessRisk(auction, profitScore)

	return &AuctionOpportunity{
		Auction:        auction,
		EstimatedValue: estimatedValue,
		ProfitScore:    profitScore,
		Risk:           risk,
		LastUpdated:    time.Now(),
	}
}

// estimateCardValue provides a rough estimate of graded card value
func (aa *AuctionAnalyzer) estimateCardValue(auction ebay.Auction) float64 {
	baseValue := auction.CurrentBid
	multiplier := 1.0

	titleLower := strings.ToLower(auction.Title)

	// Skip very low value cards that aren't worth grading
	if containsAnySubstring(titleLower, []string{"energy", "trainer", "basic"}) && baseValue < 5.0 {
		return 0 // Not worth grading
	}

	// Higher multipliers for valuable card types
	if containsAnySubstring(titleLower, []string{"charizard", "pikachu", "mew", "lugia"}) {
		multiplier *= 3.0
	} else if containsAnySubstring(titleLower, []string{"holo", "rare", "ex", "gx", "v", "vmax"}) {
		multiplier *= 2.0
	}

	// First edition bonus
	if containsAnySubstring(titleLower, []string{"1st edition", "shadowless", "base set"}) {
		multiplier *= 1.5
	}

	// Japanese cards premium
	if containsAnySubstring(titleLower, []string{"japanese", "japan", "jpn"}) {
		multiplier *= 1.2
	}

	return baseValue * multiplier
}

// assessRisk evaluates the risk level of an auction opportunity
func (aa *AuctionAnalyzer) assessRisk(auction ebay.Auction, profitScore float64) string {
	riskFactors := 0

	// High bid count = more competition
	if auction.BidCount > 10 {
		riskFactors++
	} else if auction.BidCount > 5 {
		riskFactors++ // Medium competition
	}

	// Low seller rating
	if auction.SellerRating < 95 {
		riskFactors++
	}

	// Very high profit score might be too good to be true
	if profitScore > 200 {
		riskFactors++
	}

	// Time pressure
	timeToEnd := time.Until(auction.EndTime)
	if timeToEnd < 30*time.Minute {
		riskFactors++
	}

	switch riskFactors {
	case 0:
		return "LOW"
	case 1, 2:
		return "MEDIUM"
	default:
		return "HIGH"
	}
}

// containsAnySubstring checks if any of the substrings exist in the target string
func containsAnySubstring(target string, substrings []string) bool {
	for _, substr := range substrings {
		if strings.Contains(target, substr) {
			return true
		}
	}
	return false
}