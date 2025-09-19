package population

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/guarzo/pkmgradegap/internal/model"
	"golang.org/x/time/rate"
)

const (
	psaPopulationBaseURL = "https://www.psacard.com/pop"
	psaSearchURL         = "https://www.psacard.com/pop/search"
	userAgent            = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	scrapeRateLimit      = 1 // request per second
	cacheExpiration      = 24 * time.Hour
	searchCacheTTL       = 1 * time.Hour // Search results cache for 1 hour
	maxCacheSize         = 1000          // Maximum number of cached search results
)

// searchCacheEntry represents a cached search result with expiration
type searchCacheEntry struct {
	URL       string
	ExpiresAt time.Time
}

// searchCacheStats tracks cache hit/miss metrics
type searchCacheStats struct {
	Hits      int64
	Misses    int64
	Evictions int64
}

// stringCache implements a simple LRU cache for search results
type stringCache struct {
	entries map[string]*searchCacheEntry
	order   []string // LRU order, most recently used at end
	stats   searchCacheStats
	mu      sync.RWMutex
	maxSize int
}

// newStringCache creates a new string cache with the specified max size
func newStringCache(maxSize int) *stringCache {
	return &stringCache{
		entries: make(map[string]*searchCacheEntry),
		order:   make([]string, 0, maxSize),
		maxSize: maxSize,
	}
}

// get retrieves a value from the cache
func (sc *stringCache) get(key string) (string, bool) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	entry, exists := sc.entries[key]
	if !exists {
		sc.stats.Misses++
		return "", false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		delete(sc.entries, key)
		sc.removeFromOrder(key)
		sc.stats.Misses++
		return "", false
	}

	// Move to end (most recently used)
	sc.moveToEnd(key)
	sc.stats.Hits++
	return entry.URL, true
}

// set stores a value in the cache
func (sc *stringCache) set(key, value string, ttl time.Duration) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Remove existing entry if present
	if _, exists := sc.entries[key]; exists {
		sc.removeFromOrder(key)
	}

	// Evict if at capacity
	if len(sc.entries) >= sc.maxSize {
		sc.evictOldest()
	}

	// Add new entry
	sc.entries[key] = &searchCacheEntry{
		URL:       value,
		ExpiresAt: time.Now().Add(ttl),
	}
	sc.order = append(sc.order, key)
}

// evictOldest removes the least recently used entry
func (sc *stringCache) evictOldest() {
	if len(sc.order) == 0 {
		return
	}

	oldest := sc.order[0]
	delete(sc.entries, oldest)
	sc.order = sc.order[1:]
	sc.stats.Evictions++
}

// removeFromOrder removes a key from the order slice
func (sc *stringCache) removeFromOrder(key string) {
	for i, k := range sc.order {
		if k == key {
			sc.order = append(sc.order[:i], sc.order[i+1:]...)
			break
		}
	}
}

// moveToEnd moves a key to the end of the order slice (most recently used)
func (sc *stringCache) moveToEnd(key string) {
	sc.removeFromOrder(key)
	sc.order = append(sc.order, key)
}

// getStats returns cache hit/miss statistics
func (sc *stringCache) getStats() searchCacheStats {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.stats
}

// PSAScraper implements web scraping for PSA population data
type PSAScraper struct {
	client      *http.Client
	cache       Cache        // Use the Cache interface for population data
	searchCache *stringCache // String cache for search results
	limiter     *rate.Limiter
	debug       bool
}

// NewPSAScraper creates a new PSA web scraper
func NewPSAScraper(cacheInstance Cache) *PSAScraper {
	return &PSAScraper{
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		cache:       cacheInstance,
		searchCache: newStringCache(maxCacheSize),
		limiter:     rate.NewLimiter(rate.Limit(scrapeRateLimit), 1),
		debug:       false,
	}
}

// SetDebug enables debug logging
func (s *PSAScraper) SetDebug(debug bool) {
	s.debug = debug
}

// GetSearchCacheStats returns current search cache statistics for monitoring
func (s *PSAScraper) GetSearchCacheStats() searchCacheStats {
	return s.searchCache.getStats()
}

