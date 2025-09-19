package ebay

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Listing struct {
	Title     string
	URL       string
	Price     float64
	Condition string
	EndTime   time.Time
	BidCount  int
	BuyItNow  bool
}

// Auction represents an ending auction with profit potential
type Auction struct {
	ItemID        string
	Title         string
	URL           string
	CurrentBid    float64
	BidCount      int
	EndTime       time.Time
	EstimatedValue float64
	Condition     string
	ShippingCost  float64
	SellerRating  int
	WatchCount    int
	Category      string
}

// AuctionDetail provides comprehensive auction information
type AuctionDetail struct {
	Auction
	Description   string
	Images        []string
	SellerInfo    SellerInfo
	ShippingInfo  ShippingInfo
	BidHistory    []Bid
	LastUpdated   time.Time
}

// SellerInfo contains seller details for trust assessment
type SellerInfo struct {
	Username       string
	FeedbackScore  int
	PositivePercent float64
	PowerSeller    bool
	TopRated       bool
}

// ShippingInfo contains shipping details for cost calculation
type ShippingInfo struct {
	Cost         float64
	Service      string
	HandlingTime int
	Returns      bool
}

// Bid represents a bid in the auction history
type Bid struct {
	Amount    float64
	Bidder    string
	Timestamp time.Time
}

type Client struct {
	appID       string
	httpClient  *http.Client
	rateLimiter *rateLimiter
}

// Simple rate limiter
type rateLimiter struct {
	mu       sync.Mutex
	lastCall time.Time
	minDelay time.Duration
}

// eBay Finding API response structures
type findingResponse struct {
	FindItemsAdvancedResponse []struct {
		SearchResult []struct {
			Item []struct {
				ItemID      []string `json:"itemId"`
				Title       []string `json:"title"`
				ViewItemURL []string `json:"viewItemURL"`
				ListingType []string `json:"listingType"`
				Condition   []struct {
					ConditionDisplayName []string `json:"conditionDisplayName"`
				} `json:"condition"`
				SellingStatus []struct {
					CurrentPrice []struct {
						Value      []string `json:"__value__"`
						CurrencyID []string `json:"@currencyId"`
					} `json:"currentPrice"`
					BidCount []string `json:"bidCount"`
				} `json:"sellingStatus"`
				ListingInfo []struct {
					EndTime []string `json:"endTime"`
				} `json:"listingInfo"`
			} `json:"item"`
		} `json:"searchResult"`
	} `json:"findItemsAdvancedResponse"`
}

func NewClient(appID string) *Client {
	return &Client{
		appID:      appID,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		rateLimiter: &rateLimiter{
			minDelay: 1 * time.Second, // eBay Finding API has 5000 calls/day limit = ~1 call per 17 seconds, but we'll be conservative
		},
	}
}

func (r *rateLimiter) wait() {
	r.mu.Lock()
	defer r.mu.Unlock()

	timeSinceLastCall := time.Since(r.lastCall)
	if timeSinceLastCall < r.minDelay {
		time.Sleep(r.minDelay - timeSinceLastCall)
	}
	r.lastCall = time.Now()
}

func (c *Client) Available() bool {
	return c.appID != ""
}

