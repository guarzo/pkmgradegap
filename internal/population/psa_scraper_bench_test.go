package population

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// BenchmarkStringCache benchmarks the string cache implementation
func BenchmarkStringCache(b *testing.B) {
	cache := newStringCache(1000)

	// Test data
	keys := make([]string, 100)
	values := make([]string, 100)
	for i := 0; i < 100; i++ {
		keys[i] = normalizeString(fmt.Sprintf("set_%d_card_%d_number_%d", i%10, i%20, i))
		values[i] = fmt.Sprintf("https://psacard.com/pop/pokemon/spec/%d", i)
	}

	b.ResetTimer()

	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := keys[i%len(keys)]
			value := values[i%len(values)]
			cache.set(key, value, time.Hour)
		}
	})

	// Pre-populate cache for get benchmarks
	for i := 0; i < len(keys); i++ {
		cache.set(keys[i], values[i], time.Hour)
	}

	b.Run("Get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := keys[i%len(keys)]
			_, _ = cache.get(key)
		}
	})

	b.Run("Mixed", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := keys[i%len(keys)]
			if i%4 == 0 {
				// 25% writes
				value := values[i%len(values)]
				cache.set(key, value, time.Hour)
			} else {
				// 75% reads
				_, _ = cache.get(key)
			}
		}
	})
}

// BenchmarkSearchCardCaching benchmarks search card performance with caching
func BenchmarkSearchCardCaching(b *testing.B) {
	// Create a mock cache that always returns data to avoid HTTP calls
	mockCache := &MockPopulationCache{}
	scraper := NewPSAScraper(mockCache)
	scraper.SetDebug(false) // Disable debug logging for benchmarks

	ctx := context.Background()

	// Test data representing different cards
	testCards := []struct {
		set    string
		number string
		name   string
	}{
		{"Base Set", "001", "Bulbasaur"},
		{"Base Set", "004", "Charmander"},
		{"Base Set", "007", "Squirtle"},
		{"Jungle", "001", "Clefable"},
		{"Fossil", "001", "Aerodactyl"},
		{"Team Rocket", "001", "Dark Alakazam"},
		{"Gym Heroes", "001", "Brock's Rhydon"},
		{"Gym Challenge", "001", "Koga's Arbok"},
		{"Base Set 2", "001", "Alakazam"},
		{"Neo Genesis", "001", "Ampharos"},
	}

	b.ResetTimer()

	b.Run("ColdCache", func(b *testing.B) {
		// Reset cache for each run
		scraper.searchCache = newStringCache(maxCacheSize)

		for i := 0; i < b.N; i++ {
			card := testCards[i%len(testCards)]
			// This will miss cache and make HTTP call
			_, _ = scraper.SearchCard(ctx, card.set, card.number, card.name)
		}
	})

	b.Run("WarmCache", func(b *testing.B) {
		// Pre-warm cache
		for _, card := range testCards {
			cacheKey := fmt.Sprintf("search_%s_%s_%s",
				normalizeString(card.set),
				normalizeString(card.number),
				normalizeString(card.name))
			scraper.searchCache.set(cacheKey, "https://psacard.com/pop/cached", time.Hour)
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			card := testCards[i%len(testCards)]
			// This should hit cache and be fast
			_, _ = scraper.SearchCard(ctx, card.set, card.number, card.name)
		}
	})
}

// MockPopulationCache is a simple mock that implements the Cache interface
type MockPopulationCache struct{}

func (m *MockPopulationCache) Get(key string) (*PopulationData, bool) {
	return nil, false
}

func (m *MockPopulationCache) Set(key string, data *PopulationData, ttl time.Duration) error {
	return nil
}

func (m *MockPopulationCache) GetSet(key string) (*SetPopulationData, bool) {
	return nil, false
}

func (m *MockPopulationCache) SetSet(key string, data *SetPopulationData, ttl time.Duration) error {
	return nil
}

func (m *MockPopulationCache) Clear() error {
	return nil
}
