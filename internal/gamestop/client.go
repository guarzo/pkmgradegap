package gamestop

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/guarzo/pkmgradegap/internal/cache"
	"github.com/guarzo/pkmgradegap/internal/model"
	"github.com/guarzo/pkmgradegap/internal/ratelimit"
)

// GameStopClient implements the Provider interface for GameStop
type GameStopClient struct {
	config  Config
	client  *http.Client
	cache   *cache.Cache
	limiter *ratelimit.Limiter
}

// NewGameStopClient creates a new GameStop client
func NewGameStopClient(config Config) *GameStopClient {
	client := &http.Client{
		Timeout: config.RequestTimeout,
	}

	var c *cache.Cache
	if config.CacheEnabled {
		var err error
		c, err = cache.New("/tmp/gamestop_cache.json")
		if err != nil {
			// Continue without cache
			c = nil
		}
	}

	limiter := ratelimit.NewLimiter(config.RateLimitPerMin, time.Minute)

	return &GameStopClient{
		config:  config,
		client:  client,
		cache:   c,
		limiter: limiter,
	}
}

func (g *GameStopClient) Available() bool {
	return true // GameStop is always available with our bypass method
}

func (g *GameStopClient) GetProviderName() string {
	return "GameStop"
}

func (g *GameStopClient) GetListings(setName, cardName, number string) (*ListingData, error) {
	// Try cache first
	if g.cache != nil {
		var data ListingData
		key := fmt.Sprintf("gamestop:%s:%s:%s", setName, cardName, number)
		if found, _ := g.cache.Get(key, &data); found {
			return &data, nil
		}
	}

	// Build search query
	query := g.buildSearchQuery(setName, cardName, number)

	// Rate limit
	g.limiter.Wait()

	// Fetch listings
	listings, err := g.searchWithRetry(query)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Filter and process results
	filtered := g.filterRelevantListings(listings, setName, cardName, number)

	if len(filtered) == 0 {
		return &ListingData{
			Card: model.Card{
				Name:   cardName,
				Number: number,
			},
			ActiveList:   []Listing{},
			ListingCount: 0,
			LastUpdated:  time.Now(),
			DataSource:   "GameStop",
		}, nil
	}

	// Calculate stats
	data := g.calculateListingStats(filtered, setName, cardName, number)

	// Cache the result
	if g.cache != nil {
		key := fmt.Sprintf("gamestop:%s:%s:%s", setName, cardName, number)
		_ = g.cache.Put(key, data, time.Duration(g.config.CacheTTLMinutes)*time.Minute)
	}

	return data, nil
}

func (g *GameStopClient) SearchCards(query string) ([]Listing, error) {
	g.limiter.Wait()
	return g.searchWithRetry(query)
}

func (g *GameStopClient) GetBulkListings(cards []model.Card) (map[string]*ListingData, error) {
	results := make(map[string]*ListingData)

	for _, card := range cards {
		key := fmt.Sprintf("%s-%s", card.Name, card.Number)

		data, err := g.GetListings(card.SetName, card.Name, card.Number)
		if err != nil {
			// Log error but continue with other cards
			continue
		}

		results[key] = data

		// Small delay between requests to be respectful
		time.Sleep(g.config.RequestDelay)
	}

	return results, nil
}

func (g *GameStopClient) buildSearchQuery(setName, cardName, number string) string {
	// Build a search query that's likely to find the card
	parts := []string{"pokemon", "graded"}

	if setName != "" {
		parts = append(parts, setName)
	}
	if cardName != "" {
		parts = append(parts, cardName)
	}
	if number != "" {
		parts = append(parts, "#"+number)
	}

	return strings.Join(parts, " ")
}

func (g *GameStopClient) searchWithRetry(query string) ([]Listing, error) {
	var lastErr error

	for attempt := 0; attempt <= g.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := time.Duration(attempt*attempt) * time.Second
			time.Sleep(delay)
		}

		listings, err := g.performSearch(query)
		if err == nil {
			return listings, nil
		}

		lastErr = err
	}

	return nil, fmt.Errorf("search failed after %d attempts: %w", g.config.MaxRetries+1, lastErr)
}

