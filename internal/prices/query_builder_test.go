package prices

import (
	"strings"
	"testing"
)

func TestQueryBuilder_SetBase(t *testing.T) {
	tests := []struct {
		name     string
		setName  string
		cardName string
		number   string
		expected string
	}{
		{
			name:     "basic query",
			setName:  "Surging Sparks",
			cardName: "Pikachu ex",
			number:   "250",
			expected: "pokemon Surging Sparks Pikachu ex #250",
		},
		{
			name:     "set with special chars",
			setName:  "Sword & Shield: Base",
			cardName: "Zacian V",
			number:   "138",
			expected: "pokemon Sword & Shield Base Zacian V #138",
		},
		{
			name:     "no number",
			setName:  "Promo",
			cardName: "Charizard",
			number:   "",
			expected: "pokemon Promo Charizard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewQueryBuilder().SetBase(tt.setName, tt.cardName, tt.number)
			result := qb.Build()
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestQueryBuilder_WithVariant(t *testing.T) {
	tests := []struct {
		variant  string
		contains string
	}{
		{"1st Edition", "1st edition"},
		{"First Edition", "1st edition"},
		{"Shadowless", "shadowless"},
		{"Reverse Holo", "reverse holo"},
		{"Staff Promo", "staff"},
		{"Prerelease", "prerelease"},
		{"Custom Variant", "Custom Variant"},
	}

	for _, tt := range tests {
		t.Run(tt.variant, func(t *testing.T) {
			qb := NewQueryBuilder().
				SetBase("Base Set", "Charizard", "4").
				WithVariant(tt.variant)
			result := qb.Build()
			if !strings.Contains(result, tt.contains) {
				t.Errorf("Expected query to contain '%s', got '%s'", tt.contains, result)
			}
		})
	}
}

func TestQueryBuilder_WithRegion(t *testing.T) {
	tests := []struct {
		region   string
		contains string
		language string
	}{
		{"Japan", "japanese", "Japanese"},
		{"Japanese", "japanese", "Japanese"},
		{"USA", "", "English"},
		{"Europe", "european", ""},
		{"Korea", "korean", "Korean"},
		{"Unknown Region", "Unknown Region", ""},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			qb := NewQueryBuilder().
				SetBase("Set", "Card", "1").
				WithRegion(tt.region)
			result := qb.Build()

			if tt.contains != "" && !strings.Contains(result, tt.contains) {
				t.Errorf("Expected query to contain '%s', got '%s'", tt.contains, result)
			}

			if tt.language != "" && qb.language != tt.language {
				t.Errorf("Expected language '%s', got '%s'", tt.language, qb.language)
			}
		})
	}
}

func TestQueryBuilder_WithLanguage(t *testing.T) {
	tests := []struct {
		language string
		contains string
	}{
		{"Japanese", "japanese"},
		{"Korean", "korean"},
		{"French", "french"},
		{"German", "german"},
		{"Spanish", "spanish"},
		{"Italian", "italian"},
		{"English", ""}, // English is default, no filter added
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			qb := NewQueryBuilder().
				SetBase("Set", "Card", "1").
				WithLanguage(tt.language)
			result := qb.Build()

			if tt.contains != "" && !strings.Contains(result, tt.contains) {
				t.Errorf("Expected query to contain '%s', got '%s'", tt.contains, result)
			}
		})
	}
}

func TestQueryBuilder_WithCondition(t *testing.T) {
	tests := []struct {
		condition string
		contains  string
	}{
		{"Mint", "mint"},
		{"Near Mint", "near mint"},
		{"Excellent", "excellent"},
		{"Good", "good"},
		{"Poor", "poor"},
		{"Graded", "graded"},
	}

	for _, tt := range tests {
		t.Run(tt.condition, func(t *testing.T) {
			qb := NewQueryBuilder().
				SetBase("Set", "Card", "1").
				WithCondition(tt.condition)
			result := qb.Build()

			if !strings.Contains(result, tt.contains) {
				t.Errorf("Expected query to contain '%s', got '%s'", tt.contains, result)
			}
		})
	}
}

func TestQueryBuilder_WithGrader(t *testing.T) {
	tests := []struct {
		grader   string
		contains string
	}{
		{"PSA", "PSA"},
		{"BGS", "BGS"},
		{"Beckett", "BGS"},
		{"CGC", "CGC"},
		{"SGC", "SGC"},
	}

	for _, tt := range tests {
		t.Run(tt.grader, func(t *testing.T) {
			qb := NewQueryBuilder().
				SetBase("Set", "Card", "1").
				WithGrader(tt.grader)
			result := qb.Build()

			if !strings.Contains(result, tt.contains) {
				t.Errorf("Expected query to contain '%s', got '%s'", tt.contains, result)
			}
		})
	}
}

