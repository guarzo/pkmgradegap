package cards

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/guarzo/pkmgradegap/internal/cache"
	"github.com/guarzo/pkmgradegap/internal/model"
)

type PokeTCGIO struct {
	apiKey string
	cache  *cache.Cache
	client *http.Client
}

func NewPokeTCGIO(apiKey string, c *cache.Cache) *PokeTCGIO {
	return &PokeTCGIO{
		apiKey: apiKey,
		cache:  c,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (p *PokeTCGIO) ListSets() ([]model.Set, error) {
	// Try cache first
	if p.cache != nil {
		var sets []model.Set
		if found, _ := p.cache.Get(cache.SetsKey(), &sets); found {
			return sets, nil
		}
	}

	// https://api.pokemontcg.io/v2/sets?orderBy=name
	u := "https://api.pokemontcg.io/v2/sets?orderBy=name"
	var out struct {
		Data []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			ReleaseDate string `json:"releaseDate"`
		} `json:"data"`
	}
	if err := p.get(u, &out); err != nil {
		return nil, err
	}
	sets := make([]model.Set, 0, len(out.Data))
	for _, s := range out.Data {
		sets = append(sets, model.Set{
			ID:          s.ID,
			Name:        s.Name,
			ReleaseDate: s.ReleaseDate,
		})
	}

	// Cache the results
	if p.cache != nil {
		_ = p.cache.Put(cache.SetsKey(), sets, 24*time.Hour)
	}

	return sets, nil
}

func (p *PokeTCGIO) CardsBySetID(setID string) ([]model.Card, error) {
	// Try cache first
	if p.cache != nil {
		var cards []model.Card
		if found, _ := p.cache.Get(cache.CardsKey(setID), &cards); found {
			return cards, nil
		}
	}

	// GET /v2/cards?q=set.id:SWxxxx&pageSize=250&page=N
	page := 1
	pageSize := 250
	cards := []model.Card{}

	for {
		q := url.QueryEscape("set.id:" + setID)
		u := fmt.Sprintf("https://api.pokemontcg.io/v2/cards?q=%s&pageSize=%d&page=%d", q, pageSize, page)

		var resp struct {
			Data []struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Number string `json:"number"`
				Rarity string `json:"rarity"`
				Set    struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"set"`
				TCG *struct {
					URL     string `json:"url"`
					Updated string `json:"updatedAt"`
					Prices  map[string]struct {
						Low       *float64 `json:"low,omitempty"`
						Mid       *float64 `json:"mid,omitempty"`
						High      *float64 `json:"high,omitempty"`
						Market    *float64 `json:"market,omitempty"`
						DirectLow *float64 `json:"directLow,omitempty"`
					} `json:"prices"`
				} `json:"tcgplayer"`
				CM *struct {
					URL     string `json:"url"`
					Updated string `json:"updatedAt"`
					Prices  struct {
						AverageSellPrice *float64 `json:"averageSellPrice"`
						TrendPrice       *float64 `json:"trendPrice"`
						Avg7             *float64 `json:"avg7"`
						Avg30            *float64 `json:"avg30"`
						ReverseHoloTrend *float64 `json:"reverseHoloTrend"`
						ReverseHoloAvg7  *float64 `json:"reverseHoloAvg7"`
						ReverseHoloAvg30 *float64 `json:"reverseHoloAvg30"`
					} `json:"prices"`
				} `json:"cardmarket"`
			} `json:"data"`
			Page       int `json:"page"`
			PageSize   int `json:"pageSize"`
			Count      int `json:"count"`
			TotalCount int `json:"totalCount"`
		}

		if err := p.get(u, &resp); err != nil {
			return nil, err
		}
		for _, c := range resp.Data {
			var t *model.TCGPlayerBlock
			if c.TCG != nil {
				t = &model.TCGPlayerBlock{
					URL:     c.TCG.URL,
					Updated: c.TCG.Updated,
					Prices:  c.TCG.Prices,
				}
			}
			var cm *model.CardmarketBlock
			if c.CM != nil {
				cm = &model.CardmarketBlock{
					URL:     c.CM.URL,
					Updated: c.CM.Updated,
				}
				cm.Prices.AverageSellPrice = c.CM.Prices.AverageSellPrice
				cm.Prices.TrendPrice = c.CM.Prices.TrendPrice
				cm.Prices.Avg7 = c.CM.Prices.Avg7
				cm.Prices.Avg30 = c.CM.Prices.Avg30
				cm.Prices.ReverseHoloTrend = c.CM.Prices.ReverseHoloTrend
				cm.Prices.ReverseHoloAvg7 = c.CM.Prices.ReverseHoloAvg7
				cm.Prices.ReverseHoloAvg30 = c.CM.Prices.ReverseHoloAvg30
			}

			cards = append(cards, model.Card{
				ID:         c.ID,
				Name:       c.Name,
				Number:     c.Number,
				Rarity:     c.Rarity,
				SetID:      c.Set.ID,
				SetName:    c.Set.Name,
				TCGPlayer:  t,
				Cardmarket: cm,
			})
		}

		got := resp.Page * resp.PageSize
		if got >= resp.TotalCount {
			break
		}
		page++
	}

	// Cache the results
	if p.cache != nil {
		_ = p.cache.Put(cache.CardsKey(setID), cards, 4*time.Hour)
	}

	return cards, nil
}

func (p *PokeTCGIO) get(u string, into any) error {
	maxRetries := 3
	baseDelay := 1 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			return err
		}

		if p.apiKey != "" {
			req.Header.Set("X-Api-Key", p.apiKey)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "pkmgradegap/1.0")

		resp, err := p.client.Do(req)
		if err != nil {
			log.Printf("Attempt %d/%d failed for %s: %v", attempt+1, maxRetries, u, err)
			if attempt < maxRetries-1 {
				delay := baseDelay * time.Duration(1<<attempt) // Exponential backoff
				log.Printf("Retrying in %v...", delay)
				time.Sleep(delay)
				continue
			}
			return fmt.Errorf("after %d attempts: %w", maxRetries, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == 429 {
			log.Printf("Rate limited, waiting before retry...")
			if attempt < maxRetries-1 {
				time.Sleep(5 * time.Second)
				continue
			}
		}

		if resp.StatusCode == 504 || resp.StatusCode == 502 || resp.StatusCode == 503 {
			log.Printf("Server error %d, retrying...", resp.StatusCode)
			if attempt < maxRetries-1 {
				delay := baseDelay * time.Duration(1<<attempt)
				time.Sleep(delay)
				continue
			}
		}

		if resp.StatusCode/100 != 2 {
			b, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("pokemontcg.io %s: %s", strconv.Itoa(resp.StatusCode), string(b))
		}

		return json.NewDecoder(resp.Body).Decode(into)
	}

	return fmt.Errorf("failed after %d attempts", maxRetries)
}
