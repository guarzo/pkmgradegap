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
)

// MultiLayerCache implements a two-tier caching system
type MultiLayerCache struct {
	l1 *MemoryCache // Hot data - fast access
	l2 *DiskCache   // Warm data - persistent

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
	L2Path        string        // Path for disk cache
	EnablePredict bool          // Enable predictive caching
	CompressL2    bool          // Compress L2 cache entries
}

// CacheStats tracks cache performance metrics
type CacheStats struct {
	L1Hits         int64     `json:"l1_hits"`
	L1Misses       int64     `json:"l1_misses"`
	L1HitRate      float64   `json:"l1_hit_rate"`
	L2Hits         int64     `json:"l2_hits"`
	L2Misses       int64     `json:"l2_misses"`
	L2HitRate      float64   `json:"l2_hit_rate"`
	OverallHitRate float64   `json:"overall_hit_rate"`
	Evictions      int64     `json:"evictions"`
	Prefetches     int64     `json:"prefetches"`
	StartTime      time.Time `json:"start_time"`
	mu             sync.RWMutex
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

	// Initialize predictor if enabled
	if config.EnablePredict {
		cache.predictor = NewCachePredictor()
	}

	return cache, nil
}

// CachePriority defines priority and TTL for cache entries
type CachePriority struct {
	TTL      time.Duration
	Priority int  // 1-3, higher is more important
	Volatile bool // True for frequently changing data
}

// GetEntry retrieves an entry from the cache (for use with CacheEntry type)
func (m *MultiLayerCache) GetEntry(key string) *CacheEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try L1 first
	if data, found := m.l1.Get(key); found {
		m.stats.recordL1Hit()
		return &CacheEntry{
			Key:  key,
			Data: data,
		}
	}
	m.stats.recordL1Miss()

	// Try L2
	if data, found := m.l2.Get(key); found {
		m.stats.recordL2Hit()
		// Promote to L1
		m.l1.Set(key, data, m.config.L1TTL)
		return &CacheEntry{
			Key:  key,
			Data: data,
		}
	}
	m.stats.recordL2Miss()

	return nil
}

// Put stores an entry in the cache with priority
func (m *MultiLayerCache) Put(key string, data interface{}, priority CachePriority) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Store in L1 if high priority or volatile
	if priority.Priority >= 2 || priority.Volatile {
		if err := m.l1.Set(key, data, priority.TTL); err != nil {
			return err
		}
	}

	// Always store in L2 for persistence
	if priority.Priority >= 1 {
		if err := m.l2.Set(key, data, m.config.L2TTL); err != nil {
			// Log but don't fail
			fmt.Printf("Warning: L2 cache set failed: %v\n", err)
		}
	}

	// Predictive caching for related items
	if m.predictor != nil && priority.Priority >= 2 {
		m.predictor.RecordAccess(key)
	}

	return nil
}

// Stats recording methods
func (s *CacheStats) recordL1Hit() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.L1Hits++
	s.updateRates()
}

func (s *CacheStats) recordL1Miss() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.L1Misses++
	s.updateRates()
}

func (s *CacheStats) recordL2Hit() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.L2Hits++
	s.updateRates()
}

func (s *CacheStats) recordL2Miss() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.L2Misses++
	s.updateRates()
}

func (s *CacheStats) updateRates() {
	totalL1 := s.L1Hits + s.L1Misses
	if totalL1 > 0 {
		s.L1HitRate = float64(s.L1Hits) / float64(totalL1)
	}

	totalL2 := s.L2Hits + s.L2Misses
	if totalL2 > 0 {
		s.L2HitRate = float64(s.L2Hits) / float64(totalL2)
	}

	totalHits := s.L1Hits + s.L2Hits
	totalRequests := totalL1 // Only count initial requests
	if totalRequests > 0 {
		s.OverallHitRate = float64(totalHits) / float64(totalRequests)
	}
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
func (c *MultiLayerCache) GetStats() *CacheStats {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()
	return c.stats
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
