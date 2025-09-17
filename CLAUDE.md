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

# Build and run executable
./pkmgradegap --set "Surging Sparks" --analysis rank --top 10
```

### Testing
```bash
# Run all tests
go test ./...

# Test with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./internal/analysis/

# Test specific package
go test ./internal/ebay/

# Run specific test
go test -run TestScoreRow ./internal/analysis/
```

### Development
```bash
# Initialize/update dependencies
go mod tidy

# Build without creating executable (syntax check)
go build ./cmd/pkmgradegap

# Run with environment variables
PRICECHARTING_TOKEN="token" EBAY_APP_ID="app_id" POKEMON_PRICE_TRACKER_API_KEY="key" go run ./cmd/pkmgradegap --set "Surging Sparks"

# Mock mode for testing without API keys
PRICECHARTING_TOKEN="test" EBAY_APP_ID="mock" POKEMON_PRICE_TRACKER_API_KEY="test" ./pkmgradegap --set "Surging Sparks" --with-ebay --with-pop --with-sales

# Advanced analysis modes
./pkmgradegap --set "Surging Sparks" --analysis alerts --compare-snapshots "old.json,new.json"
./pkmgradegap --set "Surging Sparks" --analysis trends --trends-csv "trends.csv"
./pkmgradegap --set "Surging Sparks" --analysis market-timing
./pkmgradegap --set "Surging Sparks" --analysis bulk-optimize
```

## Architecture Overview

This is a CLI tool that analyzes Pokemon card price gaps between raw (ungraded) and graded conditions to identify profitable grading opportunities. It uses a clean provider-based architecture with comprehensive monitoring and optimization capabilities.

### Core Components

**Provider Pattern**: Interface-based providers for different data sources:
- `cards.PokeTCGIO`: Fetches card and set data from Pokemon TCG API with embedded TCGPlayer/Cardmarket prices
- `prices.PriceCharting`: Fetches graded card prices with specific condition mappings
- `ebay.Client`: Optional eBay Finding API integration for live market validation
- `population.PSAProvider`: PSA population report lookups with scarcity scoring
- `monitoring.AlertEngine`: Price change detection and volatility alerts
- `monitoring.HistoryAnalyzer`: Historical trend analysis with linear regression
- `monitoring.TimingAnalyzer`: Market timing recommendations with seasonal patterns

**Data Flow**:
1. CLI flags determine operation mode and parameters
2. Card provider fetches all cards for a set with pagination (250 cards/page)
3. Price provider looks up graded prices using query pattern: "pokemon {SetName} {CardName} #{Number}"
4. Population provider (optional) adds PSA grading population data for scarcity scoring
5. Analysis module normalizes data into `Row` structs and applies enhanced scoring algorithm
6. Results output as CSV with optional history tracking, snapshots, and monitoring reports

### Key Data Structures

- **`model.Card`**: Central card representation with embedded TCGPlayer/Cardmarket price blocks
- **`analysis.Row`**: Normalized data containing card info, raw price, graded prices, population data, and volatility
- **`population.PopulationData`**: PSA grading population counts with scarcity classifications
- **`monitoring.Alert`**: Price change alerts with severity levels and recommendations
- **`monitoring.TrendAnalysis`**: Historical trend data with regression analysis and momentum indicators
- **`analysis.Grades`**: Struct holding PSA10, Grade9 (PSA/BGS 9), Grade95 (CGC/BGS 9.5), BGS10 prices
- **`analysis.ScoredRow`**: Extended Row with calculated score, break-even price, and scoring factors

### Price Mapping

PriceCharting API response fields map to specific grades:
- `manual-only-price` → PSA 10
- `graded-price` → Grade 9 (PSA/BGS 9)
- `box-only-price` → Grade 9.5 (CGC/BGS 9.5)
- `bgs-10-price` → BGS 10
- `loose-price` → Ungraded/Raw

### Analysis Modes

1. **rank** (default): Deterministic scoring algorithm for finding best grading opportunities
2. **raw-vs-psa10**: Simple dollar difference between raw and PSA 10 prices
3. **psa9-cgc95-bgs95-vs-psa10**: Multiple grades shown as percentages of PSA 10 value
4. **crossgrade**: CGC/BGS 9.5 to PSA 10 crossgrade ROI analysis
5. **alerts**: Compare price snapshots for significant changes with volatility analysis
6. **trends**: Analyze historical performance with linear regression and seasonal patterns
7. **bulk-optimize**: Optimize cards for PSA bulk submission batches with service level recommendations
8. **market-timing**: Seasonal and market timing recommendations with confidence scoring

### Scoring Algorithm (Rank Mode)

The scoring system uses multiple factors:
- Base score: (PSA10 - Raw - Total Costs)
- Premium lift bonus: Additional points for cards with steep PSA10 premiums
- Japanese card multiplier: Configurable weight for Japanese cards (better centering)
- Population scarcity: Bonus for low PSA10 population counts
- Volatility penalty: Reduced score for highly volatile prices

## Advanced Features

### Caching System
- Local JSON cache with TTL management (`internal/cache/`)
- Reduces API calls for frequently accessed data
- Configurable cache path via `--cache` flag

### Snapshot System
- Save complete price data for reproducible analysis
- Load snapshots for offline analysis
- Compare snapshots for price alerts

### Monitoring & Optimization (`internal/monitoring/`)
- **AlertEngine**: Price drop, opportunity, and volatility spike alerts with severity filtering
- **HistoryAnalyzer**: Historical trend analysis with linear regression, moving averages, and momentum
- **BulkOptimizer**: PSA submission batch optimization with service level and timing recommendations
- **TimingAnalyzer**: Market timing recommendations with seasonal patterns and confidence scoring

### Population Data Integration (`internal/population/`)
- **PSAProvider**: Real PSA population API integration (future)
- **MockProvider**: Deterministic mock population data for development
- **Scarcity Scoring**: Population-based bonus points (0-15 points) integrated into ranking algorithm
- **Population Caching**: TTL-based caching to minimize API calls

### eBay Integration (`internal/ebay/`)
- Live market validation with Finding API
- Mock mode for testing without API key
- Configurable max listings per card

## Environment Configuration

Required for full functionality:
```bash
export PRICECHARTING_TOKEN="your_token"    # Required for graded prices
export EBAY_APP_ID="your_app_id"           # Optional for eBay listings
export POKEMONTCGIO_API_KEY="optional_key" # Optional, increases rate limits
export POKEMON_PRICE_TRACKER_API_KEY="key" # Optional for sales data
```

## Important Implementation Details

- **Price Storage**: Prices stored as cents (integers) in PriceCharting lookups to avoid float precision issues
- **Set Matching**: Case-insensitive with exact match preference, falls back to partial match
- **Error Resilience**: Continues processing on individual card lookup failures
- **Fallback Pricing**: Prefers TCGPlayer USD market price, falls back to Cardmarket EUR trend
- **Rate Limiting**: Built-in rate limiter for API calls (`internal/ratelimit/`)
- **Volatility Tracking**: 30-day rolling price volatility calculation (`internal/volatility/`)
- **Population Integration**: Optional PSA population data with graceful degradation when unavailable
- **Progress Indicators**: Real-time progress bars with ETA estimation for all data operations
- **Pagination**: Automatic handling of large sets (250 cards per page)

## Extending the System

To add new data sources:
1. Implement provider interfaces in respective packages
2. Card providers need: `ListSets()` and `CardsBySetID()`
3. Price providers need: `Available()` and `LookupCard()`

To add analysis modes:
1. Create new report function in `internal/analysis/` or `internal/monitoring/`
2. Add case in main.go switch statement
3. Follow existing CSV output patterns for consistency
4. Add progress indicators for long-running operations
5. Include graceful error handling and fallback modes

## New CLI Flags and Features

### Population Data
- `--with-pop`: Include PSA population data in scoring (improves accuracy)
- Automatic fallback to mock data when real population API unavailable
- Scarcity bonus: 15 points (≤10 PSA10s), 10 points (≤50), 5 points (≤100), 2 points (≤500)

### Advanced Analytics
- `--analysis alerts`: Compare snapshots for price change alerts
- `--analysis trends`: Historical trend analysis with regression
- `--analysis market-timing`: Seasonal timing recommendations
- `--analysis bulk-optimize`: PSA batch optimization

### Alert System
- `--alert-threshold-usd`: Dollar change threshold (default $5)
- `--alert-threshold-pct`: Percentage change threshold (default 10%)
- `--alert-csv`: Export alerts to CSV file
- `--compare-snapshots`: Compare two snapshot files

### Trend Analysis
- `--trends-csv`: Export trend analysis to CSV
- `--analyze-trends`: Analyze historical trends from tracking CSV
- Linear regression, moving averages, momentum analysis

### Market Timing
- `--market-timing`: Get timing recommendations
- Seasonal pattern detection
- Confidence scoring for buy/sell/hold recommendations

### Bulk Optimization
- `--optimize-bulk`: Optimize for PSA bulk submission
- Service level recommendations (Regular/Express/Super Express)
- Batch optimization for submission timing