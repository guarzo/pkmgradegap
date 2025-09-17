package analysis

import (
	"testing"
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
)

func BenchmarkReportRank(b *testing.B) {
	// Generate test data for benchmarking
	rows := generateTestRows(1000) // 1000 cards
	config := Config{
		MaxAgeYears:    10,
		MinDeltaUSD:    25.0,
		MinRawUSD:      5.0,
		TopN:           25,
		GradingCost:    25.0,
		ShippingCost:   20.0,
		FeePct:         0.13,
		JapaneseWeight: 1.0,
	}
	set := &model.Set{
		Name:        "Benchmark Set",
		ID:          "benchmark1",
		ReleaseDate: "2023-01-01",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ReportRank(rows, set, config)
	}
}

func BenchmarkReportRankSmall(b *testing.B) {
	// Benchmark with smaller dataset (typical set size)
	rows := generateTestRows(250) // Typical set size
	config := Config{
		MaxAgeYears:    10,
		MinDeltaUSD:    25.0,
		MinRawUSD:      5.0,
		TopN:           25,
		GradingCost:    25.0,
		ShippingCost:   20.0,
		FeePct:         0.13,
		JapaneseWeight: 1.0,
	}
	set := &model.Set{
		Name:        "Benchmark Set Small",
		ID:          "benchmark_small",
		ReleaseDate: "2023-01-01",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ReportRank(rows, set, config)
	}
}

func BenchmarkScoring(b *testing.B) {
	// Benchmark just the scoring logic without CSV generation
	rows := generateTestRows(1000)
	config := Config{
		MaxAgeYears:    10,
		MinDeltaUSD:    25.0,
		MinRawUSD:      5.0,
		TopN:           1000, // Don't limit to test full scoring
		GradingCost:    25.0,
		ShippingCost:   20.0,
		FeePct:         0.13,
		JapaneseWeight: 1.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scoreAndFilter(rows, config)
	}
}

func BenchmarkJapaneseDetection(b *testing.B) {
	// Benchmark Japanese character detection
	testStrings := []string{
		"Pikachu",
		"ピカチュウ",
		"Pokemon Card",
		"ポケモンカード",
		"Charizard ex",
		"リザードンex",
		"Mixed ピカチュウ Text",
		"Normal English Text",
		"日本語のテキスト",
		"English with numbers 123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, text := range testStrings {
			containsJapanese(text)
		}
	}
}

func BenchmarkExtractUngradedUSD(b *testing.B) {
	// Benchmark price extraction logic
	cards := generateTestCards(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, card := range cards {
			ExtractUngradedUSD(card)
		}
	}
}

func BenchmarkRowConstruction(b *testing.B) {
	// Benchmark the creation of analysis rows
	cards := generateTestCards(250)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows := make([]Row, 0, len(cards))
		for _, card := range cards {
			rawUSD, rawSrc, rawNote := ExtractUngradedUSD(card)
			grades := Grades{
				PSA10:   100.0 + float64(i%50), // Vary prices
				Grade9:  70.0 + float64(i%30),
				Grade95: 85.0 + float64(i%40),
				BGS10:   110.0 + float64(i%60),
			}

			row := Row{
				Card:       card,
				RawUSD:     rawUSD,
				RawSrc:     rawSrc,
				RawNote:    rawNote,
				Grades:     grades,
				Population: nil,
				Volatility: 0.15,
			}
			rows = append(rows, row)
		}
	}
}

// Helper functions for benchmark test data generation

