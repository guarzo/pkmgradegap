package monitoring

import (
	"testing"
	"time"

	"github.com/guarzo/pkmgradegap/internal/ebay"
)

func TestNewAuctionAnalyzer(t *testing.T) {
	client := ebay.NewClient("test_app_id")

	config := AuctionAnalyzerConfig{
		EndingWithinMinutes:   60,
		MinProfitThresholdPct: 30.0,
		MaxBidThresholdUSD:    500.0,
	}

	analyzer := NewAuctionAnalyzer(client, config)

	if analyzer == nil {
		t.Fatal("NewAuctionAnalyzer returned nil")
	}

	if analyzer.config.GradingCostUSD != 20 {
		t.Errorf("Expected default GradingCostUSD to be 20, got %f", analyzer.config.GradingCostUSD)
	}

	if analyzer.config.EbayFeePct != 0.1295 {
		t.Errorf("Expected default EbayFeePct to be 0.1295, got %f", analyzer.config.EbayFeePct)
	}
}

func TestAuctionAnalyzer_IsCardMatch(t *testing.T) {
	client := ebay.NewClient("test_app_id")
	analyzer := NewAuctionAnalyzer(client, AuctionAnalyzerConfig{})

	tests := []struct {
		name     string
		title    string
		setName  string
		cardName string
		number   string
		expected bool
	}{
		{
			name:     "Exact card match with number",
			title:    "Pokemon Charizard Base Set #4/102 Holo",
			setName:  "Base Set",
			cardName: "Charizard",
			number:   "4",
			expected: true,
		},
		{
			name:     "Card match without number",
			title:    "Pokemon Pikachu Yellow Version Promo",
			setName:  "Promo",
			cardName: "Pikachu",
			number:   "",
			expected: true,
		},
		{
			name:     "Wrong card name",
			title:    "Pokemon Blastoise Base Set Holo",
			setName:  "Base Set",
			cardName: "Charizard",
			number:   "4",
			expected: false,
		},
		{
			name:     "Card match with different set",
			title:    "Pokemon Charizard Evolutions Holo",
			setName:  "Base Set",
			cardName: "Charizard",
			number:   "",
			expected: false, // Set doesn't match
		},
		{
			name:     "Case insensitive match",
			title:    "pokemon charizard base set holo",
			setName:  "Base Set",
			cardName: "Charizard",
			number:   "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.isCardMatch(tt.title, tt.setName, tt.cardName, tt.number)
			if result != tt.expected {
				t.Errorf("isCardMatch(%q, %q, %q, %q) = %v, expected %v",
					tt.title, tt.setName, tt.cardName, tt.number, result, tt.expected)
			}
		})
	}
}

func TestAuctionAnalyzer_ScoreAuction(t *testing.T) {
	client := ebay.NewClient("test_app_id")
	config := AuctionAnalyzerConfig{
		GradingCostUSD:  20.0,
		ShippingCostUSD: 5.0,
		EbayFeePct:      0.1295,
	}
	analyzer := NewAuctionAnalyzer(client, config)

	tests := []struct {
		name     string
		auction  ebay.Auction
		expected bool // whether opportunity should be created
	}{
		{
			name: "Profitable Charizard auction",
			auction: ebay.Auction{
				ItemID:       "123456789",
				Title:        "Pokemon Charizard Base Set 1st Edition",
				CurrentBid:   50.0,
				BidCount:     3,
				EndTime:      time.Now().Add(30 * time.Minute),
				Condition:    "Used",
				ShippingCost: 5.0,
				SellerRating: 98,
			},
			expected: true,
		},
		{
			name: "Low value common card",
			auction: ebay.Auction{
				ItemID:       "987654321",
				Title:        "Pokemon Energy Card",
				CurrentBid:   1.0,
				BidCount:     1,
				EndTime:      time.Now().Add(45 * time.Minute),
				Condition:    "Used",
				ShippingCost: 3.0,
				SellerRating: 95,
			},
			expected: false, // Low estimated value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opportunity := analyzer.scoreAuction(tt.auction)

			if tt.expected && opportunity == nil {
				t.Error("Expected opportunity to be created, but got nil")
			} else if !tt.expected && opportunity != nil {
				t.Error("Expected no opportunity, but got one")
			}

			if opportunity != nil {
				if opportunity.EstimatedValue <= tt.auction.CurrentBid {
					t.Errorf("Expected estimated value (%f) > current bid (%f)",
						opportunity.EstimatedValue, tt.auction.CurrentBid)
				}

				if opportunity.Risk == "" {
					t.Error("Risk assessment should be set")
				}

				if opportunity.ProfitScore == 0 {
					t.Error("Profit score should be calculated")
				}
			}
		})
	}
}

