package testutil

import (
	"os"
	"strconv"
)

const (
	// Test token environment variables
	TestPriceChartingToken = "TEST_PRICECHARTING_TOKEN"
	TestEbayAppID          = "TEST_EBAY_APP_ID"
	TestPokemonAPIKey      = "TEST_POKEMON_API_KEY"
	TestPSAAPIKey          = "TEST_PSA_API_KEY"

	// Default test values when environment variables are not set
	DefaultTestToken = "test-token"
	DefaultTestKey   = "test-key"
)

// GetTestToken returns a test token from environment variable or default
func GetTestToken(envVar, defaultValue string) string {
	if token := os.Getenv(envVar); token != "" {
		return token
	}
	return defaultValue
}

// GetTestPriceChartingToken returns test token for PriceCharting API
func GetTestPriceChartingToken() string {
	return GetTestToken(TestPriceChartingToken, DefaultTestToken)
}

// GetTestEbayAppID returns test app ID for eBay API
func GetTestEbayAppID() string {
	return GetTestToken(TestEbayAppID, DefaultTestKey)
}

// GetTestPokemonAPIKey returns test API key for Pokemon TCG API
func GetTestPokemonAPIKey() string {
	return GetTestToken(TestPokemonAPIKey, DefaultTestKey)
}

// GetTestPSAAPIKey returns test API key for PSA API
func GetTestPSAAPIKey() string {
	return GetTestToken(TestPSAAPIKey, DefaultTestKey)
}

// IsTestMode returns true if we're running in test mode
func IsTestMode() bool {
	testMode := os.Getenv("TEST_MODE")
	if testMode == "" {
		return true // Default to test mode if not specified
	}

	enabled, _ := strconv.ParseBool(testMode)
	return enabled
}

// GetTestBaseURL returns a test base URL for the given service
func GetTestBaseURL(service string) string {
	switch service {
	case "ebay":
		return "https://api.ebay.test"
	case "pricecharting":
		return "https://api.pricecharting.test"
	case "pokemon":
		return "https://api.pokemontcg.test"
	case "psa":
		return "https://api.psa.test"
	default:
		return "https://api.test.local"
	}
}
