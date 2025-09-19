# PriceCharting API Field Documentation

## Overview
This document provides comprehensive documentation of all available fields from the PriceCharting API as implemented in the pkmgradegap tool.

## API Response Fields

### Core Product Fields
| Field Name | Type | Description | Maps To |
|------------|------|-------------|---------|
| `id` | string | Unique product identifier | `PCMatch.ID` |
| `product-name` | string | Full product name | `PCMatch.ProductName` |
| `console` | string | Platform/category (e.g., "Pokemon") | Not currently extracted |
| `release-date` | string | Product release date | Not currently extracted |
| `genre` | string | Product genre/type | Not currently extracted |
| `upc` | string | Universal Product Code | To be added in Sprint 4 |

### Price Fields - Graded Cards
| Field Name | Type | Description | Maps To |
|------------|------|-------------|---------|
| `loose-price` | int | Ungraded/raw card price in cents | `PCMatch.LooseCents` |
| `graded-price` | int | PSA/BGS 9 graded price | `PCMatch.Grade9Cents` |
| `box-only-price` | int | CGC/BGS 9.5 price | `PCMatch.Grade95Cents` |
| `manual-only-price` | int | PSA 10 price | `PCMatch.PSA10Cents` |
| `bgs-10-price` | int | BGS 10 Black Label price | `PCMatch.BGS10Cents` |

### Price Fields - Other Conditions
| Field Name | Type | Description | Maps To |
|------------|------|-------------|---------|
| `new-price` | int | Sealed product price | `PCMatch.NewPriceCents` |
| `cib-price` | int | Complete In Box price | `PCMatch.CIBPriceCents` |
| `manual-price` | int | Manual/instructions only | `PCMatch.ManualPriceCents` |
| `box-price` | int | Box/packaging only | `PCMatch.BoxPriceCents` |

### Sales Data Fields
| Field Name | Type | Description | Maps To |
|------------|------|-------------|---------|
| `sales-volume` | int | Number of recent sales | `PCMatch.SalesVolume` |
| `last-sold-date` | string | Date of most recent sale | `PCMatch.LastSoldDate` |
| `sales-data` | array | Detailed sales transactions | `PCMatch.RecentSales` |

#### Sales Data Array Structure
Each element in `sales-data` contains:
| Field Name | Type | Description |
|------------|------|-------------|
| `sale-price` | int | Transaction price in cents |
| `sale-date` | string | Date of transaction |
| `grade` | string | Card grade (if applicable) |
| `source` | string | Marketplace (eBay, PWCC, etc.) |

### Retail Pricing Fields
| Field Name | Type | Description | Maps To |
|------------|------|-------------|---------|
| `retail-buy-price` | int | Dealer/store buy price | `PCMatch.RetailBuyPrice` |
| `retail-sell-price` | int | Dealer/store sell price | `PCMatch.RetailSellPrice` |

### Future Fields (Planned)
These fields are available in the API but not yet extracted:

| Field Name | Type | Description | Planned Sprint |
|------------|------|-------------|---------------|
| `price-history` | array | Historical price data points | Sprint 5 |
| `marketplace-listings` | array | Active marketplace offers | Sprint 3 |
| `lowest-listing` | int | Lowest current listing price | Sprint 3 |
| `average-listing` | int | Average of all listings | Sprint 3 |
| `listing-count` | int | Number of active listings | Sprint 3 |
| `trend-30d` | string | 30-day price trend | Sprint 5 |
| `trend-90d` | string | 90-day price trend | Sprint 5 |
| `volatility-score` | float | Price volatility metric | Sprint 5 |

## API Endpoints

### Currently Used
1. **Product Search (Single Match)**
   - Endpoint: `/api/product`
   - Parameters: `q` (query string), `t` (token)
   - Returns: Single best matching product

2. **Product Search (Multiple)**
   - Endpoint: `/api/products`
   - Parameters: `q` (query string), `t` (token)
   - Returns: List of matching products

3. **Product by ID**
   - Endpoint: `/api/product`
   - Parameters: `id` (product ID), `t` (token)
   - Returns: Specific product details

### Planned Endpoints
1. **Marketplace/Offers API** (Sprint 3)
   - Endpoint: `/api/offers`
   - Purpose: Real-time marketplace listings

