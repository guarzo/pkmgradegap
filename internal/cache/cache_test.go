package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestCache_PutGetWithTTL(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_cache.json")

	cache, err := New(cachePath)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Test basic put/get operations
	testData := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": map[string]string{"nested": "data"},
	}

	// Put values with 1 hour TTL
	ttl := time.Hour
	for key, value := range testData {
		if err := cache.Put(key, value, ttl); err != nil {
			t.Errorf("Failed to put %s: %v", key, err)
		}
	}

	// Get values back
	var result string
	found, err := cache.Get("key1", &result)
	if err != nil {
		t.Errorf("Failed to get key1: %v", err)
	}
	if !found {
		t.Error("Expected to find key1")
	}
	if result != "value1" {
		t.Errorf("Expected 'value1', got '%s'", result)
	}

	var intResult int
	found, err = cache.Get("key2", &intResult)
	if err != nil {
		t.Errorf("Failed to get key2: %v", err)
	}
	if !found {
		t.Error("Expected to find key2")
	}
	if intResult != 42 {
		t.Errorf("Expected 42, got %d", intResult)
	}

	var mapResult map[string]string
	found, err = cache.Get("key3", &mapResult)
	if err != nil {
		t.Errorf("Failed to get key3: %v", err)
	}
	if !found {
		t.Error("Expected to find key3")
	}
	if mapResult["nested"] != "data" {
		t.Errorf("Expected 'data', got '%s'", mapResult["nested"])
	}
}

