package monitoring

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	reportpkg "github.com/guarzo/pkmgradegap/internal/report"
)

// HistoryEntry represents a single entry in the history CSV
type HistoryEntry struct {
	Timestamp   time.Time
	Card        string
	Number      string
	Set         string
	RawUSD float64
	PSA10USD    float64
	DeltaUSD    float64
	Score       float64
	Notes       string
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
	if len(records) > 0 && (records[0][0] == "Timestamp" || records[0][0] == "Date") {
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
			"RawUSD", "PSA10USD", "DeltaUSD", "Score", "Notes",
		}
		if err := writer.Write(reportpkg.EscapeCSVRow(header)); err != nil {
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
			fmt.Sprintf("%.2f", entry.RawUSD),
			fmt.Sprintf("%.2f", entry.PSA10USD),
			fmt.Sprintf("%.2f", entry.DeltaUSD),
			fmt.Sprintf("%.2f", entry.Score),
			entry.Notes,
		}
		if err := writer.Write(reportpkg.EscapeCSVRow(record)); err != nil {
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
		TotalEntries: len(ha.entries),
		UniqueCards:  make(map[string]int),
		SetFrequency: make(map[string]int),
		TimeAnalysis: ha.analyzeTimePatterns(),
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

	// Advanced trend detection
	report.TrendDetection = ha.detectTrends()
	report.MovingAverages = ha.calculateMovingAverages()
	report.SeasonalPatterns = ha.analyzeSeasonalPatterns()
	report.MomentumAnalysis = ha.analyzeMomentum()

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
			CurrentRawPrice: currentCard.RawUSD,
			CurrentPSA10:    currentCard.PSA10Price,
			RawPriceChange:  currentCard.RawUSD - entry.RawUSD,
			PSA10Change:     currentCard.PSA10Price - entry.PSA10USD,
		}

		// Calculate if recommendation was good
		originalROI := entry.DeltaUSD / entry.RawUSD * 100
		currentROI := (currentCard.PSA10Price - currentCard.RawUSD) / currentCard.RawUSD * 100

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
	TotalEntries     int
	UniqueCards      map[string]int
	SetFrequency     map[string]int
	TopPerformers    []HistoryEntry
	ConsistentPicks  []string
	AverageScore     float64
	AverageDelta     float64
	TimeAnalysis     map[string]interface{}
	TrendDetection   *TrendDetection
	MovingAverages   *MovingAverages
	SeasonalPatterns *SeasonalAnalysis
	MomentumAnalysis *MomentumAnalysis
}

// TrendDetection contains price trend analysis
type TrendDetection struct {
	OverallTrend    string               // "BULLISH", "BEARISH", "SIDEWAYS"
	TrendStrength   float64              // 0-100
	CardTrends      map[string]CardTrend // Per-card trend analysis
	SetTrends       map[string]string    // Per-set trend direction
	ConfidenceScore float64              // 0-100
}

// CardTrend represents trend analysis for a specific card
type CardTrend struct {
	Direction       string  // "UP", "DOWN", "STABLE"
	Slope           float64 // Linear regression slope
	Correlation     float64 // R-squared value
	RecentPrices    []float64
	PriceVolatility float64
}

// MovingAverages contains moving average calculations
type MovingAverages struct {
	MA7Days         float64
	MA30Days        float64
	MA90Days        float64
	CurrentAboveMA7 bool
	TrendSignal     string // "BUY", "SELL", "HOLD"
}

// SeasonalAnalysis identifies seasonal patterns
type SeasonalAnalysis struct {
	BestMonths     []string
	WorstMonths    []string
	SeasonalFactor map[string]float64 // Month -> multiplier
	YearOverYear   map[int]float64    // Year -> average score
	Cyclical       bool               // Whether patterns repeat
}