func (c *Client) SearchRawListings(setName, cardName, number string, max int) ([]Listing, error) {
	if !c.Available() {
		return nil, fmt.Errorf("eBay app ID not configured")
	}

	// Apply rate limiting
	c.rateLimiter.wait()

	// Build a more targeted query
	query := fmt.Sprintf("pokemon \"%s\" \"%s\" #%s -(graded,slab,psa,bgs,cgc,ace)",
		setName, cardName, number)

	// eBay Finding API endpoint
	endpoint := "https://svcs.ebay.com/services/search/FindingService/v1"

	params := url.Values{}
	params.Set("OPERATION-NAME", "findItemsAdvanced")
	params.Set("SERVICE-VERSION", "1.0.0")
	params.Set("SECURITY-APPNAME", c.appID)
	params.Set("RESPONSE-DATA-FORMAT", "JSON")
	params.Set("keywords", query)
	params.Set("categoryId", "183454") // Trading Card Games category

	// Filters
	params.Set("itemFilter(0).name", "Condition")
	params.Set("itemFilter(0).value(0)", "New")
	params.Set("itemFilter(0).value(1)", "Used")

	params.Set("itemFilter(1).name", "ListingType")
	params.Set("itemFilter(1).value(0)", "All")

	params.Set("itemFilter(2).name", "ExcludeCategory")
	params.Set("itemFilter(2).value(0)", "267") // Exclude Books & Magazines

	params.Set("paginationInput.entriesPerPage", strconv.Itoa(max))
	params.Set("sortOrder", "BestMatch")

	// Execute request
	fullURL := endpoint + "?" + params.Encode()

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Add required headers
	req.Header.Set("X-EBAY-SOA-SERVICE-NAME", "FindingService")
	req.Header.Set("X-EBAY-SOA-OPERATION-NAME", "findItemsAdvanced")
	req.Header.Set("X-EBAY-SOA-SERVICE-VERSION", "1.0.0")
	req.Header.Set("X-EBAY-SOA-SECURITY-APPNAME", c.appID)
	req.Header.Set("X-EBAY-SOA-RESPONSE-DATA-FORMAT", "JSON")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eBay API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Try to parse error response
		var errorResp struct {
			ErrorMessage []struct {
				Error []struct {
					Message []string `json:"message"`
				} `json:"error"`
			} `json:"errorMessage"`
		}

		if err := json.Unmarshal(bodyBytes, &errorResp); err == nil &&
			len(errorResp.ErrorMessage) > 0 &&
			len(errorResp.ErrorMessage[0].Error) > 0 &&
			len(errorResp.ErrorMessage[0].Error[0].Message) > 0 {
			errMsg := errorResp.ErrorMessage[0].Error[0].Message[0]
			if strings.Contains(errMsg, "exceeded the number of times") {
				return nil, fmt.Errorf("eBay API rate limit exceeded. Please try again later")
			}
			return nil, fmt.Errorf("eBay API error: %s", errMsg)
		}

		return nil, fmt.Errorf("eBay API returned status %d", resp.StatusCode)
	}

	var ebayResp findingResponse
	if err := json.Unmarshal(bodyBytes, &ebayResp); err != nil {
		return nil, fmt.Errorf("parse eBay response: %w", err)
	}

	// Parse listings from response
	var listings []Listing

	if len(ebayResp.FindItemsAdvancedResponse) > 0 &&
		len(ebayResp.FindItemsAdvancedResponse[0].SearchResult) > 0 {

		searchResult := ebayResp.FindItemsAdvancedResponse[0].SearchResult[0]

		for _, item := range searchResult.Item {
			listing, err := c.parseItem(item)
			if err != nil {
				continue // Skip malformed items
			}

			// Filter out graded cards by title
			if c.isGradedCard(listing.Title) {
				continue
			}

			listings = append(listings, listing)
		}
	}

	// Sort to prefer Buy It Now over auctions
	c.sortByListingType(listings)

	// Return only the requested number of listings
	if len(listings) > max {
		listings = listings[:max]
	}

	return listings, nil
}

