package cache

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
)

// MultiLayerCache implements a three-tier caching system
type MultiLayerCache struct {
	l1 *MemoryCache // Hot data - fast access
	l2 *DiskCache   // Warm data - persistent
	l3 *RemoteCache // Cold data - network storage (optional)

	config    CacheConfig
	stats     *CacheStats
	predictor *CachePredictor
	mu        sync.RWMutex
}

// CacheConfig holds configuration for the multi-layer cache
type CacheConfig struct {
	L1MaxSize     int           // Max items in memory cache
	L1TTL         time.Duration // TTL for L1 cache
	L2MaxSize     int64         // Max size in bytes for disk cache
	L2TTL         time.Duration // TTL for L2 cache
	L3TTL         time.Duration // TTL for L3 cache
	L2Path        string        // Path for disk cache
	L3Config      *RemoteConfig // Remote cache configuration
	EnablePredict bool          // Enable predictive caching
	CompressL2    bool          // Compress L2 cache entries
}

// RemoteConfig holds remote cache configuration
type RemoteConfig struct {
	Provider string // "s3", "redis", etc.
	Endpoint string
	Bucket   string
	Prefix   string
}

// CacheStats tracks cache performance metrics
type CacheStats struct {
	L1Hits     int64
	L1Misses   int64
	L2Hits     int64
	L2Misses   int64
	L3Hits     int64
	L3Misses   int64
	Evictions  int64
	Prefetches int64
	StartTime  time.Time
	mu         sync.RWMutex
}

// CacheEntry represents a cached item
type CacheEntry struct {
	Key        string        `json:"key"`
	Data       interface{}   `json:"data"`
	CreatedAt  time.Time     `json:"created_at"`
	AccessedAt time.Time     `json:"accessed_at"`
	TTL        time.Duration `json:"ttl"`
	Size       int64         `json:"size"`
	Layer      int           `json:"layer"` // 1, 2, or 3
	Compressed bool          `json:"compressed"`
}

// NewMultiLayerCache creates a new multi-layer cache
func NewMultiLayerCache(config CacheConfig) (*MultiLayerCache, error) {
	// Set defaults
	if config.L1MaxSize == 0 {
		config.L1MaxSize = 1000
	}
	if config.L1TTL == 0 {
		config.L1TTL = 1 * time.Hour
	}
	if config.L2TTL == 0 {
		config.L2TTL = 24 * time.Hour
	}
	if config.L3TTL == 0 {
		config.L3TTL = 7 * 24 * time.Hour
	}
	if config.L2Path == "" {
		config.L2Path = "./cache"
	}

	cache := &MultiLayerCache{
		config: config,
		stats: &CacheStats{
			StartTime: time.Now(),
		},
	}

	// Initialize L1 cache (memory)
	cache.l1 = NewMemoryCache(config.L1MaxSize, config.L1TTL)

	// Initialize L2 cache (disk)
	var err error
	cache.l2, err = NewDiskCache(config.L2Path, config.L2MaxSize, config.L2TTL, config.CompressL2)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize disk cache: %w", err)
	}

	// Initialize L3 cache (remote) if configured
	if config.L3Config != nil {
		cache.l3, err = NewRemoteCache(*config.L3Config, config.L3TTL)
		if err != nil {
			// Log warning but don't fail - L3 is optional
			fmt.Printf("Warning: failed to initialize remote cache: %v\n", err)
		}
	}

	// Initialize predictor if enabled
	if config.EnablePredict {
		cache.predictor = NewCachePredictor()
	}

	return cache, nil
}

// Get retrieves an item from the cache, checking L1 -> L2 -> L3
func (c *MultiLayerCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Track access pattern for prediction
	if c.predictor != nil {
		c.predictor.RecordAccess(key)
	}

	// Try L1 first (memory cache)
	if data, found := c.l1.Get(key); found {
		c.recordHit(1)
		return data, true
	}
	c.recordMiss(1)

	// Try L2 (disk cache)
	if data, found := c.l2.Get(key); found {
		c.recordHit(2)
		// Promote to L1
		c.l1.Set(key, data, c.config.L1TTL)
		return data, true
	}
	c.recordMiss(2)

	// Try L3 (remote cache) if available
	if c.l3 != nil {
		if data, found := c.l3.Get(key); found {
			c.recordHit(3)
			// Promote to L2 and L1
			c.l2.Set(key, data, c.config.L2TTL)
			c.l1.Set(key, data, c.config.L1TTL)
			return data, true
		}
		c.recordMiss(3)
	}

	return nil, false
}

// Set stores an item in all cache layers
func (c *MultiLayerCache) Set(key string, data interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Store in L1 (memory)
	if err := c.l1.Set(key, data, c.config.L1TTL); err != nil {
		return fmt.Errorf("L1 cache set failed: %w", err)
	}

	// Store in L2 (disk)
	if err := c.l2.Set(key, data, c.config.L2TTL); err != nil {
		// Log warning but don't fail
		fmt.Printf("Warning: L2 cache set failed: %v\n", err)
	}

	// Store in L3 (remote) if available
	if c.l3 != nil {
		if err := c.l3.Set(key, data, c.config.L3TTL); err != nil {
			// Log warning but don't fail
			fmt.Printf("Warning: L3 cache set failed: %v\n", err)
		}
	}

	// Update prediction model
	if c.predictor != nil {
		c.predictor.RecordSet(key, data)
	}

	return nil
}

