package cache

import (
	"container/list"
	"sync"
	"time"
)

// MemoryCache implements an in-memory LRU cache with TTL support
type MemoryCache struct {
	maxSize    int
	defaultTTL time.Duration
	items      map[string]*list.Element
	lru        *list.List
	mu         sync.RWMutex
}

// MemoryCacheItem represents an item in the memory cache
type MemoryCacheItem struct {
	Key        string
	Value      interface{}
	ExpiresAt  time.Time
	AccessedAt time.Time
	Size       int64
}

// NewMemoryCache creates a new memory cache
func NewMemoryCache(maxSize int, defaultTTL time.Duration) *MemoryCache {
	return &MemoryCache{
		maxSize:    maxSize,
		defaultTTL: defaultTTL,
		items:      make(map[string]*list.Element),
		lru:        list.New(),
	}
}

// Get retrieves an item from the memory cache
func (m *MemoryCache) Get(key string) (interface{}, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	element, exists := m.items[key]
	if !exists {
		return nil, false
	}

	item := element.Value.(*MemoryCacheItem)

	// Check if expired
	if time.Now().After(item.ExpiresAt) {
		m.removeElement(element)
		return nil, false
	}

	// Update access time and move to front (most recently used)
	item.AccessedAt = time.Now()
	m.lru.MoveToFront(element)

	return item.Value, true
}

// Set stores an item in the memory cache
func (m *MemoryCache) Set(key string, value interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ttl == 0 {
		ttl = m.defaultTTL
	}

	now := time.Now()
	item := &MemoryCacheItem{
		Key:        key,
		Value:      value,
		ExpiresAt:  now.Add(ttl),
		AccessedAt: now,
		Size:       m.estimateSize(value),
	}

	// Check if item already exists
	if element, exists := m.items[key]; exists {
		// Update existing item
		element.Value = item
		m.lru.MoveToFront(element)
	} else {
		// Add new item
		element := m.lru.PushFront(item)
		m.items[key] = element

		// Evict if necessary
		m.evictIfNecessary()
	}

	return nil
}

// Delete removes an item from the memory cache
func (m *MemoryCache) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if element, exists := m.items[key]; exists {
		m.removeElement(element)
	}

	return nil
}

// Clear removes all items from the memory cache
func (m *MemoryCache) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.items = make(map[string]*list.Element)
	m.lru.Init()

	return nil
}

// Clean removes expired items from the memory cache
func (m *MemoryCache) Clean() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	var toRemove []*list.Element

	// Collect expired items
	for element := m.lru.Back(); element != nil; element = element.Prev() {
		item := element.Value.(*MemoryCacheItem)
		if now.After(item.ExpiresAt) {
			toRemove = append(toRemove, element)
		}
	}

	// Remove expired items
	for _, element := range toRemove {
		m.removeElement(element)
	}

	return nil
}

// Size returns the current number of items in cache
func (m *MemoryCache) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.items)
}

// Stats returns cache statistics
func (m *MemoryCache) Stats() MemoryCacheStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var totalSize int64
	var expiredCount int
	now := time.Now()

	for _, element := range m.items {
		item := element.Value.(*MemoryCacheItem)
		totalSize += item.Size
		if now.After(item.ExpiresAt) {
			expiredCount++
		}
	}

	return MemoryCacheStats{
		TotalItems:     len(m.items),
		TotalSize:      totalSize,
		MaxSize:        m.maxSize,
		ExpiredItems:   expiredCount,
		UtilizationPct: float64(len(m.items)) / float64(m.maxSize) * 100,
	}
}

// Helper methods

func (m *MemoryCache) removeElement(element *list.Element) {
	item := element.Value.(*MemoryCacheItem)
	delete(m.items, item.Key)
	m.lru.Remove(element)
}

func (m *MemoryCache) evictIfNecessary() {
	for len(m.items) > m.maxSize {
		// Remove least recently used item
		oldest := m.lru.Back()
		if oldest != nil {
			m.removeElement(oldest)
		}
	}
}

