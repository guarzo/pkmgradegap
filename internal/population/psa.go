package population

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
)

// PopulationProvider interface for loading population data
// Future implementation could use web scraping or manual CSV imports
type PopulationProvider interface {
	LoadFromCSV(path string) error
	GetPopulation(cardID string) (*model.PSAPopulation, error)
}

// CSVPopulationProvider loads population data from CSV files
type CSVPopulationProvider struct {
	data map[string]*model.PSAPopulation
}

// NewCSVPopulationProvider creates a new CSV-based population provider
func NewCSVPopulationProvider() *CSVPopulationProvider {
	return &CSVPopulationProvider{
		data: make(map[string]*model.PSAPopulation),
	}
}

// LoadFromCSV loads population data from a CSV file
// Expected CSV format: CardID,TotalGraded,PSA10,PSA9,PSA8
func (p *CSVPopulationProvider) LoadFromCSV(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("read CSV: %w", err)
	}

	// Skip header row if present
	startRow := 0
	if len(records) > 0 && records[0][0] == "CardID" {
		startRow = 1
	}

	for i := startRow; i < len(records); i++ {
		record := records[i]
		if len(record) < 5 {
			continue // Skip malformed rows
		}

		cardID := record[0]
		totalGraded, err := strconv.Atoi(record[1])
		if err != nil {
			continue
		}

		psa10, err := strconv.Atoi(record[2])
		if err != nil {
			continue
		}

		psa9, err := strconv.Atoi(record[3])
		if err != nil {
			continue
		}

		psa8, err := strconv.Atoi(record[4])
		if err != nil {
			continue
		}

		p.data[cardID] = &model.PSAPopulation{
			TotalGraded: totalGraded,
			PSA10:       psa10,
			PSA9:        psa9,
			PSA8:        psa8,
			LastUpdated: time.Now(),
		}
	}

	return nil
}

// GetPopulation retrieves population data for a specific card
func (p *CSVPopulationProvider) GetPopulation(cardID string) (*model.PSAPopulation, error) {
	if pop, exists := p.data[cardID]; exists {
		return pop, nil
	}
	return nil, nil // No data available
}

// Available returns true if population data is loaded
func (p *CSVPopulationProvider) Available() bool {
	return len(p.data) > 0
}