// MomentumAnalysis analyzes price momentum
type MomentumAnalysis struct {
	CurrentMomentum  string    // "STRONG_UP", "WEAK_UP", "NEUTRAL", "WEAK_DOWN", "STRONG_DOWN"
	MomentumScore    float64   // -100 to 100
	Acceleration     float64   // Rate of change of momentum
	SupportLevels    []float64 // Key support price levels
	ResistanceLevels []float64 // Key resistance price levels
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
	// Support both 8-field (our sample) and 9-field formats
	if len(record) < 8 {
		return HistoryEntry{}, fmt.Errorf("insufficient fields")
	}

	timestamp, err := time.Parse(time.RFC3339, record[0])
	if err != nil {
		// Try other formats
		timestamp, err = time.Parse("2006-01-02 15:04:05", record[0])
		if err != nil {
			// Try date-only format
			timestamp, err = time.Parse("2006-01-02", record[0])
			if err != nil {
				timestamp = time.Now() // Fallback
			}
		}
	}

	rawPrice, _ := strconv.ParseFloat(record[4], 64)
	psa10Price, _ := strconv.ParseFloat(record[5], 64)

	// For 8-field format, calculate delta; for 9-field, use provided delta
	var delta float64
	var score float64
	if len(record) >= 9 {
		delta, _ = strconv.ParseFloat(record[6], 64)
		score, _ = strconv.ParseFloat(record[7], 64)
	} else {
		// 8-field format: Score is in field 6
		score, _ = strconv.ParseFloat(record[6], 64)
		delta = psa10Price - rawPrice - 25 // Assuming standard grading cost
	}

	// Notes field is optional
	notes := ""
	if len(record) >= 9 && len(record) == 9 {
		notes = record[8]
	} else if len(record) == 8 {
		notes = record[7]
	}

	return HistoryEntry{
		Timestamp:   timestamp,
		Card:        record[1],
		Number:      record[2],
		Set:         record[3],
		RawUSD: rawPrice,
		PSA10USD:    psa10Price,
		DeltaUSD:    delta,
		Score:       score,
		Notes:       notes,
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

		roi := (rec.PSA10Change - rec.RawPriceChange) / rec.Entry.RawUSD * 100
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

// Advanced trend detection methods

// detectTrends analyzes overall market trends using linear regression
func (ha *HistoryAnalyzer) detectTrends() *TrendDetection {
	if len(ha.entries) < 5 {
		return &TrendDetection{
			OverallTrend:    "INSUFFICIENT_DATA",
			TrendStrength:   0,
			CardTrends:      make(map[string]CardTrend),
			SetTrends:       make(map[string]string),
			ConfidenceScore: 0,
		}
	}

	detection := &TrendDetection{
		CardTrends: make(map[string]CardTrend),
		SetTrends:  make(map[string]string),
	}

	// Calculate overall trend using PSA10 prices
	prices := make([]float64, 0)
	times := make([]float64, 0)

	for i, entry := range ha.entries {
		if entry.PSA10USD > 0 {
			prices = append(prices, entry.PSA10USD)
			times = append(times, float64(i)) // Use index as time proxy
		}
	}

	if len(prices) >= 3 {
		slope, rSquared := ha.linearRegression(times, prices)
		detection.TrendStrength = abs(slope) * 10 // Scale to 0-100
		detection.ConfidenceScore = rSquared * 100

		if slope > 0.5 {
			detection.OverallTrend = "BULLISH"
		} else if slope < -0.5 {
			detection.OverallTrend = "BEARISH"
		} else {
			detection.OverallTrend = "SIDEWAYS"
		}
	}

	// Analyze per-card trends
	cardPrices := make(map[string][]float64)
	for _, entry := range ha.entries {
		cardKey := fmt.Sprintf("%s #%s", entry.Card, entry.Number)
		if entry.PSA10USD > 0 {
			cardPrices[cardKey] = append(cardPrices[cardKey], entry.PSA10USD)
		}
	}

	for cardKey, prices := range cardPrices {
		if len(prices) >= 3 {
			times := make([]float64, len(prices))
			for i := range times {
				times[i] = float64(i)
			}

			slope, rSquared := ha.linearRegression(times, prices)
			volatility := ha.calculateVolatility(prices)

			direction := "STABLE"
			if slope > 1.0 {
				direction = "UP"
			} else if slope < -1.0 {
				direction = "DOWN"
			}

			detection.CardTrends[cardKey] = CardTrend{
				Direction:       direction,
				Slope:           slope,
				Correlation:     rSquared,
				RecentPrices:    prices,
				PriceVolatility: volatility,
			}
		}
	}

	return detection
}

// calculateMovingAverages computes moving averages for trend analysis
func (ha *HistoryAnalyzer) calculateMovingAverages() *MovingAverages {
	if len(ha.entries) == 0 {
		return &MovingAverages{TrendSignal: "INSUFFICIENT_DATA"}
	}

	// Get price series (PSA10 prices)
	prices := make([]float64, 0)
	for _, entry := range ha.entries {
		if entry.PSA10USD > 0 {
			prices = append(prices, entry.PSA10USD)
		}
	}

	if len(prices) == 0 {
		return &MovingAverages{TrendSignal: "NO_PRICE_DATA"}
	}

	ma := &MovingAverages{}

	// Calculate moving averages
	if len(prices) >= 7 {
		ma.MA7Days = ha.simpleMovingAverage(prices, 7)
	}
	if len(prices) >= 30 {
		ma.MA30Days = ha.simpleMovingAverage(prices, 30)
	}
	if len(prices) >= 90 {
		ma.MA90Days = ha.simpleMovingAverage(prices, 90)
	}

	// Current price vs MA7
	currentPrice := prices[len(prices)-1]
	if ma.MA7Days > 0 {
		ma.CurrentAboveMA7 = currentPrice > ma.MA7Days

		// Generate trend signal
		if ma.MA7Days > ma.MA30Days && ma.MA30Days > ma.MA90Days {
			ma.TrendSignal = "BUY"
		} else if ma.MA7Days < ma.MA30Days && ma.MA30Days < ma.MA90Days {
			ma.TrendSignal = "SELL"
		} else {
			ma.TrendSignal = "HOLD"
		}
	}

	return ma
}

// analyzeSeasonalPatterns identifies seasonal patterns in recommendations
func (ha *HistoryAnalyzer) analyzeSeasonalPatterns() *SeasonalAnalysis {
	analysis := &SeasonalAnalysis{
		SeasonalFactor: make(map[string]float64),
		YearOverYear:   make(map[int]float64),
	}

	if len(ha.entries) < 12 {
		return analysis
	}

	// Analyze by month
	monthScores := make(map[string][]float64)
	yearScores := make(map[int][]float64)

	for _, entry := range ha.entries {
		month := entry.Timestamp.Month().String()
		year := entry.Timestamp.Year()

		monthScores[month] = append(monthScores[month], entry.Score)
		yearScores[year] = append(yearScores[year], entry.Score)
	}

	// Calculate seasonal factors
	overallAvg := ha.calculateAverageScore()
	bestScore := 0.0
	worstScore := 1000.0

	for month, scores := range monthScores {
		if len(scores) > 0 {
			monthAvg := ha.arrayAverage(scores)
			factor := monthAvg / overallAvg
			analysis.SeasonalFactor[month] = factor

			if monthAvg > bestScore {
				bestScore = monthAvg
				analysis.BestMonths = []string{month}
			} else if monthAvg == bestScore {
				analysis.BestMonths = append(analysis.BestMonths, month)
			}

			if monthAvg < worstScore {
				worstScore = monthAvg
				analysis.WorstMonths = []string{month}
			} else if monthAvg == worstScore {
				analysis.WorstMonths = append(analysis.WorstMonths, month)
			}
		}
	}

	// Year-over-year analysis
	for year, scores := range yearScores {
		if len(scores) > 0 {
			analysis.YearOverYear[year] = ha.arrayAverage(scores)
		}
	}

	// Check for cyclical patterns
	analysis.Cyclical = len(analysis.SeasonalFactor) >= 6 // Need data for at least 6 months

	return analysis
}

// analyzeMomentum analyzes price momentum and support/resistance levels
func (ha *HistoryAnalyzer) analyzeMomentum() *MomentumAnalysis {
	analysis := &MomentumAnalysis{
		SupportLevels:    make([]float64, 0),
		ResistanceLevels: make([]float64, 0),
	}

	if len(ha.entries) < 10 {
		analysis.CurrentMomentum = "INSUFFICIENT_DATA"
		return analysis
	}

	// Get recent price data
	recentPrices := make([]float64, 0)
	for i := len(ha.entries) - 10; i < len(ha.entries); i++ {
		if i >= 0 && ha.entries[i].PSA10USD > 0 {
			recentPrices = append(recentPrices, ha.entries[i].PSA10USD)
		}
	}

	if len(recentPrices) < 5 {
		analysis.CurrentMomentum = "NO_PRICE_DATA"
		return analysis
	}

	// Calculate momentum using rate of change
	momentum := ha.calculateMomentum(recentPrices)
	analysis.MomentumScore = momentum

	// Classify momentum
	if momentum > 15 {
		analysis.CurrentMomentum = "STRONG_UP"
	} else if momentum > 5 {
		analysis.CurrentMomentum = "WEAK_UP"
	} else if momentum > -5 {
		analysis.CurrentMomentum = "NEUTRAL"
	} else if momentum > -15 {
		analysis.CurrentMomentum = "WEAK_DOWN"
	} else {
		analysis.CurrentMomentum = "STRONG_DOWN"
	}

	// Calculate acceleration (momentum of momentum)
	if len(recentPrices) >= 6 {
		midMomentum := ha.calculateMomentum(recentPrices[:len(recentPrices)/2])
		analysis.Acceleration = momentum - midMomentum
	}

	// Identify support and resistance levels
	analysis.SupportLevels, analysis.ResistanceLevels = ha.findSupportResistance(recentPrices)

	return analysis
}

// Helper mathematical functions

func (ha *HistoryAnalyzer) linearRegression(x, y []float64) (slope, rSquared float64) {
	if len(x) != len(y) || len(x) < 2 {
		return 0, 0
	}

	n := float64(len(x))
	var sumX, sumY, sumXY, sumXX float64

	for i := 0; i < len(x); i++ {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumXX += x[i] * x[i]
	}

	slope = (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)

	// Calculate R-squared
	meanY := sumY / n
	var totalSumSquares, residualSumSquares float64

	for i := 0; i < len(y); i++ {
		predicted := slope*x[i] + (sumY-slope*sumX)/n
		totalSumSquares += (y[i] - meanY) * (y[i] - meanY)
		residualSumSquares += (y[i] - predicted) * (y[i] - predicted)
	}

	if totalSumSquares > 0 {
		rSquared = 1 - (residualSumSquares / totalSumSquares)
	}

	return slope, rSquared
}

func (ha *HistoryAnalyzer) simpleMovingAverage(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}

	start := len(prices) - period
	sum := 0.0
	for i := start; i < len(prices); i++ {
		sum += prices[i]
	}

	return sum / float64(period)
}

func (ha *HistoryAnalyzer) calculateVolatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}

	mean := ha.arrayAverage(prices)
	variance := 0.0

	for _, price := range prices {
		diff := price - mean
		variance += diff * diff
	}

	variance /= float64(len(prices) - 1)
	return variance // Return variance; square root would give standard deviation
}

