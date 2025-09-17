package monitoring

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// HistoryEntry represents a single entry in the history CSV
type HistoryEntry struct {
	Timestamp    time.Time
	Card         string
	Number       string
	Set          string
	RawPriceUSD  float64
	PSA10USD     float64
	DeltaUSD     float64
	Score        float64
	Notes        string
}

// HistoryAnalyzer analyzes historical tracking data
type HistoryAnalyzer struct {
	entries []HistoryEntry
}

// NewHistoryAnalyzer creates a new history analyzer
func NewHistoryAnalyzer() *HistoryAnalyzer {
	return &HistoryAnalyzer{
		entries: []HistoryEntry{},
	}
}

// LoadHistory loads entries from a CSV file
func (ha *HistoryAnalyzer) LoadHistory(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, that's okay
			return nil
		}
		return fmt.Errorf("opening history file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("reading CSV: %w", err)
	}

	// Skip header if present
	startIdx := 0
	if len(records) > 0 && records[0][0] == "Timestamp" {
		startIdx = 1
	}

	for i := startIdx; i < len(records); i++ {
		entry, err := ha.parseHistoryEntry(records[i])
		if err != nil {
			continue // Skip malformed entries
		}
		ha.entries = append(ha.entries, entry)
	}

	return nil
}

// AppendHistory adds new entries to the history file
func (ha *HistoryAnalyzer) AppendHistory(path string, entries []HistoryEntry) error {
	// Check if file exists to determine if we need header
	needsHeader := false
	if _, err := os.Stat(path); os.IsNotExist(err) {
		needsHeader = true
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening history file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header if needed
	if needsHeader {
		header := []string{
			"Timestamp", "Card", "Number", "Set",
			"RawPriceUSD", "PSA10USD", "DeltaUSD", "Score", "Notes",
		}
		if err := writer.Write(header); err != nil {
			return fmt.Errorf("writing header: %w", err)
		}
	}

	// Write entries
	for _, entry := range entries {
		record := []string{
			entry.Timestamp.Format(time.RFC3339),
			entry.Card,
			entry.Number,
			entry.Set,
			fmt.Sprintf("%.2f", entry.RawPriceUSD),
			fmt.Sprintf("%.2f", entry.PSA10USD),
			fmt.Sprintf("%.2f", entry.DeltaUSD),
			fmt.Sprintf("%.2f", entry.Score),
			entry.Notes,
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("writing record: %w", err)
		}
	}

	// Add these entries to our in-memory list
	ha.entries = append(ha.entries, entries...)

	return nil
}

// AnalyzeTrends identifies patterns in historical picks
func (ha *HistoryAnalyzer) AnalyzeTrends() *TrendReport {
	if len(ha.entries) == 0 {
		return nil
	}

	report := &TrendReport{
		TotalEntries:  len(ha.entries),
		UniqueCards:   make(map[string]int),
		SetFrequency:  make(map[string]int),
		TimeAnalysis:  ha.analyzeTimePatterns(),
	}

	// Analyze card and set frequencies
	for _, entry := range ha.entries {
		cardKey := fmt.Sprintf("%s #%s", entry.Card, entry.Number)
		report.UniqueCards[cardKey]++
		report.SetFrequency[entry.Set]++
	}

	// Find top performers
	report.TopPerformers = ha.findTopPerformers(5)
	report.ConsistentPicks = ha.findConsistentPicks(3)

	// Calculate success metrics
	report.AverageScore = ha.calculateAverageScore()
	report.AverageDelta = ha.calculateAverageDelta()

	return report
}

// TrackPerformance compares past recommendations to current prices
func (ha *HistoryAnalyzer) TrackPerformance(currentSnapshot *Snapshot) *PerformanceReport {
	report := &PerformanceReport{
		Recommendations: []RecommendationOutcome{},
	}

	for _, entry := range ha.entries {
		cardKey := fmt.Sprintf("%s-%s", entry.Number, entry.Card)
		currentCard, exists := currentSnapshot.Cards[cardKey]
		if !exists {
			continue
		}

		outcome := RecommendationOutcome{
			Entry:           entry,
			CurrentRawPrice: currentCard.RawPriceUSD,
			CurrentPSA10:    currentCard.PSA10Price,
			RawPriceChange:  currentCard.RawPriceUSD - entry.RawPriceUSD,
			PSA10Change:     currentCard.PSA10Price - entry.PSA10USD,
		}

		// Calculate if recommendation was good
		originalROI := entry.DeltaUSD / entry.RawPriceUSD * 100
		currentROI := (currentCard.PSA10Price - currentCard.RawPriceUSD) / currentCard.RawPriceUSD * 100

		if currentROI > originalROI {
			outcome.Outcome = "IMPROVED"
		} else if currentROI > 0 && originalROI > 0 {
			outcome.Outcome = "PROFITABLE"
		} else {
			outcome.Outcome = "DECLINED"
		}

		report.Recommendations = append(report.Recommendations, outcome)
	}

	// Calculate statistics
	report.CalculateStats()

	return report
}

// Helper types and methods

type TrendReport struct {
	TotalEntries    int
	UniqueCards     map[string]int
	SetFrequency    map[string]int
	TopPerformers   []HistoryEntry
	ConsistentPicks []string
	AverageScore    float64
	AverageDelta    float64
	TimeAnalysis    map[string]interface{}
}

type PerformanceReport struct {
	Recommendations []RecommendationOutcome
	SuccessRate     float64
	AverageROI      float64
	BestPick        *RecommendationOutcome
	WorstPick       *RecommendationOutcome
}

type RecommendationOutcome struct {
	Entry           HistoryEntry
	CurrentRawPrice float64
	CurrentPSA10    float64
	RawPriceChange  float64
	PSA10Change     float64
	Outcome         string // "IMPROVED", "PROFITABLE", "DECLINED"
}

func (ha *HistoryAnalyzer) parseHistoryEntry(record []string) (HistoryEntry, error) {
	if len(record) < 9 {
		return HistoryEntry{}, fmt.Errorf("insufficient fields")
	}

	timestamp, err := time.Parse(time.RFC3339, record[0])
	if err != nil {
		// Try other formats
		timestamp, err = time.Parse("2006-01-02 15:04:05", record[0])
		if err != nil {
			timestamp = time.Now() // Fallback
		}
	}

	rawPrice, _ := strconv.ParseFloat(record[4], 64)
	psa10Price, _ := strconv.ParseFloat(record[5], 64)
	delta, _ := strconv.ParseFloat(record[6], 64)
	score, _ := strconv.ParseFloat(record[7], 64)

	return HistoryEntry{
		Timestamp:    timestamp,
		Card:         record[1],
		Number:       record[2],
		Set:          record[3],
		RawPriceUSD:  rawPrice,
		PSA10USD:     psa10Price,
		DeltaUSD:     delta,
		Score:        score,
		Notes:        record[8],
	}, nil
}

func (ha *HistoryAnalyzer) findTopPerformers(n int) []HistoryEntry {
	// Sort by score
	sorted := make([]HistoryEntry, len(ha.entries))
	copy(sorted, ha.entries)

	// Simple bubble sort for small dataset
	for i := 0; i < len(sorted)-1 && i < n; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].Score < sorted[j+1].Score {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	if len(sorted) < n {
		return sorted
	}
	return sorted[:n]
}

func (ha *HistoryAnalyzer) findConsistentPicks(minAppearances int) []string {
	cardCounts := make(map[string]int)
	for _, entry := range ha.entries {
		key := fmt.Sprintf("%s #%s", entry.Card, entry.Number)
		cardCounts[key]++
	}

	var consistent []string
	for card, count := range cardCounts {
		if count >= minAppearances {
			consistent = append(consistent, card)
		}
	}

	return consistent
}

func (ha *HistoryAnalyzer) calculateAverageScore() float64 {
	if len(ha.entries) == 0 {
		return 0
	}

	sum := 0.0
	for _, entry := range ha.entries {
		sum += entry.Score
	}
	return sum / float64(len(ha.entries))
}

func (ha *HistoryAnalyzer) calculateAverageDelta() float64 {
	if len(ha.entries) == 0 {
		return 0
	}

	sum := 0.0
	for _, entry := range ha.entries {
		sum += entry.DeltaUSD
	}
	return sum / float64(len(ha.entries))
}

func (ha *HistoryAnalyzer) analyzeTimePatterns() map[string]interface{} {
	patterns := make(map[string]interface{})

	if len(ha.entries) == 0 {
		return patterns
	}

	// Day of week analysis
	dayCount := make(map[string]int)
	for _, entry := range ha.entries {
		day := entry.Timestamp.Weekday().String()
		dayCount[day]++
	}

	// Hour analysis
	hourCount := make(map[int]int)
	for _, entry := range ha.entries {
		hour := entry.Timestamp.Hour()
		hourCount[hour]++
	}

	patterns["by_day"] = dayCount
	patterns["by_hour"] = hourCount

	// Find most active day
	maxDay := ""
	maxDayCount := 0
	for day, count := range dayCount {
		if count > maxDayCount {
			maxDay = day
			maxDayCount = count
		}
	}
	patterns["most_active_day"] = maxDay

	return patterns
}

func (pr *PerformanceReport) CalculateStats() {
	if len(pr.Recommendations) == 0 {
		return
	}

	successful := 0
	totalROI := 0.0

	for i, rec := range pr.Recommendations {
		if rec.Outcome == "IMPROVED" || rec.Outcome == "PROFITABLE" {
			successful++
		}

		roi := (rec.PSA10Change - rec.RawPriceChange) / rec.Entry.RawPriceUSD * 100
		totalROI += roi

		// Track best and worst
		if pr.BestPick == nil || roi > pr.BestPick.PSA10Change {
			pr.BestPick = &pr.Recommendations[i]
		}
		if pr.WorstPick == nil || roi < pr.WorstPick.PSA10Change {
			pr.WorstPick = &pr.Recommendations[i]
		}
	}

	pr.SuccessRate = float64(successful) / float64(len(pr.Recommendations)) * 100
	pr.AverageROI = totalROI / float64(len(pr.Recommendations))
}

// FormatTrendReport creates a human-readable trend report
func FormatTrendReport(report *TrendReport) string {
	if report == nil {
		return "No historical data available"
	}

	output := strings.Builder{}
	output.WriteString("HISTORICAL TREND ANALYSIS\n")
	output.WriteString("=========================\n\n")

	output.WriteString(fmt.Sprintf("Total Recommendations: %d\n", report.TotalEntries))
	output.WriteString(fmt.Sprintf("Unique Cards: %d\n", len(report.UniqueCards)))
	output.WriteString(fmt.Sprintf("Average Score: %.2f\n", report.AverageScore))
	output.WriteString(fmt.Sprintf("Average Delta: $%.2f\n\n", report.AverageDelta))

	if len(report.TopPerformers) > 0 {
		output.WriteString("Top Historical Picks:\n")
		for i, entry := range report.TopPerformers {
			output.WriteString(fmt.Sprintf("%d. %s #%s (Score: %.2f)\n",
				i+1, entry.Card, entry.Number, entry.Score))
		}
		output.WriteString("\n")
	}

	if len(report.ConsistentPicks) > 0 {
		output.WriteString("Frequently Recommended:\n")
		for _, card := range report.ConsistentPicks {
			output.WriteString(fmt.Sprintf("- %s\n", card))
		}
	}

	return output.String()
}