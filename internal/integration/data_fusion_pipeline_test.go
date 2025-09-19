package integration

import (
	"context"
	"testing"
	"time"

	"github.com/guarzo/pkmgradegap/internal/analysis"
	"github.com/guarzo/pkmgradegap/internal/cache"
	"github.com/guarzo/pkmgradegap/internal/concurrent"
	// "github.com/guarzo/pkmgradegap/internal/fusion" // TODO: Update when fusion package is refactored
	"github.com/guarzo/pkmgradegap/internal/model"
	"github.com/guarzo/pkmgradegap/internal/pipeline"
	"github.com/guarzo/pkmgradegap/internal/population"
)

// TestDataFusionIntegration tests the complete data fusion workflow
// TODO: Update when fusion package is refactored
func TestDataFusionIntegration(t *testing.T) {
	// Create fusion engine
	// engine := fusion.NewFusionEngine()
	t.Skip("Skipping fusion test until fusion package is refactored")

	// Create test price data from multiple sources
	// prices := map[string][]fusion.PriceData{
	// 	"raw": {
	// 		{
	// 			Value:    45.0,
	// 			Currency: "USD",
	// 			Source: fusion.DataSource{
	// 				Name:       "TCGPlayer",
	// 				Type:       fusion.SourceTypeListing,
	// 				Freshness:  2 * time.Hour,
	// 				Volume:     25,
	// 				Confidence: 0.8,
	// 				Timestamp:  time.Now().Add(-2 * time.Hour),
	// 			},
	// 		},
	// 		{
	// 			Value:    43.0,
	// 			Currency: "USD",
	// 			Source: fusion.DataSource{
	// 				Name:       "eBay_Sales",
	// 				Type:       fusion.SourceTypeSale,
	// 				Freshness:  6 * time.Hour,
	// 				Volume:     12,
	// 				Confidence: 0.9,
	// 				Timestamp:  time.Now().Add(-6 * time.Hour),
	// 			},
	// 		},
	// 	},
	// 	"psa10": {
	// 		{
	// 			Value:    180.0,
	// 			Currency: "USD",
	// 			Source: fusion.DataSource{
	// 				Name:       "PriceCharting",
	// 				Type:       fusion.SourceTypeSale,
	// 				Freshness:  1 * time.Hour,
	// 				Volume:     8,
	// 				Confidence: 0.85,
	// 				Timestamp:  time.Now().Add(-1 * time.Hour),
	// 			},
	// 		},
	// 	},
	// }

	// Create test card
	// card := model.Card{
	// 	Name:    "Charizard ex",
	// 	SetName: "Surging Sparks",
	// 	Number:  "223",
	// 	Rarity:  "Special Illustration Rare",
	// }

	// Create test population data
	// popData := &model.PSAPopulation{
	// 	TotalGraded: 500,
	// 	PSA10:       23,
	// 	PSA9:        45,
	// 	PSA8:        67,
	// 	LastUpdated: time.Now(),
	// }

	// Create test sales data
	// salesData := []fusion.SaleData{
	// 	{
	// 		Price:     175.0,
	// 		Date:      time.Now().Add(-24 * time.Hour),
	// 		Platform:  "eBay",
	// 		Condition: "PSA 10",
	// 	},
	// 	{
	// 		Price:     42.0,
	// 		Date:      time.Now().Add(-12 * time.Hour),
	// 		Platform:  "TCGPlayer",
	// 		Condition: "NM",
	// 	},
	// }

	// Test data fusion
	// fusedData := engine.FuseCardData(card, prices, popData, salesData)

	// Verify fusion results
	// if fusedData.Card.Name != card.Name {
	// 	t.Errorf("Expected card name %s, got %s", card.Name, fusedData.Card.Name)
	// }
	//
	// if fusedData.RawPrice.Value == 0 {
	// 	t.Error("Raw price fusion failed")
	// }
	//
	// if fusedData.PSA10Price.Value == 0 {
	// 	t.Error("PSA10 price fusion failed")
	// }
	//
	// if fusedData.Confidence.Overall == 0 {
	// 	t.Error("Confidence calculation failed")
	// }
	//
	// // Check confidence factors
	// if len(fusedData.Confidence.Factors) == 0 {
	// 	t.Error("No confidence factors calculated")
	// }
	//
	// t.Logf("Fusion Results:")
	// t.Logf("  Raw Price: $%.2f (confidence: %.2f)", fusedData.RawPrice.Value, fusedData.RawPrice.Confidence)
	// t.Logf("  PSA10 Price: $%.2f (confidence: %.2f)", fusedData.PSA10Price.Value, fusedData.PSA10Price.Confidence)
	// t.Logf("  Overall Confidence: %.2f", fusedData.Confidence.Overall)
	// t.Logf("  Warnings: %v", fusedData.Confidence.Warnings)
}

