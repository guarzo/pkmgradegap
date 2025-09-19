package webcache

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/guarzo/pkmgradegap/internal/analysis"
	"github.com/guarzo/pkmgradegap/internal/cards"
	"github.com/guarzo/pkmgradegap/internal/model"
	"github.com/guarzo/pkmgradegap/internal/population"
	"github.com/guarzo/pkmgradegap/internal/prices"
	"github.com/guarzo/pkmgradegap/internal/volatility"
)

// RefreshService handles generating pre-computed cache data
type RefreshService struct {
	webCache   *WebCache
	cardProv   *cards.PokeTCGIO
	priceProv  *prices.PriceCharting
	popProv    population.Provider
	volTracker *volatility.Tracker
}

// RefreshOptions configures the refresh process
type RefreshOptions struct {
	TopN             int      `json:"topN"`
	MinRawUSD        float64  `json:"minRawUSD"`
	MinDeltaUSD      float64  `json:"minDeltaUSD"`
	MaxAgeYears      int      `json:"maxAgeYears"`
	GradingCost      float64  `json:"gradingCost"`
	ShippingCost     float64  `json:"shippingCost"`
	FeePct           float64  `json:"feePct"`
	JapaneseWeight   float64  `json:"japaneseWeight"`
	WithPopulation   bool     `json:"withPopulation"`
	ExcludeSets      []string `json:"excludeSets"`
	MaxSetsToProcess int      `json:"maxSetsToProcess"`
}

// DefaultRefreshOptions returns sensible defaults
func DefaultRefreshOptions() RefreshOptions {
	return RefreshOptions{
		TopN:             1000,
		MinRawUSD:        5.0,
		MinDeltaUSD:      25.0,
		MaxAgeYears:      10,
		GradingCost:      25.0,
		ShippingCost:     20.0,
		FeePct:           0.13,
		JapaneseWeight:   1.0,
		WithPopulation:   false,
		MaxSetsToProcess: 100, // Process more sets for better coverage
	}
}

// NewRefreshService creates a new refresh service
func NewRefreshService(webCache *WebCache, cardProv *cards.PokeTCGIO, priceProv *prices.PriceCharting, popProv population.Provider, volTracker *volatility.Tracker) *RefreshService {
	return &RefreshService{
		webCache:   webCache,
		cardProv:   cardProv,
		priceProv:  priceProv,
		popProv:    popProv,
		volTracker: volTracker,
	}
}

// RefreshIfNeeded checks if refresh is needed and performs it
func (rs *RefreshService) RefreshIfNeeded(ctx context.Context, source string) error {
	if !rs.webCache.NeedsRefresh() {
		log.Println("Web cache is fresh, skipping refresh")
		return nil
	}

	log.Println("Web cache is stale or missing, performing refresh...")
	return rs.PerformRefresh(ctx, source, DefaultRefreshOptions())
}

