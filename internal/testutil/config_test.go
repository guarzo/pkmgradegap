package testutil

import (
	"os"
	"testing"
)

func TestGetTestToken(t *testing.T) {
	// Test with environment variable set
	os.Setenv("TEST_VAR", "env-value")
	defer os.Unsetenv("TEST_VAR")

	result := GetTestToken("TEST_VAR", "default-value")
	if result != "env-value" {
		t.Errorf("expected env-value, got %s", result)
	}

	// Test with environment variable unset
	result = GetTestToken("UNSET_VAR", "default-value")
	if result != "default-value" {
		t.Errorf("expected default-value, got %s", result)
	}
}

func TestGetTestPriceChartingToken(t *testing.T) {
	// Test default value
	token := GetTestPriceChartingToken()
	if token == "" {
		t.Error("token should not be empty")
	}

	// Test with environment variable
	os.Setenv(TestPriceChartingToken, "custom-token")
	defer os.Unsetenv(TestPriceChartingToken)

	token = GetTestPriceChartingToken()
	if token != "custom-token" {
		t.Errorf("expected custom-token, got %s", token)
	}
}

func TestIsTestMode(t *testing.T) {
	// Test default (should be true)
	if !IsTestMode() {
		t.Error("test mode should default to true")
	}

	// Test explicit true
	os.Setenv("TEST_MODE", "true")
	defer os.Unsetenv("TEST_MODE")

	if !IsTestMode() {
		t.Error("test mode should be true when explicitly set")
	}

	// Test explicit false
	os.Setenv("TEST_MODE", "false")
	if IsTestMode() {
		t.Error("test mode should be false when explicitly set to false")
	}
}

func TestGetTestBaseURL(t *testing.T) {
	tests := []struct {
		service  string
		expected string
	}{
		{"ebay", "https://api.ebay.test"},
		{"pricecharting", "https://api.pricecharting.test"},
		{"pokemon", "https://api.pokemontcg.test"},
		{"psa", "https://api.psa.test"},
		{"unknown", "https://api.test.local"},
	}

	for _, test := range tests {
		result := GetTestBaseURL(test.service)
		if result != test.expected {
			t.Errorf("for service %s, expected %s, got %s", test.service, test.expected, result)
		}
	}
}
