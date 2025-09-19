package marketplace

import (
	"time"
)

// MarketplaceProvider defines the interface for accessing marketplace data
type MarketplaceProvider interface {
	Available() bool
	GetProviderName() string
	GetActiveListings(productID string) (*MarketListings, error)
	GetPriceDistribution(productID string) (*PriceStats, error)
	GetSellerMetrics(productID string) (*SellerData, error)
}

// MarketListings represents active marketplace listings for a product
type MarketListings struct {
	ProductID         string    `json:"product_id"`
	TotalListings     int       `json:"total_listings"`
	LowestPriceCents  int       `json:"lowest_price_cents"`
	HighestPriceCents int       `json:"highest_price_cents"`
	AveragePriceCents int       `json:"average_price_cents"`
	MedianPriceCents  int       `json:"median_price_cents"`
	Listings          []Listing `json:"listings"`
	LastUpdated       time.Time `json:"last_updated"`
}

// Listing represents a single marketplace listing
type Listing struct {
	SellerID      string    `json:"seller_id"`
	SellerName    string    `json:"seller_name"`
	PriceCents    int       `json:"price_cents"`
	Condition     string    `json:"condition"`
	Quantity      int       `json:"quantity"`
	ShippingCents int       `json:"shipping_cents"`
	ListedDate    time.Time `json:"listed_date"`
	URL           string    `json:"url"`
}

// PriceStats represents price distribution statistics
type PriceStats struct {
	ProductID         string     `json:"product_id"`
	Mean              int        `json:"mean"`
	Median            int        `json:"median"`
	Mode              int        `json:"mode"`
	StandardDeviation float64    `json:"standard_deviation"`
	Percentile25      int        `json:"percentile_25"`
	Percentile75      int        `json:"percentile_75"`
	Percentile90      int        `json:"percentile_90"`
	PriceRange        PriceRange `json:"price_range"`
	OutlierCount      int        `json:"outlier_count"`
	LastUpdated       time.Time  `json:"last_updated"`
}

// PriceRange represents the min and max price range
type PriceRange struct {
	MinCents int `json:"min_cents"`
	MaxCents int `json:"max_cents"`
}

// SellerData represents seller metrics for a product
type SellerData struct {
	ProductID        string         `json:"product_id"`
	TotalSellers     int            `json:"total_sellers"`
	TopSellers       []SellerMetric `json:"top_sellers"`
	AverageInventory float64        `json:"average_inventory"`
	CompetitionLevel string         `json:"competition_level"` // LOW, MEDIUM, HIGH
	MarketDominance  float64        `json:"market_dominance"`  // % of market by top seller
	LastUpdated      time.Time      `json:"last_updated"`
}

// SellerMetric represents metrics for a single seller
type SellerMetric struct {
	SellerID          string  `json:"seller_id"`
	SellerName        string  `json:"seller_name"`
	ListingCount      int     `json:"listing_count"`
	AveragePriceCents int     `json:"average_price_cents"`
	MarketShare       float64 `json:"market_share"` // percentage
	Rating            float64 `json:"rating"`
	SalesVelocity     float64 `json:"sales_velocity"` // sales per day
}

// MarketAnalysis represents comprehensive market analysis
type MarketAnalysis struct {
	ProductID           string    `json:"product_id"`
	ListingVelocity     float64   `json:"listing_velocity"`      // new listings per day
	SalesVelocity       float64   `json:"sales_velocity"`        // sales per day
	DaysOnMarket        float64   `json:"days_on_market"`        // average days to sell
	PriceVolatility     float64   `json:"price_volatility"`      // coefficient of variation
	SupplyDemandRatio   float64   `json:"supply_demand_ratio"`   // listings/sales
	OptimalListingPrice int       `json:"optimal_listing_price"` // recommended price
	PriceAnomalies      []Anomaly `json:"price_anomalies"`
	LastUpdated         time.Time `json:"last_updated"`
}

// Anomaly represents a pricing anomaly
type Anomaly struct {
	Type        string    `json:"type"` // UNDERPRICED, OVERPRICED, OUTLIER
	PriceCents  int       `json:"price_cents"`
	Deviation   float64   `json:"deviation"` // % from mean
	SellerID    string    `json:"seller_id"`
	DetectedAt  time.Time `json:"detected_at"`
	Description string    `json:"description"`
}

// MarketTiming represents timing recommendations
type MarketTiming struct {
	ProductID          string              `json:"product_id"`
	CurrentTrend       string              `json:"current_trend"` // BULLISH, BEARISH, NEUTRAL
	BestBuyTime        string              `json:"best_buy_time"` // e.g., "Weekend mornings"
	BestSellTime       string              `json:"best_sell_time"`
	SeasonalFactors    []SeasonalFactor    `json:"seasonal_factors"`
	EventDrivenFactors []EventDrivenFactor `json:"event_driven_factors"`
	Recommendation     string              `json:"recommendation"`
	Confidence         float64             `json:"confidence"` // 0.0 to 1.0
	LastUpdated        time.Time           `json:"last_updated"`
}

// SeasonalFactor represents seasonal price influences
type SeasonalFactor struct {
	Period      string  `json:"period"` // e.g., "December", "Q1", "Summer"
	Impact      float64 `json:"impact"` // percentage change
	Description string  `json:"description"`
}

// EventDrivenFactor represents event-based price influences
type EventDrivenFactor struct {
	EventType   string    `json:"event_type"` // TOURNAMENT, RELEASE, ANNIVERSARY
	EventDate   time.Time `json:"event_date"`
	Impact      float64   `json:"impact"` // percentage change
	Description string    `json:"description"`
}
