package ebay

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// TradingAPIClient handles eBay Trading API operations (XML-based)
type TradingAPIClient struct {
	httpClient   *http.Client
	oauthManager *OAuthManager
	sandbox      bool
	appID        string // eBay App ID for Trading API
}

// NewTradingAPIClient creates a new Trading API client
func NewTradingAPIClient(oauthManager *OAuthManager, appID string, sandbox bool) *TradingAPIClient {
	return &TradingAPIClient{
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		oauthManager: oauthManager,
		sandbox:      sandbox,
		appID:        appID,
	}
}

// GetMyeBaySellingRequest represents the XML request structure
type GetMyeBaySellingRequest struct {
	XMLName              xml.Name `xml:"GetMyeBaySellingRequest"`
	Xmlns                string   `xml:"xmlns,attr"`
	RequesterCredentials struct {
		EBayAuthToken string `xml:"eBayAuthToken"`
	} `xml:"RequesterCredentials"`
	ActiveList struct {
		Include    bool `xml:"Include"`
		Pagination struct {
			EntriesPerPage int `xml:"EntriesPerPage"`
			PageNumber     int `xml:"PageNumber"`
		} `xml:"Pagination"`
	} `xml:"ActiveList"`
	DetailLevel string `xml:"DetailLevel"`
}

// GetMyeBaySellingResponse represents the XML response structure
type GetMyeBaySellingResponse struct {
	XMLName xml.Name `xml:"GetMyeBaySellingResponse"`
	Ack     string   `xml:"Ack"`
	Version string   `xml:"Version"`
	Build   string   `xml:"Build"`
	Errors  []struct {
		ShortMessage string `xml:"ShortMessage"`
		LongMessage  string `xml:"LongMessage"`
		ErrorCode    string `xml:"ErrorCode"`
	} `xml:"Errors>Error"`
	ActiveList struct {
		ItemArray struct {
			Items []TradingAPIItem `xml:"Item"`
		} `xml:"ItemArray"`
		PaginationResult struct {
			TotalNumberOfPages   int `xml:"TotalNumberOfPages"`
			TotalNumberOfEntries int `xml:"TotalNumberOfEntries"`
		} `xml:"PaginationResult"`
	} `xml:"ActiveList"`
}

// TradingAPIItem represents an item from Trading API
type TradingAPIItem struct {
	ItemID      string `xml:"ItemID"`
	Title       string `xml:"Title"`
	ViewItemURL string `xml:"ViewItemURL"`
	ListingType string `xml:"ListingType"`
	Quantity    int    `xml:"Quantity"`
	StartPrice  struct {
		Value      float64 `xml:",chardata"`
		CurrencyID string  `xml:"currencyID,attr"`
	} `xml:"StartPrice"`
	BuyItNowPrice struct {
		Value      float64 `xml:",chardata"`
		CurrencyID string  `xml:"currencyID,attr"`
	} `xml:"BuyItNowPrice"`
	CurrentPrice struct {
		Value      float64 `xml:",chardata"`
		CurrencyID string  `xml:"currencyID,attr"`
	} `xml:"CurrentPrice"`
	ListingDetails struct {
		StartTime time.Time `xml:"StartTime"`
		EndTime   time.Time `xml:"EndTime"`
		ViewCount int       `xml:"ViewCount"`
	} `xml:"ListingDetails"`
	SellingStatus struct {
		BidCount      int    `xml:"BidCount"`
		ListingStatus string `xml:"ListingStatus"`
		QuantitySold  int    `xml:"QuantitySold"`
		CurrentPrice  struct {
			Value      float64 `xml:",chardata"`
			CurrencyID string  `xml:"currencyID,attr"`
		} `xml:"CurrentPrice"`
	} `xml:"SellingStatus"`
	PictureDetails struct {
		PictureURL []string `xml:"PictureURL"`
	} `xml:"PictureDetails"`
	ConditionDisplayName string `xml:"ConditionDisplayName"`
	PrimaryCategory      struct {
		CategoryID   string `xml:"CategoryID"`
		CategoryName string `xml:"CategoryName"`
	} `xml:"PrimaryCategory"`
	// Watch count is not directly available in Trading API
	// Would need separate GetItemTransactions call
}

