package model

type Set struct {
	ID   string
	Name string
}

// Minimal card representation we need for pricing/analysis.
// Add fields as you harden matching (rarity, variant flags, etc.).
type Card struct {
	ID         string
	Name       string
	SetID      string
	SetName    string
	Number     string
	Rarity     string
	TCGPlayer  *TCGPlayerBlock   // may be nil
	Cardmarket *CardmarketBlock  // may be nil
}

type TCGPlayerBlock struct {
	URL     string
	Updated string
	// We only care about the market price. Add more if you want low/mid/high.
	Prices map[string]struct {
		Low       *float64 `json:"low,omitempty"`
		Mid       *float64 `json:"mid,omitempty"`
		High      *float64 `json:"high,omitempty"`
		Market    *float64 `json:"market,omitempty"`
		DirectLow *float64 `json:"directLow,omitempty"`
	}
}

type CardmarketBlock struct {
	URL     string
	Updated string
	Prices struct {
		AverageSellPrice    *float64 `json:"averageSellPrice,omitempty"`
		TrendPrice          *float64 `json:"trendPrice,omitempty"`
		ReverseHoloTrend    *float64 `json:"reverseHoloTrend,omitempty"`
		Avg7                *float64 `json:"avg7,omitempty"`
		Avg30               *float64 `json:"avg30,omitempty"`
		ReverseHoloAvg7     *float64 `json:"reverseHoloAvg7,omitempty"`
		ReverseHoloAvg30    *float64 `json:"reverseHoloAvg30,omitempty"`
	} `json:"prices"`
}