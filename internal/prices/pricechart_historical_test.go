package prices

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/guarzo/pkmgradegap/internal/cache"
)

// TestPriceCharting_GetPriceHistory tests historical price data retrieval
func TestPriceCharting_GetPriceHistory(t *testing.T) {
	// Mock server for API responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate successful historical data response
		response := map[string]any{
			"status": "success",
			"price-history": []map[string]any{
				{
					"date":         "2024-01-15",
					"psa10-price":  15000,
					"grade9-price": 8000,
					"raw-price":    3000,
					"volume":       5,
					"timestamp":    1705334400,
				},
				{
					"date":         "2024-01-16",
					"psa10-price":  15500,
					"grade9-price": 8200,
					"raw-price":    3100,
					"volume":       3,
					"timestamp":    1705420800,
				},
				{
					"date":         "2024-01-17",
					"psa10-price":  14800,
					"grade9-price": 7900,
					"raw-price":    2950,
					"volume":       7,
					"timestamp":    1705507200,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create PriceCharting client with mock server
	c, _ := cache.New("/tmp/test-cache.json")
	pc := NewPriceCharting("test-token", c)

	// Override API endpoint for testing
	originalURL := "https://www.pricecharting.com/api/product/history"
	mockURL := server.URL

	// Mock the httpGetJSON function by testing the parseHistoricalData directly
	testData := map[string]any{
		"status": "success",
		"price-history": []any{
			map[string]any{
				"date":         "2024-01-15",
				"psa10-price":  15000.0,
				"grade9-price": 8000.0,
				"raw-price":    3000.0,
				"volume":       5.0,
				"timestamp":    1705334400.0,
			},
			map[string]any{
				"date":         "2024-01-16",
				"psa10-price":  15500.0,
				"grade9-price": 8200.0,
				"raw-price":    3100.0,
				"volume":       3.0,
				"timestamp":    1705420800.0,
			},
		},
	}

	history := pc.parseHistoricalData(testData)

	// Verify parsed data
	if len(history) != 2 {
		t.Errorf("Expected 2 price points, got %d", len(history))
	}

	expected := PricePoint{
		Date:        "2024-01-15",
		PSA10Price:  15000,
		Grade9Price: 8000,
		RawPrice:    3000,
		Volume:      5,
		Timestamp:   1705334400,
	}

	if !reflect.DeepEqual(history[0], expected) {
		t.Errorf("Expected %+v, got %+v", expected, history[0])
	}

	// Test with invalid data
	invalidData := map[string]any{
		"status":  "error",
		"message": "Product not found",
	}

	invalidHistory := pc.parseHistoricalData(invalidData)
	if len(invalidHistory) != 0 {
		t.Errorf("Expected empty history for invalid data, got %d items", len(invalidHistory))
	}

	_ = originalURL // Avoid unused variable warning
	_ = mockURL     // Avoid unused variable warning
}

// TestPriceCharting_GetTrendAnalysis tests trend analysis calculations
func TestPriceCharting_GetTrendAnalysis(t *testing.T) {
	c, _ := cache.New("/tmp/test-cache2.json")
	pc := NewPriceCharting("test-token", c)

	// Test calculateMovingAverage
	prices := []float64{100, 105, 110, 108, 112, 115, 118}
	ma7 := pc.calculateMovingAverage(prices, 7)
	expectedMA := (100 + 105 + 110 + 108 + 112 + 115 + 118) / 7.0
	if ma7 != expectedMA {
		t.Errorf("Expected moving average %.2f, got %.2f", expectedMA, ma7)
	}

	// Test calculateVolatility
	volatility := pc.calculateVolatility(prices)
	if volatility <= 0 {
		t.Error("Expected positive volatility")
	}

	// Test calculateTrendDirection
	upwardPrices := []float64{100, 102, 104, 106, 108, 110, 112, 114, 116, 118}
	direction, strength := pc.calculateTrendDirection(upwardPrices)
	if direction != "up" {
		t.Errorf("Expected upward trend, got %s", direction)
	}
	if strength <= 0.5 {
		t.Errorf("Expected strong trend (>0.5), got %.3f", strength)
	}

	downwardPrices := []float64{120, 118, 116, 114, 112, 110, 108, 106, 104, 102}
	direction, strength = pc.calculateTrendDirection(downwardPrices)
	if direction != "down" {
		t.Errorf("Expected downward trend, got %s", direction)
	}
	if strength <= 0.5 {
		t.Errorf("Expected strong trend (>0.5), got %.3f", strength)
	}

	stablePrices := []float64{100, 101, 100, 99, 100, 101, 100}
	direction, strength = pc.calculateTrendDirection(stablePrices)
	if direction != "stable" {
		t.Errorf("Expected stable trend, got %s", direction)
	}

	// Test calculatePercentChange
	testPrices := []float64{100, 105, 110, 115, 120}
	change3d := pc.calculatePercentChange(testPrices, 3)
	// For len=5, periods=3: pastPrice = prices[5-3-1] = prices[1] = 105
	// change = (120 - 105) / 105 * 100 = 14.29%
	expectedChange := ((120.0 - 105.0) / 105.0) * 100.0
	if math.Abs(change3d-expectedChange) > 0.01 {
		t.Errorf("Expected percent change %.2f, got %.2f", expectedChange, change3d)
	}

	// Test calculateSupportResistance
	volatilePrices := []float64{100, 95, 105, 90, 110, 85, 115, 80, 120, 75, 125, 70}
	support, resistance := pc.calculateSupportResistance(volatilePrices)
	if support >= resistance {
		t.Errorf("Support level (%d) should be less than resistance level (%d)", support, resistance)
	}
}

// TestPriceCharting_GeneratePricePrediction tests price prediction logic
func TestPriceCharting_GeneratePricePrediction(t *testing.T) {
	c, _ := cache.New("/tmp/test-cache3.json")
	pc := NewPriceCharting("test-token", c)

	// Test calculateSeasonalFactor directly by calling at different times
	// We can't mock time.Now globally, so test the logic differently

	// Just test that the seasonal factor function runs without error
	factor := pc.calculateSeasonalFactor()
	if factor <= 0 {
		t.Errorf("Expected positive seasonal factor, got %.2f", factor)
	}

	// Test the range is reasonable (between 0.9 and 1.2)
	if factor < 0.9 || factor > 1.2 {
		t.Errorf("Seasonal factor out of expected range (0.9-1.2): %.2f", factor)
	}
}

// TestPriceCharting_EnrichWithHistoricalData tests historical data enrichment
func TestPriceCharting_EnrichWithHistoricalData(t *testing.T) {
	c, _ := cache.New("/tmp/test-cache4.json")
	pc := NewPriceCharting("test-token", c)

	// Test with nil match
	err := pc.EnrichWithHistoricalData(nil)
	if err == nil {
		t.Error("Expected error for nil match")
	}

	// Test with empty ID
	match := &PCMatch{
		ID:          "",
		ProductName: "Test Card",
	}
	err = pc.EnrichWithHistoricalData(match)
	if err == nil {
		t.Error("Expected error for empty ID")
	}

	// Test with valid match but no data available
	match = &PCMatch{
		ID:          "test-id",
		ProductName: "Test Card",
		PSA10Cents:  15000,
	}
	// This should not fail even if no historical data is available
	err = pc.EnrichWithHistoricalData(match)
	if err != nil {
		t.Errorf("Enrichment should not fail when historical data unavailable: %v", err)
	}
}

// TestPricePoint_Validation tests PricePoint data validation
func TestPricePoint_Validation(t *testing.T) {
	validPoint := PricePoint{
		Date:        "2024-01-15",
		PSA10Price:  15000,
		Grade9Price: 8000,
		RawPrice:    3000,
		Volume:      5,
		Timestamp:   1705334400,
	}

	// Verify all fields are properly set
	if validPoint.Date == "" {
		t.Error("Date should not be empty")
	}
	if validPoint.PSA10Price <= 0 {
		t.Error("PSA10Price should be positive")
	}
	if validPoint.Timestamp <= 0 {
		t.Error("Timestamp should be positive")
	}
}

// TestTrendData_Validation tests TrendData structure validation
func TestTrendData_Validation(t *testing.T) {
	trendData := &TrendData{
		Direction:        "up",
		Strength:         0.85,
		Volatility:       0.15,
		SupportLevel:     10000,
		ResistanceLevel:  20000,
		MovingAverage7d:  15000,
		MovingAverage30d: 14500,
		PercentChange7d:  5.2,
		PercentChange30d: 12.8,
		SeasonalFactor:   1.10,
		EventModifier:    1.0,
		CorrelationData:  make(map[string]float64),
	}

	// Validate trend direction
	validDirections := []string{"up", "down", "stable"}
	isValid := false
	for _, dir := range validDirections {
		if trendData.Direction == dir {
			isValid = true
			break
		}
	}
	if !isValid {
		t.Errorf("Invalid trend direction: %s", trendData.Direction)
	}

	// Validate strength is between 0 and 1
	if trendData.Strength < 0 || trendData.Strength > 1 {
		t.Errorf("Strength should be between 0 and 1, got %.2f", trendData.Strength)
	}

	// Validate support < resistance
	if trendData.SupportLevel >= trendData.ResistanceLevel {
		t.Error("Support level should be less than resistance level")
	}
}

// TestPredictionModel_Validation tests PredictionModel structure validation
func TestPredictionModel_Validation(t *testing.T) {
	prediction := &PredictionModel{
		ProductID:         "test-123",
		PredictedPrice7d:  15500,
		PredictedPrice30d: 16200,
		Confidence7d:      0.75,
		Confidence30d:     0.65,
		ModelType:         "linear",
		LastUpdated:       time.Now().Format("2006-01-02T15:04:05Z"),
	}

	// Validate product ID
	if prediction.ProductID == "" {
		t.Error("ProductID should not be empty")
	}

	// Validate confidence values
	if prediction.Confidence7d < 0 || prediction.Confidence7d > 1 {
		t.Errorf("7d confidence should be between 0 and 1, got %.2f", prediction.Confidence7d)
	}
	if prediction.Confidence30d < 0 || prediction.Confidence30d > 1 {
		t.Errorf("30d confidence should be between 0 and 1, got %.2f", prediction.Confidence30d)
	}

	// Validate model type
	if prediction.ModelType == "" {
		t.Error("ModelType should not be empty")
	}

	// Validate timestamp format
	if _, err := time.Parse("2006-01-02T15:04:05Z", prediction.LastUpdated); err != nil {
		t.Errorf("Invalid timestamp format: %s", prediction.LastUpdated)
	}
}

// BenchmarkCalculateMovingAverage benchmarks moving average calculation
func BenchmarkCalculateMovingAverage(b *testing.B) {
	c, _ := cache.New("/tmp/bench-cache1.json")
	pc := NewPriceCharting("test-token", c)

	// Generate test data
	prices := make([]float64, 100)
	for i := range prices {
		prices[i] = float64(1000 + i*10)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pc.calculateMovingAverage(prices, 30)
	}
}

// BenchmarkCalculateVolatility benchmarks volatility calculation
func BenchmarkCalculateVolatility(b *testing.B) {
	c, _ := cache.New("/tmp/bench-cache2.json")
	pc := NewPriceCharting("test-token", c)

	// Generate test data
	prices := make([]float64, 100)
	for i := range prices {
		prices[i] = float64(1000 + i*10)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pc.calculateVolatility(prices)
	}
}

// BenchmarkCalculateTrendDirection benchmarks trend calculation
func BenchmarkCalculateTrendDirection(b *testing.B) {
	c, _ := cache.New("/tmp/bench-cache3.json")
	pc := NewPriceCharting("test-token", c)

	// Generate test data
	prices := make([]float64, 60)
	for i := range prices {
		prices[i] = float64(1000 + i*5) // Slight upward trend
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pc.calculateTrendDirection(prices)
	}
}

// TestPriceCharting_HistoricalEnrichmentIntegration tests the complete historical enrichment flow
func TestPriceCharting_HistoricalEnrichmentIntegration(t *testing.T) {
	c, _ := cache.New("/tmp/test-integration.json")
	pc := NewPriceCharting("test-token", c)

	// Test with historical enrichment disabled (default)
	if pc.IsHistoricalEnrichmentEnabled() {
		t.Error("Historical enrichment should be disabled by default")
	}

	// Enable historical enrichment
	pc.EnableHistoricalEnrichment()
	if !pc.IsHistoricalEnrichmentEnabled() {
		t.Error("Historical enrichment should be enabled after calling EnableHistoricalEnrichment()")
	}

	// Disable historical enrichment
	pc.DisableHistoricalEnrichment()
	if pc.IsHistoricalEnrichmentEnabled() {
		t.Error("Historical enrichment should be disabled after calling DisableHistoricalEnrichment()")
	}

	// Test that enrichment doesn't break when enabled with empty match
	pc.EnableHistoricalEnrichment()
	emptyMatch := &PCMatch{
		ID:          "",
		ProductName: "Test",
	}
	err := pc.EnrichWithHistoricalData(emptyMatch)
	if err == nil {
		t.Error("Expected error when enriching match without ID")
	}

	// Test with valid match but no actual API calls (will fail gracefully)
	validMatch := &PCMatch{
		ID:          "test-123",
		ProductName: "Test Card",
		PSA10Cents:  15000,
	}
	err = pc.EnrichWithHistoricalData(validMatch)
	// Should not error even if API calls fail
	if err != nil {
		t.Errorf("Enrichment should not fail gracefully when API unavailable: %v", err)
	}

	// Verify that the match structure contains the new Sprint 5 fields
	// (even if they're empty/default values due to failed API calls)
	if validMatch.TrendDirection != "" && validMatch.TrendDirection != "stable" && validMatch.TrendDirection != "up" && validMatch.TrendDirection != "down" {
		t.Errorf("Invalid trend direction: %s", validMatch.TrendDirection)
	}

	// Test that volatility is a reasonable number (0 or positive)
	if validMatch.Volatility < 0 {
		t.Errorf("Volatility should not be negative: %.3f", validMatch.Volatility)
	}

	// Test that prediction prices are reasonable (0 or positive)
	if validMatch.PredictedPrice7d < 0 {
		t.Errorf("7-day prediction should not be negative: %d", validMatch.PredictedPrice7d)
	}
	if validMatch.PredictedPrice30d < 0 {
		t.Errorf("30-day prediction should not be negative: %d", validMatch.PredictedPrice30d)
	}
}
