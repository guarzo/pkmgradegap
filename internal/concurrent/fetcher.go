package concurrent

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/guarzo/pkmgradegap/internal/model"
)

// Result represents the result of a concurrent operation
type Result struct {
	Card  model.Card
	Data  interface{}
	Error error
	Type  string // "card", "price", "population", "sales"
}

// ConcurrentFetcher manages concurrent data fetching operations
type ConcurrentFetcher struct {
	workers      int
	rateLimit    *rate.Limiter
	timeout      time.Duration
	errorHandler ErrorHandler
	progressChan chan Progress
	metrics      *FetchMetrics
	mu           sync.RWMutex
}

// FetcherConfig holds configuration for the concurrent fetcher
type FetcherConfig struct {
	Workers      int           // Number of concurrent workers
	RateLimit    rate.Limit    // Requests per second
	Timeout      time.Duration // Timeout per request
	ErrorHandler ErrorHandler  // Custom error handling
}

// ErrorHandler defines how to handle errors during fetching
type ErrorHandler func(card model.Card, err error, retryCount int) bool // returns true to retry

// Progress represents progress information
type Progress struct {
	Completed int
	Total     int
	Current   string
	StartTime time.Time
	Errors    int
}

// FetchMetrics tracks performance metrics
type FetchMetrics struct {
	TotalRequests  int
	SuccessfulReqs int
	FailedRequests int
	AverageLatency time.Duration
	TotalLatency   time.Duration
	CacheHits      int
	APICallsMade   int
	StartTime      time.Time
	EndTime        time.Time
	mu             sync.RWMutex
}

