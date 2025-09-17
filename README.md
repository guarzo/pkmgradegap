# Pokemon Grade Gap Analyzer

Analyze price differences between raw and graded Pokemon cards to identify profitable grading opportunities.

## Features

- **Rank Mode**: Find the best grading opportunities with deterministic scoring algorithm
- **Cost Analysis**: Account for grading fees, shipping, and marketplace selling costs
- **Set Age Filtering**: Focus on recent sets with better print quality
- **Japanese Card Weighting**: Bonus scoring for Japanese cards (typically better centering)
- **Smart Caching**: Reduce API calls with local JSON cache and TTL management
- **Reproducible Analysis**: Save/load price snapshots for consistent results
- **History Tracking**: Track top picks over time for trend analysis
- **Price Alerts**: Monitor price changes between snapshots (Phase 3)
- **Market Timing**: Get buy/sell recommendations based on historical trends (Phase 3)
- **Bulk Optimization**: Optimize PSA submission batches by service level (Phase 3)
- **Trend Analysis**: Analyze historical performance of recommendations (Phase 3)
- **eBay Integration**: Live eBay listings for market validation with mock mode for testing (Phase 4)
- **Comprehensive Testing**: Full unit test coverage with mock data providers (Phase 4)

## Quick Start

```bash
# Clone and build
git clone https://github.com/guarzo/pkmgradegap.git
cd pkmgradegap
go mod tidy
go build -o pkmgradegap ./cmd/pkmgradegap

# Set API tokens
export PRICECHARTING_TOKEN="your_token_here"  # Required - Get from pricecharting.com/api
export POKEMONTCGIO_API_KEY="optional_key"     # Optional - Higher rate limits
export EBAY_APP_ID="optional_app_id"           # Optional - For live eBay listings

# Find best grading opportunities (default mode)
./pkmgradegap --set "Surging Sparks"
```

## Usage Examples

### Basic Usage

```bash
# List all available sets
./pkmgradegap --list-sets

# Find best grading opportunities with default settings
./pkmgradegap --set "Surging Sparks"
# Defaults: --analysis rank --max-age-years 10 --min-delta-usd 25

# Customize cost parameters
./pkmgradegap --set "Surging Sparks" \
  --grading-cost 25 \
  --shipping 20 \
  --fee-pct 0.13 \
  --japanese-weight 1.15 \
  --top 25
```

### Advanced Features

```bash
# Save snapshot for reproducible analysis
./pkmgradegap --set "Surging Sparks" \
  --snapshot-out prices_20250916.json \
  --history data/targets.csv

# Load from snapshot (offline analysis)
./pkmgradegap --set "Surging Sparks" \
  --snapshot-in prices_20250916.json

# Show scoring breakdown
./pkmgradegap --set "Surging Sparks" --why

# Include live eBay listings (requires EBAY_APP_ID)
./pkmgradegap --set "Surging Sparks" --with-ebay --ebay-max 3

# Include price volatility analysis
./pkmgradegap --set "Surging Sparks" --with-volatility

# Crossgrade analysis (CGC/BGS 9.5 to PSA 10)
./pkmgradegap --set "Surging Sparks" --analysis crossgrade

# Legacy analysis modes
./pkmgradegap --set "Surging Sparks" --analysis raw-vs-psa10
./pkmgradegap --set "Surging Sparks" --analysis psa9-cgc95-bgs95-vs-psa10
```

### Phase 3: Monitoring & Alerts

```bash
# Compare price snapshots and generate alerts
./pkmgradegap --analysis alerts \
  --compare-snapshots snapshot1.json,snapshot2.json \
  --alert-threshold-pct 15 \
  --alert-threshold-usd 10

# Analyze historical trends from tracking CSV
./pkmgradegap --analysis trends \
  --history data/targets.csv

# Optimize cards for bulk PSA submission
./pkmgradegap --set "Surging Sparks" \
  --analysis bulk-optimize \
  --grading-cost 25 \
  --shipping 20

# Get market timing recommendations
./pkmgradegap --set "Surging Sparks" \
  --analysis market-timing
```

