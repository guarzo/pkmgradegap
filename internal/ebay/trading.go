package ebay

import (
	"time"
)

// TradingClient handles eBay Trading API and Inventory API operations
type TradingClient struct {
	tradingAPI   *TradingAPIClient
	oauthManager *OAuthManager
	sandbox      bool
}

// UserListing represents a user's active eBay listing
type UserListing struct {
	ItemID         string    `json:"itemId"`
	SKU            string    `json:"sku"`
	Title          string    `json:"title"`
	CardName       string    `json:"cardName"`
	SetName        string    `json:"setName"`
	CardNumber     string    `json:"cardNumber"`
	CurrentPrice   float64   `json:"currentPrice"`
	Quantity       int       `json:"quantity"`
	ViewCount      int       `json:"viewCount"`
	WatchCount     int       `json:"watchCount"`
	ListingURL     string    `json:"listingUrl"`
	ImageURL       string    `json:"imageUrl"`
	StartTime      time.Time `json:"startTime"`
	EndTime        time.Time `json:"endTime"`
	ListingType    string    `json:"listingType"`
	Condition      string    `json:"condition"`
	CategoryID     string    `json:"categoryId"`
	LastModified   time.Time `json:"lastModified"`
	ListingStatus  string    `json:"listingStatus"`
	SoldQuantity   int       `json:"soldQuantity"`
	TimeLeft       string    `json:"timeLeft"`
	PrimaryCatName string    `json:"primaryCategoryName"`
	DaysActive     int       `json:"daysActive"`
}

// ListingSummary provides overview statistics
type ListingSummary struct {
	TotalActive      int     `json:"totalActive"`
	TotalViews       int     `json:"totalViews"`
	TotalWatchers    int     `json:"totalWatchers"`
	TotalValue       float64 `json:"totalValue"`
	AvgDaysListed    float64 `json:"avgDaysListed"`
	SoldThisMonth    int     `json:"soldThisMonth"`
	RevenueThisMonth float64 `json:"revenueThisMonth"`
}

// NewTradingClient creates a new Trading API client
func NewTradingClient(oauthManager *OAuthManager, appID string, sandbox bool) *TradingClient {
	return &TradingClient{
		tradingAPI:   NewTradingAPIClient(oauthManager, appID, sandbox),
		oauthManager: oauthManager,
		sandbox:      sandbox,
	}
}

// GetMyListings fetches user's active eBay listings using Trading API
func (c *TradingClient) GetMyListings(userID string, limit int, offset int) ([]UserListing, error) {
	// Calculate page number from offset
	page := (offset / limit) + 1
	if page < 1 {
		page = 1
	}

	// Use the new Trading API client
	return c.tradingAPI.GetMyListings(userID, page, limit)
}

// This method is no longer needed - Trading API provides all details

// UpdateListingPrice updates the price of a listing using Trading API
func (c *TradingClient) UpdateListingPrice(userID, itemID string, newPrice float64) error {
	return c.tradingAPI.UpdateListingPrice(userID, itemID, newPrice)
}

// BulkUpdatePrices updates multiple listing prices
func (c *TradingClient) BulkUpdatePrices(userID string, updates map[string]float64) (map[string]error, error) {
	results := make(map[string]error)

	for itemID, newPrice := range updates {
		err := c.UpdateListingPrice(userID, itemID, newPrice)
		if err != nil {
			results[itemID] = err
		}
	}

	return results, nil
}

// GetListingSummary returns overview statistics for user's listings
func (c *TradingClient) GetListingSummary(userID string) (*ListingSummary, error) {
	return c.tradingAPI.GetListingSummary(userID)
}

// This method is now implemented in TradingAPIClient

// GetCompetitorPrices fetches competitor prices for similar items
func (c *TradingClient) GetCompetitorPrices(title string, condition string) ([]float64, error) {
	// This would use the Finding API (already implemented in ebay.go)
	// Reuse existing Client for Finding API calls

	// For now, return empty slice (would integrate with existing Finding API)
	return []float64{}, nil
}