func (c *Client) parseItem(item struct {
	ItemID      []string `json:"itemId"`
	Title       []string `json:"title"`
	ViewItemURL []string `json:"viewItemURL"`
	ListingType []string `json:"listingType"`
	Condition   []struct {
		ConditionDisplayName []string `json:"conditionDisplayName"`
	} `json:"condition"`
	SellingStatus []struct {
		CurrentPrice []struct {
			Value      []string `json:"__value__"`
			CurrencyID []string `json:"@currencyId"`
		} `json:"currentPrice"`
		BidCount []string `json:"bidCount"`
	} `json:"sellingStatus"`
	ListingInfo []struct {
		EndTime []string `json:"endTime"`
	} `json:"listingInfo"`
}) (Listing, error) {
	listing := Listing{}

	// Extract title
	if len(item.Title) > 0 {
		listing.Title = item.Title[0]
	}

	// Extract URL
	if len(item.ViewItemURL) > 0 {
		listing.URL = item.ViewItemURL[0]
	}

	// Extract condition
	if len(item.Condition) > 0 && len(item.Condition[0].ConditionDisplayName) > 0 {
		listing.Condition = item.Condition[0].ConditionDisplayName[0]
	}

	// Extract price and bid count
	if len(item.SellingStatus) > 0 {
		sellingStatus := item.SellingStatus[0]

		if len(sellingStatus.CurrentPrice) > 0 && len(sellingStatus.CurrentPrice[0].Value) > 0 {
			if price, err := strconv.ParseFloat(sellingStatus.CurrentPrice[0].Value[0], 64); err == nil {
				listing.Price = price
			}
		}

		if len(sellingStatus.BidCount) > 0 {
			if count, err := strconv.Atoi(sellingStatus.BidCount[0]); err == nil {
				listing.BidCount = count
			}
		}
	}

	// Extract end time
	if len(item.ListingInfo) > 0 && len(item.ListingInfo[0].EndTime) > 0 {
		if endTime, err := time.Parse(time.RFC3339, item.ListingInfo[0].EndTime[0]); err == nil {
			listing.EndTime = endTime
		}
	}

	// Determine if Buy It Now
	if len(item.ListingType) > 0 {
		listing.BuyItNow = strings.Contains(strings.ToLower(item.ListingType[0]), "fixedprice")
	}

	return listing, nil
}

var gradedPattern = regexp.MustCompile(`(?i)\b(psa|bgs|cgc|sgc|beckett|graded|slab|slabbed|authenticated|gem\s+mint|pristine|black\s+label|perfect\s+10|mint\s+9|nm-mt\s+8)\b`)

func (c *Client) isGradedCard(title string) bool {
	return gradedPattern.MatchString(title)
}

func (c *Client) sortByListingType(listings []Listing) {
	sort.Slice(listings, func(i, j int) bool {
		// Prefer Buy It Now listings over auctions
		if listings[i].BuyItNow && !listings[j].BuyItNow {
			return true
		}
		if !listings[i].BuyItNow && listings[j].BuyItNow {
			return false
		}
		// If both are the same type, maintain original order
		return false
	})
}

// GetEndingAuctions fetches Pokemon card auctions ending within the specified time frame
func (c *Client) GetEndingAuctions(minutesRemaining int, category string) ([]Auction, error) {
	if !c.Available() {
		return nil, fmt.Errorf("eBay app ID not configured")
	}

	// Apply rate limiting
	c.rateLimiter.wait()

	// eBay Finding API endpoint
	endpoint := "https://svcs.ebay.com/services/search/FindingService/v1"

	params := url.Values{}
	params.Set("OPERATION-NAME", "findItemsAdvanced")
	params.Set("SERVICE-VERSION", "1.0.0")
	params.Set("SECURITY-APPNAME", c.appID)
	params.Set("RESPONSE-DATA-FORMAT", "JSON")
	params.Set("keywords", "pokemon")
	params.Set("categoryId", "183454") // Trading Card Games category

	// Filter for auctions only (ending within specified time)
	params.Set("itemFilter(0).name", "ListingType")
	params.Set("itemFilter(0).value(0)", "Auction")

	// Filter by end time (auctions ending within minutesRemaining)
	params.Set("itemFilter(1).name", "EndTimeFrom")
	params.Set("itemFilter(1).value(0)", time.Now().Format(time.RFC3339))

	endTimeTo := time.Now().Add(time.Duration(minutesRemaining) * time.Minute)
	params.Set("itemFilter(2).name", "EndTimeTo")
	params.Set("itemFilter(2).value(0)", endTimeTo.Format(time.RFC3339))

	// Filter for Pokemon cards only
	params.Set("itemFilter(3).name", "Condition")
	params.Set("itemFilter(3).value(0)", "New")
	params.Set("itemFilter(3).value(1)", "Used")

	// Exclude graded cards in query to focus on raw cards
	params.Set("itemFilter(4).name", "ExcludeCategory")
	params.Set("itemFilter(4).value(0)", "267") // Exclude Books & Magazines

	params.Set("paginationInput.entriesPerPage", "100")
	params.Set("sortOrder", "EndTimeSoonest")

	// Execute request
	fullURL := endpoint + "?" + params.Encode()

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Add required headers
	req.Header.Set("X-EBAY-SOA-SERVICE-NAME", "FindingService")
	req.Header.Set("X-EBAY-SOA-OPERATION-NAME", "findItemsAdvanced")
	req.Header.Set("X-EBAY-SOA-SERVICE-VERSION", "1.0.0")
	req.Header.Set("X-EBAY-SOA-SECURITY-APPNAME", c.appID)
	req.Header.Set("X-EBAY-SOA-RESPONSE-DATA-FORMAT", "JSON")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eBay API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("eBay API returned status %d", resp.StatusCode)
	}

	var ebayResp findingResponse
	if err := json.Unmarshal(bodyBytes, &ebayResp); err != nil {
		return nil, fmt.Errorf("parse eBay response: %w", err)
	}

	// Parse auctions from response
	var auctions []Auction

	if len(ebayResp.FindItemsAdvancedResponse) > 0 &&
		len(ebayResp.FindItemsAdvancedResponse[0].SearchResult) > 0 {

		searchResult := ebayResp.FindItemsAdvancedResponse[0].SearchResult[0]

		for _, item := range searchResult.Item {
			auction, err := c.parseAuctionItem(item)
			if err != nil {
				continue // Skip malformed items
			}

			// Filter out graded cards by title
			if c.isGradedCard(auction.Title) {
				continue
			}

			// Additional Pokemon card filtering
			if !c.isPokemonCard(auction.Title) {
				continue
			}

			auctions = append(auctions, auction)
		}
	}

	return auctions, nil
}