func generateTestRows(count int) []Row {
	var rows []Row

	for i := 0; i < count; i++ {
		// Create variety in card data
		cardName := generateCardName(i)
		number := generateCardNumber(i)

		// Vary prices to create realistic distribution
		rawPrice := 10.0 + float64(i%100)*0.5  // $10-$60 range
		psa10Price := rawPrice * (1.5 + float64(i%10)*0.2) // 1.5x to 3.5x multiplier

		row := Row{
			Card: model.Card{
				Name:   cardName,
				Number: number,
				TCGPlayer: &model.TCGPlayerBlock{
					Prices: map[string]struct {
						Low       *float64 `json:"low,omitempty"`
						Mid       *float64 `json:"mid,omitempty"`
						High      *float64 `json:"high,omitempty"`
						Market    *float64 `json:"market,omitempty"`
						DirectLow *float64 `json:"directLow,omitempty"`
					}{
						"normal": {Market: &rawPrice},
					},
				},
			},
			RawUSD: rawPrice,
			RawSrc: "tcgplayer.market",
			RawNote: "USD",
			Grades: Grades{
				PSA10:   psa10Price,
				Grade9:  psa10Price * 0.7,  // 70% of PSA10
				Grade95: psa10Price * 0.85, // 85% of PSA10
				BGS10:   psa10Price * 1.1,  // 110% of PSA10
			},
			Population: &model.PSAPopulation{
				TotalGraded: 1000 + i%2000,
				PSA10:       50 + i%200,
				PSA9:        300 + i%500,
				PSA8:        200 + i%300,
				LastUpdated: time.Now(),
			},
			Volatility: 0.05 + float64(i%20)*0.01, // 5% to 25% volatility
		}

		rows = append(rows, row)
	}

	return rows
}

func generateTestCards(count int) []model.Card {
	var cards []model.Card

	for i := 0; i < count; i++ {
		marketPrice := 10.0 + float64(i%100)*0.5
		card := model.Card{
			Name:   generateCardName(i),
			Number: generateCardNumber(i),
			TCGPlayer: &model.TCGPlayerBlock{
				Prices: map[string]struct {
					Low       *float64 `json:"low,omitempty"`
					Mid       *float64 `json:"mid,omitempty"`
					High      *float64 `json:"high,omitempty"`
					Market    *float64 `json:"market,omitempty"`
					DirectLow *float64 `json:"directLow,omitempty"`
				}{
					"normal": {Market: &marketPrice},
				},
			},
		}
		cards = append(cards, card)
	}

	return cards
}

func generateCardName(index int) string {
	// Create variety in card names including some Japanese
	names := []string{
		"Pikachu", "Charizard", "Blastoise", "Venusaur", "Mew",
		"Lugia", "Ho-Oh", "Rayquaza", "Dialga", "Palkia",
		"ピカチュウ", "リザードン", "フシギダネ", "ゼニガメ", "ミュウ",
		"Garchomp", "Lucario", "Darkrai", "Arceus", "Reshiram",
		"Zekrom", "Kyurem", "Xerneas", "Yveltal", "Zygarde",
	}

	baseName := names[index%len(names)]

	// Add variety with ex, V, VMAX suffixes
	suffixes := []string{"", " ex", " V", " VMAX", " VSTAR", " GX"}
	suffix := suffixes[index%len(suffixes)]

	return baseName + suffix
}

func generateCardNumber(index int) string {
	// Generate realistic card numbers
	if index%20 == 0 {
		// Some special/promo numbers
		return "PROMO"
	}
	if index%15 == 0 {
		// Some secret rare numbers
		return "SR"
	}
	// Regular numbers 1-300
	return string(rune('0' + (index%10))) + string(rune('0' + ((index/10)%10))) + string(rune('0' + ((index/100)%3)))
}

// Benchmark memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	config := Config{
		MaxAgeYears:    10,
		MinDeltaUSD:    25.0,
		MinRawUSD:      5.0,
		TopN:           25,
		GradingCost:    25.0,
		ShippingCost:   20.0,
		FeePct:         0.13,
		JapaneseWeight: 1.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// This tests memory allocation patterns in real usage
		rows := generateTestRows(250)

		// Simulate the full analysis pipeline
		scoredRows := scoreAndFilter(rows, config)

		// This simulates CSV generation memory usage
		csvRows := make([][]string, len(scoredRows)+1)
		csvRows[0] = []string{"Card", "No", "RawUSD", "PSA10USD", "DeltaUSD", "CostUSD", "BreakEvenUSD", "Score", "Notes"}

		for j, sr := range scoredRows {
			if j >= config.TopN {
				break
			}
			csvRows[j+1] = []string{
				sr.Row.Card.Name,
				sr.Row.Card.Number,
				"$0.00", "$0.00", "$0.00", "$0.00", "$0.00", "0.0", "",
			}
		}

		// Force usage to prevent optimization
		_ = len(csvRows)
	}
}