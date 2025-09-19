# Pokemon Grade Gap Analyzer

Analyze price differences between raw and graded Pokemon cards to identify profitable grading opportunities with multi-source data fusion.

## Features

- **Multi-Source Data Fusion**: Combine prices from PriceCharting, eBay, GameStop, and sales data
- **Intelligent Scoring**: Advanced algorithm with population scarcity and volatility factors
- **GameStop Integration**: Trade-in values and buylist pricing for arbitrage opportunities
- **Cost Analysis**: Account for grading fees, shipping, and marketplace selling costs
- **Web Interface**: Interactive dashboard with Chart.js visualizations and real-time updates
- **Japanese Card Weighting**: Bonus scoring for Japanese cards (better centering)
- **Smart Caching**: Multi-layer cache with predictive loading and TTL management
- **Price Alerts**: Monitor price changes between snapshots with severity levels
- **Market Timing**: Seasonal recommendations based on historical trends
- **Bulk Optimization**: Optimize PSA submission batches by service level
- **Population Data**: PSA population integration for scarcity scoring
- **eBay Integration**: Live eBay listings for market validation
- **eBay Listing Manager**: Manage your own eBay listings with intelligent repricing
- **Sales History**: Actual transaction data from multiple marketplaces
- **Load Testing**: Performance validation framework included

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

# For eBay Listing Manager (optional):
export EBAY_CLIENT_ID="your_client_id"         # eBay OAuth App ID
export EBAY_CLIENT_SECRET="your_secret"        # eBay OAuth Secret
export EBAY_REDIRECT_URI="http://localhost:8080/api/ebay/callback"

# GameStop integration uses web scraping (no API key needed)
export POKEMON_PRICE_TRACKER_API_KEY="key"     # Optional - For sales data

# Find best grading opportunities (default mode)
./pkmgradegap --set "Surging Sparks"

# Or start the web interface
./pkmgradegap server --port 8080 --auto-open
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
# Include GameStop trade-in values (web scraping)
./pkmgradegap --set "Surging Sparks" --with-gamestop
# Note: No API key required, but may fail if website structure changes

# Enable data fusion from multiple sources
./pkmgradegap --set "Surging Sparks" --fusion-mode --with-sales --with-gamestop

# Include population data for scarcity scoring
./pkmgradegap --set "Surging Sparks" --with-pop

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

### Web Interface

```bash
# Start the web server
./pkmgradegap server --port 8080

# Auto-open browser
./pkmgradegap server --auto-open

# The web interface provides:
# - Interactive dashboard with real-time updates
# - Chart.js visualizations for price trends
# - Advanced filtering and search
# - Batch analysis for multiple sets
# - Export to CSV/JSON/PDF
# - Theme selection (auto/light/dark)
```

### Monitoring & Alerts

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
- `--set STRING`: Set name to analyze (or use `server` to start web interface)

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

### Data Sources
- `--with-ebay`: Fetch current eBay listings (requires EBAY_APP_ID)
- `--with-gamestop`: Include GameStop trade-in values
- `--with-pop`: Include PSA population data
- `--with-sales`: Include sales transaction data
- `--with-volatility`: Include 30-day price volatility data
- `--fusion-mode`: Enable multi-source data fusion
- `--ebay-max INT`: Max listings per card (default: 3)

### Data Management
- `--snapshot-out PATH`: Save price data for reproducibility
- `--snapshot-in PATH`: Load price data from snapshot
- `--cache PATH`: Cache file location (default: data/cache.json)
- `--cache-ttl DURATION`: Cache time-to-live (default: 24h)
- `--history PATH`: Append top picks here (default: data/targets.csv)

### Monitoring & Alerts
- `--compare-snapshots PATH1,PATH2`: Compare two snapshots for price alerts
- `--alert-threshold-pct FLOAT`: Alert threshold for percentage change (default: 10.0)
- `--alert-threshold-usd FLOAT`: Alert threshold for dollar change (default: 5.0)
- `--alert-csv PATH`: Export alerts to CSV file

### Server Options
- `--port INT`: Web server port (default: 8080)
- `--auto-open`: Auto-open browser on server start

### Utility
- `--list-sets`: List all available sets and exit
- `--verbose`: Enable verbose logging
- `--debug`: Enable debug mode

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
| eBay Finding | ❌ Optional | Live eBay listings | Developer account required |
| GameStop | ❌ Optional | Trade-in values (web scraping) | No API - uses web scraping |
| Pokemon Price Tracker | ❌ Optional | Sales transaction data | API subscription |
| PSA Population | ❌ Optional | Population reports | Future implementation |