// GetMyListings fetches the user's active eBay listings using Trading API
func (c *TradingAPIClient) GetMyListings(userID string, page int, pageSize int) ([]UserListing, error) {
	token, err := c.oauthManager.GetValidToken(userID)
	if err != nil {
		return nil, fmt.Errorf("getting valid token: %w", err)
	}

	// Build XML request
	request := GetMyeBaySellingRequest{
		Xmlns: "urn:ebay:apis:eBLBaseComponents",
	}
	request.RequesterCredentials.EBayAuthToken = token.AccessToken
	request.ActiveList.Include = true
	request.ActiveList.Pagination.EntriesPerPage = pageSize
	request.ActiveList.Pagination.PageNumber = page
	request.DetailLevel = "ReturnAll"

	// Marshal to XML
	xmlData, err := xml.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshaling XML request: %w", err)
	}

	// Add XML declaration
	xmlRequest := `<?xml version="1.0" encoding="utf-8"?>` + string(xmlData)

	// Determine endpoint
	endpoint := "https://api.ebay.com/ws/api.dll"
	if c.sandbox {
		endpoint = "https://api.sandbox.ebay.com/ws/api.dll"
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader([]byte(xmlRequest)))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set Trading API headers
	req.Header.Set("X-EBAY-API-COMPATIBILITY-LEVEL", "967")
	req.Header.Set("X-EBAY-API-DEV-NAME", c.appID) // Use App ID as Dev Name
	req.Header.Set("X-EBAY-API-APP-NAME", c.appID)
	req.Header.Set("X-EBAY-API-CERT-NAME", c.appID) // In production, use separate cert ID
	req.Header.Set("X-EBAY-API-CALL-NAME", "GetMyeBaySelling")
	req.Header.Set("X-EBAY-API-SITEID", "0") // US site
	req.Header.Set("Content-Type", "text/xml")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	// Parse XML response
	var response GetMyeBaySellingResponse
	if err := xml.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parsing XML response: %w", err)
	}

	// Check for API errors
	if response.Ack != "Success" && response.Ack != "Warning" {
		errorMsg := "Unknown error"
		if len(response.Errors) > 0 {
			errorMsg = response.Errors[0].LongMessage
		}
		return nil, fmt.Errorf("eBay API error: %s", errorMsg)
	}

	// Convert Trading API items to UserListings
	listings := make([]UserListing, 0, len(response.ActiveList.ItemArray.Items))
	for _, item := range response.ActiveList.ItemArray.Items {
		listing := c.convertTradingAPIItem(item)
		listings = append(listings, listing)
	}

	return listings, nil
}

// convertTradingAPIItem converts Trading API item to UserListing
func (c *TradingAPIClient) convertTradingAPIItem(item TradingAPIItem) UserListing {
	listing := UserListing{
		ItemID:         item.ItemID,
		Title:          item.Title,
		Quantity:       item.Quantity,
		ViewCount:      item.ListingDetails.ViewCount,
		ListingURL:     item.ViewItemURL,
		StartTime:      item.ListingDetails.StartTime,
		EndTime:        item.ListingDetails.EndTime,
		ListingType:    item.ListingType,
		Condition:      item.ConditionDisplayName,
		CategoryID:     item.PrimaryCategory.CategoryID,
		PrimaryCatName: item.PrimaryCategory.CategoryName,
		ListingStatus:  item.SellingStatus.ListingStatus,
		SoldQuantity:   item.SellingStatus.QuantitySold,
	}

	// Determine current price
	if item.ListingType == "FixedPriceItem" && item.BuyItNowPrice.Value > 0 {
		listing.CurrentPrice = item.BuyItNowPrice.Value
	} else if item.SellingStatus.CurrentPrice.Value > 0 {
		listing.CurrentPrice = item.SellingStatus.CurrentPrice.Value
	} else {
		listing.CurrentPrice = item.StartPrice.Value
	}

	// Set image URL
	if len(item.PictureDetails.PictureURL) > 0 {
		listing.ImageURL = item.PictureDetails.PictureURL[0]
	}

	// Extract card details from title
	listing.CardName, listing.SetName, listing.CardNumber = c.extractCardDetails(item.Title)

	// Calculate days active
	if !listing.StartTime.IsZero() {
		listing.DaysActive = int(time.Since(listing.StartTime).Hours() / 24)
	}

	// Calculate time left
	if !listing.EndTime.IsZero() {
		timeLeft := time.Until(listing.EndTime)
		if timeLeft > 0 {
			if timeLeft.Hours() > 24 {
				listing.TimeLeft = fmt.Sprintf("%dd", int(timeLeft.Hours()/24))
			} else {
				listing.TimeLeft = fmt.Sprintf("%dh", int(timeLeft.Hours()))
			}
		} else {
			listing.TimeLeft = "Ended"
		}
	}

	return listing
}