// Delete removes an item from all cache layers
func (c *MultiLayerCache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errors []error

	if err := c.l1.Delete(key); err != nil {
		errors = append(errors, fmt.Errorf("L1 delete failed: %w", err))
	}

	if err := c.l2.Delete(key); err != nil {
		errors = append(errors, fmt.Errorf("L2 delete failed: %w", err))
	}

	if c.l3 != nil {
		if err := c.l3.Delete(key); err != nil {
			errors = append(errors, fmt.Errorf("L3 delete failed: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cache delete errors: %v", errors)
	}

	return nil
}

// Clear removes all items from all cache layers
func (c *MultiLayerCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errors []error

	if err := c.l1.Clear(); err != nil {
		errors = append(errors, fmt.Errorf("L1 clear failed: %w", err))
	}

	if err := c.l2.Clear(); err != nil {
		errors = append(errors, fmt.Errorf("L2 clear failed: %w", err))
	}

	if c.l3 != nil {
		if err := c.l3.Clear(); err != nil {
			errors = append(errors, fmt.Errorf("L3 clear failed: %w", err))
		}
	}

	// Reset stats
	c.stats = &CacheStats{StartTime: time.Now()}

	if len(errors) > 0 {
		return fmt.Errorf("cache clear errors: %v", errors)
	}

	return nil
}

// Prefetch loads data into cache proactively
func (c *MultiLayerCache) Prefetch(ctx context.Context, targets []PrefetchTarget) error {
	for _, target := range targets {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := c.prefetchTarget(target); err != nil {
				fmt.Printf("Warning: prefetch failed for %s: %v\n", target.Key, err)
			}
		}
	}
	return nil
}

// GetPredictedTargets returns suggested prefetch targets
func (c *MultiLayerCache) GetPredictedTargets() []PrefetchTarget {
	if c.predictor == nil {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.predictor.GetPredictions()
}

// GetStats returns cache performance statistics
func (c *MultiLayerCache) GetStats() CacheStats {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()

	stats := *c.stats

	// Calculate hit rates
	l1Total := stats.L1Hits + stats.L1Misses
	if l1Total > 0 {
		stats.L1HitRate = float64(stats.L1Hits) / float64(l1Total)
	}

	l2Total := stats.L2Hits + stats.L2Misses
	if l2Total > 0 {
		stats.L2HitRate = float64(stats.L2Hits) / float64(l2Total)
	}

	l3Total := stats.L3Hits + stats.L3Misses
	if l3Total > 0 {
		stats.L3HitRate = float64(stats.L3Hits) / float64(l3Total)
	}

	overallHits := stats.L1Hits + stats.L2Hits + stats.L3Hits
	overallTotal := l1Total + l2Total + l3Total
	if overallTotal > 0 {
		stats.OverallHitRate = float64(overallHits) / float64(overallTotal)
	}

	return stats
}

// Optimize performs cache optimization operations
func (c *MultiLayerCache) Optimize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clean expired entries
	if err := c.l1.Clean(); err != nil {
		fmt.Printf("Warning: L1 cleanup failed: %v\n", err)
	}

	if err := c.l2.Clean(); err != nil {
		fmt.Printf("Warning: L2 cleanup failed: %v\n", err)
	}

	if c.l3 != nil {
		if err := c.l3.Clean(); err != nil {
			fmt.Printf("Warning: L3 cleanup failed: %v\n", err)
		}
	}

	// Run prediction model optimization
	if c.predictor != nil {
		c.predictor.Optimize()
	}

	return nil
}

// Helper methods

func (c *MultiLayerCache) recordHit(layer int) {
	c.stats.mu.Lock()
	defer c.stats.mu.Unlock()

	switch layer {
	case 1:
		c.stats.L1Hits++
	case 2:
		c.stats.L2Hits++
	case 3:
		c.stats.L3Hits++
	}
}

func (c *MultiLayerCache) recordMiss(layer int) {
	c.stats.mu.Lock()
	defer c.stats.mu.Unlock()

	switch layer {
	case 1:
		c.stats.L1Misses++
	case 2:
		c.stats.L2Misses++
	case 3:
		c.stats.L3Misses++
	}
}

func (c *MultiLayerCache) prefetchTarget(target PrefetchTarget) error {
	// Check if already cached
	if _, found := c.Get(target.Key); found {
		return nil // Already cached
	}

	// If we have a loader function, use it
	if target.Loader != nil {
		data, err := target.Loader()
		if err != nil {
			return err
		}
		return c.Set(target.Key, data, target.TTL)
	}

	return nil
}

// Extended CacheStats with calculated fields
type CacheStats struct {
	L1Hits         int64     `json:"l1_hits"`
	L1Misses       int64     `json:"l1_misses"`
	L1HitRate      float64   `json:"l1_hit_rate"`
	L2Hits         int64     `json:"l2_hits"`
	L2Misses       int64     `json:"l2_misses"`
	L2HitRate      float64   `json:"l2_hit_rate"`
	L3Hits         int64     `json:"l3_hits"`
	L3Misses       int64     `json:"l3_misses"`
	L3HitRate      float64   `json:"l3_hit_rate"`
	OverallHitRate float64   `json:"overall_hit_rate"`
	Evictions      int64     `json:"evictions"`
	Prefetches     int64     `json:"prefetches"`
	StartTime      time.Time `json:"start_time"`
	mu             sync.RWMutex
}

// PrefetchTarget represents something that should be preloaded
type PrefetchTarget struct {
	Key         string
	Priority    int
	TTL         time.Duration
	Loader      func() (interface{}, error)
	Probability float64 // Probability this will be accessed
}

// DiskCache implements persistent disk-based caching
type DiskCache struct {
	basePath    string
	maxSize     int64
	ttl         time.Duration
	compress    bool
	currentSize int64
	mu          sync.RWMutex
}

// NewDiskCache creates a new disk cache
func NewDiskCache(basePath string, maxSize int64, ttl time.Duration, compress bool) (*DiskCache, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, err
	}

	return &DiskCache{
		basePath: basePath,
		maxSize:  maxSize,
		ttl:      ttl,
		compress: compress,
	}, nil
}

