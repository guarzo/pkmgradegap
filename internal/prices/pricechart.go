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

// Result normalized to cents (integers) to avoid float issues.
type PCMatch struct {
	ID            string
	ProductName   string
	LooseCents    int // "loose-price" (ungraded)
	Grade9Cents   int // "graded-price" (Grade 9)
	Grade95Cents  int // "box-only-price" (Grade 9.5)
	PSA10Cents    int // "manual-only-price" (PSA 10)
	BGS10Cents    int // "bgs-10-price" (BGS 10)
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
	return &PCMatch{
		ID:           fmt.Sprint(m["id"]),
		ProductName:  fmt.Sprint(m["product-name"]),
		LooseCents:   get("loose-price"),
		Grade9Cents:  get("graded-price"),
		Grade95Cents: get("box-only-price"),
		PSA10Cents:   get("manual-only-price"),
		BGS10Cents:   get("bgs-10-price"),
	}
}