// extractCardDetails parses card information from listing title
func (c *TradingAPIClient) extractCardDetails(title string) (cardName, setName, number string) {
	// Updated patterns for better Pokemon card detection
	patterns := []struct {
		regex *regexp.Regexp
		name  string
	}{
		{
			// Pokemon [CardName] [SetName] #[Number]
			regex: regexp.MustCompile(`(?i)pokemon\s+(.+?)\s+(.+?)\s+#?(\d+(?:/\d+)?)`),
			name:  "standard",
		},
		{
			// [CardName] #[Number] [SetName] Pokemon
			regex: regexp.MustCompile(`(?i)(.+?)\s+#?(\d+(?:/\d+)?)\s+(.+?)\s+pokemon`),
			name:  "reverse",
		},
		{
			// [CardName] [Special] - [SetName] #[Number]
			regex: regexp.MustCompile(`(?i)(.+?)\s+(VMAX|VSTAR|V|GX|EX|Prime|BREAK)\s*-?\s*(.+?)\s+#?(\d+)`),
			name:  "special",
		},
		{
			// [SetName] [CardName] #[Number]
			regex: regexp.MustCompile(`(?i)(.+?)\s+(.+?)\s+#?(\d+(?:/\d+)?)(?:\s|$)`),
			name:  "simple",
		},
	}

	// Try each pattern
	for _, pattern := range patterns {
		if matches := pattern.regex.FindStringSubmatch(title); matches != nil {
			switch pattern.name {
			case "standard":
				if len(matches) >= 4 {
					return strings.TrimSpace(matches[1]), strings.TrimSpace(matches[2]), strings.TrimSpace(matches[3])
				}
			case "reverse":
				if len(matches) >= 4 {
					return strings.TrimSpace(matches[1]), strings.TrimSpace(matches[3]), strings.TrimSpace(matches[2])
				}
			case "special":
				if len(matches) >= 5 {
					cardName := strings.TrimSpace(matches[1]) + " " + strings.TrimSpace(matches[2])
					return cardName, strings.TrimSpace(matches[3]), strings.TrimSpace(matches[4])
				}
			case "simple":
				if len(matches) >= 4 {
					// Guess which is set vs card name (sets often have recognizable names)
					setKeywords := []string{"base", "jungle", "fossil", "team rocket", "gym", "neo", "expedition", "aquapolis", "skyridge", "ruby", "sapphire", "emerald", "diamond", "pearl", "platinum", "heartgold", "soulsilver", "black", "white", "xy", "sun", "moon", "sword", "shield", "brilliant", "astral", "lost", "silver", "crown", "fusion", "chilling", "darkness", "vivid", "battle", "rebel", "cosmic", "unified", "evolving", "paldea", "obsidian", "surging", "temporal", "twilight", "shrouded", "stellar"}

					part1 := strings.ToLower(matches[1])

					// Check if part1 looks like a set name
					for _, keyword := range setKeywords {
						if strings.Contains(part1, keyword) {
							return strings.TrimSpace(matches[2]), strings.TrimSpace(matches[1]), strings.TrimSpace(matches[3])
						}
					}

					// Default: assume first part is card name
					return strings.TrimSpace(matches[1]), strings.TrimSpace(matches[2]), strings.TrimSpace(matches[3])
				}
			}
		}
	}

	// Fallback: try to extract just the number
	if numMatch := regexp.MustCompile(`#?(\d+(?:/\d+)?)`).FindStringSubmatch(title); numMatch != nil {
		number = numMatch[1]
	}

	// If no patterns match, try to extract pokemon-related words as card name
	words := strings.Fields(title)
	for i, word := range words {
		if strings.ToLower(word) == "pokemon" && i > 0 {
			cardName = strings.Join(words[:i], " ")
			break
		}
	}

	return cardName, setName, number
}