// NewConcurrentFetcher creates a new concurrent fetcher
func NewConcurrentFetcher(config FetcherConfig) *ConcurrentFetcher {
	workers := config.Workers
	if workers == 0 {
		workers = runtime.NumCPU()
		if workers > 10 {
			workers = 10 // Cap at 10 to be respectful to APIs
		}
	}

	rateLimit := config.RateLimit
	if rateLimit == 0 {
		rateLimit = rate.Limit(5) // 5 requests per second default
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &ConcurrentFetcher{
		workers:      workers,
		rateLimit:    rate.NewLimiter(rateLimit, workers),
		timeout:      timeout,
		errorHandler: config.ErrorHandler,
		progressChan: make(chan Progress, 100),
		metrics:      &FetchMetrics{StartTime: time.Now()},
	}
}

// FetchAll fetches data for all cards concurrently
func (f *ConcurrentFetcher) FetchAll(ctx context.Context, cards []model.Card, fetcher DataFetcher) []Result {
	if len(cards) == 0 {
		return nil
	}

	f.metrics.TotalRequests = len(cards)
	f.metrics.StartTime = time.Now()

	// Create channels for coordination
	jobs := make(chan model.Card, len(cards))
	results := make(chan Result, len(cards))

	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < f.workers; w++ {
		wg.Add(1)
		go f.worker(ctx, w, jobs, results, fetcher, &wg)
	}

	// Send jobs
	go func() {
		defer close(jobs)
		for _, card := range cards {
			select {
			case jobs <- card:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Start progress tracker
	progressCtx, progressCancel := context.WithCancel(ctx)
	go f.trackProgress(progressCtx, len(cards))

	// Collect results
	allResults := f.collectResults(ctx, results, len(cards))

	// Clean up
	wg.Wait()
	close(results)
	progressCancel()

	f.metrics.EndTime = time.Now()
	return allResults
}

// worker processes jobs from the jobs channel
func (f *ConcurrentFetcher) worker(ctx context.Context, id int, jobs <-chan model.Card, results chan<- Result, fetcher DataFetcher, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case card, ok := <-jobs:
			if !ok {
				return // Jobs channel closed
			}

			// Rate limit
			if err := f.rateLimit.Wait(ctx); err != nil {
				select {
				case results <- Result{Card: card, Error: err, Type: "rate_limit"}:
				case <-ctx.Done():
				}
				continue
			}

			// Fetch with timeout
			result := f.fetchWithTimeout(ctx, card, fetcher)
			f.updateMetrics(result)

			select {
			case results <- result:
			case <-ctx.Done():
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

// fetchWithTimeout fetches data with a timeout and retry logic
func (f *ConcurrentFetcher) fetchWithTimeout(ctx context.Context, card model.Card, fetcher DataFetcher) Result {
	timeoutCtx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	start := time.Now()
	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		data, err := fetcher.Fetch(timeoutCtx, card)

		if err == nil {
			return Result{
				Card: card,
				Data: data,
				Type: fetcher.Type(),
			}
		}

		lastErr = err

		// Check if we should retry
		if f.errorHandler != nil && attempt < maxRetries-1 {
			if f.errorHandler(card, err, attempt) {
				// Wait before retry with exponential backoff
				backoff := time.Duration(1<<uint(attempt)) * time.Second
				select {
				case <-time.After(backoff):
					continue
				case <-timeoutCtx.Done():
					return Result{Card: card, Error: timeoutCtx.Err(), Type: fetcher.Type()}
				}
			}
		}

		break
	}

	latency := time.Since(start)
	f.recordLatency(latency)

	return Result{
		Card:  card,
		Error: fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr),
		Type:  fetcher.Type(),
	}
}

// collectResults collects all results and tracks progress
func (f *ConcurrentFetcher) collectResults(ctx context.Context, results <-chan Result, total int) []Result {
	allResults := make([]Result, 0, total)
	completed := 0

	for completed < total {
		select {
		case result, ok := <-results:
			if !ok {
				return allResults
			}

			allResults = append(allResults, result)
			completed++

			// Send progress update
			select {
			case f.progressChan <- Progress{
				Completed: completed,
				Total:     total,
				Current:   fmt.Sprintf("%s #%s", result.Card.Name, result.Card.Number),
				StartTime: f.metrics.StartTime,
				Errors:    f.getErrorCount(),
			}:
			default:
				// Don't block if progress channel is full
			}

		case <-ctx.Done():
			return allResults
		}
	}

	return allResults
}

// trackProgress sends periodic progress updates
func (f *ConcurrentFetcher) trackProgress(ctx context.Context, total int) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Progress updates are sent from collectResults
		case <-ctx.Done():
			return
		}
	}
}

// DataFetcher interface for different types of data fetching
type DataFetcher interface {
	Fetch(ctx context.Context, card model.Card) (interface{}, error)
	Type() string
}

// ProgressChannel returns the progress channel for monitoring
func (f *ConcurrentFetcher) ProgressChannel() <-chan Progress {
	return f.progressChan
}

// GetMetrics returns current performance metrics
func (f *ConcurrentFetcher) GetMetrics() *FetchMetrics {
	f.metrics.mu.RLock()
	defer f.metrics.mu.RUnlock()

	metrics := FetchMetrics{
		TotalRequests:  f.metrics.TotalRequests,
		SuccessfulReqs: f.metrics.SuccessfulReqs,
		FailedRequests: f.metrics.FailedRequests,
		AverageLatency: f.metrics.AverageLatency,
		TotalLatency:   f.metrics.TotalLatency,
		CacheHits:      f.metrics.CacheHits,
		APICallsMade:   f.metrics.APICallsMade,
		StartTime:      f.metrics.StartTime,
		EndTime:        f.metrics.EndTime,
	}
	if !metrics.EndTime.IsZero() && metrics.SuccessfulReqs > 0 {
		metrics.AverageLatency = metrics.TotalLatency / time.Duration(metrics.SuccessfulReqs)
	}

	return &metrics
}

// updateMetrics updates performance metrics
func (f *ConcurrentFetcher) updateMetrics(result Result) {
	f.metrics.mu.Lock()
	defer f.metrics.mu.Unlock()

	if result.Error != nil {
		f.metrics.FailedRequests++
	} else {
		f.metrics.SuccessfulReqs++
	}
}

// recordLatency records request latency
func (f *ConcurrentFetcher) recordLatency(latency time.Duration) {
	f.metrics.mu.Lock()
	defer f.metrics.mu.Unlock()

	f.metrics.TotalLatency += latency
}

// getErrorCount returns current error count
func (f *ConcurrentFetcher) getErrorCount() int {
	f.metrics.mu.RLock()
	defer f.metrics.mu.RUnlock()
	return f.metrics.FailedRequests
}

// BatchFetcher handles fetching different types of data in parallel
type BatchFetcher struct {
	concurrentFetcher *ConcurrentFetcher
	cardFetcher       DataFetcher
	priceFetcher      DataFetcher
	populationFetcher DataFetcher
	salesFetcher      DataFetcher
}

// NewBatchFetcher creates a new batch fetcher
func NewBatchFetcher(config FetcherConfig) *BatchFetcher {
	return &BatchFetcher{
		concurrentFetcher: NewConcurrentFetcher(config),
	}
}

// SetFetchers configures the different data fetchers
func (b *BatchFetcher) SetFetchers(card, price, population, sales DataFetcher) {
	b.cardFetcher = card
	b.priceFetcher = price
	b.populationFetcher = population
	b.salesFetcher = sales
}

// FetchAllData fetches all types of data for cards concurrently
func (b *BatchFetcher) FetchAllData(ctx context.Context, cards []model.Card) *BatchResults {
	results := &BatchResults{
		Cards:      make(map[string]interface{}),
		Prices:     make(map[string]interface{}),
		Population: make(map[string]interface{}),
		Sales:      make(map[string]interface{}),
		Errors:     make(map[string][]error),
	}

	// Create separate contexts for each type of fetch
	// This allows different timeout/retry policies per data type

	var wg sync.WaitGroup

	// Fetch cards
	if b.cardFetcher != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cardResults := b.concurrentFetcher.FetchAll(ctx, cards, b.cardFetcher)
			b.processResults(cardResults, results.Cards, results.Errors, "cards")
		}()
	}

	// Fetch prices
	if b.priceFetcher != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			priceResults := b.concurrentFetcher.FetchAll(ctx, cards, b.priceFetcher)
			b.processResults(priceResults, results.Prices, results.Errors, "prices")
		}()
	}

	// Fetch population (may be filtered by targeting)
	if b.populationFetcher != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			popResults := b.concurrentFetcher.FetchAll(ctx, cards, b.populationFetcher)
			b.processResults(popResults, results.Population, results.Errors, "population")
		}()
	}

	// Fetch sales
	if b.salesFetcher != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			salesResults := b.concurrentFetcher.FetchAll(ctx, cards, b.salesFetcher)
			b.processResults(salesResults, results.Sales, results.Errors, "sales")
		}()
	}

	wg.Wait()
	return results
}

