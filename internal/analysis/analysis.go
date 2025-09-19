package analysis

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
)

type Grades struct {
	PSA10   float64
	Grade9  float64
	Grade95 float64 // maps to 9.5 (PC "box-only-price")
	BGS10   float64
}

type Row struct {
	Card       model.Card
	RawUSD     float64
	RawSrc     string
	RawNote    string
	Grades     Grades
	Population *model.PSAPopulation // Optional population data
	Volatility float64              // 30-day price variance (0-1 scale)

	// Sprint 3: Marketplace fields
	ActiveListings      int     // Current marketplace listings
	LowestListing       float64 // Lowest available price in USD
	ListingVelocity     float64 // Sales per day
	CompetitionLevel    string  // LOW, MEDIUM, HIGH
	OptimalListingPrice float64 // Recommended listing price in USD
	MarketTrend         string  // BULLISH, BEARISH, NEUTRAL
	SupplyDemandRatio   float64 // listings/sales ratio

	// Sprint 4: UPC & Advanced Search fields
	UPC             string  // Universal Product Code
	MatchConfidence float64 // Match confidence (0.0 to 1.0)
	MatchMethod     string  // How the match was found ("upc", "id", "search", "fuzzy")
	Variant         string  // Card variant (1st Edition, Shadowless, etc.)
	Language        string  // Card language

	// Sprint 1: Auction fields
	AuctionOpportunities int     // Number of ending auctions found
	BestAuctionBid      float64 // Current bid of most profitable auction
	BestAuctionProfit   float64 // Estimated profit of best auction
	BestAuctionURL      string  // URL to best auction opportunity
	BestAuctionRisk     string  // Risk level of best auction (LOW/MEDIUM/HIGH)
}

type Config struct {
	MaxAgeYears      int
	MinDeltaUSD      float64
	MinRawUSD        float64
	TopN             int
	GradingCost      float64
	ShippingCost     float64
	FeePct           float64
	JapaneseWeight   float64
	ShowWhy          bool
	WithEbay         bool
	EbayMax          int
	WithAuctions     bool // Sprint 1: Include auction opportunities
	WithVolatility   bool // Include volatility data
	AllowThinPremium bool // Allow PSA9/PSA10 > 0.75
	WithMarketplace  bool // Sprint 3: Include marketplace data
}

type ScoredRow struct {
	Row
	Score          float64
	BreakEvenUSD   float64
	NetProfitUSD   float64
	TotalCostUSD   float64
	IsJapanese     bool
	SetAgeYears    int
	ScoreBreakdown string
	PSA10Rate      float64 // Calculated from population
	PSA9Rate       float64 // Calculated from population
}

// Extract USD prices from TCGplayer only (no EUR fallback)
func ExtractUngradedUSD(c model.Card) (value float64, source string, note string) {
	// Try TCGplayer markets in order of likeliest printing buckets.
	if c.TCGPlayer != nil && c.TCGPlayer.Prices != nil {
		typeNameOrder := []string{"normal", "holofoil", "reverseHolofoil", "1stEditionHolofoil", "1stEditionNormal"}
		for _, t := range typeNameOrder {
			if p, ok := c.TCGPlayer.Prices[t]; ok && p.Market != nil && *p.Market > 0 {
				return round2(*p.Market), "tcgplayer.market", "USD"
			}
		}
	}
	// Return 0 if no USD price available (skip EUR prices)
	return 0, "", ""
}

func ReportRawVsPSA10(rows []Row) [][]string {
	out := [][]string{
		{"Card", "Number", "RawUSD", "RawSource", "PSA10_USD", "Delta_USD", "Notes"},
	}
	for _, r := range rows {
		if r.RawUSD <= 0 || r.Grades.PSA10 <= 0 {
			continue
		}
		delta := r.Grades.PSA10 - r.RawUSD
		notes := r.RawNote
		out = append(out, []string{
			fmt.Sprintf("%s", r.Card.Name),
			r.Card.Number,
			money(r.RawUSD),
			r.RawSrc,
			money(r.Grades.PSA10),
			money(delta),
			notes,
		})
	}
	return out
}

func ReportMultiVsPSA10(rows []Row) [][]string {
	out := [][]string{
		{"Card", "Number", "PSA9_USD", "CGC/BGS_9.5_USD", "BGS10_USD", "PSA10_USD", "PSA9/10_%", "9.5/10_%", "BGS10/PSA10_%"},
	}
	for _, r := range rows {
		if r.Grades.PSA10 <= 0 {
			continue
		}
		psa9pct := pct(r.Grades.Grade9, r.Grades.PSA10)
		g95pct := pct(r.Grades.Grade95, r.Grades.PSA10)
		bgs10pct := pct(r.Grades.BGS10, r.Grades.PSA10)
		out = append(out, []string{
			r.Card.Name,
			r.Card.Number,
			money(r.Grades.Grade9),
			money(r.Grades.Grade95),
			money(r.Grades.BGS10),
			money(r.Grades.PSA10),
			psa9pct,
			g95pct,
			bgs10pct,
		})
	}
	return out
}

