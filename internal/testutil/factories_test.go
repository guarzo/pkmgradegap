package testutil

import (
	"strings"
	"testing"
	"time"
)

func TestNewTestDataFactory(t *testing.T) {
	// Test with fixed seed
	factory1 := NewTestDataFactory(12345)
	factory2 := NewTestDataFactory(12345)

	// Should generate same values with same seed
	token1 := factory1.GenerateTestToken()
	token2 := factory2.GenerateTestToken()

	if token1 != token2 {
		t.Errorf("factories with same seed should generate same values, got %s and %s", token1, token2)
	}

	// Test with different seeds
	factory3 := NewTestDataFactory(54321)
	token3 := factory3.GenerateTestToken()

	if token1 == token3 {
		t.Error("factories with different seeds should generate different values")
	}
}

func TestGenerateTestToken(t *testing.T) {
	factory := NewTestDataFactory(0)
	token := factory.GenerateTestToken()

	if !strings.HasPrefix(token, "test-token-") {
		t.Errorf("token should start with 'test-token-', got %s", token)
	}

	if len(token) < 15 {
		t.Errorf("token should be longer than 15 characters, got %s", token)
	}
}

func TestGenerateTestURL(t *testing.T) {
	factory := NewTestDataFactory(0)
	url := factory.GenerateTestURL("ebay", "charizard")

	if !strings.Contains(url, "ebay.test.local") {
		t.Errorf("URL should contain service domain, got %s", url)
	}

	if !strings.Contains(url, "charizard") {
		t.Errorf("URL should contain resource, got %s", url)
	}

	if !strings.HasPrefix(url, "https://") {
		t.Errorf("URL should use HTTPS, got %s", url)
	}
}

func TestGenerateTestCardNumber(t *testing.T) {
	factory := NewTestDataFactory(0)
	number := factory.GenerateTestCardNumber()

	if len(number) != 3 {
		t.Errorf("card number should be 3 digits, got %s", number)
	}

	// Should be between 001 and 300
	if number < "001" || number > "300" {
		t.Errorf("card number should be between 001 and 300, got %s", number)
	}
}

func TestGenerateTestSetName(t *testing.T) {
	factory := NewTestDataFactory(0)
	setName := factory.GenerateTestSetName()

	if !strings.HasPrefix(setName, "Test ") {
		t.Errorf("set name should start with 'Test ', got %s", setName)
	}
}

func TestGenerateTestCardName(t *testing.T) {
	factory := NewTestDataFactory(0)
	cardName := factory.GenerateTestCardName()

	if !strings.HasPrefix(cardName, "Test ") {
		t.Errorf("card name should start with 'Test ', got %s", cardName)
	}
}

func TestGenerateTestPrice(t *testing.T) {
	factory := NewTestDataFactory(0)
	price := factory.GenerateTestPrice()

	if price < 500 || price > 50500 {
		t.Errorf("price should be between 500 and 50500 cents, got %d", price)
	}
}

func TestGenerateTestDate(t *testing.T) {
	factory := NewTestDataFactory(0)
	date := factory.GenerateTestDate()
	now := time.Now()

	// Should be within the last year
	oneYearAgo := now.AddDate(-1, 0, 0)
	if date.Before(oneYearAgo) || date.After(now) {
		t.Errorf("date should be within last year, got %v", date)
	}
}

func TestGenerateTestGrade(t *testing.T) {
	factory := NewTestDataFactory(0)
	grade := factory.GenerateTestGrade()

	validGrades := []string{"Raw", "PSA 8", "PSA 9", "PSA 10", "BGS 9", "BGS 10"}
	found := false
	for _, valid := range validGrades {
		if grade == valid {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("grade should be one of valid grades, got %s", grade)
	}
}

func TestGenerateTestMarketplace(t *testing.T) {
	factory := NewTestDataFactory(0)
	marketplace := factory.GenerateTestMarketplace()

	if !strings.HasPrefix(marketplace, "Test ") {
		t.Errorf("marketplace should start with 'Test ', got %s", marketplace)
	}
}