func TestQueryBuilder_BuildWithConfidence(t *testing.T) {
	tests := []struct {
		name               string
		builder            *QueryBuilder
		expectedConfidence float64
		tolerance          float64
	}{
		{
			name:               "base query only",
			builder:            NewQueryBuilder().SetBase("Set", "Card", "1"),
			expectedConfidence: 0.7,
			tolerance:          0.05,
		},
		{
			name: "with variant",
			builder: NewQueryBuilder().
				SetBase("Set", "Card", "1").
				WithVariant("1st Edition"),
			expectedConfidence: 0.85,
			tolerance:          0.05,
		},
		{
			name: "with multiple filters",
			builder: NewQueryBuilder().
				SetBase("Set", "Card", "1").
				WithVariant("1st Edition").
				WithLanguage("Japanese").
				WithCondition("Mint"),
			expectedConfidence: 1.0,
			tolerance:          0.05,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, confidence := tt.builder.BuildWithConfidence()
			diff := confidence - tt.expectedConfidence
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tolerance {
				t.Errorf("Expected confidence ~%.2f, got %.2f", tt.expectedConfidence, confidence)
			}
		})
	}
}

func TestQueryBuilder_NormalizeSetName(t *testing.T) {
	qb := NewQueryBuilder()

	tests := []struct {
		input    string
		expected string
	}{
		{"Base Set: Unlimited", "Base Set Unlimited"},
		{"Sword-Shield", "Sword Shield"},
		{"Legends & Myths", "Legends & Myths"},
		{"SWSH01", "Sword Shield01"},
		{"SM Base", "Sun Moon Base"},
		{"SV01", "Scarlet Violet01"},
		{"  Extra  Spaces  ", "Extra  Spaces"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := qb.normalizeSetName(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestQueryBuilder_NormalizeCardName(t *testing.T) {
	qb := NewQueryBuilder()

	tests := []struct {
		input    string
		expected string
	}{
		{"Pikachu ex", "Pikachu ex"},
		{"Charizard VMAX", "Charizard VMAX"},
		{"Mewtwo V", "Mewtwo V"},
		{"Alakazam Prime", "Alakazam Prime"},
		{"Ho-Oh LEGEND", "Ho-Oh LEGEND"},
		{"Plain Card", "Plain Card"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := qb.normalizeCardName(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestBuildAdvancedQuery(t *testing.T) {
	tests := []struct {
		name     string
		setName  string
		cardName string
		number   string
		options  QueryOptions
		expected string
	}{
		{
			name:     "no options",
			setName:  "Surging Sparks",
			cardName: "Pikachu",
			number:   "250",
			options:  QueryOptions{},
			expected: "pokemon Surging Sparks Pikachu #250",
		},
		{
			name:     "with variant",
			setName:  "Base Set",
			cardName: "Charizard",
			number:   "4",
			options:  QueryOptions{Variant: "1st Edition"},
			expected: "pokemon Base Set Charizard #4 1st edition",
		},
		{
			name:     "with language",
			setName:  "VMAX Climax",
			cardName: "Charizard",
			number:   "3",
			options:  QueryOptions{Language: "Japanese"},
			expected: "pokemon VMAX Climax Charizard #3 japanese",
		},
		{
			name:     "with exact match",
			setName:  "Crown Zenith",
			cardName: "Pikachu",
			number:   "160",
			options:  QueryOptions{ExactMatch: true},
			expected: "\"pokemon Crown Zenith Pikachu #160\"",
		},
		{
			name:     "all options",
			setName:  "Base Set",
			cardName: "Blastoise",
			number:   "2",
			options: QueryOptions{
				Variant:   "Shadowless",
				Language:  "English",
				Condition: "Mint",
				Grader:    "PSA",
			},
			expected: "pokemon Base Set Blastoise #2 shadowless mint PSA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildAdvancedQuery(tt.setName, tt.cardName, tt.number, tt.options)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestQueryBuilder_EmptyBase(t *testing.T) {
	qb := NewQueryBuilder()
	result := qb.Build()
	if result != "" {
		t.Errorf("Expected empty string for empty base query, got '%s'", result)
	}
}
