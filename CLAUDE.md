# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Build and Run
```bash
# Build the CLI tool
go build -o pkmgradegap ./cmd/pkmgradegap

# Run directly with go run
go run ./cmd/pkmgradegap --help
go run ./cmd/pkmgradegap --list-sets
go run ./cmd/pkmgradegap --set "Surging Sparks" --analysis rank

# Start web server
go run ./cmd/pkmgradegap server --port 8080
```

### Testing
```bash
# Run all tests
go test ./...

# Test with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/gamestop/
go test ./internal/monitoring/
go test ./internal/fusion/

# Run integration tests
go test ./internal/integration/

# Run load tests
go run scripts/load_test.go
# or use the wrapper script
./scripts/run_load_test.sh
```

### Development
```bash
# Initialize/update dependencies
go mod tidy

# Build without creating executable (syntax check)
go build ./cmd/pkmgradegap

# Run with environment variables
PRICECHARTING_TOKEN="token" EBAY_APP_ID="app_id" POKEMON_PRICE_TRACKER_API_KEY="key" go run ./cmd/pkmgradegap --set "Surging Sparks"

# Testing mode without API keys (except GameStop which uses web scraping)
PRICECHARTING_TOKEN="test" EBAY_APP_ID="mock" POKEMON_PRICE_TRACKER_API_KEY="test" ./pkmgradegap --set "Surging Sparks" --with-ebay --with-pop --with-sales --with-gamestop
```

## Architecture Overview

CLI tool that analyzes Pokemon card price gaps between raw and graded conditions to identify profitable grading opportunities. Uses provider-based architecture for extensible data source integration.

### Core Components

**Provider Pattern**: Interface-based providers for different data sources:
- `cards.PokeTCGIO`: Pokemon TCG API with embedded TCGPlayer/Cardmarket prices
- `prices.PriceCharting`: Graded card prices with specific condition mappings
- `gamestop.GameStopClient`: GameStop web scraping for graded card listings and market data
- `ebay.Client`: eBay Finding API for live market validation
- `population.PSAProvider`: PSA population report lookups with web scraping capabilities
- `sales.Provider`: Sales transaction data from marketplaces
- `fusion.Engine`: Multi-source price data fusion with confidence scoring
- `monitoring.AlertEngine`: Price change detection, volatility alerts, and opportunity identification
- `cache.MultilayerCache`: File and memory-based caching with TTL management

**Data Flow**:
1. CLI flags determine operation mode and parameters
2. Card provider fetches cards with pagination (250 cards/page)
3. Price providers lookup using query pattern: "pokemon {SetName} {CardName} #{Number}"
4. Analysis module normalizes data into `Row` structs and applies scoring algorithm
5. Results output as CSV with optional monitoring reports

### Key Data Structures

- **`model.Card`**: Central card representation with embedded price blocks
- **`analysis.Row`**: Normalized data containing card info, prices, population data
- **`analysis.ScoredRow`**: Extended Row with calculated score and scoring factors
- **`gamestop.TradeInData`**: GameStop specific pricing and trade values
- **`fusion.FusedPrice`**: Multi-source price fusion with confidence intervals
- **`monitoring.Alert`**: Price change alerts with severity levels
- **`population.PopulationData`**: PSA grading population data with grade breakdowns, scarcity levels, and trends

### Price Mapping

PriceCharting API response fields:
- `manual-only-price` → PSA 10
- `graded-price` → Grade 9 (PSA/BGS 9)
- `box-only-price` → Grade 9.5 (CGC/BGS 9.5)
- `bgs-10-price` → BGS 10
- `loose-price` → Ungraded/Raw

### PSA Population API Response Format

**Individual Card Population Response:**
```json
{
  "success": true,
  "data": {
    "specId": 123456,
    "description": "2022 Pokemon Japanese VMAX Climax Charizard VMAX #003/184",
    "brand": "Pokemon",
    "category": "Pokemon TCG",
    "sport": "Non-Sport",
    "year": "2022",
    "setName": "VMAX Climax",
    "population": {
      "total": 8769,
      "auth": 45,
      "grades": {
        "1": 12, "2": 23, "3": 67, "4": 128, "5": 234,
        "6": 445, "7": 890, "8": 1920, "9": 2800, "10": 1250
      },
      "lastUpdate": "2024-01-15"
    }
  }
}
```

**Set Population Search Response:**
```json
{
  "success": true,
  "results": [
    {
      "specId": 123456,
      "description": "Card description with #number",
      "setName": "Set Name",
      "population": { /* same structure as above */ }
    }
  ]
}
```

**Error Response:**
```json
{
  "success": false,
  "error": "Card not found in PSA database"
}
```

**Population Data Processing:**
- Grades 1-10 map to "PSA 1" through "PSA 10" keys
- Qualified grades (e.g., "10Q") map to qualifier counts
- Scarcity levels: ULTRA_RARE (≤10 PSA 10s), RARE (≤50), UNCOMMON (≤500), COMMON (>500)
- Trends calculated from total population and last update age

### Analysis Modes

1. **rank**: Default scoring algorithm for grading opportunities
2. **raw-vs-psa10**: Simple dollar difference analysis
3. **crossgrade**: CGC/BGS to PSA crossgrade ROI
4. **alerts**: Compare snapshots for price changes
5. **trends**: Historical trend analysis
6. **bulk-optimize**: PSA submission batch optimization
7. **market-timing**: Seasonal timing recommendations

### Scoring Algorithm