// TestPopulationTargetingSystem tests the population targeting system
func TestPopulationTargetingSystem(t *testing.T) {
	// Create targeting engine
	config := population.TargetingConfig{
		MinRawValue:      1.0,
		MinPredictedROI:  0.2,
		RarityFilter:     []string{"Secret Rare", "Ultra Rare", "Special Illustration Rare"},
		EnableHeuristics: true,
	}
	targeting := population.NewTargetingEngine(config)

	// Test cards
	testCards := []model.Card{
		{
			Name:    "Charizard ex",
			SetName: "Surging Sparks",
			Number:  "223",
			Rarity:  "Special Illustration Rare",
		},
		{
			Name:    "Fire Energy",
			SetName: "Surging Sparks",
			Number:  "230",
			Rarity:  "Common",
		},
		{
			Name:    "Pikachu ex",
			SetName: "Surging Sparks",
			Number:  "004",
			Rarity:  "Ultra Rare",
		},
		{
			Name:    "Professor's Research",
			SetName: "Surging Sparks",
			Number:  "225",
			Rarity:  "Uncommon",
		},
	}

	// Test targeting decisions
	var targetedCards []model.Card
	for _, card := range testCards {
		if targeting.ShouldFetchPopulation(card) {
			targetedCards = append(targetedCards, card)
			t.Logf("Targeting: %s (%s) - %s", card.Name, card.Number, card.Rarity)
		} else {
			t.Logf("Skipping: %s (%s) - %s", card.Name, card.Number, card.Rarity)
		}
	}

	// Verify targeting results
	expectedTargeted := 2 // Charizard ex and Pikachu ex should be targeted
	if len(targetedCards) != expectedTargeted {
		t.Errorf("Expected %d targeted cards, got %d", expectedTargeted, len(targetedCards))
	}

	// Test batch targeting
	batchTargeting := population.NewBatchTargeting(targeting, 10)
	worthFetching := batchTargeting.ProcessCards(testCards)

	if len(worthFetching) != len(targetedCards) {
		t.Errorf("Batch targeting results don't match individual results")
	}

	// Test targeting report
	report := batchTargeting.GetTargetingReport(testCards)
	if report.TotalCards != len(testCards) {
		t.Errorf("Expected total cards %d, got %d", len(testCards), report.TotalCards)
	}

	if report.TargetedCards != len(targetedCards) {
		t.Errorf("Expected targeted cards %d, got %d", len(targetedCards), report.TargetedCards)
	}

	t.Logf("Targeting Report:")
	t.Logf("  Total Cards: %d", report.TotalCards)
	t.Logf("  Targeted: %d", report.TargetedCards)
	t.Logf("  Skipped: %d", report.SkippedCards)
	t.Logf("  Targeting Rate: %.2f%%", report.TargetingRate*100)
	t.Logf("  By Reason: %v", report.ByReason)
}

