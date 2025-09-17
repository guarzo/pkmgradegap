package ebay

// Provider defines the interface for eBay listing providers
type Provider interface {
	Available() bool
	SearchRawListings(setName, cardName, number string, max int) ([]Listing, error)
}

// Ensure Client implements Provider
var _ Provider = (*Client)(nil)