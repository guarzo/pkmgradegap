package testutil

import (
	"fmt"
	"math/rand"
	"time"
)

// TestDataFactory provides methods for generating dynamic test data
type TestDataFactory struct {
	rand *rand.Rand
}

// NewTestDataFactory creates a new test data factory with a seeded random generator
func NewTestDataFactory(seed int64) *TestDataFactory {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	return &TestDataFactory{
		rand: rand.New(rand.NewSource(seed)),
	}
}

// GenerateTestToken generates a random test token
func (f *TestDataFactory) GenerateTestToken() string {
	return fmt.Sprintf("test-token-%d", f.rand.Int63())
}

// GenerateTestURL generates a test URL for the given service and resource
func (f *TestDataFactory) GenerateTestURL(service, resource string) string {
	return fmt.Sprintf("https://%s.test.local/%s/%d", service, resource, f.rand.Int63())
}

// GenerateTestCardNumber generates a random card number for testing
func (f *TestDataFactory) GenerateTestCardNumber() string {
	return fmt.Sprintf("%03d", f.rand.Intn(300)+1)
}

// GenerateTestSetName generates a random test set name
func (f *TestDataFactory) GenerateTestSetName() string {
	sets := []string{"Test Base Set", "Test Jungle", "Test Fossil", "Test Rocket", "Test Gym"}
	return sets[f.rand.Intn(len(sets))]
}

// GenerateTestCardName generates a random test card name
func (f *TestDataFactory) GenerateTestCardName() string {
	names := []string{"Test Pikachu", "Test Charizard", "Test Blastoise", "Test Venusaur", "Test Mewtwo"}
	return names[f.rand.Intn(len(names))]
}

// GenerateTestPrice generates a random price in cents
func (f *TestDataFactory) GenerateTestPrice() int {
	return f.rand.Intn(50000) + 500 // Between $5 and $500
}

// GenerateTestDate generates a random date within the last year
func (f *TestDataFactory) GenerateTestDate() time.Time {
	days := f.rand.Intn(365)
	return time.Now().AddDate(0, 0, -days)
}

// GenerateTestGrade generates a random card grade
func (f *TestDataFactory) GenerateTestGrade() string {
	grades := []string{"Raw", "PSA 8", "PSA 9", "PSA 10", "BGS 9", "BGS 10"}
	return grades[f.rand.Intn(len(grades))]
}

// GenerateTestMarketplace generates a random marketplace name
func (f *TestDataFactory) GenerateTestMarketplace() string {
	marketplaces := []string{"Test eBay", "Test PWCC", "Test Goldin", "Test Heritage"}
	return marketplaces[f.rand.Intn(len(marketplaces))]
}