// processResults processes results and organizes them by card
func (b *BatchFetcher) processResults(results []Result, dataMap map[string]interface{}, errorMap map[string][]error, dataType string) {
	for _, result := range results {
		cardKey := fmt.Sprintf("%s-%s", result.Card.Number, result.Card.Name)

		if result.Error != nil {
			if errorMap[cardKey] == nil {
				errorMap[cardKey] = make([]error, 0)
			}
			errorMap[cardKey] = append(errorMap[cardKey], fmt.Errorf("%s: %w", dataType, result.Error))
		} else {
			dataMap[cardKey] = result.Data
		}
	}
}

// BatchResults contains all fetched data organized by type
type BatchResults struct {
	Cards      map[string]interface{} // cardKey -> card data
	Prices     map[string]interface{} // cardKey -> price data
	Population map[string]interface{} // cardKey -> population data
	Sales      map[string]interface{} // cardKey -> sales data
	Errors     map[string][]error     // cardKey -> errors
}

// GetCardResults returns results for a specific card
func (b *BatchResults) GetCardResults(cardKey string) CardResults {
	return CardResults{
		Card:       b.Cards[cardKey],
		Price:      b.Prices[cardKey],
		Population: b.Population[cardKey],
		Sales:      b.Sales[cardKey],
		Errors:     b.Errors[cardKey],
	}
}

// CardResults contains all data for a single card
type CardResults struct {
	Card       interface{}
	Price      interface{}
	Population interface{}
	Sales      interface{}
	Errors     []error
}

// HasErrors returns true if there are any errors for this card
func (c *CardResults) HasErrors() bool {
	return len(c.Errors) > 0
}

// IsComplete returns true if all expected data is present
func (c *CardResults) IsComplete() bool {
	return c.Card != nil && c.Price != nil && !c.HasErrors()
}

// DefaultErrorHandler provides a sensible default error handling strategy
func DefaultErrorHandler(card model.Card, err error, retryCount int) bool {
	// Retry on timeout or temporary network errors
	if err != nil {
		errStr := err.Error()
		if contains(errStr, "timeout") ||
			contains(errStr, "temporary") ||
			contains(errStr, "connection reset") {
			return retryCount < 2 // Retry up to 2 times
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findInString(s, substr))))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
