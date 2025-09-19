package marketplace

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/guarzo/pkmgradegap/internal/cache"
	"golang.org/x/time/rate"
)

// PriceChartingMarketplace implements MarketplaceProvider using PriceCharting API
type PriceChartingMarketplace struct {
	apiKey      string
	httpClient  *http.Client
	baseURL     string
	cache       *cache.Cache
	rateLimiter *rate.Limiter
}

// NewPriceChartingMarketplace creates a new PriceCharting marketplace provider
func NewPriceChartingMarketplace(apiKey string, c *cache.Cache) *PriceChartingMarketplace {
	if apiKey == "" || apiKey == "test" || apiKey == "mock" {
		return nil
	}

	return &PriceChartingMarketplace{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:     "https://www.pricecharting.com",
		cache:       c,
		rateLimiter: rate.NewLimiter(rate.Every(time.Second), 5), // 5 requests per second
	}
}

// Available returns true if the provider is configured and available
func (p *PriceChartingMarketplace) Available() bool {
	return p != nil && p.apiKey != "" && p.apiKey != "test" && p.apiKey != "mock"
}

// GetProviderName returns the provider name
func (p *PriceChartingMarketplace) GetProviderName() string {
	return "PriceCharting Marketplace"
}

// GetActiveListings fetches active marketplace listings for a product
func (p *PriceChartingMarketplace) GetActiveListings(productID string) (*MarketListings, error) {
	if !p.Available() {
		return nil, fmt.Errorf("marketplace provider not available")
	}

	// Check cache first
	cacheKey := fmt.Sprintf("marketplace:listings:%s", productID)
	if cached := p.getCached(cacheKey); cached != nil {
		if listings, ok := cached.(*MarketListings); ok {
			return listings, nil
		}
	}

	// Wait for rate limiter with proper context
	ctx := context.Background()
	if err := p.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Make API request to offers endpoint
	url := fmt.Sprintf("%s/api/offers?id=%s&api_key=%s", p.baseURL, productID, p.apiKey)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var apiResponse struct {
		Status string `json:"status"`
		Offers []struct {
			SellerID   string  `json:"seller-id"`
			SellerName string  `json:"seller-name"`
			Price      float64 `json:"price"`
			Shipping   float64 `json:"shipping"`
			Condition  string  `json:"condition"`
			Quantity   int     `json:"quantity"`
			ListedDate string  `json:"listed-date"`
			URL        string  `json:"url"`
		} `json:"offers"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Convert to our format
	listings := &MarketListings{
		ProductID:     productID,
		TotalListings: len(apiResponse.Offers),
		Listings:      make([]Listing, 0, len(apiResponse.Offers)),
		LastUpdated:   time.Now(),
	}

	var totalPrice, lowestPrice, highestPrice int
	prices := make([]int, 0, len(apiResponse.Offers))

	for _, offer := range apiResponse.Offers {
		priceCents := int(offer.Price * 100)
		shippingCents := int(offer.Shipping * 100)

		listing := Listing{
			SellerID:      offer.SellerID,
			SellerName:    offer.SellerName,
			PriceCents:    priceCents,
			Condition:     offer.Condition,
			Quantity:      offer.Quantity,
			ShippingCents: shippingCents,
			URL:           offer.URL,
		}

		// Parse listed date
		if offer.ListedDate != "" {
			if t, err := time.Parse("2006-01-02", offer.ListedDate); err == nil {
				listing.ListedDate = t
			}
		}

		listings.Listings = append(listings.Listings, listing)

		totalPrice += priceCents
		prices = append(prices, priceCents)

		if lowestPrice == 0 || priceCents < lowestPrice {
			lowestPrice = priceCents
		}
		if priceCents > highestPrice {
			highestPrice = priceCents
		}
	}

	if len(prices) > 0 {
		listings.LowestPriceCents = lowestPrice
		listings.HighestPriceCents = highestPrice
		listings.AveragePriceCents = totalPrice / len(prices)

		// Calculate median
		sort.Ints(prices)
		if len(prices)%2 == 0 {
			listings.MedianPriceCents = (prices[len(prices)/2-1] + prices[len(prices)/2]) / 2
		} else {
			listings.MedianPriceCents = prices[len(prices)/2]
		}
	}

	// Cache the result
	p.setCache(cacheKey, listings, 15*time.Minute)

	return listings, nil
}

// GetPriceDistribution calculates price distribution statistics
func (p *PriceChartingMarketplace) GetPriceDistribution(productID string) (*PriceStats, error) {
	// Get listings first
	listings, err := p.GetActiveListings(productID)
	if err != nil {
		return nil, err
	}

	if len(listings.Listings) == 0 {
		return nil, fmt.Errorf("no listings available for price distribution")
	}

	prices := make([]float64, 0, len(listings.Listings))
	for _, listing := range listings.Listings {
		prices = append(prices, float64(listing.PriceCents))
	}

	sort.Float64s(prices)

	stats := &PriceStats{
		ProductID:   productID,
		LastUpdated: time.Now(),
	}

	// Calculate mean
	var sum float64
	for _, price := range prices {
		sum += price
	}
	stats.Mean = int(sum / float64(len(prices)))

	// Calculate median
	if len(prices)%2 == 0 {
		stats.Median = int((prices[len(prices)/2-1] + prices[len(prices)/2]) / 2)
	} else {
		stats.Median = int(prices[len(prices)/2])
	}

	// Calculate mode (most frequent price)
	priceCount := make(map[int]int)
	for _, price := range prices {
		priceCount[int(price)]++
	}
	maxCount := 0
	for price, count := range priceCount {
		if count > maxCount {
			maxCount = count
			stats.Mode = price
		}
	}

	// Calculate standard deviation
	var variance float64
	mean := float64(stats.Mean)
	for _, price := range prices {
		variance += math.Pow(price-mean, 2)
	}
	variance /= float64(len(prices))
	stats.StandardDeviation = math.Sqrt(variance)

	// Calculate percentiles
	stats.Percentile25 = int(prices[len(prices)*25/100])
	stats.Percentile75 = int(prices[len(prices)*75/100])
	stats.Percentile90 = int(prices[len(prices)*90/100])

	// Set price range
	stats.PriceRange = PriceRange{
		MinCents: int(prices[0]),
		MaxCents: int(prices[len(prices)-1]),
	}

	// Count outliers (prices beyond 1.5 * IQR)
	iqr := float64(stats.Percentile75 - stats.Percentile25)
	lowerBound := float64(stats.Percentile25) - 1.5*iqr
	upperBound := float64(stats.Percentile75) + 1.5*iqr
	for _, price := range prices {
		if price < lowerBound || price > upperBound {
			stats.OutlierCount++
		}
	}

	return stats, nil
}

// GetSellerMetrics calculates seller metrics for a product
func (p *PriceChartingMarketplace) GetSellerMetrics(productID string) (*SellerData, error) {
	// Get listings first
	listings, err := p.GetActiveListings(productID)
	if err != nil {
		return nil, err
	}

	if len(listings.Listings) == 0 {
		return nil, fmt.Errorf("no listings available for seller metrics")
	}

	// Aggregate seller data
	sellerStats := make(map[string]*SellerMetric)
	for _, listing := range listings.Listings {
		if seller, exists := sellerStats[listing.SellerID]; exists {
			seller.ListingCount++
			seller.AveragePriceCents = (seller.AveragePriceCents*seller.ListingCount + listing.PriceCents) / (seller.ListingCount + 1)
		} else {
			sellerStats[listing.SellerID] = &SellerMetric{
				SellerID:          listing.SellerID,
				SellerName:        listing.SellerName,
				ListingCount:      1,
				AveragePriceCents: listing.PriceCents,
			}
		}
	}

	// Calculate market share
	totalListings := len(listings.Listings)
	for _, seller := range sellerStats {
		seller.MarketShare = float64(seller.ListingCount) / float64(totalListings) * 100
	}

	// Sort sellers by listing count
	sellers := make([]*SellerMetric, 0, len(sellerStats))
	for _, seller := range sellerStats {
		sellers = append(sellers, seller)
	}
	sort.Slice(sellers, func(i, j int) bool {
		return sellers[i].ListingCount > sellers[j].ListingCount
	})

	// Take top 10 sellers
	topSellers := sellers
	if len(topSellers) > 10 {
		topSellers = topSellers[:10]
	}

	// Calculate average inventory
	var totalInventory int
	for _, seller := range sellers {
		totalInventory += seller.ListingCount
	}
	avgInventory := float64(totalInventory) / float64(len(sellers))

	// Determine competition level
	competitionLevel := "LOW"
	if len(sellers) > 5 {
		competitionLevel = "MEDIUM"
	}
	if len(sellers) > 10 {
		competitionLevel = "HIGH"
	}

	// Calculate market dominance (top seller's market share)
	marketDominance := 0.0
	if len(sellers) > 0 {
		marketDominance = sellers[0].MarketShare
	}

	return &SellerData{
		ProductID:        productID,
		TotalSellers:     len(sellerStats),
		TopSellers:       convertToSellerMetrics(topSellers),
		AverageInventory: avgInventory,
		CompetitionLevel: competitionLevel,
		MarketDominance:  marketDominance,
		LastUpdated:      time.Now(),
	}, nil
}

// Helper function to convert seller metrics
func convertToSellerMetrics(sellers []*SellerMetric) []SellerMetric {
	result := make([]SellerMetric, len(sellers))
	for i, seller := range sellers {
		result[i] = *seller
	}
	return result
}

// getCached retrieves a cached value
func (p *PriceChartingMarketplace) getCached(key string) interface{} {
	if p.cache == nil {
		return nil
	}

	var data interface{}
	if found, _ := p.cache.Get(key, &data); found {
		return data
	}
	return nil
}

// setCache stores a value in cache
func (p *PriceChartingMarketplace) setCache(key string, value interface{}, ttl time.Duration) {
	if p.cache != nil {
		_ = p.cache.Put(key, value, ttl)
	}
}