// SearchCard searches for a card and returns the population report URL or identifier
func (s *PSAScraper) SearchCard(ctx context.Context, set, number, name string) (string, error) {
	// Create cache key for search results
	cacheKey := fmt.Sprintf("search_%s_%s_%s",
		normalizeString(set),
		normalizeString(number),
		normalizeString(name))

	// Check search cache first
	if cachedURL, found := s.searchCache.get(cacheKey); found {
		if s.debug {
			stats := s.searchCache.getStats()
			log.Printf("PSAScraper: Cache HIT for search '%s %s %s' (Stats: %d hits, %d misses, %d evictions)",
				set, number, name, stats.Hits, stats.Misses, stats.Evictions)
		}
		return cachedURL, nil
	}

	if s.debug {
		stats := s.searchCache.getStats()
		log.Printf("PSAScraper: Cache MISS for search '%s %s %s' (Stats: %d hits, %d misses, %d evictions)",
			set, number, name, stats.Hits, stats.Misses, stats.Evictions)
	}

	// Rate limit
	if err := s.limiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limiter error: %w", err)
	}

	// Build search query
	searchQuery := fmt.Sprintf("%s %s %s", set, number, name)
	searchURL := fmt.Sprintf("%s?q=%s&category=pokemon", psaSearchURL, url.QueryEscape(searchQuery))

	if s.debug {
		log.Printf("PSAScraper: Searching for card: %s", searchQuery)
	}

	// Make request
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("performing search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("search returned status %d", resp.StatusCode)
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("parsing search results: %w", err)
	}

	// Look for population report link
	// PSA search results typically have links like: /pop/pokemon/2020/sword-shield/001
	var popURL string
	doc.Find("a[href*='/pop/pokemon']").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		// Check if this link matches our card
		linkText := strings.ToLower(s.Text())
		if strings.Contains(linkText, strings.ToLower(number)) {
			// Found a potential match
			if !strings.HasPrefix(href, "http") {
				href = "https://www.psacard.com" + href
			}
			popURL = href
		}
	})

	if popURL == "" {
		// Try alternative search patterns
		// Look for spec number in data attributes
		doc.Find("tr[data-spec], div[data-spec]").Each(func(i int, s *goquery.Selection) {
			spec, exists := s.Attr("data-spec")
			if exists && spec != "" {
				popURL = fmt.Sprintf("%s/%s", psaPopulationBaseURL, spec)
			}
		})
	}

	if popURL == "" {
		return "", fmt.Errorf("no population report found for %s %s %s", set, number, name)
	}

	// Cache the search result for future lookups
	s.searchCache.set(cacheKey, popURL, searchCacheTTL)

	if s.debug {
		stats := s.searchCache.getStats()
		log.Printf("PSAScraper: Cached search result for '%s %s %s' -> %s (Stats: %d hits, %d misses, %d evictions)",
			set, number, name, popURL, stats.Hits, stats.Misses, stats.Evictions)
	}

	return popURL, nil
}

