package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Entry struct {
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
	TTL       time.Duration   `json:"ttl"`
}

type Cache struct {
	path    string
	entries map[string]Entry
	mu      sync.RWMutex
}

func New(path string) (*Cache, error) {
	c := &Cache{
		path:    path,
		entries: make(map[string]Entry),
	}

	// Load existing cache if file exists
	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read cache: %w", err)
		}
		if len(data) > 0 {
			if err := json.Unmarshal(data, &c.entries); err != nil {
				// Ignore corrupt cache, start fresh
				c.entries = make(map[string]Entry)
			}
		}
	}

	return c, nil
}

func (c *Cache) Get(key string, target interface{}) (bool, error) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	if !ok {
		c.mu.RUnlock()
		return false, nil
	}

	// Check TTL
	expired := entry.TTL > 0 && time.Since(entry.Timestamp) > entry.TTL
	if !expired {
		// Entry is valid, unmarshal and return
		err := json.Unmarshal(entry.Data, target)
		c.mu.RUnlock()
		if err != nil {
			return false, fmt.Errorf("unmarshal cache entry: %w", err)
		}
		return true, nil
	}
	c.mu.RUnlock()

	// Entry expired, need write lock to delete
	c.mu.Lock()
	// Double-check the entry still exists and is expired
	if e, exists := c.entries[key]; exists && e.TTL > 0 && time.Since(e.Timestamp) > e.TTL {
		delete(c.entries, key)
	}
	c.mu.Unlock()

	return false, nil
}

func (c *Cache) Put(key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}

	c.mu.Lock()
	c.entries[key] = Entry{
		Data:      data,
		Timestamp: time.Now(),
		TTL:       ttl,
	}
	c.mu.Unlock()

	return c.saveLocked()
}

// saveLocked saves the cache to disk without additional locking
// Call this when you already have a lock or after releasing it
func (c *Cache) saveLocked() error {
	// Create parent directory if needed
	if dir := filepath.Dir(c.path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create cache dir: %w", err)
		}
	}

	// Take a read lock to safely marshal entries
	c.mu.RLock()
	data, err := json.MarshalIndent(c.entries, "", "  ")
	c.mu.RUnlock()

	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}

	return os.WriteFile(c.path, data, 0644)
}

// save is deprecated, use saveLocked instead
func (c *Cache) save() error {
	return c.saveLocked()
}

// Clear removes all cache entries
func (c *Cache) Clear() error {
	c.mu.Lock()
	c.entries = make(map[string]Entry)
	c.mu.Unlock()
	return c.saveLocked()
}

// Remove deletes a specific cache entry
func (c *Cache) Remove(key string) error {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
	return c.saveLocked()
}

// BuildKey creates semantic cache keys
func BuildKey(parts ...string) string {
	key := ""
	for i, part := range parts {
		if i > 0 {
			key += "|"
		}
		key += part
	}
	return key
}

// Common cache keys
func SetsKey() string {
	return "sets:v2"
}

func CardsKey(setID string) string {
	return BuildKey("cards", "set", setID)
}

func PriceChartingKey(setName, cardName, number string) string {
	return BuildKey("pc", setName, cardName, number)
}