func pct(a, b float64) string {
	if a <= 0 || b <= 0 {
		return ""
	}
	return fmt.Sprintf("%.1f%%", (a/b)*100.0)
}

func money(v float64) string {
	if v <= 0 {
		return ""
	}
	return "$" + strconv.FormatFloat(round2(v), 'f', 2, 64)
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}

// formatEbayListings formats eBay listings for CSV output
func formatEbayListings(listings []EbayListing, maxLinks int) string {
	if len(listings) == 0 {
		return ""
	}

	var links []string
	count := 0

	for _, listing := range listings {
		if count >= maxLinks {
			break
		}

		// Format: Price | Title | URL
		link := fmt.Sprintf("$%.2f|%s|%s",
			listing.Price,
			truncateTitle(listing.Title, 30),
			listing.URL)
		links = append(links, link)
		count++
	}

	return strings.Join(links, " ; ")
}

func truncateTitle(title string, maxLen int) string {
	if len(title) <= maxLen {
		return title
	}
	return title[:maxLen-3] + "..."
}

// EbayProvider interface for eBay listing lookup
type EbayProvider interface {
	Available() bool
	SearchRawListings(setName, cardName, number string, max int) ([]EbayListing, error)
}

// AuctionProvider interface for auction analysis
type AuctionProvider interface {
	Available() bool
	GetAuctionOpportunities(setName, cardName, number string) ([]AuctionOpportunity, error)
}

// AuctionOpportunity represents an auction opportunity for the analysis package
type AuctionOpportunity struct {
	CurrentBid     float64
	EstimatedValue float64
	ProfitScore    float64
	Risk           string
	URL            string
	TimeRemaining  string
}

// EbayListing represents an eBay listing with necessary fields for analysis
type EbayListing struct {
	Price float64
	Title string
	URL   string
}

// ReportRank generates the ranked opportunities report
func ReportRank(rows []Row, set *model.Set, config Config) [][]string {
	return ReportRankWithEbay(rows, set, config, nil)
}