// parseAuctionItem converts eBay API response item to Auction struct
func (c *Client) parseAuctionItem(item struct {
	ItemID      []string `json:"itemId"`
	Title       []string `json:"title"`
	ViewItemURL []string `json:"viewItemURL"`
	ListingType []string `json:"listingType"`
	Condition   []struct {
		ConditionDisplayName []string `json:"conditionDisplayName"`
	} `json:"condition"`
	SellingStatus []struct {
		CurrentPrice []struct {
			Value      []string `json:"__value__"`
			CurrencyID []string `json:"@currencyId"`
		} `json:"currentPrice"`
		BidCount []string `json:"bidCount"`
	} `json:"sellingStatus"`
	ListingInfo []struct {
		EndTime []string `json:"endTime"`
	} `json:"listingInfo"`
}) (Auction, error) {
	auction := Auction{}

	// Extract Item ID
	if len(item.ItemID) > 0 {
		auction.ItemID = item.ItemID[0]
	}

	// Extract title
	if len(item.Title) > 0 {
		auction.Title = item.Title[0]
	}

	// Extract URL
	if len(item.ViewItemURL) > 0 {
		auction.URL = item.ViewItemURL[0]
	}

	// Extract condition
	if len(item.Condition) > 0 && len(item.Condition[0].ConditionDisplayName) > 0 {
		auction.Condition = item.Condition[0].ConditionDisplayName[0]
	}

	// Extract current bid and bid count
	if len(item.SellingStatus) > 0 {
		sellingStatus := item.SellingStatus[0]

		if len(sellingStatus.CurrentPrice) > 0 && len(sellingStatus.CurrentPrice[0].Value) > 0 {
			if price, err := strconv.ParseFloat(sellingStatus.CurrentPrice[0].Value[0], 64); err == nil {
				auction.CurrentBid = price
			}
		}

		if len(sellingStatus.BidCount) > 0 {
			if count, err := strconv.Atoi(sellingStatus.BidCount[0]); err == nil {
				auction.BidCount = count
			}
		}
	}

	// Extract end time
	if len(item.ListingInfo) > 0 && len(item.ListingInfo[0].EndTime) > 0 {
		if endTime, err := time.Parse(time.RFC3339, item.ListingInfo[0].EndTime[0]); err == nil {
			auction.EndTime = endTime
		}
	}

	// Set category (default to Pokemon for this search)
	auction.Category = "Pokemon"

	return auction, nil
}