func TestCache_TTLExpiration(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_ttl_cache.json")

	cache, err := New(cachePath)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Put value with very short TTL
	shortTTL := 50 * time.Millisecond
	if err := cache.Put("short_ttl", "will_expire", shortTTL); err != nil {
		t.Fatalf("Failed to put short TTL value: %v", err)
	}

	// Put value with no TTL (permanent)
	if err := cache.Put("no_ttl", "permanent", 0); err != nil {
		t.Fatalf("Failed to put permanent value: %v", err)
	}

	// Immediately check - should be available
	var result string
	found, err := cache.Get("short_ttl", &result)
	if err != nil {
		t.Errorf("Failed to get short_ttl: %v", err)
	}
	if !found {
		t.Error("Expected to find short_ttl immediately")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Check expired value - should not be found
	found, err = cache.Get("short_ttl", &result)
	if err != nil {
		t.Errorf("Failed to get expired short_ttl: %v", err)
	}
	if found {
		t.Error("Expected short_ttl to be expired")
	}

	// Check permanent value - should still be available
	found, err = cache.Get("no_ttl", &result)
	if err != nil {
		t.Errorf("Failed to get no_ttl: %v", err)
	}
	if !found {
		t.Error("Expected to find permanent value")
	}
	if result != "permanent" {
		t.Errorf("Expected 'permanent', got '%s'", result)
	}
}

func TestCache_Persistence(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_persistence_cache.json")

	// Create cache and add data
	cache1, err := New(cachePath)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	testValue := "persistent_data"
	if err := cache1.Put("persist_key", testValue, time.Hour); err != nil {
		t.Fatalf("Failed to put persistent value: %v", err)
	}

	// Create new cache instance pointing to same file
	cache2, err := New(cachePath)
	if err != nil {
		t.Fatalf("Failed to create second cache: %v", err)
	}

	// Should load existing data
	var result string
	found, err := cache2.Get("persist_key", &result)
	if err != nil {
		t.Errorf("Failed to get persistent value: %v", err)
	}
	if !found {
		t.Error("Expected to find persistent value")
	}
	if result != testValue {
		t.Errorf("Expected '%s', got '%s'", testValue, result)
	}
}

func TestCache_SemanticKeys(t *testing.T) {
	// Test SetsKey
	key := SetsKey()
	expected := "sets:v2"
	if key != expected {
		t.Errorf("Expected SetsKey '%s', got '%s'", expected, key)
	}

	// Test CardsKey
	setID := "base1"
	key = CardsKey(setID)
	expected = "cards|set|base1"
	if key != expected {
		t.Errorf("Expected CardsKey '%s', got '%s'", expected, key)
	}

	// Test PriceChartingKey
	setName := "Base Set"
	cardName := "Charizard"
	number := "4"
	key = PriceChartingKey(setName, cardName, number)
	expected = "pc|Base Set|Charizard|4"
	if key != expected {
		t.Errorf("Expected PriceChartingKey '%s', got '%s'", expected, key)
	}

	// Test BuildKey with various inputs
	tests := []struct {
		parts    []string
		expected string
	}{
		{[]string{"a"}, "a"},
		{[]string{"a", "b"}, "a|b"},
		{[]string{"a", "b", "c"}, "a|b|c"},
		{[]string{}, ""},
	}

	for _, test := range tests {
		result := BuildKey(test.parts...)
		if result != test.expected {
			t.Errorf("BuildKey(%v) = '%s', expected '%s'", test.parts, result, test.expected)
		}
	}
}

func TestCache_Concurrent(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_concurrent_cache.json")

	cache, err := New(cachePath)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Test concurrent access safety
	const numGoroutines = 10
	const numOperations = 100
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("concurrent_%d_%d", id, j)
				value := fmt.Sprintf("value_%d_%d", id, j)
				if err := cache.Put(key, value, time.Hour); err != nil {
					t.Errorf("Concurrent put failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all values were written
	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < numOperations; j++ {
			key := fmt.Sprintf("concurrent_%d_%d", i, j)
			expected := fmt.Sprintf("value_%d_%d", i, j)

			var result string
			found, err := cache.Get(key, &result)
			if err != nil {
				t.Errorf("Failed to get concurrent value %s: %v", key, err)
			}
			if !found {
				t.Errorf("Expected to find concurrent value %s", key)
			}
			if result != expected {
				t.Errorf("Expected '%s', got '%s'", expected, result)
			}
		}
	}
}

func TestCache_ClearAndRemove(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_clear_cache.json")

	cache, err := New(cachePath)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Add test data
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for key, value := range testData {
		if err := cache.Put(key, value, time.Hour); err != nil {
			t.Errorf("Failed to put %s: %v", key, err)
		}
	}

	// Test Remove
	if err := cache.Remove("key2"); err != nil {
		t.Errorf("Failed to remove key2: %v", err)
	}

	var result string
	found, err := cache.Get("key2", &result)
	if err != nil {
		t.Errorf("Failed to get removed key: %v", err)
	}
	if found {
		t.Error("Expected key2 to be removed")
	}

	// Other keys should still exist
	found, err = cache.Get("key1", &result)
	if err != nil {
		t.Errorf("Failed to get key1: %v", err)
	}
	if !found {
		t.Error("Expected key1 to still exist")
	}

	// Test Clear
	if err := cache.Clear(); err != nil {
		t.Errorf("Failed to clear cache: %v", err)
	}

	// No keys should exist
	for key := range testData {
		found, err := cache.Get(key, &result)
		if err != nil {
			t.Errorf("Failed to get cleared key %s: %v", key, err)
		}
		if found {
			t.Errorf("Expected key %s to be cleared", key)
		}
	}
}

func TestCache_CorruptedFile(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_corrupted_cache.json")

	// Create corrupted cache file
	corruptedData := []byte("{invalid json}")
	if err := os.WriteFile(cachePath, corruptedData, 0644); err != nil {
		t.Fatalf("Failed to create corrupted file: %v", err)
	}

	// Should handle corrupted file gracefully
	cache, err := New(cachePath)
	if err != nil {
		t.Fatalf("Failed to create cache with corrupted file: %v", err)
	}

	// Should be able to add new data
	if err := cache.Put("test", "value", time.Hour); err != nil {
		t.Errorf("Failed to put value after corruption: %v", err)
	}

	var result string
	found, err := cache.Get("test", &result)
	if err != nil {
		t.Errorf("Failed to get value after corruption: %v", err)
	}
	if !found {
		t.Error("Expected to find value after corruption recovery")
	}
	if result != "value" {
		t.Errorf("Expected 'value', got '%s'", result)
	}
}

func TestCache_NonExistentKey(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_nonexistent_cache.json")

	cache, err := New(cachePath)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Try to get non-existent key
	var result string
	found, err := cache.Get("nonexistent", &result)
	if err != nil {
		t.Errorf("Unexpected error for non-existent key: %v", err)
	}
	if found {
		t.Error("Expected not to find non-existent key")
	}
}
