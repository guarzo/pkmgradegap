package ebay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

type Client struct {
	appID      string
	httpClient *http.Client
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
	}
}

func (c *Client) Available() bool {
	return c.appID != ""
}

func (c *Client) SearchRawListings(setName, cardName, number string, max int) ([]Listing, error) {
	if !c.Available() {
		return nil, fmt.Errorf("eBay app ID not configured")
	}

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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("eBay API returned status %d", resp.StatusCode)
	}

	var ebayResp findingResponse
	if err := json.NewDecoder(resp.Body).Decode(&ebayResp); err != nil {
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

			// Break if we have enough listings
			if len(listings) >= max {
				break
			}
		}
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

func (c *Client) isGradedCard(title string) bool {
	lower := strings.ToLower(title)
	gradedTerms := []string{
		"psa", "bgs", "cgc", "ace", "graded", "slab", "slabbed",
		"authenticated", "gem mint", "pristine", "black label",
		"perfect 10", "mint 9", "nm-mt 8",
	}

	for _, term := range gradedTerms {
		if strings.Contains(lower, term) {
			return true
		}
	}

	return false
}