func (m *MemoryCache) estimateSize(value interface{}) int64 {
	// Simple size estimation - could be improved with reflection
	switch v := value.(type) {
	case string:
		return int64(len(v))
	case []byte:
		return int64(len(v))
	case int, int32, int64, float32, float64:
		return 8
	case bool:
		return 1
	default:
		return 256 // Default estimate for complex objects
	}
}

// MemoryCacheStats contains memory cache statistics
type MemoryCacheStats struct {
	TotalItems     int     `json:"total_items"`
	TotalSize      int64   `json:"total_size"`
	MaxSize        int     `json:"max_size"`
	ExpiredItems   int     `json:"expired_items"`
	UtilizationPct float64 `json:"utilization_pct"`
}

// TTLCache is a simple TTL-only cache (no LRU eviction)
type TTLCache struct {
	items map[string]*TTLCacheItem
	mu    sync.RWMutex
}

// TTLCacheItem represents an item in the TTL cache
type TTLCacheItem struct {
	Value     interface{}
	ExpiresAt time.Time
}

// NewTTLCache creates a new TTL-only cache
func NewTTLCache() *TTLCache {
	return &TTLCache{
		items: make(map[string]*TTLCacheItem),
	}
}

// Get retrieves an item from the TTL cache
func (t *TTLCache) Get(key string) (interface{}, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	item, exists := t.items[key]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(item.ExpiresAt) {
		// Don't remove here to avoid upgrading the lock
		return nil, false
	}

	return item.Value, true
}

// Set stores an item in the TTL cache
func (t *TTLCache) Set(key string, value interface{}, ttl time.Duration) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.items[key] = &TTLCacheItem{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}

	return nil
}

// Delete removes an item from the TTL cache
func (t *TTLCache) Delete(key string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.items, key)
	return nil
}

// Clean removes expired items from the TTL cache
func (t *TTLCache) Clean() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	for key, item := range t.items {
		if now.After(item.ExpiresAt) {
			delete(t.items, key)
		}
	}

	return nil
}

// Size returns the current number of items in cache
func (t *TTLCache) Size() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.items)
}

// ThreadSafeMap is a generic thread-safe map with optional TTL
type ThreadSafeMap struct {
	data map[string]interface{}
	ttl  map[string]time.Time
	mu   sync.RWMutex
}

// NewThreadSafeMap creates a new thread-safe map
func NewThreadSafeMap() *ThreadSafeMap {
	return &ThreadSafeMap{
		data: make(map[string]interface{}),
		ttl:  make(map[string]time.Time),
	}
}

// Get retrieves a value from the map
func (m *ThreadSafeMap) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check TTL if set
	if expiry, hasTTL := m.ttl[key]; hasTTL {
		if time.Now().After(expiry) {
			return nil, false
		}
	}

	value, exists := m.data[key]
	return value, exists
}

// Set stores a value in the map
func (m *ThreadSafeMap) Set(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = value
	delete(m.ttl, key) // Remove any existing TTL
}

// SetWithTTL stores a value in the map with TTL
func (m *ThreadSafeMap) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = value
	m.ttl[key] = time.Now().Add(ttl)
}

// Delete removes a value from the map
func (m *ThreadSafeMap) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
	delete(m.ttl, key)
}

// Keys returns all keys in the map
func (m *ThreadSafeMap) Keys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.data))
	now := time.Now()

	for key := range m.data {
		// Check TTL
		if expiry, hasTTL := m.ttl[key]; hasTTL {
			if now.After(expiry) {
				continue // Skip expired items
			}
		}
		keys = append(keys, key)
	}

	return keys
}

// Size returns the number of items in the map
func (m *ThreadSafeMap) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	now := time.Now()

	for key := range m.data {
		// Check TTL
		if expiry, hasTTL := m.ttl[key]; hasTTL {
			if now.After(expiry) {
				continue // Skip expired items
			}
		}
		count++
	}

	return count
}

// Clean removes expired items
func (m *ThreadSafeMap) Clean() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for key, expiry := range m.ttl {
		if now.After(expiry) {
			delete(m.data, key)
			delete(m.ttl, key)
		}
	}
}
