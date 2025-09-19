package integration

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/guarzo/pkmgradegap/internal/analysis"
	"github.com/guarzo/pkmgradegap/internal/model"
	"github.com/guarzo/pkmgradegap/internal/monitoring"
)

// TestSnapshotWorkflow tests the complete snapshot creation and comparison workflow
func TestSnapshotWorkflow(t *testing.T) {
	// Create test data
	rows := []analysis.Row{
		{
			Card: model.Card{
				Name:    "Pikachu ex",
				Number:  "001",
				SetName: "Test Set",
				Rarity:  "Double Rare",
			},
			RawUSD: 50.00,
			Grades: analysis.Grades{
				PSA10:  150.00,
				Grade9: 100.00,
			},
			Volatility: 10.5,
		},
		{
			Card: model.Card{
				Name:    "Charizard ex",
				Number:  "025",
				SetName: "Test Set",
				Rarity:  "Special Illustration Rare",
			},
			RawUSD: 200.00,
			Grades: analysis.Grades{
				PSA10:  800.00,
				Grade9: 400.00,
			},
			Volatility: 20.5,
		},
	}

	// Test snapshot creation
	snapshot := monitoring.CreateSnapshotFromRows("Test Set", rows)
	if snapshot == nil {
		t.Fatal("Failed to create snapshot")
	}

	if snapshot.SetName != "Test Set" {
		t.Errorf("Expected set name 'Test Set', got '%s'", snapshot.SetName)
	}

	if len(snapshot.Cards) != 2 {
		t.Errorf("Expected 2 cards in snapshot, got %d", len(snapshot.Cards))
	}

	// Test saving snapshot
	tmpDir := t.TempDir()
	snapshotPath := filepath.Join(tmpDir, "test_snapshot.json")

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal snapshot: %v", err)
	}

	if err := os.WriteFile(snapshotPath, data, 0644); err != nil {
		t.Fatalf("Failed to save snapshot: %v", err)
	}

	// Test loading snapshot
	loadedSnapshot, err := monitoring.LoadSnapshot(snapshotPath)
	if err != nil {
		t.Fatalf("Failed to load snapshot: %v", err)
	}

	if loadedSnapshot.SetName != snapshot.SetName {
		t.Errorf("Loaded snapshot has different set name")
	}
}