// Get retrieves an item from disk cache
func (d *DiskCache) Get(key string) (interface{}, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	filePath := d.getFilePath(key)

	// Check if file exists and is not expired
	if !d.isValid(filePath) {
		return nil, false
	}

	data, err := d.readFile(filePath)
	if err != nil {
		return nil, false
	}

	// Update access time
	d.updateAccessTime(filePath)

	return data, true
}

// Set stores an item in disk cache
func (d *DiskCache) Set(key string, data interface{}, ttl time.Duration) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	filePath := d.getFilePath(key)

	// Create entry
	entry := CacheEntry{
		Key:        key,
		Data:       data,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
		TTL:        ttl,
		Layer:      2,
		Compressed: d.compress,
	}

	return d.writeFile(filePath, entry)
}

// Delete removes an item from disk cache
func (d *DiskCache) Delete(key string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	filePath := d.getFilePath(key)
	return os.Remove(filePath)
}

// Clear removes all items from disk cache
func (d *DiskCache) Clear() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	return os.RemoveAll(d.basePath)
}

// Clean removes expired entries
func (d *DiskCache) Clean() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	return filepath.Walk(d.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && !d.isValid(path) {
			os.Remove(path)
		}

		return nil
	})
}

// Helper methods for DiskCache

func (d *DiskCache) getFilePath(key string) string {
	// Create a safe filename from the key
	filename := fmt.Sprintf("%x", []byte(key))
	if d.compress {
		filename += ".gz"
	} else {
		filename += ".json"
	}
	return filepath.Join(d.basePath, filename)
}

func (d *DiskCache) isValid(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	// Check if expired
	if time.Since(info.ModTime()) > d.ttl {
		return false
	}

	return true
}

func (d *DiskCache) readFile(filePath string) (interface{}, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var reader io.Reader = file

	if d.compress {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, err
		}
		defer gzReader.Close()
		reader = gzReader
	}

	var entry CacheEntry
	if err := json.NewDecoder(reader).Decode(&entry); err != nil {
		return nil, err
	}

	return entry.Data, nil
}

func (d *DiskCache) writeFile(filePath string, entry CacheEntry) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var writer io.Writer = file

	if d.compress {
		gzWriter := gzip.NewWriter(file)
		defer gzWriter.Close()
		writer = gzWriter
	}

	return json.NewEncoder(writer).Encode(entry)
}

func (d *DiskCache) updateAccessTime(filePath string) {
	now := time.Now()
	os.Chtimes(filePath, now, now)
}

// RemoteCache implements network-based caching (placeholder)
type RemoteCache struct {
	config RemoteConfig
	ttl    time.Duration
}

func NewRemoteCache(config RemoteConfig, ttl time.Duration) (*RemoteCache, error) {
	// This is a placeholder - would implement actual remote cache (S3, Redis, etc.)
	return &RemoteCache{
		config: config,
		ttl:    ttl,
	}, nil
}

func (r *RemoteCache) Get(key string) (interface{}, bool) {
	// Placeholder implementation
	return nil, false
}

func (r *RemoteCache) Set(key string, data interface{}, ttl time.Duration) error {
	// Placeholder implementation
	return nil
}

func (r *RemoteCache) Delete(key string) error {
	// Placeholder implementation
	return nil
}

func (r *RemoteCache) Clear() error {
	// Placeholder implementation
	return nil
}

func (r *RemoteCache) Clean() error {
	// Placeholder implementation
	return nil
}