func TestAuctionAnalyzer_EstimateCardValue(t *testing.T) {
	client := ebay.NewClient("test_app_id")
	analyzer := NewAuctionAnalyzer(client, AuctionAnalyzerConfig{})

	tests := []struct {
		name           string
		auction        ebay.Auction
		expectedMinVal float64
	}{
		{
			name: "Charizard should get high multiplier",
			auction: ebay.Auction{
				Title:      "Pokemon Charizard Base Set Holo",
				CurrentBid: 100.0,
			},
			expectedMinVal: 200.0, // Should get at least 2x multiplier
		},
		{
			name: "First edition should get bonus",
			auction: ebay.Auction{
				Title:      "Pokemon Pikachu 1st Edition",
				CurrentBid: 50.0,
			},
			expectedMinVal: 75.0, // Should get 1.5x multiplier minimum
		},
		{
			name: "Holo card should get bonus",
			auction: ebay.Auction{
				Title:      "Pokemon Venusaur Holo Rare",
				CurrentBid: 30.0,
			},
			expectedMinVal: 60.0, // Should get 2x multiplier
		},
		{
			name: "Japanese card should get bonus",
			auction: ebay.Auction{
				Title:      "Pokemon Japanese Pikachu",
				CurrentBid: 25.0,
			},
			expectedMinVal: 30.0, // Should get 1.2x multiplier
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := analyzer.estimateCardValue(tt.auction)
			if value < tt.expectedMinVal {
				t.Errorf("Expected value >= %f, got %f", tt.expectedMinVal, value)
			}
		})
	}
}

func TestAuctionAnalyzer_AssessRisk(t *testing.T) {
	client := ebay.NewClient("test_app_id")
	analyzer := NewAuctionAnalyzer(client, AuctionAnalyzerConfig{})

	tests := []struct {
		name         string
		auction      ebay.Auction
		profitScore  float64
		expectedRisk string
	}{
		{
			name: "Low risk auction",
			auction: ebay.Auction{
				BidCount:     2,
				SellerRating: 98,
				EndTime:      time.Now().Add(2 * time.Hour),
			},
			profitScore:  50.0,
			expectedRisk: "LOW",
		},
		{
			name: "Medium risk auction",
			auction: ebay.Auction{
				BidCount:     8,
				SellerRating: 92,
				EndTime:      time.Now().Add(1 * time.Hour),
			},
			profitScore:  75.0,
			expectedRisk: "MEDIUM",
		},
		{
			name: "High risk auction",
			auction: ebay.Auction{
				BidCount:     15,
				SellerRating: 85,
				EndTime:      time.Now().Add(20 * time.Minute),
			},
			profitScore:  300.0, // Too good to be true
			expectedRisk: "HIGH",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			risk := analyzer.assessRisk(tt.auction, tt.profitScore)
			if risk != tt.expectedRisk {
				t.Errorf("Expected risk %s, got %s", tt.expectedRisk, risk)
			}
		})
	}
}

func TestContainsAnySubstring(t *testing.T) {
	tests := []struct {
		target     string
		substrings []string
		expected   bool
	}{
		{"Pokemon Charizard Base Set", []string{"charizard", "blastoise"}, true},
		{"Pokemon Pikachu Card", []string{"charizard", "blastoise"}, false},
		{"Normal Card", []string{"holo", "rare"}, false},
		{"Rare Holo Card", []string{"holo", "rare"}, true},
		{"japanese pokemon", []string{"japan", "jpn"}, true},
	}

	for _, tt := range tests {
		result := containsAnySubstring(tt.target, tt.substrings)
		if result != tt.expected {
			t.Errorf("containsAnySubstring(%q, %v) = %v, expected %v",
				tt.target, tt.substrings, result, tt.expected)
		}
	}
}