package ratelimit

import (
	"sync"
	"testing"
	"time"
)

func TestLimiter_Allow(t *testing.T) {
	// Create limiter with 3 tokens, refill every 100ms
	limiter := NewLimiter(3, 100*time.Millisecond)

	// Should allow 3 requests immediately
	for i := 0; i < 3; i++ {
		if !limiter.Allow() {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 4th request should be denied
	if limiter.Allow() {
		t.Error("4th request should be denied")
	}

	// Wait for refill and try again
	time.Sleep(150 * time.Millisecond)
	if !limiter.Allow() {
		t.Error("Request after refill should be allowed")
	}
}

func TestLimiter_TokenRefill(t *testing.T) {
	limiter := NewLimiter(2, 50*time.Millisecond)

	// Consume all tokens
	limiter.Allow()
	limiter.Allow()

	// Should be empty
	if limiter.TokensAvailable() != 0 {
		t.Errorf("Expected 0 tokens, got %d", limiter.TokensAvailable())
	}

	// Wait for one refill cycle
	time.Sleep(60 * time.Millisecond)

	// Should have 1 token
	available := limiter.TokensAvailable()
	if available != 1 {
		t.Errorf("Expected 1 token after refill, got %d", available)
	}

	// Wait for another refill cycle
	time.Sleep(60 * time.Millisecond)

	// Should be back to max (2 tokens)
	available = limiter.TokensAvailable()
	if available != 2 {
		t.Errorf("Expected 2 tokens (max), got %d", available)
	}
}

func TestLimiter_Wait(t *testing.T) {
	limiter := NewLimiter(1, 100*time.Millisecond)

	// Consume the token
	if !limiter.Allow() {
		t.Fatal("First request should be allowed")
	}

	// Wait should block and then succeed
	start := time.Now()
	limiter.Wait()
	elapsed := time.Since(start)

	// Should have waited approximately the refill time
	if elapsed < 90*time.Millisecond || elapsed > 200*time.Millisecond {
		t.Errorf("Wait took %v, expected ~100ms", elapsed)
	}

	// Should have consumed the refilled token
	if limiter.Allow() {
		t.Error("Token should have been consumed by Wait()")
	}
}

func TestLimiter_WaitWithTimeout(t *testing.T) {
	limiter := NewLimiter(1, 200*time.Millisecond)

	// Consume the token
	limiter.Allow()

	// Wait with short timeout - should fail
	start := time.Now()
	success := limiter.WaitWithTimeout(50 * time.Millisecond)
	elapsed := time.Since(start)

	if success {
		t.Error("WaitWithTimeout should have failed with short timeout")
	}

	if elapsed < 40*time.Millisecond || elapsed > 80*time.Millisecond {
		t.Errorf("Timeout took %v, expected ~50ms", elapsed)
	}

	// Wait with long timeout - should succeed
	start = time.Now()
	success = limiter.WaitWithTimeout(300 * time.Millisecond)
	elapsed = time.Since(start)

	if !success {
		t.Error("WaitWithTimeout should have succeeded with long timeout")
	}

	if elapsed < 180*time.Millisecond || elapsed > 350*time.Millisecond {
		t.Errorf("Wait took %v, expected ~200ms", elapsed)
	}
}

func TestLimiter_Concurrent(t *testing.T) {
	limiter := NewLimiter(5, 10*time.Millisecond)

	const numGoroutines = 10
	const requestsPerGoroutine = 10

	var wg sync.WaitGroup
	var totalAllowed int64
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var localAllowed int64

			for j := 0; j < requestsPerGoroutine; j++ {
				if limiter.Allow() {
					localAllowed++
				}
				time.Sleep(1 * time.Millisecond) // Small delay between requests
			}

			mu.Lock()
			totalAllowed += localAllowed
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Should have allowed some requests (at least the initial bucket)
	// but not all requests (due to rate limiting)
	totalRequests := int64(numGoroutines * requestsPerGoroutine)
	if totalAllowed == 0 {
		t.Error("No requests were allowed")
	}
	if totalAllowed >= totalRequests {
		t.Error("All requests were allowed, rate limiting didn't work")
	}

	t.Logf("Allowed %d/%d requests", totalAllowed, totalRequests)
}

func TestLimiter_BurstBehavior(t *testing.T) {
	// Create limiter that allows 3 requests initially, then 1 every 100ms
	limiter := NewLimiter(3, 100*time.Millisecond)

	// Should allow burst of 3 requests
	for i := 0; i < 3; i++ {
		if !limiter.Allow() {
			t.Errorf("Burst request %d should be allowed", i+1)
		}
	}

	// 4th request should be denied (no burst capacity left)
	if limiter.Allow() {
		t.Error("Request beyond burst should be denied")
	}

	// Wait for one refill and try again
	time.Sleep(150 * time.Millisecond)
	if !limiter.Allow() {
		t.Error("Request after refill should be allowed")
	}
}

func TestNewDefaultRateLimiters(t *testing.T) {
	limiters := NewDefaultRateLimiters()

	if limiters.PokemonTCGIO == nil {
		t.Error("PokemonTCGIO limiter should not be nil")
	}
	if limiters.PriceCharting == nil {
		t.Error("PriceCharting limiter should not be nil")
	}
	if limiters.EBay == nil {
		t.Error("EBay limiter should not be nil")
	}

	// Test that they work
	if !limiters.PokemonTCGIO.Allow() {
		t.Error("PokemonTCGIO limiter should allow first request")
	}
	if !limiters.PriceCharting.Allow() {
		t.Error("PriceCharting limiter should allow first request")
	}
	if !limiters.EBay.Allow() {
		t.Error("EBay limiter should allow first request")
	}
}

func TestNewCustomRateLimiters(t *testing.T) {
	customLimiters := NewCustomRateLimiters(
		500*time.Millisecond, // Pokemon TCG
		1*time.Second,        // PriceCharting
		30*time.Second,       // eBay
	)

	if customLimiters.PokemonTCGIO == nil {
		t.Error("Custom PokemonTCGIO limiter should not be nil")
	}
	if customLimiters.PriceCharting == nil {
		t.Error("Custom PriceCharting limiter should not be nil")
	}
	if customLimiters.EBay == nil {
		t.Error("Custom EBay limiter should not be nil")
	}

	// Test basic functionality
	if !customLimiters.PokemonTCGIO.Allow() {
		t.Error("Custom PokemonTCGIO limiter should allow first request")
	}
}

func TestLimiter_EdgeCases(t *testing.T) {
	// Test with very fast refill
	fastLimiter := NewLimiter(1, 1*time.Millisecond)
	fastLimiter.Allow() // Consume token

	// Should refill very quickly
	time.Sleep(5 * time.Millisecond)
	if !fastLimiter.Allow() {
		t.Error("Fast limiter should have refilled")
	}

	// Test with very slow refill
	slowLimiter := NewLimiter(2, 1*time.Hour)
	slowLimiter.Allow()
	slowLimiter.Allow()

	// Should not refill quickly
	time.Sleep(10 * time.Millisecond)
	if slowLimiter.Allow() {
		t.Error("Slow limiter should not have refilled yet")
	}
}
