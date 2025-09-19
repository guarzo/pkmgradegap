package webcache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/guarzo/pkmgradegap/internal/analysis"
)

const (
	DefaultCacheDir      = "data/cache/web_cache"
	TopOpportunitiesFile = "top_opportunities.json"
	AllCardsFile         = "all_cards.json"
	MetadataFile         = "metadata.json"
	SetsSummaryFile      = "sets_summary.json"
)

// WebCache manages pre-computed web data
type WebCache struct {
	CacheDir string
}

// CachedResult represents pre-computed analysis results
type CachedResult struct {
	Columns   []string         `json:"columns"`
	Rows      []map[string]any `json:"rows"`
	Params    map[string]any   `json:"params"`
	Timestamp time.Time        `json:"timestamp"`
	SetName   string           `json:"setName"`
	CardCount int              `json:"cardCount"`
}

// CacheMetadata tracks cache freshness and statistics
type CacheMetadata struct {
	LastRefresh     time.Time `json:"lastRefresh"`
	TotalCards      int       `json:"totalCards"`
	TotalSets       int       `json:"totalSets"`
	RefreshSource   string    `json:"refreshSource"` // "startup" or "manual"
	Version         string    `json:"version"`
	TopNCards       int       `json:"topNCards"`
	RefreshDuration string    `json:"refreshDuration"`
}

// SetSummary provides quick stats for each set
type SetSummary struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	CardCount        int       `json:"cardCount"`
	TopScore         float64   `json:"topScore"`
	AvgRawPrice      float64   `json:"avgRawPrice"`
	AvgPSA10Price    float64   `json:"avgPSA10Price"`
	LastUpdated      time.Time `json:"lastUpdated"`
	OpportunityCount int       `json:"opportunityCount"` // Cards with score > threshold
}

// NewWebCache creates a new web cache instance
func NewWebCache(cacheDir string) *WebCache {
	if cacheDir == "" {
		cacheDir = DefaultCacheDir
	}
	return &WebCache{
		CacheDir: cacheDir,
	}
}

// IsStale checks if the cache is older than 24 hours
func (wc *WebCache) IsStale() bool {
	metadata, err := wc.LoadMetadata()
	if err != nil {
		return true // If no metadata, consider stale
	}

	return time.Since(metadata.LastRefresh) > 24*time.Hour
}

// NeedsRefresh checks if cache needs refreshing (stale or missing)
func (wc *WebCache) NeedsRefresh() bool {
	// Check if metadata file exists
	metadataPath := filepath.Join(wc.CacheDir, MetadataFile)
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		return true
	}

	// Check if top opportunities file exists
	topOppsPath := filepath.Join(wc.CacheDir, TopOpportunitiesFile)
	if _, err := os.Stat(topOppsPath); os.IsNotExist(err) {
		return true
	}

	return wc.IsStale()
}

// SaveTopOpportunities saves the top N cards across all sets
func (wc *WebCache) SaveTopOpportunities(result *CachedResult) error {
	if err := os.MkdirAll(wc.CacheDir, 0755); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}

	path := filepath.Join(wc.CacheDir, TopOpportunitiesFile)
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal top opportunities: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// LoadTopOpportunities loads the cached top opportunities
func (wc *WebCache) LoadTopOpportunities() (*CachedResult, error) {
	path := filepath.Join(wc.CacheDir, TopOpportunitiesFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read top opportunities: %w", err)
	}

	var result CachedResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal top opportunities: %w", err)
	}

	return &result, nil
}

// SaveAllCards saves complete dataset for filtering
func (wc *WebCache) SaveAllCards(result *CachedResult) error {
	if err := os.MkdirAll(wc.CacheDir, 0755); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}

	path := filepath.Join(wc.CacheDir, AllCardsFile)
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal all cards: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// LoadAllCards loads the complete cached dataset
func (wc *WebCache) LoadAllCards() (*CachedResult, error) {
	path := filepath.Join(wc.CacheDir, AllCardsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read all cards: %w", err)
	}

	var result CachedResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal all cards: %w", err)
	}

	return &result, nil
}