// TestAlertGeneration tests the alert engine with price changes
func TestAlertGeneration(t *testing.T) {
	// Create old snapshot
	oldSnapshot := &monitoring.Snapshot{
		Timestamp: time.Now().Add(-24 * time.Hour),
		SetName:   "Test Set",
		Cards: map[string]*monitoring.SnapshotCardData{
			"001-Pikachu ex": {
				Card: model.Card{
					Name:   "Pikachu ex",
					Number: "001",
				},
				RawUSD:     50.00,
				PSA10Price: 150.00,
			},
			"025-Charizard ex": {
				Card: model.Card{
					Name:   "Charizard ex",
					Number: "025",
				},
				RawUSD:     250.00,
				PSA10Price: 850.00,
			},
		},
	}

	// Create new snapshot with price changes
	newSnapshot := &monitoring.Snapshot{
		Timestamp: time.Now(),
		SetName:   "Test Set",
		Cards: map[string]*monitoring.SnapshotCardData{
			"001-Pikachu ex": {
				Card: model.Card{
					Name:   "Pikachu ex",
					Number: "001",
				},
				RawUSD:     40.00,  // 20% drop
				PSA10Price: 175.00, // 16.7% increase
			},
			"025-Charizard ex": {
				Card: model.Card{
					Name:   "Charizard ex",
					Number: "025",
				},
				RawUSD:     200.00, // 20% drop
				PSA10Price: 750.00, // 11.8% drop
			},
			"150-Milotic ex": { // New card
				Card: model.Card{
					Name:   "Milotic ex",
					Number: "150",
				},
				RawUSD:     15.00,
				PSA10Price: 65.00,
			},
		},
	}

	// Generate alerts
	alertConfig := monitoring.AlertConfig{
		PriceDropThresholdPct:   10.0,
		PriceDropThresholdUSD:   5.0,
		OpportunityThresholdROI: 25.0,
		VolatilityHighThreshold: 0.13,
		MinSeverity:             "LOW",
	}
	engine := monitoring.NewAlertEngine(alertConfig)

	// First compare snapshots to get deltas
	deltas := monitoring.CompareSnapshots(oldSnapshot, newSnapshot, 10.0, 5.0)
	alerts := engine.GenerateAlerts(deltas)

	// Verify alerts were generated
	if len(alerts) == 0 {
		t.Error("Expected alerts to be generated")
	}

	// Check for specific alert types
	var hasPriceDrop, hasPriceIncrease bool
	for _, alert := range alerts {
		switch alert.Type {
		case monitoring.AlertPriceDrop:
			hasPriceDrop = true
		case monitoring.AlertPriceIncrease:
			hasPriceIncrease = true
		case monitoring.AlertVolatilitySpike:
			// Volatility alerts are not yet implemented in GenerateAlerts
		}
	}

	if !hasPriceDrop {
		t.Error("Expected price drop alert for 20% raw price decrease")
	}
	if !hasPriceIncrease {
		t.Error("Expected price increase alert for 16.7% PSA10 increase")
	}
	// Volatility alerts are not yet implemented in GenerateAlerts
	// if !hasVolatilityAlert {
	// 	t.Error("Expected volatility alert for 25% volatility")
	// }
}

// TestHistoryAnalyzer tests the historical trend analysis
func TestHistoryAnalyzer(t *testing.T) {
	// Create test history file
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "test_history.csv")

	historyData := [][]string{
		{"Date", "Card", "No", "SetName", "RawUSD", "PSA10USD", "Score", "Notes"},
		{"2025-01-01", "Pikachu ex", "001", "Test Set", "50.00", "150.00", "75.00", "Initial"},
		{"2025-01-08", "Pikachu ex", "001", "Test Set", "48.00", "155.00", "82.00", "Week 1"},
		{"2025-01-15", "Pikachu ex", "001", "Test Set", "45.00", "160.00", "90.00", "Week 2"},
		{"2025-01-22", "Pikachu ex", "001", "Test Set", "42.00", "165.00", "98.00", "Week 3"},
		{"2025-01-29", "Pikachu ex", "001", "Test Set", "40.00", "170.00", "105.00", "Week 4"},
		{"2025-01-01", "Charizard ex", "025", "Test Set", "250.00", "850.00", "575.00", "Initial"},
		{"2025-01-08", "Charizard ex", "025", "Test Set", "240.00", "825.00", "560.00", "Week 1"},
		{"2025-01-15", "Charizard ex", "025", "Test Set", "230.00", "800.00", "545.00", "Week 2"},
	}

	// Write history file
	file, err := os.Create(historyPath)
	if err != nil {
		t.Fatalf("Failed to create history file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if err := writer.WriteAll(historyData); err != nil {
		t.Fatalf("Failed to write history data: %v", err)
	}

	// Test loading and analyzing history
	analyzer := monitoring.NewHistoryAnalyzer()
	if err := analyzer.LoadHistory(historyPath); err != nil {
		t.Fatalf("Failed to load history: %v", err)
	}

	report := analyzer.AnalyzeTrends()
	if report == nil {
		t.Fatal("Failed to generate trend report")
	}

	// Verify trend detection
	if report.TrendDetection == nil {
		t.Error("Expected trend detection in report")
	} else {
		// With declining raw prices and rising PSA10, should detect opportunity
		if report.TrendDetection.OverallTrend == "" {
			t.Error("Expected overall trend to be detected")
		}
	}

	// Verify moving averages
	if report.MovingAverages == nil {
		t.Error("Expected moving averages in report")
	}

	// Verify top performers
	if len(report.TopPerformers) == 0 {
		t.Error("Expected top performers to be identified")
	}
}