// isPokemonCard checks if the title indicates a Pokemon card
func (c *Client) isPokemonCard(title string) bool {
	titleLower := strings.ToLower(title)

	// Must contain "pokemon" and some card-related terms
	if !strings.Contains(titleLower, "pokemon") {
		return false
	}

	// Look for card indicators
	cardIndicators := []string{"card", "tcg", "holo", "foil", "rare", "promo", "shadowless",
		"1st edition", "unlimited", "base set", "jungle", "fossil", "team rocket"}

	for _, indicator := range cardIndicators {
		if strings.Contains(titleLower, indicator) {
			return true
		}
	}

	// Check for set names (basic ones)
	setNames := []string{"base", "jungle", "fossil", "rocket", "gym", "neo", "expedition",
		"aquapolis", "skyridge", "ruby", "sapphire", "emerald", "diamond", "pearl",
		"platinum", "black", "white", "x", "y", "sun", "moon", "sword", "shield",
		"brilliant", "astral", "lost", "crown", "silver", "fusion", "chilling",
		"darkness", "evolving", "battle", "rebel", "ghost", "crimson", "temporal",
		"paradox", "obsidian", "surging", "twilight", "stellar"}

	for _, setName := range setNames {
		if strings.Contains(titleLower, setName) {
			return true
		}
	}

	return false
}

// GetAuctionDetails fetches comprehensive details for a specific auction
func (c *Client) GetAuctionDetails(itemID string) (*AuctionDetail, error) {
	if !c.Available() {
		return nil, fmt.Errorf("eBay app ID not configured")
	}

	// Apply rate limiting
	c.rateLimiter.wait()

	// For now, use the Finding API to get basic auction info
	// In a production environment, you'd want to use the Shopping API or Trading API
	// for more detailed information
	endpoint := "https://svcs.ebay.com/services/search/FindingService/v1"

	params := url.Values{}
	params.Set("OPERATION-NAME", "findItemsAdvanced")
	params.Set("SERVICE-VERSION", "1.0.0")
	params.Set("SECURITY-APPNAME", c.appID)
	params.Set("RESPONSE-DATA-FORMAT", "JSON")
	params.Set("keywords", fmt.Sprintf("item:%s", itemID))

	// Execute request
	fullURL := endpoint + "?" + params.Encode()

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Add required headers
	req.Header.Set("X-EBAY-SOA-SERVICE-NAME", "FindingService")
	req.Header.Set("X-EBAY-SOA-OPERATION-NAME", "findItemsAdvanced")
	req.Header.Set("X-EBAY-SOA-SERVICE-VERSION", "1.0.0")
	req.Header.Set("X-EBAY-SOA-SECURITY-APPNAME", c.appID)
	req.Header.Set("X-EBAY-SOA-RESPONSE-DATA-FORMAT", "JSON")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eBay API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("eBay API returned status %d", resp.StatusCode)
	}

	var ebayResp findingResponse
	if err := json.Unmarshal(bodyBytes, &ebayResp); err != nil {
		return nil, fmt.Errorf("parse eBay response: %w", err)
	}

	// Parse auction details from response
	if len(ebayResp.FindItemsAdvancedResponse) > 0 &&
		len(ebayResp.FindItemsAdvancedResponse[0].SearchResult) > 0 &&
		len(ebayResp.FindItemsAdvancedResponse[0].SearchResult[0].Item) > 0 {

		item := ebayResp.FindItemsAdvancedResponse[0].SearchResult[0].Item[0]

		// Parse basic auction info
		auction, err := c.parseAuctionItem(item)
		if err != nil {
			return nil, fmt.Errorf("parse auction item: %w", err)
		}

		// Create detailed auction object
		detail := &AuctionDetail{
			Auction:     auction,
			Description: "Description not available via Finding API",
			Images:      []string{}, // Images not available via Finding API
			SellerInfo: SellerInfo{
				Username:       "N/A",
				FeedbackScore:  0,
				PositivePercent: 0.0,
				PowerSeller:    false,
				TopRated:       false,
			},
			ShippingInfo: ShippingInfo{
				Cost:         0.0,
				Service:      "Standard",
				HandlingTime: 1,
				Returns:      false,
			},
			BidHistory:  []Bid{}, // Bid history not available via Finding API
			LastUpdated: time.Now(),
		}

		return detail, nil
	}

	return nil, fmt.Errorf("auction with ID %s not found", itemID)
}
