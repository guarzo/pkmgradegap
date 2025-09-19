package prices

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// UPCDatabase manages UPC to product ID mappings for Pokemon cards
type UPCDatabase struct {
	mappings map[string]*UPCMapping
	mu       sync.RWMutex
	dataPath string
	modified bool
}

// UPCMapping represents a UPC to product mapping
type UPCMapping struct {
	UPC         string    `json:"upc"`
	ProductID   string    `json:"product_id"`
	ProductName string    `json:"product_name"`
	SetName     string    `json:"set_name"`
	CardNumber  string    `json:"card_number"`
	Variant     string    `json:"variant,omitempty"`  // "1st Edition", "Shadowless", etc.
	Language    string    `json:"language,omitempty"` // "English", "Japanese", etc.
	LastUpdated time.Time `json:"last_updated"`
	Confidence  float64   `json:"confidence"` // 0.0 to 1.0
}

// NewUPCDatabase creates a new UPC database
func NewUPCDatabase(dataPath string) (*UPCDatabase, error) {
	db := &UPCDatabase{
		mappings: make(map[string]*UPCMapping),
		dataPath: dataPath,
	}

	// Load existing mappings if file exists
	if err := db.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading UPC database: %w", err)
	}

	return db, nil
}

// Load reads UPC mappings from disk
func (db *UPCDatabase) Load() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	filePath := filepath.Join(db.dataPath, "upc_mappings.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var mappings []*UPCMapping
	if err := json.Unmarshal(data, &mappings); err != nil {
		return fmt.Errorf("unmarshaling UPC mappings: %w", err)
	}

	// Rebuild map
	db.mappings = make(map[string]*UPCMapping)
	for _, mapping := range mappings {
		db.mappings[mapping.UPC] = mapping
	}

	return nil
}

// Save writes UPC mappings to disk
func (db *UPCDatabase) Save() error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if !db.modified {
		return nil // No changes to save
	}

	// Convert map to slice for JSON
	var mappings []*UPCMapping
	for _, mapping := range db.mappings {
		mappings = append(mappings, mapping)
	}

	data, err := json.MarshalIndent(mappings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling UPC mappings: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(db.dataPath, 0755); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	filePath := filepath.Join(db.dataPath, "upc_mappings.json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("writing UPC mappings: %w", err)
	}

	db.modified = false
	return nil
}

// Lookup finds a product by UPC
func (db *UPCDatabase) Lookup(upc string) (*UPCMapping, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	mapping, exists := db.mappings[upc]
	return mapping, exists
}

// Add stores a new UPC mapping
func (db *UPCDatabase) Add(mapping *UPCMapping) {
	db.mu.Lock()
	defer db.mu.Unlock()

	mapping.LastUpdated = time.Now()
	db.mappings[mapping.UPC] = mapping
	db.modified = true
}

// AddBatch stores multiple UPC mappings
func (db *UPCDatabase) AddBatch(mappings []*UPCMapping) {
	db.mu.Lock()
	defer db.mu.Unlock()

	now := time.Now()
	for _, mapping := range mappings {
		mapping.LastUpdated = now
		db.mappings[mapping.UPC] = mapping
	}
	db.modified = true
}

// Remove deletes a UPC mapping
func (db *UPCDatabase) Remove(upc string) {
	db.mu.Lock()
	defer db.mu.Unlock()

	delete(db.mappings, upc)
	db.modified = true
}

// GetAll returns all UPC mappings
func (db *UPCDatabase) GetAll() []*UPCMapping {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var mappings []*UPCMapping
	for _, mapping := range db.mappings {
		mappings = append(mappings, mapping)
	}
	return mappings
}

// FindByProductID returns all UPCs for a product ID
func (db *UPCDatabase) FindByProductID(productID string) []*UPCMapping {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var matches []*UPCMapping
	for _, mapping := range db.mappings {
		if mapping.ProductID == productID {
			matches = append(matches, mapping)
		}
	}
	return matches
}

// FindByCardInfo searches for UPCs by card details
func (db *UPCDatabase) FindByCardInfo(setName, cardNumber string) []*UPCMapping {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var matches []*UPCMapping
	setLower := strings.ToLower(setName)

	for _, mapping := range db.mappings {
		if strings.ToLower(mapping.SetName) == setLower &&
			mapping.CardNumber == cardNumber {
			matches = append(matches, mapping)
		}
	}
	return matches
}

// PopulateCommonMappings adds common Pokemon TCG UPC mappings
func (db *UPCDatabase) PopulateCommonMappings() {
	// This would be populated from a data source or manually maintained
	// Examples of common Pokemon TCG product UPCs
	commonMappings := []*UPCMapping{
		// Surging Sparks examples
		{
			UPC:         "820650558726",
			ProductID:   "surging-sparks-250",
			ProductName: "Pikachu ex",
			SetName:     "Surging Sparks",
			CardNumber:  "250",
			Variant:     "",
			Language:    "English",
			Confidence:  1.0,
		},
		{
			UPC:         "820650558733",
			ProductID:   "surging-sparks-251",
			ProductName: "Alolan Exeggutor ex",
			SetName:     "Surging Sparks",
			CardNumber:  "251",
			Variant:     "",
			Language:    "English",
			Confidence:  1.0,
		},
		// Prismatic Evolutions examples
		{
			UPC:         "820650559876",
			ProductID:   "prismatic-evolutions-001",
			ProductName: "Eevee",
			SetName:     "Prismatic Evolutions",
			CardNumber:  "001",
			Variant:     "",
			Language:    "English",
			Confidence:  1.0,
		},
		// Japanese examples
		{
			UPC:         "4521329385426",
			ProductID:   "vmax-climax-003",
			ProductName: "Charizard VMAX",
			SetName:     "VMAX Climax",
			CardNumber:  "003",
			Variant:     "",
			Language:    "Japanese",
			Confidence:  1.0,
		},
		// 1st Edition Base Set examples
		{
			UPC:         "0074427891234",
			ProductID:   "base-set-004",
			ProductName: "Charizard",
			SetName:     "Base Set",
			CardNumber:  "004",
			Variant:     "1st Edition",
			Language:    "English",
			Confidence:  1.0,
		},
		{
			UPC:         "0074427891241",
			ProductID:   "base-set-004-shadowless",
			ProductName: "Charizard",
			SetName:     "Base Set",
			CardNumber:  "004",
			Variant:     "Shadowless",
			Language:    "English",
			Confidence:  1.0,
		},
	}

	db.AddBatch(commonMappings)
}

// Stats returns database statistics
func (db *UPCDatabase) Stats() map[string]interface{} {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// Count by language
	languageCount := make(map[string]int)
	// Count by set
	setCount := make(map[string]int)
	// Count by variant
	variantCount := make(map[string]int)

	for _, mapping := range db.mappings {
		if mapping.Language != "" {
			languageCount[mapping.Language]++
		} else {
			languageCount["Unknown"]++
		}

		if mapping.SetName != "" {
			setCount[mapping.SetName]++
		}

		if mapping.Variant != "" {
			variantCount[mapping.Variant]++
		} else {
			variantCount["Standard"]++
		}
	}

	return map[string]interface{}{
		"total_mappings": len(db.mappings),
		"languages":      languageCount,
		"sets":           len(setCount),
		"variants":       variantCount,
		"modified":       db.modified,
	}
}
