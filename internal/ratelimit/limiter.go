package ratelimit

import (
	"sync"
	"time"
)

// Limiter implements a token bucket rate limiter
type Limiter struct {
	tokens     int
	maxTokens  int
	refillRate time.Duration
	mu         sync.Mutex
	lastRefill time.Time
}

// NewLimiter creates a new token bucket rate limiter
// maxTokens: maximum number of tokens in the bucket
// refillRate: how often to add one token to the bucket
func NewLimiter(maxTokens int, refillRate time.Duration) *Limiter {
	return &Limiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request can proceed immediately
// Returns true if a token is available and consumed
func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.refillTokens()

	if l.tokens > 0 {
		l.tokens--
		return true
	}

	return false
}

// Wait blocks until a token is available
func (l *Limiter) Wait() {
	for !l.Allow() {
		// Sleep for a short time before checking again
		time.Sleep(l.refillRate / time.Duration(l.maxTokens))
	}
}

// WaitWithTimeout waits for a token with a timeout
// Returns true if token was acquired, false if timeout exceeded
func (l *Limiter) WaitWithTimeout(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if l.Allow() {
			return true
		}

		// Sleep for a short time before checking again
		sleepTime := l.refillRate / time.Duration(l.maxTokens)
		if sleepTime > time.Until(deadline) {
			sleepTime = time.Until(deadline)
		}
		if sleepTime > 0 {
			time.Sleep(sleepTime)
		}
	}

	return false
}

// TokensAvailable returns the current number of tokens available
func (l *Limiter) TokensAvailable() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.refillTokens()
	return l.tokens
}

// refillTokens adds tokens based on elapsed time
// Must be called with mutex held
func (l *Limiter) refillTokens() {
	now := time.Now()
	elapsed := now.Sub(l.lastRefill)

	// Calculate how many tokens to add
	tokensToAdd := int(elapsed / l.refillRate)

	if tokensToAdd > 0 {
		l.tokens = min(l.maxTokens, l.tokens+tokensToAdd)
		l.lastRefill = now
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RateLimiterConfig holds configuration for API rate limiters
type RateLimiterConfig struct {
	PokemonTCGIO   *Limiter
	PriceCharting  *Limiter
	EBay           *Limiter
}

// NewDefaultRateLimiters creates rate limiters with sensible defaults for each API
func NewDefaultRateLimiters() *RateLimiterConfig {
	return &RateLimiterConfig{
		// Pokemon TCG API: 20,000 requests per hour = ~5.5 requests per second
		// Conservative: 1 request per 300ms = ~3.3 requests per second
		PokemonTCGIO: NewLimiter(10, 300*time.Millisecond),

		// PriceCharting: Assume similar limits, be conservative
		// 1 request per 500ms = 2 requests per second
		PriceCharting: NewLimiter(5, 500*time.Millisecond),

		// eBay Finding API: 5,000 requests per day for basic tier
		// ~0.06 requests per second, so 1 request per 16 seconds
		// But allow bursts with small bucket
		EBay: NewLimiter(3, 16*time.Second),
	}
}

// NewCustomRateLimiters creates rate limiters with custom configurations
func NewCustomRateLimiters(pokemonTCGRate, priceChartingRate, eBayRate time.Duration) *RateLimiterConfig {
	return &RateLimiterConfig{
		PokemonTCGIO:  NewLimiter(10, pokemonTCGRate),
		PriceCharting: NewLimiter(5, priceChartingRate),
		EBay:          NewLimiter(3, eBayRate),
	}
}