// PerformRefresh generates fresh cache data
func (rs *RefreshService) PerformRefresh(ctx context.Context, source string, options RefreshOptions) error {
	startTime := time.Now()
	log.Printf("Starting web cache refresh (source: %s)", source)

	// Get all sets
	allSets, err := rs.cardProv.ListSets()
	if err != nil {
		return fmt.Errorf("failed to list sets: %w", err)
	}

	// Filter sets by age if specified
	filteredSets := rs.filterSetsByAge(allSets, options.MaxAgeYears)

	// Limit number of sets to process for reasonable refresh times
	if len(filteredSets) > options.MaxSetsToProcess {
		// Sort by release date (newest first) and take the most recent sets
		sort.Slice(filteredSets, func(i, j int) bool {
			return filteredSets[i].ReleaseDate > filteredSets[j].ReleaseDate
		})
		filteredSets = filteredSets[:options.MaxSetsToProcess]
		log.Printf("Limited processing to %d most recent sets (out of %d total)", options.MaxSetsToProcess, len(allSets))
	}

	log.Printf("Processing %d sets for cache refresh", len(filteredSets))

	var allScoredRows []analysis.ScoredRow
	var setSummaries []SetSummary
	totalCardsProcessed := 0
	setStartTime := time.Now()

	// Process each set
	for i, set := range filteredSets {
		setProcessStart := time.Now()

		// Calculate ETA based on average time per set
		var eta string
		if i > 0 {
			avgTimePerSet := time.Since(setStartTime) / time.Duration(i)
			remaining := time.Duration(len(filteredSets)-i) * avgTimePerSet
			eta = fmt.Sprintf(" (ETA: %s)", formatDuration(remaining))
		}

		log.Printf("\n=== Processing set %d/%d: %s%s ===",
			i+1, len(filteredSets), set.Name, eta)

		// Check for cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("refresh cancelled")
		default:
		}

		scoredRows, err := rs.processSet(ctx, set, options)
		if err != nil {
			log.Printf("Warning: failed to process set %s: %v", set.Name, err)
			continue
		}

		if len(scoredRows) == 0 {
			log.Printf("No opportunities found in set %s", set.Name)
			continue
		}

		// Add to global results
		allScoredRows = append(allScoredRows, scoredRows...)
		totalCardsProcessed += len(scoredRows)

		// Create set summary
		summary := ConvertScoredRowsToSummary(set.ID, set.Name, scoredRows)
		setSummaries = append(setSummaries, summary)

		setDuration := time.Since(setProcessStart)
		log.Printf("  ✓ Completed %s: %d opportunities found (top score: %.1f) in %s",
			set.Name, len(scoredRows), summary.TopScore, formatDuration(setDuration))
	}

	// Sort all results by score descending
	sort.Slice(allScoredRows, func(i, j int) bool {
		return allScoredRows[i].Score > allScoredRows[j].Score
	})

	// Limit to top N for the main cache
	topRows := allScoredRows
	if len(topRows) > options.TopN {
		topRows = topRows[:options.TopN]
	}

	log.Printf("Generated %d total opportunities, keeping top %d", len(allScoredRows), len(topRows))

	// Convert to web format
	topOpportunities := ConvertRowsToWebFormat(topRows, "Multiple Sets", totalCardsProcessed)
	allOpportunities := ConvertRowsToWebFormat(allScoredRows, "Multiple Sets", totalCardsProcessed)

	// Save cache data
	if err := rs.webCache.SaveTopOpportunities(topOpportunities); err != nil {
		return fmt.Errorf("failed to save top opportunities: %w", err)
	}

	if err := rs.webCache.SaveAllCards(allOpportunities); err != nil {
		return fmt.Errorf("failed to save all cards: %w", err)
	}

	if err := rs.webCache.SaveSetsSummary(setSummaries); err != nil {
		return fmt.Errorf("failed to save sets summary: %w", err)
	}

	// Save metadata
	metadata := &CacheMetadata{
		LastRefresh:     time.Now(),
		TotalCards:      totalCardsProcessed,
		TotalSets:       len(filteredSets),
		RefreshSource:   source,
		Version:         "1.0",
		TopNCards:       len(topRows),
		RefreshDuration: time.Since(startTime).String(),
	}

	if err := rs.webCache.SaveMetadata(metadata); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	log.Printf("Web cache refresh completed in %v (processed %d cards from %d sets)",
		time.Since(startTime), totalCardsProcessed, len(filteredSets))

	return nil
}

// processSet processes a single set and returns scored opportunities
func (rs *RefreshService) processSet(ctx context.Context, set model.Set, options RefreshOptions) ([]analysis.ScoredRow, error) {
	// Fetch cards in set
	log.Printf("  Fetching cards in %s...", set.Name)
	cards, err := rs.cardProv.CardsBySetID(set.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cards: %w", err)
	}

	if len(cards) == 0 {
		return nil, nil
	}

	log.Printf("  Processing %d cards (this may take a while)...", len(cards))

	// Build analysis rows with progress indicator
	var rows []analysis.Row
	for i, card := range cards {
		// Show progress every 10 cards or for the last card
		if i%10 == 0 || i == len(cards)-1 {
			log.Printf("    Progress: %d/%d cards (%.0f%%) - Current: %s #%s",
				i+1, len(cards), float64(i+1)/float64(len(cards))*100,
				card.Name, card.Number)
		}

		row := rs.buildAnalysisRow(ctx, card, set.Name, options)
		rows = append(rows, row)

		// Check for cancellation periodically
		if i%5 == 0 {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("processing cancelled")
			default:
			}
		}
	}

	// Apply sanitization
	sanitizeConfig := analysis.DefaultSanitizeConfig()
	rows = analysis.SanitizeRows(rows, sanitizeConfig)

	// Filter rows
	analysisConfig := analysis.Config{
		MinRawUSD:      options.MinRawUSD,
		MinDeltaUSD:    options.MinDeltaUSD,
		MaxAgeYears:    options.MaxAgeYears,
		GradingCost:    options.GradingCost,
		ShippingCost:   options.ShippingCost,
		FeePct:         options.FeePct,
		JapaneseWeight: options.JapaneseWeight,
		TopN:           0, // Don't limit here, we'll limit globally
	}

	filteredRows := rs.filterRows(rows, analysisConfig)

	// Score and sort
	scoredRows := rs.scoreRows(filteredRows, analysisConfig)

	return scoredRows, nil
}