// SaveMetadata saves cache metadata
func (wc *WebCache) SaveMetadata(metadata *CacheMetadata) error {
	if err := os.MkdirAll(wc.CacheDir, 0755); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}

	path := filepath.Join(wc.CacheDir, MetadataFile)
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// LoadMetadata loads cache metadata
func (wc *WebCache) LoadMetadata() (*CacheMetadata, error) {
	path := filepath.Join(wc.CacheDir, MetadataFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read metadata: %w", err)
	}

	var metadata CacheMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

// SaveSetsSummary saves per-set summary data
func (wc *WebCache) SaveSetsSummary(summaries []SetSummary) error {
	if err := os.MkdirAll(wc.CacheDir, 0755); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}

	path := filepath.Join(wc.CacheDir, SetsSummaryFile)
	data, err := json.MarshalIndent(summaries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sets summary: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// LoadSetsSummary loads per-set summary data
func (wc *WebCache) LoadSetsSummary() ([]SetSummary, error) {
	path := filepath.Join(wc.CacheDir, SetsSummaryFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read sets summary: %w", err)
	}

	var summaries []SetSummary
	if err := json.Unmarshal(data, &summaries); err != nil {
		return nil, fmt.Errorf("unmarshal sets summary: %w", err)
	}

	return summaries, nil
}

// ConvertRowsToWebFormat converts analysis rows to web cache format
func ConvertRowsToWebFormat(rows []analysis.ScoredRow, setName string, totalCards int) *CachedResult {
	columns := []string{"Rank", "Set", "Card", "No", "Rarity", "RawUSD", "BestGrade", "Delta", "Score"}
	var webRows []map[string]any

	for i, row := range rows {
		// Determine best grade price and label
		bestPrice := row.Grades.PSA10
		bestLabel := "PSA10"
		if row.Grades.BGS10 > bestPrice {
			bestPrice = row.Grades.BGS10
			bestLabel = "BGS10"
		}

		webRow := map[string]any{
			"Rank":      i + 1,
			"Set":       setName,
			"Card":      row.Card.Name,
			"No":        row.Card.Number,
			"Rarity":    row.Card.Rarity,
			"RawUSD":    fmt.Sprintf("%.2f", row.RawUSD),
			"BestGrade": fmt.Sprintf("%.2f (%s)", bestPrice, bestLabel),
			"Delta":     fmt.Sprintf("%.2f", bestPrice-row.RawUSD),
			"Score":     fmt.Sprintf("%.1f", row.Score),
		}

		// Add population data if available
		if row.Population != nil {
			columns = appendIfNotExists(columns, "PSA10Pop")
			webRow["PSA10Pop"] = row.Population.PSA10
		}

		// Add volatility if available
		if row.Volatility > 0 {
			columns = appendIfNotExists(columns, "Volatility")
			webRow["Volatility"] = fmt.Sprintf("%.1f%%", row.Volatility*100)
		}

		webRows = append(webRows, webRow)
	}

	return &CachedResult{
		Columns: columns,
		Rows:    webRows,
		Params: map[string]any{
			"analysis":   "rank",
			"cached":     true,
			"totalCards": totalCards,
		},
		Timestamp: time.Now(),
		SetName:   setName,
		CardCount: len(webRows),
	}
}

// ConvertScoredRowsToSummary creates a set summary from scored rows
func ConvertScoredRowsToSummary(setID, setName string, rows []analysis.ScoredRow) SetSummary {
	summary := SetSummary{
		ID:          setID,
		Name:        setName,
		CardCount:   len(rows),
		LastUpdated: time.Now(),
	}

	if len(rows) == 0 {
		return summary
	}

	var totalRaw, totalPSA10 float64
	opportunityCount := 0

	summary.TopScore = rows[0].Score // Assumes rows are sorted by score desc

	for _, row := range rows {
		totalRaw += row.RawUSD
		totalPSA10 += row.Grades.PSA10

		if row.Score > 25.0 { // Opportunity threshold
			opportunityCount++
		}
	}

	summary.AvgRawPrice = totalRaw / float64(len(rows))
	summary.AvgPSA10Price = totalPSA10 / float64(len(rows))
	summary.OpportunityCount = opportunityCount

	return summary
}

// appendIfNotExists adds a string to slice if it doesn't exist
func appendIfNotExists(slice []string, item string) []string {
	for _, existing := range slice {
		if existing == item {
			return slice
		}
	}
	return append(slice, item)
}
