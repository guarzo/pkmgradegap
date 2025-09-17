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
	WithVolatility   bool    // Include volatility data
	AllowThinPremium bool    // Allow PSA9/PSA10 > 0.75
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

		// Population multipliers removed - no public PSA API available
		psa10Rate := float64(0)
		psa9Rate := float64(0)

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
			scoredRow.ScoreBreakdown = fmt.Sprintf("Profit:%.2f Premium:%.2f JPN:%v",
				netProfit, score-netProfit, isJapanese)
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
	if config.WithVolatility {
		header = append(header, "Volatility30D")
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
			ebayLinks := ""
			if ebayClient != nil && ebayClient.Available() && set != nil {
				listings, err := ebayClient.SearchRawListings(set.Name, sr.Card.Name, sr.Card.Number, config.EbayMax)
				if err != nil {
					// Log error but continue processing
					errMsg := err.Error()
					if len(errMsg) > 50 {
						errMsg = errMsg[:50] + "..."
					}
					ebayLinks = fmt.Sprintf("Error: %s", errMsg)
				} else if len(listings) > 0 {
					ebayLinks = formatEbayListings(listings, config.EbayMax)
				} else {
					ebayLinks = "No listings found"
				}
			} else {
				ebayLinks = "eBay not available"
			}
			row = append(row, ebayLinks)
		}

		if config.WithVolatility {
			row = append(row, fmt.Sprintf("%.1f%%", sr.Volatility*100))
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
	crossgradeCost := 30.0  // PSA crossgrade service
	shippingCost := 20.0    // Round trip shipping
	sellingFeePct := 0.13   // eBay/TCG fees

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