// ReportRankWithEbay generates the ranked opportunities report with eBay integration
func ReportRankWithEbay(rows []Row, set *model.Set, config Config, ebayClient EbayProvider) [][]string {
	// Calculate set age if filtering is enabled
	setAge := 0
	if config.MaxAgeYears > 0 && set != nil && set.ReleaseDate != "" {
		setAge = calculateSetAge(set.ReleaseDate)
		if setAge > config.MaxAgeYears {
			// Return empty result with header for old sets
			return [][]string{
				{"Card", "No", "RawUSD", "PSA10USD", "DeltaUSD", "CostUSD", "BreakEvenUSD", "Score", "Notes", "EBayLinks"},
				{fmt.Sprintf("Set %s is %d years old, exceeds --max-age-years %d", set.Name, setAge, config.MaxAgeYears), "", "", "", "", "", "", "", "", ""},
			}
		}
	}

	// Score and filter rows
	scoredRows := []ScoredRow{}
	for _, r := range rows {
		// Skip if no prices
		if r.RawUSD <= 0 || r.Grades.PSA10 <= 0 {
			continue
		}

		// Apply minimum filters
		if r.RawUSD < config.MinRawUSD {
			continue
		}

		// Skip negative ROI cards
		if r.Grades.PSA10 <= r.RawUSD {
			continue
		}

		delta := r.Grades.PSA10 - r.RawUSD
		if delta < config.MinDeltaUSD {
			continue
		}

		// Filter thin premium unless allowed
		if !config.AllowThinPremium && r.Grades.Grade9 > 0 && r.Grades.PSA10 > 0 {
			if r.Grades.Grade9/r.Grades.PSA10 > 0.75 {
				continue
			}
		}

		// Calculate costs and score
		totalCost := r.RawUSD + config.GradingCost + config.ShippingCost
		sellingFees := r.Grades.PSA10 * config.FeePct
		netProfit := r.Grades.PSA10 - totalCost - sellingFees
		breakEven := totalCost / (1 - config.FeePct)

		// Base score is net profit
		score := netProfit

		// Add premium lift bonus (rewards steep PSA10 premium)
		if r.Grades.PSA10 > 0 && r.Grades.Grade9 > 0 {
			premiumLift := (1 - r.Grades.Grade9/r.Grades.PSA10) * 10
			score += premiumLift
		}

		// Check if Japanese (simple heuristic: card name contains Japanese characters or set is Japanese)
		isJapanese := containsJapanese(r.Card.Name)
		if isJapanese {
			score *= config.JapaneseWeight
		}

		// Population-based scoring factors
		psa10Rate := float64(0)
		psa9Rate := float64(0)

		// Apply population scarcity bonus if population data is available
		if r.Population != nil {
			// Calculate PSA 10 rate (success rate)
			if r.Population.TotalGraded > 0 {
				psa10Rate = float64(r.Population.PSA10) / float64(r.Population.TotalGraded)
				psa9Rate = float64(r.Population.PSA9) / float64(r.Population.TotalGraded)
			}

			// Apply scarcity bonus based on PSA 10 population
			scarcityBonus := calculateScarcityBonus(r.Population.PSA10)
			score += scarcityBonus

			// Apply population quality bonus (high PSA 10 rate indicates good card quality)
			if r.Population.TotalGraded >= 100 { // Need sufficient sample size
				qualityBonus := psa10Rate * 5 // Bonus for cards that grade well
				score += qualityBonus
			}
		}

		// Apply volatility penalty if available
		if config.WithVolatility && r.Volatility > 0 {
			if r.Volatility > 0.2 {
				score *= 0.9
			}
		}

		scoredRow := ScoredRow{
			Row:          r,
			Score:        score,
			BreakEvenUSD: breakEven,
			NetProfitUSD: netProfit,
			TotalCostUSD: totalCost,
			IsJapanese:   isJapanese,
			SetAgeYears:  setAge,
			PSA10Rate:    psa10Rate,
			PSA9Rate:     psa9Rate,
		}

		if config.ShowWhy {
			breakdown := fmt.Sprintf("Profit:%.2f", netProfit)

			// Add premium lift breakdown
			if r.Grades.PSA10 > 0 && r.Grades.Grade9 > 0 {
				premiumLift := (1 - r.Grades.Grade9/r.Grades.PSA10) * 10
				breakdown += fmt.Sprintf(" Premium:%.2f", premiumLift)
			}

			// Add population factors if available
			if r.Population != nil {
				scarcityBonus := calculateScarcityBonus(r.Population.PSA10)
				breakdown += fmt.Sprintf(" Scarcity:%.2f", scarcityBonus)

				if r.Population.TotalGraded >= 100 {
					qualityBonus := psa10Rate * 5
					breakdown += fmt.Sprintf(" Quality:%.2f", qualityBonus)
				}
			}

			// Add other factors
			breakdown += fmt.Sprintf(" JPN:%v", isJapanese)

			if config.WithVolatility && r.Volatility > 0.2 {
				breakdown += " Vol:0.9x"
			}

			scoredRow.ScoreBreakdown = breakdown
		}

		scoredRows = append(scoredRows, scoredRow)
	}

	// Sort by score descending
	sort.Slice(scoredRows, func(i, j int) bool {
		return scoredRows[i].Score > scoredRows[j].Score
	})

	// Limit to top N
	if config.TopN > 0 && len(scoredRows) > config.TopN {
		scoredRows = scoredRows[:config.TopN]
	}

	// Build output
	header := []string{"Card", "No", "RawUSD", "PSA10USD", "DeltaUSD", "CostUSD", "BreakEvenUSD", "Score", "Notes"}
	if config.WithEbay {
		header = append(header, "EBayLinks")
	}
	if config.WithAuctions {
		header = append(header, "AuctionCount", "BestBid", "AuctionProfit%", "AuctionRisk", "AuctionURL")
	}
	if config.WithVolatility {
		header = append(header, "Volatility30D")
	}
	if config.WithMarketplace {
		header = append(header, "ActiveListings", "LowestListing", "OptimalPrice", "CompetitionLevel", "MarketTrend", "ListingVelocity")
	}
	if config.ShowWhy {
		header = append(header, "Why")
	}

	out := [][]string{header}

	for _, sr := range scoredRows {
		notes := sr.RawNote
		if sr.IsJapanese {
			notes += " [JPN]"
		}

		row := []string{
			sr.Card.Name,
			sr.Card.Number,
			money(sr.RawUSD),
			money(sr.Grades.PSA10),
			money(sr.Grades.PSA10 - sr.RawUSD),
			money(sr.TotalCostUSD),
			money(sr.BreakEvenUSD),
			fmt.Sprintf("%.1f", sr.Score),
			notes,
		}

		if config.WithEbay {
			// Add empty eBay links column for now
			// This would be populated if ebayClient was provided
			row = append(row, "")
		}

		if config.WithAuctions {
			// Add auction data columns
			auctionCount := fmt.Sprintf("%d", sr.AuctionOpportunities)
			bestBid := ""
			auctionProfit := ""
			auctionRisk := ""
			auctionURL := ""

			if sr.BestAuctionBid > 0 {
				bestBid = money(sr.BestAuctionBid)
				auctionProfit = fmt.Sprintf("%.1f%%", sr.BestAuctionProfit)
				auctionRisk = sr.BestAuctionRisk
				auctionURL = sr.BestAuctionURL
			}

			row = append(row, auctionCount, bestBid, auctionProfit, auctionRisk, auctionURL)
		}

		if config.WithVolatility {
			row = append(row, fmt.Sprintf("%.1f%%", sr.Volatility*100))
		}

		if config.WithMarketplace {
			// Add marketplace columns
			row = append(row,
				fmt.Sprintf("%d", sr.ActiveListings),
				money(sr.LowestListing),
				money(sr.OptimalListingPrice),
				sr.CompetitionLevel,
				sr.MarketTrend,
				fmt.Sprintf("%.1f", sr.ListingVelocity),
			)
		}

		if config.ShowWhy {
			row = append(row, sr.ScoreBreakdown)
		}

		out = append(out, row)
	}

	return out
}