## Command-Line Flags

### Required
- `--set STRING`: Set name to analyze

### Analysis Options
- `--analysis STRING`: Mode: rank|raw-vs-psa10|psa9-cgc95-bgs95-vs-psa10|crossgrade|alerts|trends|bulk-optimize|market-timing (default: rank)
- `--max-age-years INT`: Only sets released within N years (default: 10, 0=disable)
- `--min-delta-usd FLOAT`: Minimum PSA10-Raw gap required (default: 25)
- `--min-raw-usd FLOAT`: Minimum raw card price (default: 5)
- `--top INT`: Show top N results (default: 25)
- `--allow-thin-premium`: Allow cards with PSA9/PSA10 > 0.75

### Cost Parameters
- `--grading-cost FLOAT`: PSA grading fee per card (default: 25)
- `--shipping FLOAT`: Round-trip shipping cost (default: 20)
- `--fee-pct FLOAT`: Selling fee percentage (default: 0.13)

### Scoring Modifiers
- `--japanese-weight FLOAT`: Multiplier for Japanese cards (default: 1.0)
- `--why`: Show scoring factor breakdown

### Data Management
- `--snapshot-out PATH`: Save price data for reproducibility
- `--snapshot-in PATH`: Load price data from snapshot
- `--cache PATH`: Cache file location (default: data/cache.json)
- `--history PATH`: Append top picks here (default: data/targets.csv)
- `--with-ebay`: Fetch current eBay listings (requires EBAY_APP_ID)
- `--ebay-max INT`: Max listings per card (default: 3)
- `--with-volatility`: Include 30-day price volatility data

### Phase 3: Monitoring & Alerts
- `--compare-snapshots PATH1,PATH2`: Compare two snapshots for price alerts
- `--alert-threshold-pct FLOAT`: Alert threshold for percentage change (default: 10.0)
- `--alert-threshold-usd FLOAT`: Alert threshold for dollar change (default: 5.0)

### Utility
- `--list-sets`: List all available sets and exit

## Output Format

### Rank Mode (Default)
```csv
Card,No,RawUSD,PSA10USD,DeltaUSD,CostUSD,BreakEvenUSD,Score,Notes
Pikachu ex,238,$45.00,$125.00,$80.00,$90.00,$103.45,42.5,USD [JPN]
```

**With Optional Columns:**
```csv
Card,No,RawUSD,PSA10USD,DeltaUSD,CostUSD,BreakEvenUSD,Score,Notes,EBayLinks,Volatility30D
Pikachu ex,238,$45.00,$125.00,$80.00,$90.00,$103.45,42.5,USD [JPN],$43.50|NM Card|ebay.com/123,8.5%
```

- **Card**: Card name
- **No**: Card number in set
- **RawUSD**: Current raw/ungraded price
- **PSA10USD**: PSA 10 graded price
- **DeltaUSD**: Simple price difference (PSA10 - Raw)
- **CostUSD**: Total investment (Raw + Grading + Shipping)
- **BreakEvenUSD**: Minimum sale price needed for profit
- **Score**: Opportunity score (higher is better)
- **Notes**: Additional info ([JPN] for Japanese cards)
- **EBayLinks**: Live eBay listings (Price|Title|URL format, optional)
- **Volatility30D**: 30-day price volatility percentage (optional)

### Raw vs PSA 10 Analysis
```csv
Card,Number,RawUSD,RawSource,PSA10_USD,Delta_USD,Notes
Pikachu ex,238,$45.00,tcgplayer.market,$125.00,$80.00,USD
```

### Multi-Grade Comparison
```csv
Card,Number,PSA9_USD,CGC/BGS_9.5_USD,BGS10_USD,PSA10_USD,PSA9/10_%,9.5/10_%,BGS10/PSA10_%
Pikachu ex,238,$65.00,$85.00,$150.00,$125.00,52.0%,68.0%,120.0%
```