### Integration Options
- **Web Scraping**: GameStop integration uses web scraping (may break if site changes)
- **Data Fusion**: Automatically combines available sources for best accuracy
- **Graceful Degradation**: Tool works with only PriceCharting API

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
├── cmd/pkmgradegap/
│   ├── main.go                  # CLI interface
│   └── server.go                 # Web server implementation
├── internal/
│   ├── analysis/                 # Scoring and reporting logic
│   ├── cache/                    # Multi-layer caching system
│   ├── cards/                    # Pokemon TCG API client
│   ├── prices/                   # PriceCharting API client
│   ├── gamestop/                 # GameStop integration
│   ├── ebay/                     # eBay Finding API
│   ├── population/               # PSA population data
│   ├── sales/                    # Sales transaction data
│   ├── fusion/                   # Multi-source data fusion
│   ├── monitoring/               # Alerts and analysis
│   ├── volatility/               # Price volatility tracking
│   └── model/                    # Data structures
├── scripts/
│   ├── load_test.go              # Performance testing
│   └── run_load_test.sh          # Test runner
├── web/                          # Web interface assets
├── data/                         # Cache and history files
└── docs/                         # Documentation
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

## eBay Listing Manager

The eBay Listing Manager provides a comprehensive web interface for managing your eBay Pokemon card listings with intelligent repricing suggestions.

### Features

- **OAuth Authentication**: Secure connection to your eBay seller account
- **Bulk Listing Management**: View and manage all your active listings
- **Intelligent Repricing**: AI-driven price suggestions based on:
  - Current market prices from TCGPlayer and PriceCharting
  - Competitor analysis from active eBay listings
  - Sales velocity and demand indicators
  - Population scarcity data
  - Days on market (staleness detection)
- **Confidence Scoring**: Each suggestion includes a confidence percentage
- **Batch Operations**: Apply price changes to multiple listings at once
- **Performance Dashboard**: Track views, watchers, and sales metrics
- **Export Functionality**: Download listing data and suggestions as CSV

### Setup

1. **Register an eBay Developer Account**:
   - Go to https://developer.ebay.com
   - Create an application for the Production environment
   - Select "Authorization Code Grant" as the auth type

2. **Configure Environment Variables**:
```bash
# Required for eBay Listing Manager
export EBAY_CLIENT_ID="your_app_id"          # eBay Application ID (OAuth Client ID)
export EBAY_CLIENT_SECRET="your_cert_id"     # eBay Certificate ID (OAuth Secret)
export EBAY_REDIRECT_URI="http://localhost:8080/api/ebay/callback"  # OAuth callback URL
export EBAY_SANDBOX_MODE="false"             # Use "true" for testing with sandbox

# Also needed for Trading API calls
export EBAY_APP_ID="your_app_id"             # Same as EBAY_CLIENT_ID (for Trading API headers)
```

**Important Notes**:
- You need **both** OAuth credentials (for authentication) AND a Trading API App ID
- The `EBAY_APP_ID` should be the same value as `EBAY_CLIENT_ID`
- For production, ensure `EBAY_SANDBOX_MODE="false"`
- The redirect URI must exactly match what's configured in your eBay app

3. **Access the Interface**:
```bash
# Start the server
./pkmgradegap server --port 8080

# Open browser to:
# http://localhost:8080/ebay
```

### Usage Workflow

1. **Connect to eBay**: Click "Connect to eBay" and authorize the application
2. **Sync Listings**: Fetch your active Pokemon card listings
3. **Analyze Prices**: Click "Analyze Prices" to generate repricing suggestions
4. **Review Suggestions**:
   - **DECREASE**: Price is above market, reducing recommended
   - **INCREASE**: Price is below market, can increase
   - **HOLD**: Price is optimal
5. **Apply Changes**: Select listings and apply suggested prices in bulk

### Price Suggestion Algorithm

The repricing engine considers multiple factors:

- **Competition Level**: Number of similar active listings
- **Market Prices**: TCGPlayer and PriceCharting reference prices
- **Days Active**: Penalizes stale listings (>30 days)
- **Engagement Rate**: Views-to-watch ratio indicates demand
- **Recent Sales**: Actual selling prices from completed listings
- **Population Rarity**: PSA population data affects pricing power
- **Market Trends**: Bullish/bearish market detection

### API Endpoints

```bash
# OAuth Flow
GET  /api/ebay/auth              # Initiate OAuth
GET  /api/ebay/callback          # OAuth callback

# Listing Management
GET  /api/ebay/listings          # Get user's listings
PUT  /api/ebay/listings/:id      # Update single listing
POST /api/ebay/analyze           # Generate price suggestions
POST /api/ebay/reprice           # Bulk price updates

# Analytics
GET  /api/ebay/dashboard         # Overview statistics
GET  /api/ebay/competitors/:id  # Competitor analysis
```

## Testing

Run the test suite to verify functionality:

```bash
# Run all tests
go test ./...

# Test with coverage
go test -cover ./...

# Test specific modules
go test ./internal/gamestop/
go test ./internal/fusion/
go test ./internal/monitoring/

# Integration tests
go test ./internal/integration/

# Load testing
./scripts/run_load_test.sh
# Or with custom parameters
go run scripts/load_test.go -users 50 -duration 60s
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