package prices

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/guarzo/pkmgradegap/internal/cache"
	"github.com/guarzo/pkmgradegap/internal/model"
)

// BenchmarkLookupCard benchmarks single card lookup
func BenchmarkLookupCard(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockSingleProductResponse)
	}))
	defer server.Close()

	// Setup cache
	cacheDir := b.TempDir()
	cacheFile := filepath.Join(cacheDir, "bench_cache.json")
	testCache, _ := cache.New(cacheFile)

	pc := NewPriceCharting("test-token", testCache)
	card := model.Card{Name: "Pikachu", Number: "025"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pc.LookupCard("Surging Sparks", card)
	}
}

// BenchmarkLookupCardWithCache benchmarks cached lookups
func BenchmarkLookupCardWithCache(b *testing.B) {
	// Setup cache with pre-populated data
	cacheDir := b.TempDir()
	cacheFile := filepath.Join(cacheDir, "bench_cache.json")
	testCache, _ := cache.New(cacheFile)

	// Pre-populate cache
	key := cache.PriceChartingKey("Surging Sparks", "Pikachu", "025")
	testCache.Put(key, &PCMatch{
		ID:          "12345",
		ProductName: "Pokemon Surging Sparks Pikachu #025",
		LooseCents:  850,
		PSA10Cents:  2500,
	}, 1*time.Hour)

	pc := NewPriceCharting("test-token", testCache)
	card := model.Card{Name: "Pikachu", Number: "025"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pc.LookupCard("Surging Sparks", card)
	}
}

// BenchmarkLookupBatch benchmarks batch processing
func BenchmarkLookupBatch(b *testing.B) {
	benchmarks := []struct {
		name      string
		numCards  int
		batchSize int
	}{
		{"Small-10cards-batch5", 10, 5},
		{"Small-10cards-batch10", 10, 10},
		{"Medium-50cards-batch10", 50, 10},
		{"Medium-50cards-batch20", 50, 20},
		{"Large-100cards-batch20", 100, 20},
		{"Large-200cards-batch20", 200, 20},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Simulate API delay
				time.Sleep(10 * time.Millisecond)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(mockSingleProductResponse)
			}))
			defer server.Close()

			// Generate cards
			cards := make([]model.Card, bm.numCards)
			for i := 0; i < bm.numCards; i++ {
				cards[i] = model.Card{
					Name:   fmt.Sprintf("Card%d", i),
					Number: fmt.Sprintf("%03d", i),
				}
			}

			// Setup cache
			cacheDir := b.TempDir()
			cacheFile := filepath.Join(cacheDir, "bench_cache.json")
			testCache, _ := cache.New(cacheFile)

			pc := NewPriceCharting("test-token", testCache)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = pc.LookupBatch("Test Set", cards, bm.batchSize)
			}
		})
	}
}

// BenchmarkCacheHitRate benchmarks cache effectiveness
func BenchmarkCacheHitRate(b *testing.B) {
	benchmarks := []struct {
		name       string
		cacheRatio float64 // Percentage of cards to pre-cache
	}{
		{"NoCache-0%", 0.0},
		{"LowCache-25%", 0.25},
		{"MediumCache-50%", 0.50},
		{"HighCache-75%", 0.75},
		{"FullCache-100%", 1.0},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(5 * time.Millisecond) // Simulate API delay
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(mockSingleProductResponse)
			}))
			defer server.Close()

			// Generate 100 cards
			numCards := 100
			cards := make([]model.Card, numCards)
			for i := 0; i < numCards; i++ {
				cards[i] = model.Card{
					Name:   fmt.Sprintf("Card%d", i),
					Number: fmt.Sprintf("%03d", i),
				}
			}

			// Setup cache with pre-populated data
			cacheDir := b.TempDir()
			cacheFile := filepath.Join(cacheDir, "bench_cache.json")
			testCache, _ := cache.New(cacheFile)

			// Pre-cache specified percentage
			numToCahce := int(float64(numCards) * bm.cacheRatio)
			setName := "Test Set"
			for i := 0; i < numToCahce; i++ {
				card := cards[i]
				key := cache.PriceChartingKey(setName, card.Name, card.Number)
				testCache.Put(key, &PCMatch{
					ID:          fmt.Sprintf("cached-%d", i),
					ProductName: fmt.Sprintf("%s #%s", card.Name, card.Number),
					LooseCents:  100,
					PSA10Cents:  1000,
				}, 1*time.Hour)
			}

			pc := NewPriceCharting("test-token", testCache)

			b.ResetTimer()
			b.StartTimer()
			results, _ := pc.LookupBatch(setName, cards, 20)
			b.StopTimer()

			// Report cache hit statistics
			cachedCount := 0
			for _, res := range results {
				if res.Cached {
					cachedCount++
				}
			}

			stats := pc.GetStats()
			b.ReportMetric(float64(cachedCount), "cached_hits")
			b.ReportMetric(float64(numCards-cachedCount), "api_calls")
			if apiReq, ok := stats["api_requests"].(int64); ok {
				b.ReportMetric(float64(apiReq), "actual_api_requests")
			}
			if cacheRate, ok := stats["cache_hit_rate"].(string); ok {
				b.Logf("Cache hit rate: %s", cacheRate)
			}
		})
	}
}