// TestConcurrentFetchingSystem tests the concurrent fetching system
func TestConcurrentFetchingSystem(t *testing.T) {
	// Create test data fetcher
	testFetcher := &TestDataFetcher{}

	// Create concurrent fetcher
	config := concurrent.FetcherConfig{
		Workers:      3,
		RateLimit:    10, // 10 requests per second
		Timeout:      5 * time.Second,
		ErrorHandler: concurrent.DefaultErrorHandler,
	}
	fetcher := concurrent.NewConcurrentFetcher(config)

	// Test cards
	testCards := []model.Card{
		{Name: "Card 1", SetName: "Test Set", Number: "001"},
		{Name: "Card 2", SetName: "Test Set", Number: "002"},
		{Name: "Card 3", SetName: "Test Set", Number: "003"},
		{Name: "Card 4", SetName: "Test Set", Number: "004"},
		{Name: "Card 5", SetName: "Test Set", Number: "005"},
	}

	ctx := context.Background()

	// Test concurrent fetching
	start := time.Now()
	results := fetcher.FetchAll(ctx, testCards, testFetcher)
	duration := time.Since(start)

	// Verify results
	if len(results) != len(testCards) {
		t.Errorf("Expected %d results, got %d", len(testCards), len(results))
	}

	successCount := 0
	for _, result := range results {
		if result.Error == nil {
			successCount++
		} else {
			t.Logf("Error for %s: %v", result.Card.Name, result.Error)
		}
	}

	t.Logf("Concurrent Fetch Results:")
	t.Logf("  Total Cards: %d", len(testCards))
	t.Logf("  Successful: %d", successCount)
	t.Logf("  Duration: %v", duration)

	// Get metrics
	metrics := fetcher.GetMetrics()
	t.Logf("  Metrics:")
	t.Logf("    Total Requests: %d", metrics.TotalRequests)
	t.Logf("    Successful: %d", metrics.SuccessfulReqs)
	t.Logf("    Failed: %d", metrics.FailedRequests)
	if metrics.SuccessfulReqs > 0 {
		t.Logf("    Average Latency: %v", metrics.AverageLatency)
	}

	// Verify concurrent execution was faster than sequential
	expectedSequentialTime := time.Duration(len(testCards)) * 500 * time.Millisecond // 500ms per card
	if duration >= expectedSequentialTime {
		t.Logf("Warning: Concurrent execution (%v) not faster than expected sequential (%v)", duration, expectedSequentialTime)
	}
}