Multiple factors in ranking:
- Base score: (PSA10 - Raw - Total Costs)
- Premium lift bonus for steep PSA10 premiums
- Japanese card multiplier (configurable)
- Population scarcity bonus (0-15 points)
- Volatility penalty for unstable prices
- GameStop trade bonus when applicable

## Environment Configuration

```bash
export PRICECHARTING_TOKEN="your_token"         # Required for graded prices
export EBAY_APP_ID="your_app_id"                # Optional for eBay listings
export POKEMONTCGIO_API_KEY="optional_key"      # Optional, increases rate limits
export POKEMON_PRICE_TRACKER_API_KEY="key"      # Optional for sales data
export PSA_POPULATION_API_KEY="psa_key"         # Optional for PSA population
# GameStop integration uses web scraping (no API key needed)
# Warning: May break if GameStop changes their website structure
```

## Important Implementation Details

- **Price Storage**: Prices stored as cents (integers) to avoid float precision issues
- **Set Matching**: Case-insensitive with exact match preference
- **Query Pattern**: GameStop uses specific SKU pattern: "pokemon-tcg-{normalized-set}-{card-name}-{number}"
- **Rate Limiting**: Built-in rate limiter for all API calls
- **Concurrent Processing**: Parallel card processing with configurable worker pools
- **Progress Indicators**: Real-time progress bars with ETA estimation
- **Memory Management**: LRU cache eviction and memory pressure monitoring
- **Error Resilience**: Continues processing on individual card failures

## Web Interface (Sprint 4)

### Server Mode
```bash
# Start server with default port 8080
./pkmgradegap server

# Custom port
./pkmgradegap server --port 9090

# With auto-open browser
./pkmgradegap server --auto-open
```

### API Endpoints
- `GET /api/health` - Health check with provider status
- `GET /api/sets` - List available sets
- `POST /api/analyze` - Start analysis job
- `GET /api/jobs/{id}` - Get job status
- `GET /api/jobs/{id}/stream` - SSE stream for real-time updates
- `DELETE /api/cache` - Clear cache
- `GET /api/metrics` - Prometheus metrics

### Frontend Features
- Chart.js visualizations for price trends and ROI
- Advanced filtering with saved presets
- Batch analysis for multiple sets
- Export to CSV/JSON/PDF
- Theme selection (auto/light/dark)
- Virtual scrolling for large datasets
- Real-time progress with SSE

### Load Testing
```bash
# Run comprehensive load test
./scripts/run_load_test.sh

# Custom parameters
go run scripts/load_test.go -users 50 -duration 60s -ramp 10s
```

## Extending the System

### Adding Data Sources (Provider Development Guide)

Creating a new data provider requires implementing the standard provider interface and following established patterns:

#### 1. Provider Interface Implementation
```go
type Provider interface {
    Available() bool
    GetProviderName() string
    // Additional methods specific to provider type (prices, population, etc.)
}
```

#### 2. Directory Structure
```
internal/newprovider/
├── provider.go        # Main provider implementation
├── provider_test.go   # Unit tests
├── mock.go           # Mock implementation for testing
├── client.go         # HTTP client (if needed)
└── types.go          # Provider-specific data structures
```

#### 3. Implementation Pattern
```go
package newprovider

type Config struct {
    APIKey         string
    BaseURL        string
    RequestTimeout time.Duration
    CacheEnabled   bool
}

type NewProvider struct {
    config Config
    client *http.Client
    cache  *cache.Cache
    limiter *ratelimit.Limiter
}

func NewProvider(config Config) *NewProvider {
    // Initialize client, cache, rate limiter
    return &NewProvider{...}
}

func (p *NewProvider) Available() bool {
    // Check if provider is properly configured and accessible
    return p.config.APIKey != ""
}
```

#### 4. Integration Steps
1. Add provider to main application configuration
2. Update fusion rules in `internal/fusion/engine.go`
3. Add provider health checks in web server
4. Create integration tests in `internal/integration/`
5. Add mock provider for testing scenarios

#### 5. Testing Requirements
- Unit tests with >80% coverage
- Mock provider for isolated testing
- Integration tests with real API responses
- Error handling and edge case testing
- Rate limiting and caching behavior tests

### Mock vs Production Provider Usage

#### Mock Provider Architecture
Mock providers are isolated test implementations that:
- Implement the same interface as production providers
- Return static, predictable data for testing
- Are activated only in test environments
- Never affect production data or analytics

#### Environment-Based Provider Selection
```go
// Environment variable controls mock usage
if os.Getenv("PROVIDER_MOCK") == "true" {
    provider = NewMockProvider()
} else {
    provider = NewProductionProvider(config)
}
```

#### Mock Provider Guidelines
1. **Activation**: Require explicit environment variable (`PROVIDER_MOCK=true`)
2. **Data Isolation**: Use clearly identifiable test data
3. **Logging**: Log when mock providers are active
4. **Interface Compliance**: Implement identical interface as production provider
5. **Error Simulation**: Include error conditions for testing

#### Production Safety
- Runtime checks prevent mock usage in production
- Configuration validation at startup
- Clear logging when mocks are enabled
- Separate mock data generation for dynamic testing

### Adding Analysis Modes
1. Create report function in `internal/analysis/`
2. Add case in main.go switch statement
3. Follow CSV output patterns
4. Include progress indicators

### Adding Monitoring Features
1. Implement analyzer in `internal/monitoring/`
2. Update AlertEngine for new alert types
3. Add CSV export for new metrics