### Crossgrade Analysis
```csv
Card,No,CGC95USD,PSA10USD,CrossgradeROI%,Notes
Pikachu ex,238,$85.00,$125.00,15.3%,"Investment: $135.00, Net: $108.75"
```

## How It Works

1. **Fetches card data** from Pokemon TCG API
   - Includes embedded TCGPlayer prices for raw cards
   - Caches results locally to reduce API calls

2. **Looks up graded prices** via PriceCharting API
   - Maps specific fields to grades:
     - PSA 10: `manual-only-price`
     - Grade 9: `graded-price`
     - Grade 9.5: `box-only-price`
     - BGS 10: `bgs-10-price`

3. **Calculates opportunity scores**
   - Base score = (PSA10 - Raw - All Costs)
   - Premium lift bonus for steep PSA10 premiums
   - Optional Japanese card multiplier

4. **Outputs ranked opportunities**
   - Sorted by score descending
   - Filtered by age and minimum thresholds
   - Saves history for trend tracking

## API Requirements

| API | Required | Purpose | Get Key |
|-----|----------|---------|---------|
| PriceCharting | ✅ Yes | Graded card prices | [pricecharting.com/api](https://www.pricecharting.com/api) |
| Pokemon TCG | ❌ Optional | Card data (works without key) | [pokemontcg.io](https://pokemontcg.io) |
| eBay Finding | ❌ Optional | Live eBay listings | [docs/EBAY_SETUP_GUIDE.md](docs/EBAY_SETUP_GUIDE.md) |

### eBay Integration
The tool integrates with eBay Finding API for live market data:
- **Setup**: See [docs/EBAY_SETUP_GUIDE.md](docs/EBAY_SETUP_GUIDE.md)
- **Optional**: Tool works fully without eBay integration
- **Rate Limits**: Free tier provides 5,000 calls/day

## Grading Economics

### Default Cost Structure
- **PSA Grading**: $25 (Regular service, 10-15 business days)
- **Shipping**: $20 (Round trip with insurance)
- **Selling Fees**: 13% (eBay final value + payment processing)

### Break-Even Calculation
```
Total Investment = Raw Price + Grading Cost + Shipping
Break-Even Price = Total Investment / (1 - Selling Fee %)
```

## Project Structure

```
pkmgradegap/
├── cmd/pkmgradegap/main.go      # CLI interface
├── internal/
│   ├── analysis/analysis.go     # Scoring and reporting logic
│   ├── cache/cache.go           # Local caching system
│   ├── cards/poketcgio.go       # Pokemon TCG API client
│   ├── prices/pricechart.go     # PriceCharting API client
│   └── model/types.go           # Data structures
├── data/                         # Cache and history files
├── go.mod
└── README.md
```

## Tips for Finding PSA 10 Candidates

- **Focus on recent sets**: Better print quality and centering standards
- **Look for Japanese cards**: Generally superior print quality
- **Check low population cards**: Potential for price appreciation
- **Consider high-value commons**: Often overlooked by collectors
- **Monitor set age**: Cards from sets <5 years old grade better

## Known Limitations

- **Variant matching**: May not perfectly match promo cards or special editions
- **Population data**: PSA population reports not available via public API
- **Market timing**: Doesn't account for price volatility during grading period
- **Rate limits**: Free tier limited to 20,000 requests/day for Pokemon TCG API

## Testing

Run the test suite to verify functionality:

```bash
# Run all tests
go test ./...

# Test with coverage
go test -cover ./...

# Test specific modules
go test ./internal/ebay/

# Test integration with main CLI
go test ./cmd/pkmgradegap/
```

## Contributing

Pull requests welcome! Priority areas:
- Improved card variant matching
- Web interface for broader accessibility
- Additional grading company support (BGS, CGC, SGC)
- Mobile app development
- Machine learning grade prediction

## License

MIT

## Credits

Built using:
- [Pokemon TCG API](https://pokemontcg.io) for card data
- [PriceCharting](https://www.pricecharting.com) for graded prices
- Go standard library (no external dependencies)