// TestMultiLayerCacheSystem tests the multi-layer cache system
func TestMultiLayerCacheSystem(t *testing.T) {
	// Create cache configuration
	config := cache.CacheConfig{
		L1MaxSize:     100,
		L1TTL:         1 * time.Hour,
		L2MaxSize:     1024 * 1024, // 1MB
		L2TTL:         24 * time.Hour,
		L2Path:        "./test_cache",
		EnablePredict: true,
		CompressL2:    true,
	}

	// Create multi-layer cache
	mlCache, err := cache.NewMultiLayerCache(config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer mlCache.Clear() // Cleanup

	// Test cache operations
	testKey := "test_card_data"
	testData := map[string]interface{}{
		"name":   "Charizard ex",
		"set":    "Surging Sparks",
		"number": "223",
		"price":  45.0,
		"grades": []int{10, 9, 8, 7},
	}

	// Test set
	err = mlCache.Set(testKey, testData, 1*time.Hour)
	if err != nil {
		t.Errorf("Cache set failed: %v", err)
	}

	// Test get (should hit L1)
	retrieved, found := mlCache.Get(testKey)
	if !found {
		t.Error("Cache get failed - item not found")
	}

	if retrieved == nil {
		t.Error("Cache get failed - nil data returned")
	}

	// Verify data integrity
	if retrievedMap, ok := retrieved.(map[string]interface{}); ok {
		if retrievedMap["name"] != testData["name"] {
			t.Error("Cache data corruption detected")
		}
	}

	// Test cache stats
	stats := mlCache.GetStats()
	t.Logf("Cache Stats:")
	t.Logf("  L1 Hits: %d", stats.L1Hits)
	t.Logf("  L1 Misses: %d", stats.L1Misses)
	t.Logf("  L1 Hit Rate: %.2f%%", stats.L1HitRate*100)
	t.Logf("  Overall Hit Rate: %.2f%%", stats.OverallHitRate*100)

	// Test cache prediction (simulate some access patterns)
	for i := 0; i < 5; i++ {
		mlCache.Get(testKey) // Simulate repeated access
	}

	predictions := mlCache.GetPredictedTargets()
	t.Logf("Cache Predictions: %d targets", len(predictions))
	for _, pred := range predictions {
		t.Logf("  Predicted: %s (probability: %.2f)", pred.Key, pred.Probability)
	}

	// Test cache optimization
	err = mlCache.Optimize()
	if err != nil {
		t.Errorf("Cache optimization failed: %v", err)
	}
}

// TestPipelineProcessingSystem tests the pipeline processing system
func TestPipelineProcessingSystem(t *testing.T) {
	// Create test stages
	cardStage := pipeline.NewCardFetchStage(&TestCardProvider{})
	priceStage := pipeline.NewPriceFetchStage(&TestPriceProvider{})
	// fusionStage := pipeline.NewDataFusionStage(fusion.NewFusionEngine()) // TODO: Update when fusion package is refactored
	analysisStage := pipeline.NewAnalysisStage(&TestAnalysisEngine{})

	// Create pipeline
	pipelineConfig := pipeline.PipelineConfig{
		BufferSize: 10,
		Stages: []pipeline.Stage{
			cardStage,
			priceStage,
			// fusionStage, // TODO: Update when fusion package is refactored
			analysisStage,
		},
	}
	proc := pipeline.NewPipeline(pipelineConfig)

	// Test cards
	testCards := []model.Card{
		{Name: "Test Card 1", SetName: "Test Set", Number: "001"},
		{Name: "Test Card 2", SetName: "Test Set", Number: "002"},
		{Name: "Test Card 3", SetName: "Test Set", Number: "003"},
	}

	ctx := context.Background()

	// Process cards through pipeline
	start := time.Now()
	outputChannel := proc.Process(ctx, testCards)

	// Collect results
	var results []interface{}
	for result := range outputChannel {
		results = append(results, result)
	}
	duration := time.Since(start)

	// Verify results
	if len(results) != len(testCards) {
		t.Errorf("Expected %d results, got %d", len(testCards), len(results))
	}

	t.Logf("Pipeline Results:")
	t.Logf("  Processed Cards: %d", len(results))
	t.Logf("  Duration: %v", duration)

	// Get pipeline metrics
	metrics := proc.GetMetrics()
	t.Logf("  Pipeline Metrics:")
	t.Logf("    Total Items: %d", metrics.TotalItems)
	t.Logf("    Processed: %d", metrics.ProcessedItems)
	t.Logf("    Errors: %d", metrics.ErrorCount)
	t.Logf("    Throughput: %.2f items/sec", metrics.Throughput)

	for stageName, stageMetrics := range metrics.StageMetrics {
		t.Logf("    Stage %s:", stageName)
		t.Logf("      Processed: %d", stageMetrics.ItemsProcessed)
		t.Logf("      Errors: %d", stageMetrics.ItemsErrored)
		if stageMetrics.AverageLatency > 0 {
			t.Logf("      Avg Latency: %v", stageMetrics.AverageLatency)
		}
	}
}

// Test helper types and implementations

type TestDataFetcher struct{}

func (t *TestDataFetcher) Fetch(ctx context.Context, card model.Card) (interface{}, error) {
	// Simulate some processing time
	time.Sleep(100 * time.Millisecond)
	return map[string]string{
		"name":   card.Name,
		"set":    card.SetName,
		"number": card.Number,
		"data":   "test_data",
	}, nil
}

func (t *TestDataFetcher) Type() string {
	return "test"
}

type TestCardProvider struct{}

func (t *TestCardProvider) GetCard(ctx context.Context, card model.Card) (interface{}, error) {
	return map[string]interface{}{
		"card": card,
		"type": "test_card_data",
	}, nil
}

type TestPriceProvider struct{}

func (t *TestPriceProvider) GetPrice(ctx context.Context, card model.Card) (interface{}, error) {
	return map[string]interface{}{
		"raw_price":   45.0,
		"psa10_price": 180.0,
		"currency":    "USD",
	}, nil
}

type TestAnalysisEngine struct{}

// TODO: Update when fusion package is refactored
func (t *TestAnalysisEngine) Analyze(ctx context.Context, data interface{}) (*analysis.Row, error) {
	// Return a proper analysis.Row
	return &analysis.Row{
		Card:   model.Card{Name: "Test Card"},
		RawUSD: 10.0,
		Grades: analysis.Grades{
			PSA10: 100.0,
		},
	}, nil
}
