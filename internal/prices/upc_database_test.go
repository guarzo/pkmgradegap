package prices

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUPCDatabase(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "upc_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	db, err := NewUPCDatabase(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Add and Lookup", func(t *testing.T) {
		mapping := &UPCMapping{
			UPC:         "820650558726",
			ProductID:   "surging-sparks-250",
			ProductName: "Pikachu ex",
			SetName:     "Surging Sparks",
			CardNumber:  "250",
			Language:    "English",
			Confidence:  1.0,
		}

		db.Add(mapping)

		// Test lookup
		result, found := db.Lookup("820650558726")
		if !found {
			t.Error("Expected to find UPC mapping")
		}
		if result.ProductID != "surging-sparks-250" {
			t.Errorf("Expected product ID surging-sparks-250, got %s", result.ProductID)
		}
	})

	t.Run("AddBatch", func(t *testing.T) {
		mappings := []*UPCMapping{
			{
				UPC:         "820650558733",
				ProductID:   "surging-sparks-251",
				ProductName: "Alolan Exeggutor ex",
				SetName:     "Surging Sparks",
				CardNumber:  "251",
			},
			{
				UPC:         "820650559876",
				ProductID:   "prismatic-evolutions-001",
				ProductName: "Eevee",
				SetName:     "Prismatic Evolutions",
				CardNumber:  "001",
			},
		}

		db.AddBatch(mappings)

		// Verify all were added
		if _, found := db.Lookup("820650558733"); !found {
			t.Error("Expected to find first batch UPC")
		}
		if _, found := db.Lookup("820650559876"); !found {
			t.Error("Expected to find second batch UPC")
		}
	})

	t.Run("FindByProductID", func(t *testing.T) {
		// Add duplicate product IDs with different UPCs
		db.Add(&UPCMapping{
			UPC:        "074427891234",
			ProductID:  "base-set-004",
			SetName:    "Base Set",
			CardNumber: "004",
			Variant:    "1st Edition",
		})
		db.Add(&UPCMapping{
			UPC:        "074427891241",
			ProductID:  "base-set-004",
			SetName:    "Base Set",
			CardNumber: "004",
			Variant:    "Shadowless",
		})

		results := db.FindByProductID("base-set-004")
		if len(results) != 2 {
			t.Errorf("Expected 2 mappings for product ID, got %d", len(results))
		}
	})

	t.Run("FindByCardInfo", func(t *testing.T) {
		db.Add(&UPCMapping{
			UPC:         "4521329385426",
			ProductID:   "vmax-climax-003",
			ProductName: "Charizard VMAX",
			SetName:     "VMAX Climax",
			CardNumber:  "003",
			Language:    "Japanese",
		})

		results := db.FindByCardInfo("VMAX Climax", "003")
		if len(results) != 1 {
			t.Errorf("Expected 1 mapping for card info, got %d", len(results))
		}

		// Test case insensitive search
		results = db.FindByCardInfo("vmax climax", "003")
		if len(results) != 1 {
			t.Errorf("Expected case-insensitive search to work, got %d results", len(results))
		}
	})

	t.Run("Save and Load", func(t *testing.T) {
		// Save current database
		err := db.Save()
		if err != nil {
			t.Fatal(err)
		}

		// Create new database instance and load
		db2, err := NewUPCDatabase(tmpDir)
		if err != nil {
			t.Fatal(err)
		}

		// Verify data was persisted
		result, found := db2.Lookup("820650558726")
		if !found {
			t.Error("Expected to find persisted UPC mapping")
		}
		if result.ProductName != "Pikachu ex" {
			t.Errorf("Expected persisted product name Pikachu ex, got %s", result.ProductName)
		}
	})

	t.Run("Remove", func(t *testing.T) {
		db.Remove("820650558726")

		_, found := db.Lookup("820650558726")
		if found {
			t.Error("Expected UPC to be removed")
		}
	})

	t.Run("Stats", func(t *testing.T) {
		// Populate with test data
		db.PopulateCommonMappings()

		stats := db.Stats()
		total, ok := stats["total_mappings"].(int)
		if !ok || total == 0 {
			t.Error("Expected non-zero total mappings in stats")
		}

		languages, ok := stats["languages"].(map[string]int)
		if !ok || len(languages) == 0 {
			t.Error("Expected language statistics")
		}
	})
}

func TestUPCMapping_LastUpdated(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "upc_test")
	defer os.RemoveAll(tmpDir)

	db, _ := NewUPCDatabase(tmpDir)

	before := time.Now()
	mapping := &UPCMapping{
		UPC:       "test-upc",
		ProductID: "test-product",
	}
	db.Add(mapping)
	after := time.Now()

	result, _ := db.Lookup("test-upc")
	if result.LastUpdated.Before(before) || result.LastUpdated.After(after) {
		t.Error("LastUpdated timestamp not set correctly")
	}
}

func TestUPCDatabase_NonExistentFile(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "upc_test")
	defer os.RemoveAll(tmpDir)

	// Try to load from non-existent file
	db, err := NewUPCDatabase(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Should initialize with empty database
	all := db.GetAll()
	if len(all) != 0 {
		t.Errorf("Expected empty database, got %d mappings", len(all))
	}
}

func TestUPCDatabase_InvalidJSON(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "upc_test")
	defer os.RemoveAll(tmpDir)

	// Write invalid JSON to file
	invalidFile := filepath.Join(tmpDir, "upc_mappings.json")
	os.WriteFile(invalidFile, []byte("invalid json"), 0644)

	// Should fail to load
	db, err := NewUPCDatabase(tmpDir)
	if err == nil {
		t.Error("Expected error loading invalid JSON")
	}
	if db != nil {
		t.Error("Expected nil database on error")
	}
}