func (ha *HistoryAnalyzer) arrayAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, value := range values {
		sum += value
	}

	return sum / float64(len(values))
}

func (ha *HistoryAnalyzer) calculateMomentum(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}

	firstPrice := prices[0]
	lastPrice := prices[len(prices)-1]

	return ((lastPrice - firstPrice) / firstPrice) * 100
}

func (ha *HistoryAnalyzer) findSupportResistance(prices []float64) (support, resistance []float64) {
	if len(prices) < 5 {
		return support, resistance
	}

	// Simple approach: find local minima and maxima
	for i := 1; i < len(prices)-1; i++ {
		// Local minimum (support)
		if prices[i] < prices[i-1] && prices[i] < prices[i+1] {
			support = append(support, prices[i])
		}
		// Local maximum (resistance)
		if prices[i] > prices[i-1] && prices[i] > prices[i+1] {
			resistance = append(resistance, prices[i])
		}
	}

	return support, resistance
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

	// Trend Detection
	if report.TrendDetection != nil {
		output.WriteString("TREND ANALYSIS:\n")
		output.WriteString("===============\n")
		output.WriteString(fmt.Sprintf("Overall Trend: %s\n", report.TrendDetection.OverallTrend))
		output.WriteString(fmt.Sprintf("Trend Strength: %.1f/100\n", report.TrendDetection.TrendStrength))
		output.WriteString(fmt.Sprintf("Confidence: %.1f%%\n\n", report.TrendDetection.ConfidenceScore))
	}

	// Moving Averages
	if report.MovingAverages != nil {
		output.WriteString("MOVING AVERAGES:\n")
		output.WriteString("================\n")
		if report.MovingAverages.MA7Days > 0 {
			output.WriteString(fmt.Sprintf("7-Day MA: $%.2f\n", report.MovingAverages.MA7Days))
		}
		if report.MovingAverages.MA30Days > 0 {
			output.WriteString(fmt.Sprintf("30-Day MA: $%.2f\n", report.MovingAverages.MA30Days))
		}
		if report.MovingAverages.MA90Days > 0 {
			output.WriteString(fmt.Sprintf("90-Day MA: $%.2f\n", report.MovingAverages.MA90Days))
		}
		output.WriteString(fmt.Sprintf("Signal: %s\n\n", report.MovingAverages.TrendSignal))
	}

	// Momentum Analysis
	if report.MomentumAnalysis != nil {
		output.WriteString("MOMENTUM ANALYSIS:\n")
		output.WriteString("==================\n")
		output.WriteString(fmt.Sprintf("Current Momentum: %s\n", report.MomentumAnalysis.CurrentMomentum))
		output.WriteString(fmt.Sprintf("Momentum Score: %.1f\n", report.MomentumAnalysis.MomentumScore))
		if report.MomentumAnalysis.Acceleration != 0 {
			output.WriteString(fmt.Sprintf("Acceleration: %.2f\n", report.MomentumAnalysis.Acceleration))
		}
		if len(report.MomentumAnalysis.SupportLevels) > 0 {
			output.WriteString(fmt.Sprintf("Support Levels: %v\n", report.MomentumAnalysis.SupportLevels))
		}
		if len(report.MomentumAnalysis.ResistanceLevels) > 0 {
			output.WriteString(fmt.Sprintf("Resistance Levels: %v\n", report.MomentumAnalysis.ResistanceLevels))
		}
		output.WriteString("\n")
	}

	// Seasonal Patterns
	if report.SeasonalPatterns != nil && len(report.SeasonalPatterns.BestMonths) > 0 {
		output.WriteString("SEASONAL PATTERNS:\n")
		output.WriteString("==================\n")
		output.WriteString(fmt.Sprintf("Best Months: %v\n", report.SeasonalPatterns.BestMonths))
		output.WriteString(fmt.Sprintf("Worst Months: %v\n", report.SeasonalPatterns.WorstMonths))
		output.WriteString(fmt.Sprintf("Cyclical: %t\n\n", report.SeasonalPatterns.Cyclical))
	}

	if len(report.TopPerformers) > 0 {
		output.WriteString("TOP HISTORICAL PICKS:\n")
		output.WriteString("====================\n")
		for i, entry := range report.TopPerformers {
			output.WriteString(fmt.Sprintf("%d. %s #%s (Score: %.2f)\n",
				i+1, entry.Card, entry.Number, entry.Score))
		}
		output.WriteString("\n")
	}

	if len(report.ConsistentPicks) > 0 {
		output.WriteString("FREQUENTLY RECOMMENDED:\n")
		output.WriteString("======================\n")
		for _, card := range report.ConsistentPicks {
			output.WriteString(fmt.Sprintf("- %s\n", card))
		}
	}

	return output.String()
}