// buildAnalysisRow builds a single analysis row
func (rs *RefreshService) buildAnalysisRow(ctx context.Context, card model.Card, setName string, options RefreshOptions) analysis.Row {
	// Extract raw price
	rawUSD, rawSrc, rawNote := analysis.ExtractUngradedUSD(card)

	var grades analysis.Grades

	// Look up graded prices (this is the slow part - API call)
	if rs.priceProv != nil && rs.priceProv.Available() {
		if match, err := rs.priceProv.LookupCard(setName, card); err == nil && match != nil {
			grades = analysis.Grades{
				PSA10:   float64(match.PSA10Cents) / 100.0,
				Grade9:  float64(match.Grade9Cents) / 100.0,
				Grade95: float64(match.Grade95Cents) / 100.0,
				BGS10:   float64(match.BGS10Cents) / 100.0,
			}
		} else if err != nil {
			// Log API errors but continue processing
			if strings.Contains(err.Error(), "HTTP") {
				log.Printf("      ⚠ API error for %s #%s: %v", card.Name, card.Number, err)
			}
		}
	}

	// Look up population data if enabled
	var popData *model.PSAPopulation
	if options.WithPopulation && rs.popProv != nil && rs.popProv.Available() {
		if pData, err := rs.popProv.LookupPopulation(ctx, card); err == nil && pData != nil {
			popData = &model.PSAPopulation{
				TotalGraded: pData.TotalGraded,
				PSA10:       pData.PSA10Population,
				PSA9:        pData.PSA9Population,
				PSA8:        pData.PSA8Population,
				LastUpdated: pData.LastUpdated,
			}
		}
	}

	// Calculate volatility
	var volatility float64
	if rs.volTracker != nil {
		// Add current prices to tracker
		if rawUSD > 0 {
			rs.volTracker.AddPrice(setName, card.Name, card.Number, "raw", rawUSD)
		}
		if grades.PSA10 > 0 {
			rs.volTracker.AddPrice(setName, card.Name, card.Number, "psa10", grades.PSA10)
		}

		volatility = rs.volTracker.Calculate30DayVolatility(setName, card.Name, card.Number, "psa10")
	}

	return analysis.Row{
		Card:       card,
		RawUSD:     rawUSD,
		RawSrc:     rawSrc,
		RawNote:    rawNote,
		Grades:     grades,
		Population: popData,
		Volatility: volatility,
	}
}

// filterSetsByAge filters sets by release date
func (rs *RefreshService) filterSetsByAge(sets []model.Set, maxAgeYears int) []model.Set {
	if maxAgeYears <= 0 {
		return sets
	}

	cutoffDate := time.Now().AddDate(-maxAgeYears, 0, 0)
	var filtered []model.Set

	for _, set := range sets {
		// Parse release date (format: YYYY-MM-DD)
		if releaseDate, err := time.Parse("2006-01-02", set.ReleaseDate); err == nil {
			if releaseDate.After(cutoffDate) {
				filtered = append(filtered, set)
			}
		} else {
			// If we can't parse the date, include it to be safe
			filtered = append(filtered, set)
		}
	}

	return filtered
}

// filterRows applies basic filtering logic
func (rs *RefreshService) filterRows(rows []analysis.Row, cfg analysis.Config) []analysis.Row {
	var filtered []analysis.Row

	for _, row := range rows {
		// Skip if raw price is below minimum
		if row.RawUSD < cfg.MinRawUSD {
			continue
		}

		// Skip if no graded prices (need either PSA10 or BGS10)
		targetGrade := row.Grades.PSA10
		if row.Grades.BGS10 > targetGrade {
			targetGrade = row.Grades.BGS10
		}
		if targetGrade == 0 {
			continue
		}

		// Calculate delta
		delta := targetGrade - row.RawUSD
		if delta < cfg.MinDeltaUSD {
			continue
		}

		filtered = append(filtered, row)
	}

	return filtered
}

// scoreRows applies scoring logic and sorts by score
func (rs *RefreshService) scoreRows(rows []analysis.Row, cfg analysis.Config) []analysis.ScoredRow {
	var scored []analysis.ScoredRow

	for _, row := range rows {
		sr := analysis.ScoredRow{
			Row: row,
		}

		// Basic profit calculation - use higher of PSA10 or BGS10
		targetGrade := row.Grades.PSA10
		if row.Grades.BGS10 > targetGrade {
			targetGrade = row.Grades.BGS10
		}

		totalCost := row.RawUSD + cfg.GradingCost + cfg.ShippingCost
		sellingFees := targetGrade * cfg.FeePct
		netProfit := targetGrade - totalCost - sellingFees

		// Base score is net profit
		sr.Score = netProfit

		// Apply Japanese weight if applicable
		if rs.isJapanese(row.Card) && cfg.JapaneseWeight > 1.0 {
			sr.Score *= cfg.JapaneseWeight
		}

		// Population scarcity bonus
		if row.Population != nil && row.Population.PSA10 <= 10 {
			sr.Score += 15.0 // Ultra rare bonus
		} else if row.Population != nil && row.Population.PSA10 <= 50 {
			sr.Score += 10.0 // Rare bonus
		}

		// Penalty for high volatility
		if row.Volatility > 0.3 {
			sr.Score *= (1.0 - row.Volatility*0.5)
		}

		scored = append(scored, sr)
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	return scored
}

// isJapanese determines if a card is Japanese
func (rs *RefreshService) isJapanese(card model.Card) bool {
	// Simple heuristic: Japanese cards often have specific markers
	return card.Rarity == "Common" && len(card.Number) > 0 && card.Number[0] >= '0' && card.Number[0] <= '9'
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}