2. **Historical Data API** (Sprint 5)
   - Endpoint: `/api/price-history`
   - Purpose: Historical price trends

3. **Batch API** (Sprint 2)
   - Endpoint: `/api/batch`
   - Purpose: Process multiple products in one request

## Data Processing Notes

### Price Normalization
- All prices are stored as integers in cents to avoid floating-point precision issues
- Conversion: API value → cents (multiply by 1 if already in cents)
- Display: cents → dollars (divide by 100)

### Type Conversion
The implementation handles multiple input types for price fields:
- `float64`: Converted to int
- `int`: Used directly
- `string`: Parsed as float then converted to int
- `null`/`nil`: Defaults to 0

### Error Handling
- Retry logic with exponential backoff (3 attempts)
- Delays: 1s, 2s, 4s between retries
- 4xx errors: No retry (client error)
- 5xx errors: Automatic retry
- Network errors: Automatic retry

### Caching Strategy
- Cache TTL: 2 hours for successful lookups
- Cache key format: `pricecharting:{setName}:{cardName}:{number}`
- Cache miss: Fetches from API
- Cache hit: Returns cached data immediately

## Query Formation
Cards are searched using the pattern:
```
pokemon {SetName} {CardName} #{Number}
```

Examples:
- `pokemon Base Set Charizard #4`
- `pokemon Surging Sparks Pikachu #238`
- `pokemon Celebrations Mew #025`

## Response Validation
A valid product response must have at least one of:
- `loose-price`
- `manual-only-price` (PSA 10)
- `graded-price` (Grade 9)

## Implementation Status

### ✅ Completed (Sprint 1)
- Sales data extraction (`sales-volume`, `last-sold-date`)
- Retail pricing fields (`retail-buy-price`, `retail-sell-price`)
- Additional price fields (`new-price`, `cib-price`, `manual-price`, `box-price`)
- Enhanced error handling with retry logic
- Comprehensive type conversion and null handling
- Unit and integration tests

### 🔄 In Progress
- Documentation and field mapping guide

### 📋 Planned
- Sprint 2: Batch processing to reduce API calls
- Sprint 3: Marketplace data integration
- Sprint 4: UPC lookup capabilities
- Sprint 5: Historical trend analysis
- Sprint 6: Web interface enhancements

## Testing

### Unit Tests
Located in `internal/prices/pricechart_test.go`:
- Field extraction validation
- Type conversion testing
- Null value handling
- Sales data processing
- Cache functionality

### Integration Tests
Located in `internal/integration/pricechart_integration_test.go`:
- Live API connectivity
- Data completeness verification
- Retry logic validation
- Cache performance testing
- Sales data extraction

### Running Tests
```bash
# Unit tests only
go test ./internal/prices/

# Integration tests (requires PRICECHARTING_TOKEN)
PRICECHARTING_TOKEN="your_token" go test ./internal/integration/

# All tests with coverage
go test -cover ./...
```

## Environment Configuration

### Required
```bash
export PRICECHARTING_TOKEN="your_token"
```

### Optional (for testing)
```bash
export TEST_MODE=true  # Use mock data
export DEBUG=true      # Enable verbose logging
```

## Troubleshooting

### Common Issues

1. **No price data returned**
   - Verify the card name and number are correct
   - Check if the card exists in PriceCharting database
   - Some promotional or special cards may not have price data

2. **API rate limiting**
   - Implement caching (already done)
   - Use batch processing (Sprint 2)
   - Add delays between requests in bulk operations

3. **Inconsistent field presence**
   - Not all cards have all price points
   - Sales data may not be available for all cards
   - Retail pricing is dealer-specific and may be missing

## Performance Metrics

### Current Performance
- Single card lookup: ~500ms (without cache)
- Cached lookup: <10ms
- Retry on failure: Up to 7 seconds (worst case)

### Target Performance (After Optimization)
- Batch processing: 15-20 cards per request
- 80% reduction in API calls
- 60% improvement in response time

## Contact & Support

For PriceCharting API issues:
- Documentation: https://www.pricecharting.com/api-documentation
- Support: api@pricecharting.com

For pkmgradegap implementation:
- GitHub Issues: https://github.com/guarzo/pkmgradegap/issues
- Documentation: This file and CLAUDE.md