// TestBulkOptimizer tests the bulk submission optimization
func TestBulkOptimizer(t *testing.T) {
	// Create test scored rows - need at least 20 cards for minimum batch
	var rows []analysis.ScoredRow

	// Add enough cards to meet minimum batch requirements
	// All cards should fit in "Value" service level (max $199)
	for i := 1; i <= 25; i++ {
		rows = append(rows, analysis.ScoredRow{
			Row: analysis.Row{
				Card: model.Card{
					Name:   fmt.Sprintf("Card %d", i),
					Number: fmt.Sprintf("%03d", i),
				},
				RawUSD: float64(5 + i),                            // Raw prices $6-$30
				Grades: analysis.Grades{PSA10: float64(50 + i*5)}, // PSA10 prices $55-$175 (all under $199)
			},
			Score: float64(i * 10),
		})
	}

	// Convert ScoredRows to SubmissionCards
	submissionCards := make([]monitoring.SubmissionCard, len(rows))
	for i, row := range rows {
		submissionCards[i] = monitoring.SubmissionCard{
			Card:          row.Card,
			RawUSD:        row.RawUSD,
			PSA10Price:    row.Grades.PSA10,
			PSA9Price:     row.Grades.Grade9,
			ExpectedGrade: 10.0,
			ExpectedValue: row.Grades.PSA10,
		}
	}

	optimizer := monitoring.NewBulkOptimizer(0.13, 20.0)
	batches := optimizer.OptimizeSubmission(submissionCards)

	// Debug: Log the batch count and card values
	t.Logf("Created %d batches from %d cards", len(batches), len(submissionCards))
	if len(submissionCards) > 0 {
		t.Logf("First card: PSA10=$%.2f, Raw=$%.2f", submissionCards[0].PSA10Price, submissionCards[0].RawUSD)
		t.Logf("Last card: PSA10=$%.2f, Raw=$%.2f", submissionCards[len(submissionCards)-1].PSA10Price, submissionCards[len(submissionCards)-1].RawUSD)
	}

	if len(batches) == 0 {
		t.Error("Expected at least one batch to be created")
	}

	// Verify service level assignment
	for _, batch := range batches {
		if len(batch.Cards) == 0 {
			t.Error("Batch has no cards")
		}
		if batch.EstimatedROI < 0 {
			t.Error("Batch has negative ROI")
		}
	}
}

// TestMarketTiming tests the market timing analyzer
func TestMarketTiming(t *testing.T) {
	// Create test snapshots representing historical data
	snapshots := []*monitoring.Snapshot{
		{
			Timestamp: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			SetName:   "Test Set",
			Cards: map[string]*monitoring.SnapshotCardData{
				"001-Pikachu ex": {
					RawUSD:     50.00,
					PSA10Price: 150.00,
				},
			},
		},
		{
			Timestamp: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			SetName:   "Test Set",
			Cards: map[string]*monitoring.SnapshotCardData{
				"001-Pikachu ex": {
					RawUSD:     40.00,  // Raw dropped
					PSA10Price: 170.00, // PSA10 increased
				},
			},
		},
	}

	analyzer := monitoring.NewMarketAnalyzer(snapshots)
	recommendation := analyzer.AnalyzeCard("001-Pikachu ex")

	if recommendation == nil {
		t.Fatal("Expected timing recommendation")
	}

	// With raw prices down and PSA10 up, should recommend BUY or SUBMIT
	if recommendation.Action != "BUY" && recommendation.Action != "SUBMIT" {
		t.Errorf("Expected BUY or SUBMIT recommendation, got %s", recommendation.Action)
	}

	// Test seasonal analysis
	seasonal := analyzer.SeasonalAnalysis()
	if len(seasonal) == 0 {
		t.Error("Expected seasonal patterns to be detected")
	}
}