// ExportTrendReportToCSV exports trend analysis to CSV format
func ExportTrendReportToCSV(report *TrendReport, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("creating CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write metadata section
	metadata := [][]string{
		{"Report Type", "Historical Trend Analysis"},
		{"Generated At", time.Now().Format("2006-01-02 15:04:05")},
		{"Total Entries", fmt.Sprintf("%d", report.TotalEntries)},
		{"Unique Cards", fmt.Sprintf("%d", len(report.UniqueCards))},
		{"Average Score", fmt.Sprintf("%.2f", report.AverageScore)},
		{"Average Delta", fmt.Sprintf("%.2f", report.AverageDelta)},
		{}, // Empty line
	}

	for _, row := range metadata {
		if err := writer.Write(reportpkg.EscapeCSVRow(row)); err != nil {
			return fmt.Errorf("writing metadata: %w", err)
		}
	}

	// Write trend analysis section if available
	if report.TrendDetection != nil {
		trendSection := [][]string{
			{"TREND ANALYSIS", ""},
			{"Overall Trend", report.TrendDetection.OverallTrend},
			{"Trend Strength", fmt.Sprintf("%.1f", report.TrendDetection.TrendStrength)},
			{"Confidence Score", fmt.Sprintf("%.1f", report.TrendDetection.ConfidenceScore)},
			{}, // Empty line
		}

		for _, row := range trendSection {
			if err := writer.Write(reportpkg.EscapeCSVRow(row)); err != nil {
				return fmt.Errorf("writing trend analysis: %w", err)
			}
		}
	}

	// Write moving averages section if available
	if report.MovingAverages != nil {
		maSection := [][]string{
			{"MOVING AVERAGES", ""},
			{"7-Day MA", fmt.Sprintf("%.2f", report.MovingAverages.MA7Days)},
			{"30-Day MA", fmt.Sprintf("%.2f", report.MovingAverages.MA30Days)},
			{"90-Day MA", fmt.Sprintf("%.2f", report.MovingAverages.MA90Days)},
			{"Trend Signal", report.MovingAverages.TrendSignal},
			{}, // Empty line
		}

		for _, row := range maSection {
			if err := writer.Write(reportpkg.EscapeCSVRow(row)); err != nil {
				return fmt.Errorf("writing moving averages: %w", err)
			}
		}
	}

	// Write momentum analysis section if available
	if report.MomentumAnalysis != nil {
		momentumSection := [][]string{
			{"MOMENTUM ANALYSIS", ""},
			{"Current Momentum", report.MomentumAnalysis.CurrentMomentum},
			{"Momentum Score", fmt.Sprintf("%.1f", report.MomentumAnalysis.MomentumScore)},
			{"Acceleration", fmt.Sprintf("%.2f", report.MomentumAnalysis.Acceleration)},
			{}, // Empty line
		}

		for _, row := range momentumSection {
			if err := writer.Write(reportpkg.EscapeCSVRow(row)); err != nil {
				return fmt.Errorf("writing momentum analysis: %w", err)
			}
		}
	}

	// Write seasonal patterns section if available
	if report.SeasonalPatterns != nil && len(report.SeasonalPatterns.BestMonths) > 0 {
		seasonalSection := [][]string{
			{"SEASONAL PATTERNS", ""},
			{"Best Months", strings.Join(report.SeasonalPatterns.BestMonths, ", ")},
			{"Worst Months", strings.Join(report.SeasonalPatterns.WorstMonths, ", ")},
			{"Cyclical", fmt.Sprintf("%t", report.SeasonalPatterns.Cyclical)},
			{}, // Empty line
		}

		for _, row := range seasonalSection {
			if err := writer.Write(reportpkg.EscapeCSVRow(row)); err != nil {
				return fmt.Errorf("writing seasonal patterns: %w", err)
			}
		}
	}

	// Write top performers section
	if len(report.TopPerformers) > 0 {
		topPerformersSection := [][]string{
			{"TOP PERFORMERS", "Score", "Card", "Number", "Set", "Raw Price", "PSA10 Price", "Delta"},
		}

		for i, entry := range report.TopPerformers {
			if i >= 10 { // Limit to top 10 in CSV
				break
			}
			row := []string{
				fmt.Sprintf("Rank %d", i+1),
				fmt.Sprintf("%.2f", entry.Score),
				entry.Card,
				entry.Number,
				entry.Set,
				fmt.Sprintf("%.2f", entry.RawUSD),
				fmt.Sprintf("%.2f", entry.PSA10USD),
				fmt.Sprintf("%.2f", entry.DeltaUSD),
			}
			topPerformersSection = append(topPerformersSection, row)
		}

		// Add empty line
		topPerformersSection = append(topPerformersSection, []string{})

		for _, row := range topPerformersSection {
			if err := writer.Write(reportpkg.EscapeCSVRow(row)); err != nil {
				return fmt.Errorf("writing top performers: %w", err)
			}
		}
	}

	// Write card-specific trends if available
	if report.TrendDetection != nil && len(report.TrendDetection.CardTrends) > 0 {
		cardTrendsSection := [][]string{
			{"CARD TRENDS", "Direction", "Slope", "Correlation", "Volatility"},
		}

		// Limit to top 20 cards by correlation
		cardTrends := make([]struct {
			name  string
			trend CardTrend
		}, 0)

		for cardName, trend := range report.TrendDetection.CardTrends {
			cardTrends = append(cardTrends, struct {
				name  string
				trend CardTrend
			}{cardName, trend})
		}

		// Simple sort by correlation (descending)
		for i := 0; i < len(cardTrends)-1 && i < 20; i++ {
			for j := 0; j < len(cardTrends)-i-1; j++ {
				if cardTrends[j].trend.Correlation < cardTrends[j+1].trend.Correlation {
					cardTrends[j], cardTrends[j+1] = cardTrends[j+1], cardTrends[j]
				}
			}
		}

		maxCards := 20
		if len(cardTrends) < maxCards {
			maxCards = len(cardTrends)
		}

		for i := 0; i < maxCards; i++ {
			row := []string{
				cardTrends[i].name,
				cardTrends[i].trend.Direction,
				fmt.Sprintf("%.3f", cardTrends[i].trend.Slope),
				fmt.Sprintf("%.3f", cardTrends[i].trend.Correlation),
				fmt.Sprintf("%.2f", cardTrends[i].trend.PriceVolatility),
			}
			cardTrendsSection = append(cardTrendsSection, row)
		}

		for _, row := range cardTrendsSection {
			if err := writer.Write(reportpkg.EscapeCSVRow(row)); err != nil {
				return fmt.Errorf("writing card trends: %w", err)
			}
		}
	}

	return nil
}