// ScrapePopulation scrapes population data from a PSA population report page
func (s *PSAScraper) ScrapePopulation(ctx context.Context, cardIdentifier string) (*model.PSAPopulation, error) {
	// Note: Can't use the PopulationData cache here since we return model.PSAPopulation
	// The cache interface is specific to PopulationData type

	// Rate limit
	if err := s.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Determine if cardIdentifier is a URL or spec number
	var popURL string
	if strings.HasPrefix(cardIdentifier, "http") {
		popURL = cardIdentifier
	} else {
		popURL = fmt.Sprintf("%s/%s", psaPopulationBaseURL, cardIdentifier)
	}

	if s.debug {
		log.Printf("PSAScraper: Fetching population from %s", popURL)
	}

	// Make request
	req, err := http.NewRequestWithContext(ctx, "GET", popURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("performing population request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("population page returned status %d", resp.StatusCode)
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parsing population page: %w", err)
	}

	// Parse the population table
	pop, err := s.parsePopulationTable(doc)
	if err != nil {
		return nil, fmt.Errorf("parsing population table: %w", err)
	}

	// Can't cache model.PSAPopulation with the PopulationData cache interface

	return pop, nil
}

// parsePopulationTable extracts grade distribution data from the HTML document
func (s *PSAScraper) parsePopulationTable(doc *goquery.Document) (*model.PSAPopulation, error) {
	pop := &model.PSAPopulation{
		LastUpdated: time.Now(),
	}
	gradeDistribution := make(map[string]int)

	// Look for the population table
	// PSA typically uses a table with grade headers
	found := false

	// Try to find the main population table
	doc.Find("table.pop-table, table#population-table, .population-grid table").Each(func(i int, table *goquery.Selection) {
		if found {
			return
		}

		// Look for grade headers (1, 2, 3, ... 10)
		gradeHeaders := make(map[int]int) // column index -> grade

		table.Find("thead th, tr:first-child th, tr:first-child td").Each(func(j int, header *goquery.Selection) {
			text := strings.TrimSpace(header.Text())
			// Try to parse as grade number
			if grade, err := strconv.Atoi(text); err == nil && grade >= 1 && grade <= 10 {
				gradeHeaders[j] = grade
			}
			// Also check for "PSA 10", "PSA 9", etc.
			if strings.HasPrefix(text, "PSA ") {
				gradeStr := strings.TrimPrefix(text, "PSA ")
				if grade, err := strconv.Atoi(gradeStr); err == nil && grade >= 1 && grade <= 10 {
					gradeHeaders[j] = grade
				}
			}
		})

		if len(gradeHeaders) == 0 {
			// Try alternative format: look for data in rows with grade labels
			table.Find("tr").Each(func(j int, row *goquery.Selection) {
				cells := row.Find("td")
				if cells.Length() >= 2 {
					// First cell might be grade label
					gradeText := strings.TrimSpace(cells.Eq(0).Text())
					popText := strings.TrimSpace(cells.Eq(1).Text())

					// Extract grade number
					gradeMatch := regexp.MustCompile(`\b(\d+)\b`).FindStringSubmatch(gradeText)
					if len(gradeMatch) > 1 {
						if grade, err := strconv.Atoi(gradeMatch[1]); err == nil && grade >= 1 && grade <= 10 {
							// Extract population count
							popCount := parsePopulationCount(popText)
							if popCount > 0 {
								gradeDistribution[fmt.Sprintf("PSA %d", grade)] = popCount
								if grade == 10 {
									pop.PSA10 = popCount
								} else if grade == 9 {
									pop.PSA9 = popCount
								} else if grade == 8 {
									pop.PSA8 = popCount
								}
								found = true
							}
						}
					}
				}
			})
		} else {
			// Parse data row with grade columns
			table.Find("tbody tr, tr").Each(func(j int, row *goquery.Selection) {
				if j == 0 && len(gradeHeaders) > 0 {
					// Skip header row if we already parsed it
					return
				}

				cells := row.Find("td")
				if cells.Length() > 0 {
					cells.Each(func(k int, cell *goquery.Selection) {
						if grade, exists := gradeHeaders[k]; exists {
							popText := strings.TrimSpace(cell.Text())
							popCount := parsePopulationCount(popText)
							if popCount > 0 {
								gradeDistribution[fmt.Sprintf("PSA %d", grade)] = popCount
								if grade == 10 {
									pop.PSA10 = popCount
								} else if grade == 9 {
									pop.PSA9 = popCount
								} else if grade == 8 {
									pop.PSA8 = popCount
								}
								found = true
							}
						}
					})
				}
			})
		}
	})

	// Alternative: Look for specific grade divs or spans
	if !found {
		// Try to find grade populations in div elements
		doc.Find("div[class*='grade'], span[class*='grade']").Each(func(i int, elem *goquery.Selection) {
			text := strings.TrimSpace(elem.Text())
			// Look for pattern like "PSA 10: 1,234" or "Grade 10 (1234)"
			patterns := []string{
				`PSA\s+(\d+)[:\s]+([0-9,]+)`,
				`Grade\s+(\d+)[:\s\(]+([0-9,]+)`,
			}

			for _, pattern := range patterns {
				re := regexp.MustCompile(pattern)
				matches := re.FindStringSubmatch(text)
				if len(matches) == 3 {
					if grade, err := strconv.Atoi(matches[1]); err == nil && grade >= 1 && grade <= 10 {
						popCount := parsePopulationCount(matches[2])
						if popCount > 0 {
							gradeDistribution[fmt.Sprintf("PSA %d", grade)] = popCount
							if grade == 10 {
								pop.PSA10 = popCount
							} else if grade == 9 {
								pop.PSA9 = popCount
							} else if grade == 8 {
								pop.PSA8 = popCount
							}
							found = true
						}
					}
				}
			}
		})
	}

	if !found || len(gradeDistribution) == 0 {
		return nil, fmt.Errorf("no population data found in HTML")
	}

	// Calculate total population
	pop.TotalGraded = 0
	for _, count := range gradeDistribution {
		pop.TotalGraded += count
	}

	if s.debug {
		log.Printf("PSAScraper: Parsed population - Total: %d, PSA10: %d, PSA9: %d, PSA8: %d",
			pop.TotalGraded, pop.PSA10, pop.PSA9, pop.PSA8)
	}

	return pop, nil
}

// parsePopulationCount extracts a number from text that may contain commas and other characters
func parsePopulationCount(text string) int {
	// Remove commas and extract numbers
	text = strings.ReplaceAll(text, ",", "")
	text = strings.TrimSpace(text)

	// Extract just the number part
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(text)
	if match == "" {
		return 0
	}

	count, err := strconv.Atoi(match)
	if err != nil {
		return 0
	}

	return count
}

// normalizeString normalizes a string for cache keys
func normalizeString(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)
	// Remove special characters
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	return s
}

// GetCardPopulation is a convenience method that searches and scrapes in one call
func (s *PSAScraper) GetCardPopulation(ctx context.Context, set, number, name string) (*model.PSAPopulation, error) {
	// First, search for the card
	popURL, err := s.SearchCard(ctx, set, number, name)
	if err != nil {
		if s.debug {
			log.Printf("PSAScraper: Search failed for %s %s %s: %v", set, number, name, err)
		}
		// Return nil without error to allow fallback to mock provider
		return nil, nil
	}

	// Then scrape the population data
	pop, err := s.ScrapePopulation(ctx, popURL)
	if err != nil {
		if s.debug {
			log.Printf("PSAScraper: Scraping failed for %s: %v", popURL, err)
		}
		// Return nil without error to allow fallback to mock provider
		return nil, nil
	}

	return pop, nil
}
