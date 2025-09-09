# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Build and Run
```bash
# Build the CLI tool
go build -o pokeprice ./cmd/pokeprice

# Run directly with go run
go run ./cmd/pokeprice --help
go run ./cmd/pokeprice --list-sets
go run ./cmd/pokeprice --set "Surging Sparks" --analysis raw-vs-psa10

# Build and run executable
./pokeprice --set "Surging Sparks" --analysis psa9-cgc95-bgs95-vs-psa10 > report.csv
```

### Development
```bash
# Initialize/update dependencies
go mod tidy

# Test build without creating executable
go build ./cmd/pokeprice

# Run with environment variables
PRICECHARTING_TOKEN="token" POKEMONTCGIO_API_KEY="key" go run ./cmd/pokeprice --list-sets
```

## Architecture Overview

This is a CLI tool that analyzes Pokemon card price gaps between raw (ungraded) and graded conditions using a clean provider-based architecture:

### Core Components

**Provider Pattern**: The system uses interface-based providers for different data sources:
- `cards.PokeTCGIO`: Fetches card and set data from Pokemon TCG API, includes embedded TCGPlayer/Cardmarket prices
- `prices.PriceCharting`: Fetches graded card prices with specific condition mappings

**Data Flow**:
1. CLI flags determine operation (list sets vs analyze set)
2. Card provider fetches all cards for a set with pagination
3. Price provider looks up graded prices by constructing queries like "pokemon {SetName} {CardName} #{Number}"
4. Analysis module normalizes data into `Row` structs and generates CSV output

### Key Data Structures

**`model.Card`**: Central card representation with optional TCGPlayer/Cardmarket price blocks
**`analysis.Row`**: Normalized row for analysis containing card data, raw price, and all grade prices
**`analysis.Grades`**: Struct holding PSA10, Grade9, Grade95, BGS10 prices

### Price Mapping

PriceCharting API response fields map to specific grades:
- `manual-only-price` → PSA 10
- `graded-price` → Grade 9 (PSA/BGS 9)  
- `box-only-price` → Grade 9.5
- `bgs-10-price` → BGS 10
- `loose-price` → Ungraded

### Analysis Modes

1. **raw-vs-psa10**: Shows dollar difference between raw and PSA 10 prices
2. **psa9-cgc95-bgs95-vs-psa10**: Shows multiple grades as percentages of PSA 10 value

## Environment Configuration

Required for full functionality:
```bash
export PRICECHARTING_TOKEN="your_token"    # Required for graded prices
export POKEMONTCGIO_API_KEY="optional_key" # Optional, increases rate limits
```

## Important Implementation Details

- **Price Normalization**: Prices stored as cents (integers) in PriceCharting lookups to avoid float precision issues
- **Set Matching**: Case-insensitive set name matching with exact match preference
- **Pagination**: Automatic pagination for large sets (250 cards per page)
- **Fallback Pricing**: Prefers TCGPlayer USD market price, falls back to Cardmarket EUR trend price
- **Error Handling**: Continues processing on individual card price lookup failures

## Extending the System

To add new data sources, implement the implicit provider interfaces:
- Card providers need: `ListSets() ([]model.Set, error)` and `CardsBySetID(setID string) ([]model.Card, error)`
- Price providers need: `Available() bool` and `LookupCard(setName string, c model.Card) (*PCMatch, error)`

The analysis module accepts normalized `[]analysis.Row` and can be extended with new report types.