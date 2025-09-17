package prices

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/guarzo/pkmgradegap/internal/cache"
	"github.com/guarzo/pkmgradegap/internal/model"
)

type PriceCharting struct {
	token string
	cache *cache.Cache
}

func NewPriceCharting(token string, c *cache.Cache) *PriceCharting {
	return &PriceCharting{
		token: token,
		cache: c,
	}
}

func (p *PriceCharting) Available() bool {
	return p.token != ""
}

// GetSalesFromPriceData extracts sales data from a PCMatch result
// This can be used to augment the sales provider with PriceCharting data
func (p *PriceCharting) GetSalesFromPriceData(match *PCMatch) (avgSalePrice float64, salesCount int, hasData bool) {
	if match == nil || len(match.RecentSales) == 0 {
		return 0, 0, false
	}

	// Convert cents to dollars for average price
	avgPriceCents := match.AvgSalePrice
	if avgPriceCents > 0 {
		avgSalePrice = float64(avgPriceCents) / 100.0
	}

	return avgSalePrice, match.SalesCount, true
}

// Result normalized to cents (integers) to avoid float issues.
type PCMatch struct {
	ID           string
	ProductName  string
	LooseCents   int // "loose-price" (ungraded)
	Grade9Cents  int // "graded-price" (Grade 9)
	Grade95Cents int // "box-only-price" (Grade 9.5)
	PSA10Cents   int // "manual-only-price" (PSA 10)
	BGS10Cents   int // "bgs-10-price" (BGS 10)
	// Sales data extracted from API (if available)
	RecentSales  []SaleData // Recent eBay sales tracked by PriceCharting
	SalesCount   int        // Total number of sales
	LastSoldDate string     // Date of last sale
	AvgSalePrice int        // Average sale price in cents
}

// SaleData represents a single sale tracked by PriceCharting
type SaleData struct {
	PriceCents int
	Date       string
	Grade      string
	Source     string // "eBay", "PWCC", etc.
}

func (p *PriceCharting) LookupCard(setName string, c model.Card) (*PCMatch, error) {
	// Try cache first
	if p.cache != nil {
		var match PCMatch
		key := cache.PriceChartingKey(setName, c.Name, c.Number)
		if found, _ := p.cache.Get(key, &match); found {
			return &match, nil
		}
	}

	// Heuristic query; you will likely refine this (promos, alt arts, RH).
	// Examples:
	//   "Pokemon Surging Sparks Pikachu #238"
	//   "Pokemon Vivid Voltage Charizard #25"
	q := fmt.Sprintf("pokemon %s %s #%s", setName, c.Name, c.Number)
	match, err := p.lookupByQuery(q)

	// Cache the result if successful
	if err == nil && match != nil && p.cache != nil {
		key := cache.PriceChartingKey(setName, c.Name, c.Number)
		_ = p.cache.Put(key, match, 2*time.Hour)
	}

	return match, err
}

func (p *PriceCharting) lookupByQuery(q string) (*PCMatch, error) {
	// First try /api/product?q=... (best match)
	u := fmt.Sprintf("https://www.pricecharting.com/api/product?t=%s&q=%s", url.QueryEscape(p.token), url.QueryEscape(q))
	var one map[string]any
	if err := httpGetJSON(u, &one); err == nil && strings.EqualFold(fmt.Sprint(one["status"]), "success") && hasPriceKeys(one) {
		return pcFrom(one), nil
	}
	// Fallback: /api/products to list and then pick the first
	u = fmt.Sprintf("https://www.pricecharting.com/api/products?t=%s&q=%s", url.QueryEscape(p.token), url.QueryEscape(q))
	var many struct {
		Status   string `json:"status"`
		Products []struct {
			ID          string `json:"id"`
			ProductName string `json:"product-name"`
		} `json:"products"`
	}
	if err := httpGetJSON(u, &many); err != nil {
		return nil, err
	}
	if strings.ToLower(many.Status) != "success" || len(many.Products) == 0 {
		return nil, fmt.Errorf("no product match")
	}
	// Pull full product by id
	id := many.Products[0].ID
	u = fmt.Sprintf("https://www.pricecharting.com/api/product?t=%s&id=%s", url.QueryEscape(p.token), url.QueryEscape(id))
	var full map[string]any
	if err := httpGetJSON(u, &full); err != nil {
		return nil, err
	}
	if strings.ToLower(fmt.Sprint(full["status"])) != "success" {
		return nil, fmt.Errorf("product fetch failed")
	}
	return pcFrom(full), nil
}

func httpGetJSON(u string, into any) error {
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(b))
	}
	return json.NewDecoder(resp.Body).Decode(into)
}

func hasPriceKeys(m map[string]any) bool {
	// We only need some combo to consider it card data
	_, lp := m["loose-price"]
	_, psa10 := m["manual-only-price"]
	_, g9 := m["graded-price"]
	return lp || psa10 || g9
}

func pcFrom(m map[string]any) *PCMatch {
	get := func(k string) int {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case float64:
				return int(t)
			case int:
				return t
			}
		}
		return 0
	}

	result := &PCMatch{
		ID:           fmt.Sprint(m["id"]),
		ProductName:  fmt.Sprint(m["product-name"]),
		LooseCents:   get("loose-price"),
		Grade9Cents:  get("graded-price"),
		Grade95Cents: get("box-only-price"),
		PSA10Cents:   get("manual-only-price"),
		BGS10Cents:   get("bgs-10-price"),
	}

	// Extract sales data if available
	if salesData, ok := m["sales-data"].([]interface{}); ok {
		for _, sale := range salesData {
			if saleMap, ok := sale.(map[string]interface{}); ok {
				saleInfo := SaleData{
					PriceCents: get("sale-price"),
					Date:       fmt.Sprint(saleMap["sale-date"]),
					Grade:      fmt.Sprint(saleMap["grade"]),
					Source:     "eBay", // PriceCharting primarily tracks eBay
				}
				result.RecentSales = append(result.RecentSales, saleInfo)
			}
		}
		result.SalesCount = len(result.RecentSales)
	}

	// Extract additional sales metadata
	if lastSold, ok := m["last-sold-date"].(string); ok {
		result.LastSoldDate = lastSold
	}

	// Calculate average sale price if we have sales
	if len(result.RecentSales) > 0 {
		total := 0
		for _, sale := range result.RecentSales {
			total += sale.PriceCents
		}
		result.AvgSalePrice = total / len(result.RecentSales)
	}

	return result
}