// BenchmarkQueryOptimization benchmarks query optimization
func BenchmarkQueryOptimization(b *testing.B) {
	pc := NewPriceCharting("test-token", nil)

	testCases := []struct {
		setName  string
		cardName string
		number   string
	}{
		{"Surging Sparks", "Pikachu", "025"},
		{"Surging Sparks", "Pikachu ex", "025"},
		{"Surging Sparks", "Charizard VMAX", "006"},
		{"Sword & Shield: Base Set", "Zacian V", "138"},
		{"Pokemon GO", "Mewtwo VSTAR", "079"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			_ = pc.OptimizeQuery(tc.setName, tc.cardName, tc.number)
		}
	}
}

// BenchmarkMultiLayerCache benchmarks multi-layer cache performance
func BenchmarkMultiLayerCache(b *testing.B) {
	benchmarks := []struct {
		name     string
		useMulti bool
		l1Size   int
		l2Size   int64
	}{
		{"SingleLayer", false, 0, 0},
		{"MultiLayer-Small", true, 100, 10 * 1024 * 1024},
		{"MultiLayer-Medium", true, 500, 50 * 1024 * 1024},
		{"MultiLayer-Large", true, 2000, 100 * 1024 * 1024},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			// Setup cache
			cacheDir := b.TempDir()
			cacheFile := filepath.Join(cacheDir, "bench_cache.json")
			testCache, _ := cache.New(cacheFile)

			pc := NewPriceCharting("test-token", testCache)

			// Enable multi-layer cache if specified
			if bm.useMulti {
				config := cache.CacheConfig{
					L1MaxSize:     bm.l1Size,
					L1TTL:         30 * time.Minute,
					L2MaxSize:     bm.l2Size,
					L2TTL:         24 * time.Hour,
					L2Path:        filepath.Join(cacheDir, "multilayer"),
					EnablePredict: true,
					CompressL2:    true,
				}
				_ = pc.EnableMultiLayerCache(config)
			}

			// Generate test data
			numCards := 100
			cards := make([]model.Card, numCards)
			for i := 0; i < numCards; i++ {
				cards[i] = model.Card{
					Name:   fmt.Sprintf("Card%d", i),
					Number: fmt.Sprintf("%03d", i),
				}
			}

			// Pre-warm cache with half the cards
			setName := "Test Set"
			for i := 0; i < numCards/2; i++ {
				card := cards[i]
				key := cache.PriceChartingKey(setName, card.Name, card.Number)
				match := &PCMatch{
					ID:          fmt.Sprintf("card-%d", i),
					ProductName: fmt.Sprintf("%s #%s", card.Name, card.Number),
					LooseCents:  100 + i*10,
					PSA10Cents:  1000 + i*100,
				}
				testCache.Put(key, match, 1*time.Hour)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Simulate mixed read patterns
				idx := i % numCards
				card := cards[idx]
				_, _ = pc.LookupCard(setName, card)
			}
		})
	}
}

// BenchmarkDeduplication benchmarks query deduplication
func BenchmarkDeduplication(b *testing.B) {
	pc := NewPriceCharting("test-token", nil)

	// Generate queries with duplicates
	queries := []string{
		"pokemon Surging Sparks Pikachu #025",
		"pokemon Surging Sparks Charizard #006",
		"pokemon Surging Sparks Pikachu #025", // Duplicate
		"pokemon Surging Sparks Blastoise #009",
		"pokemon Surging Sparks Charizard #006", // Duplicate
		"pokemon Surging Sparks Venusaur #003",
		"pokemon Surging Sparks Pikachu #025", // Duplicate
	}

	// Pre-populate deduplicator
	for i, q := range queries[:3] {
		pc.queryDedup.Store(q, &PCMatch{
			ID:          fmt.Sprintf("dedup-%d", i),
			ProductName: q,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, q := range queries {
			_ = pc.queryDedup.GetCached(q)
		}
	}
}
