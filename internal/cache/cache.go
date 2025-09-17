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
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		return false, nil
	}

	// Check TTL
	if entry.TTL > 0 && time.Since(entry.Timestamp) > entry.TTL {
		delete(c.entries, key)
		return false, nil
	}

	if err := json.Unmarshal(entry.Data, target); err != nil {
		return false, fmt.Errorf("unmarshal cache entry: %w", err)
	}

	return true, nil
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

	return c.save()
}

func (c *Cache) save() error {
	// Create parent directory if needed
	if dir := filepath.Dir(c.path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create cache dir: %w", err)
		}
	}

	c.mu.RLock()
	data, err := json.MarshalIndent(c.entries, "", "  ")
	c.mu.RUnlock()

	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}

	return os.WriteFile(c.path, data, 0644)
}

// Clear removes all cache entries
func (c *Cache) Clear() error {
	c.mu.Lock()
	c.entries = make(map[string]Entry)
	c.mu.Unlock()
	return c.save()
}

// Remove deletes a specific cache entry
func (c *Cache) Remove(key string) error {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
	return c.save()
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