func calculateSetAge(releaseDate string) int {
	// Parse release date (format: "YYYY-MM-DD" or "YYYY/MM/DD")
	formats := []string{"2006-01-02", "2006/01/02", "01/02/2006"}
	var parsed time.Time
	var err error

	for _, format := range formats {
		parsed, err = time.Parse(format, releaseDate)
		if err == nil {
			break
		}
	}

	if err != nil {
		// If we can't parse, assume it's old
		return 999
	}

	years := int(time.Since(parsed).Hours() / 24 / 365)
	return years
}

func containsJapanese(s string) bool {
	// Simple check for Japanese characters (Hiragana, Katakana, Kanji ranges)
	for _, r := range s {
		if (r >= 0x3040 && r <= 0x309F) || // Hiragana
			(r >= 0x30A0 && r <= 0x30FF) || // Katakana
			(r >= 0x4E00 && r <= 0x9FAF) { // Kanji
			return true
		}
	}
	return false
}

// ReportCrossgrade identifies CGC/BGS 9.5 cards worth regrading to PSA
func ReportCrossgrade(rows []Row) [][]string {
	out := [][]string{
		{"Card", "No", "CGC95USD", "PSA10USD", "CrossgradeROI%", "Notes"},
	}

	// PSA crossgrade submission costs
	crossgradeCost := 30.0 // PSA crossgrade service
	shippingCost := 20.0   // Round trip shipping
	sellingFeePct := 0.13  // eBay/TCG fees

	for _, r := range rows {
		if r.Grades.Grade95 <= 0 || r.Grades.PSA10 <= 0 {
			continue
		}

		// Calculate crossgrade economics
		totalInvestment := r.Grades.Grade95 + crossgradeCost + shippingCost
		netRevenue := r.Grades.PSA10 * (1 - sellingFeePct)
		roi := ((netRevenue - totalInvestment) / totalInvestment) * 100

		// Only show positive ROI opportunities
		if roi <= 10 { // Minimum 10% ROI to be worthwhile
			continue
		}

		notes := fmt.Sprintf("Investment: $%.2f, Net: $%.2f", totalInvestment, netRevenue)

		out = append(out, []string{
			r.Card.Name,
			r.Card.Number,
			money(r.Grades.Grade95),
			money(r.Grades.PSA10),
			fmt.Sprintf("%.1f%%", roi),
			notes,
		})
	}

	return out
}

// calculateScarcityBonus returns a bonus score based on PSA 10 population scarcity
func calculateScarcityBonus(psa10Population int) float64 {
	// Scarcity bonus tiers based on PSA 10 population
	switch {
	case psa10Population == 0:
		return 0 // No data, no bonus
	case psa10Population <= 10:
		return 15.0 // Ultra rare - huge bonus
	case psa10Population <= 50:
		return 10.0 // Very rare - large bonus
	case psa10Population <= 200:
		return 5.0 // Rare - medium bonus
	case psa10Population <= 500:
		return 2.0 // Uncommon - small bonus
	case psa10Population <= 1000:
		return 1.0 // Somewhat common - minimal bonus
	default:
		return 0.0 // Common - no bonus
	}
}