func (g *GameStopClient) performSearch(query string) ([]Listing, error) {
	// Build search URL
	baseURL := "https://www.gamestop.com/graded-trading-cards/gradedcollectibles-cards-pokemon"
	searchURL := fmt.Sprintf("%s?q=%s&limit=%d", baseURL, url.QueryEscape(query), g.config.MaxSearchResults)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set realistic browser headers
	g.setBrowserHeaders(req)

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Handle compression
	reader, err := g.getReader(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to create reader: %w", err)
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse the HTML/JSON content
	return g.parseSearchResults(string(body))
}

func (g *GameStopClient) setBrowserHeaders(req *http.Request) {
	userAgent := g.config.UserAgents[0]
	if g.config.UseRandomUA && len(g.config.UserAgents) > 1 {
		userAgent = g.config.UserAgents[rand.Intn(len(g.config.UserAgents))]
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Referer", "https://www.google.com/")
}

func (g *GameStopClient) getReader(resp *http.Response) (io.Reader, error) {
	var reader io.Reader = resp.Body

	encoding := resp.Header.Get("Content-Encoding")
	switch encoding {
	case "gzip":
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		reader = gzipReader
	case "br":
		reader = brotli.NewReader(resp.Body)
	case "deflate":
		// TODO: Add deflate support if needed
		reader = resp.Body
	}

	return reader, nil
}

func (g *GameStopClient) parseSearchResults(html string) ([]Listing, error) {
	var listings []Listing

	// Look for the window.__INITIAL_STATE__ JSON data
	if initialState := g.extractInitialState(html); initialState != "" {
		if parsed := g.parseInitialStateJSON(initialState); len(parsed) > 0 {
			listings = append(listings, parsed...)
		}
	}

	// Also try to parse HTML product tiles as fallback
	if htmlParsed := g.parseHTMLProductTiles(html); len(htmlParsed) > 0 {
		listings = append(listings, htmlParsed...)
	}

	return g.deduplicateListings(listings), nil
}

func (g *GameStopClient) extractInitialState(html string) string {
	// Find window.__INITIAL_STATE__=...
	pattern := `window\.__INITIAL_STATE__\s*=\s*(\{.*?\});`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(html)

	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

func (g *GameStopClient) parseInitialStateJSON(jsonStr string) []Listing {
	var listings []Listing

	// Parse the JSON structure
	var state map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &state); err != nil {
		return listings
	}

	// Navigate through the JSON to find products
	// The structure might be something like: products.results or searchResults.products
	if products := g.extractProductsFromState(state); len(products) > 0 {
		for _, product := range products {
			if listing := g.convertProductToListing(product); listing != nil {
				listings = append(listings, *listing)
			}
		}
	}

	return listings
}

func (g *GameStopClient) extractProductsFromState(state map[string]interface{}) []map[string]interface{} {
	var products []map[string]interface{}

	// Try various paths where products might be stored
	paths := [][]string{
		{"products", "results"},
		{"searchResults", "products"},
		{"productSearch", "results"},
		{"search", "products"},
		{"catalog", "products"},
	}

	for _, path := range paths {
		if items := g.getNestedValue(state, path); items != nil {
			if productList, ok := items.([]interface{}); ok {
				for _, item := range productList {
					if product, ok := item.(map[string]interface{}); ok {
						products = append(products, product)
					}
				}
				break
			}
		}
	}

	return products
}

func (g *GameStopClient) getNestedValue(data map[string]interface{}, path []string) interface{} {
	current := data

	for i, key := range path {
		if value, exists := current[key]; exists {
			if i == len(path)-1 {
				return value
			}
			if nextMap, ok := value.(map[string]interface{}); ok {
				current = nextMap
			} else {
				return nil
			}
		} else {
			return nil
		}
	}

	return current
}

func (g *GameStopClient) convertProductToListing(product map[string]interface{}) *Listing {
	listing := &Listing{
		Seller:  "GameStop",
		InStock: true, // Default assumption
	}

	// Extract common fields
	if name, ok := product["name"].(string); ok {
		listing.Title = name
	}
	if title, ok := product["title"].(string); ok && listing.Title == "" {
		listing.Title = title
	}

	if sku, ok := product["sku"].(string); ok {
		listing.SKU = sku
	}
	if id, ok := product["id"].(string); ok && listing.SKU == "" {
		listing.SKU = id
	}

	// Extract price
	if price := g.extractPrice(product); price > 0 {
		listing.Price = price
	}

	// Extract grade from title
	listing.Grade = g.extractGradeFromTitle(listing.Title)

	// Extract URL
	if url, ok := product["url"].(string); ok {
		listing.URL = url
	} else if productId, ok := product["id"].(string); ok {
		listing.URL = fmt.Sprintf("https://www.gamestop.com/graded-trading-cards/gradedcollectibles-cards-pokemon/%s", productId)
	}

	// Extract image
	if image := g.extractImage(product); image != "" {
		listing.ImageURL = image
	}

	// Extract stock status
	if stock := g.extractStockStatus(product); stock != nil {
		listing.InStock = *stock
	}

	// Only return if we have essential data
	if listing.Title != "" && listing.Price > 0 {
		return listing
	}

	return nil
}

func (g *GameStopClient) extractPrice(product map[string]interface{}) float64 {
	// Try various price field names
	priceFields := []string{"price", "salePrice", "listPrice", "currentPrice", "amount"}

	for _, field := range priceFields {
		if priceVal, exists := product[field]; exists {
			switch p := priceVal.(type) {
			case float64:
				return p
			case string:
				if parsed, err := strconv.ParseFloat(strings.TrimPrefix(p, "$"), 64); err == nil {
					return parsed
				}
			case map[string]interface{}:
				// Price might be nested (e.g., {amount: 99.99, currency: "USD"})
				if amount, ok := p["amount"].(float64); ok {
					return amount
				}
				if value, ok := p["value"].(float64); ok {
					return value
				}
			}
		}
	}

	return 0
}

func (g *GameStopClient) extractGradeFromTitle(title string) string {
	// Common grading patterns
	patterns := []string{
		`PSA\s+(\d+(?:\.\d+)?)`,
		`BGS\s+(\d+(?:\.\d+)?)`,
		`CGC\s+(\d+(?:\.\d+)?)`,
		`SGC\s+(\d+(?:\.\d+)?)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		if matches := re.FindStringSubmatch(title); len(matches) > 1 {
			return strings.ToUpper(re.FindString(title))
		}
	}

	return "Unknown"
}

func (g *GameStopClient) extractImage(product map[string]interface{}) string {
	imageFields := []string{"image", "imageUrl", "thumbnail", "picture"}

	for _, field := range imageFields {
		if imageVal, exists := product[field]; exists {
			if imageStr, ok := imageVal.(string); ok {
				return imageStr
			}
			if imageMap, ok := imageVal.(map[string]interface{}); ok {
				if url, ok := imageMap["url"].(string); ok {
					return url
				}
			}
		}
	}

	return ""
}

func (g *GameStopClient) extractStockStatus(product map[string]interface{}) *bool {
	stockFields := []string{"inStock", "available", "availability", "stock"}

	for _, field := range stockFields {
		if stockVal, exists := product[field]; exists {
			switch s := stockVal.(type) {
			case bool:
				return &s
			case string:
				inStock := strings.ToLower(s) == "true" || strings.ToLower(s) == "available"
				return &inStock
			}
		}
	}

	return nil
}

func (g *GameStopClient) parseHTMLProductTiles(html string) []Listing {
	// HTML parsing fallback - this would require more complex parsing
	// For now, return empty slice as JSON parsing should be sufficient
	return []Listing{}
}

func (g *GameStopClient) deduplicateListings(listings []Listing) []Listing {
	seen := make(map[string]bool)
	var unique []Listing

	for _, listing := range listings {
		key := fmt.Sprintf("%s-%s-%.2f", listing.SKU, listing.Title, listing.Price)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, listing)
		}
	}

	return unique
}

func (g *GameStopClient) filterRelevantListings(listings []Listing, setName, cardName, number string) []Listing {
	var filtered []Listing

	setLower := strings.ToLower(setName)
	cardLower := strings.ToLower(cardName)

	for _, listing := range listings {
		titleLower := strings.ToLower(listing.Title)

		// Check if listing matches the card
		hasCard := cardName == "" || strings.Contains(titleLower, cardLower)
		hasSet := setName == "" || strings.Contains(titleLower, setLower)
		hasNumber := number == "" || strings.Contains(listing.Title, "#"+number) || strings.Contains(listing.Title, number)

		// Must match card name and either set or number
		if hasCard && (hasSet || hasNumber) {
			filtered = append(filtered, listing)
		}
	}

	// Limit results
	if len(filtered) > g.config.MaxListingsPerCard {
		filtered = filtered[:g.config.MaxListingsPerCard]
	}

	return filtered
}

func (g *GameStopClient) calculateListingStats(listings []Listing, setName, cardName, number string) *ListingData {
	if len(listings) == 0 {
		return &ListingData{
			Card: model.Card{
				Name:   cardName,
				Number: number,
			},
			ActiveList:   []Listing{},
			ListingCount: 0,
			LastUpdated:  time.Now(),
			DataSource:   "GameStop",
		}
	}

	var total float64
	var lowest float64 = listings[0].Price

	for _, listing := range listings {
		total += listing.Price
		if listing.Price < lowest {
			lowest = listing.Price
		}
	}

	return &ListingData{
		Card: model.Card{
			Name:   cardName,
			Number: number,
		},
		ActiveList:   listings,
		LowestPrice:  lowest,
		AveragePrice: total / float64(len(listings)),
		ListingCount: len(listings),
		LastUpdated:  time.Now(),
		DataSource:   "GameStop",
	}
}