// UpdateListingPrice updates the price of a listing using Trading API
func (c *TradingAPIClient) UpdateListingPrice(userID, itemID string, newPrice float64) error {
	token, err := c.oauthManager.GetValidToken(userID)
	if err != nil {
		return fmt.Errorf("getting valid token: %w", err)
	}

	// For Trading API, we'd need to use ReviseItem call
	// This is complex as it requires the full item structure
	// For now, return a placeholder implementation

	// Create ReviseItem XML request
	xmlRequest := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
	<ReviseItemRequest xmlns="urn:ebay:apis:eBLBaseComponents">
		<RequesterCredentials>
			<eBayAuthToken>%s</eBayAuthToken>
		</RequesterCredentials>
		<Item>
			<ItemID>%s</ItemID>
			<StartPrice currencyID="USD">%.2f</StartPrice>
		</Item>
	</ReviseItemRequest>`, token.AccessToken, itemID, newPrice)

	// Determine endpoint
	endpoint := "https://api.ebay.com/ws/api.dll"
	if c.sandbox {
		endpoint = "https://api.sandbox.ebay.com/ws/api.dll"
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(xmlRequest))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	// Set Trading API headers
	req.Header.Set("X-EBAY-API-COMPATIBILITY-LEVEL", "967")
	req.Header.Set("X-EBAY-API-DEV-NAME", c.appID)
	req.Header.Set("X-EBAY-API-APP-NAME", c.appID)
	req.Header.Set("X-EBAY-API-CERT-NAME", c.appID)
	req.Header.Set("X-EBAY-API-CALL-NAME", "ReviseItem")
	req.Header.Set("X-EBAY-API-SITEID", "0")
	req.Header.Set("Content-Type", "text/xml")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("price update failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response for errors (simplified)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	// Check for errors in XML response
	if strings.Contains(string(body), "<Ack>Failure</Ack>") {
		return fmt.Errorf("eBay API returned failure: %s", string(body))
	}

	return nil
}

// GetListingSummary returns overview statistics
func (c *TradingAPIClient) GetListingSummary(userID string) (*ListingSummary, error) {
	// Get all listings (first page with high limit)
	listings, err := c.GetMyListings(userID, 1, 200)
	if err != nil {
		return nil, fmt.Errorf("fetching listings: %w", err)
	}

	summary := &ListingSummary{
		TotalActive: len(listings),
	}

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	var totalDays float64
	for _, listing := range listings {
		summary.TotalViews += listing.ViewCount
		summary.TotalWatchers += listing.WatchCount
		summary.TotalValue += listing.CurrentPrice * float64(listing.Quantity)

		// Calculate days listed
		if !listing.StartTime.IsZero() {
			daysListed := now.Sub(listing.StartTime).Hours() / 24
			totalDays += daysListed
		}

		// Check if sold this month
		if listing.SoldQuantity > 0 && listing.LastModified.After(monthStart) {
			summary.SoldThisMonth += listing.SoldQuantity
			summary.RevenueThisMonth += listing.CurrentPrice * float64(listing.SoldQuantity)
		}
	}

	if len(listings) > 0 {
		summary.AvgDaysListed = totalDays / float64(len(listings))
	}

	return summary, nil
}