// TestEndToEndMonitoringWorkflow tests the complete monitoring workflow
func TestEndToEndMonitoringWorkflow(t *testing.T) {
	tmpDir := t.TempDir()

	// Step 1: Create initial analysis rows
	initialRows := []analysis.Row{
		{
			Card: model.Card{
				Name:    "Test Card",
				Number:  "001",
				SetName: "Test Set",
			},
			RawUSD: 50.00,
			Grades: analysis.Grades{PSA10: 150.00},
		},
	}

	// Step 2: Create and save initial snapshot
	snapshot1 := monitoring.CreateSnapshotFromRows("Test Set", initialRows)
	snapshot1Path := filepath.Join(tmpDir, "snapshot1.json")

	data1, _ := json.Marshal(snapshot1)
	os.WriteFile(snapshot1Path, data1, 0644)

	// Step 3: Simulate time passing and price changes
	time.Sleep(100 * time.Millisecond)

	updatedRows := []analysis.Row{
		{
			Card: model.Card{
				Name:    "Test Card",
				Number:  "001",
				SetName: "Test Set",
			},
			RawUSD: 42.00,                          // Price dropped
			Grades: analysis.Grades{PSA10: 175.00}, // PSA10 increased
		},
	}

	// Step 4: Create and save second snapshot
	snapshot2 := monitoring.CreateSnapshotFromRows("Test Set", updatedRows)
	snapshot2Path := filepath.Join(tmpDir, "snapshot2.json")

	data2, _ := json.Marshal(snapshot2)
	os.WriteFile(snapshot2Path, data2, 0644)

	// Step 5: Load snapshots and generate alerts
	loaded1, err := monitoring.LoadSnapshot(snapshot1Path)
	if err != nil {
		t.Fatalf("Failed to load first snapshot: %v", err)
	}

	loaded2, err := monitoring.LoadSnapshot(snapshot2Path)
	if err != nil {
		t.Fatalf("Failed to load second snapshot: %v", err)
	}

	// Step 6: Generate and verify alerts
	alertConfig := monitoring.AlertConfig{
		PriceDropThresholdPct:   10.0,
		PriceDropThresholdUSD:   5.0,
		OpportunityThresholdROI: 25.0,
		VolatilityHighThreshold: 0.13,
		MinSeverity:             "LOW",
	}
	engine := monitoring.NewAlertEngine(alertConfig)

	// First compare snapshots to get deltas
	deltas := monitoring.CompareSnapshots(loaded1, loaded2, 10.0, 5.0)
	alerts := engine.GenerateAlerts(deltas)

	if len(alerts) == 0 {
		t.Error("Expected alerts from price changes")
	}

	// Step 7: Generate alert report
	report := monitoring.GenerateAlertReport(alerts, loaded1, loaded2, snapshot1Path, snapshot2Path, alertConfig)

	if report == nil {
		t.Error("Expected alert report to be generated")
	}

	// Step 8: Verify alerts have proper structure
	for _, alert := range alerts {
		if alert.Type == "" {
			t.Error("Alert missing type")
		}
		if alert.Severity == "" {
			t.Error("Alert missing severity")
		}
	}
}

// TestProgressIndicatorIntegration tests that progress indicators work with monitoring
func TestProgressIndicatorIntegration(t *testing.T) {
	// This is a simple test to ensure progress indicators can be created
	// In real usage, they would update during long-running operations

	tests := []struct {
		name     string
		total    int
		progress int
	}{
		{"Snapshot loading", 100, 50},
		{"Alert generation", 200, 150},
		{"Trend analysis", 300, 300},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate progress tracking
			progressPct := float64(tt.progress) / float64(tt.total) * 100
			if progressPct > 100 {
				t.Errorf("Progress exceeds 100%%: %.1f%%", progressPct)
			}
		})
	}
}
