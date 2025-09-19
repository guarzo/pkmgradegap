# Pokemon Grade Gap Analyzer - User Guide

## Getting Started

### 1. Quick Setup (5 minutes)

**Prerequisites:**
- Go 1.19+ installed
- Internet connection for API access

**Installation:**
```bash
# Clone the repository
git clone https://github.com/guarzo/pkmgradegap.git
cd pkmgradegap

# Install dependencies
go mod tidy

# Build the application
go build -o pkmgradegap ./cmd/pkmgradegap
```

**Required API Setup:**
```bash
# Get your PriceCharting API token from https://www.pricecharting.com/api
export PRICECHARTING_TOKEN="your_token_here"

# Optional: Get Pokemon TCG API key from https://pokemontcg.io
export POKEMONTCGIO_API_KEY="your_key_here"

# Optional: eBay App ID for live listings
export EBAY_APP_ID="your_app_id_here"

# GameStop integration uses web scraping (no API key needed)
# Warning: Web scraping may break if GameStop changes their website

# Optional: Pokemon Price Tracker for sales data
export POKEMON_PRICE_TRACKER_API_KEY="your_key_here"

# Optional: PSA Population API (future)
export PSA_POPULATION_API_KEY="your_key_here"
```

**Quick Test:**
```bash
# List available sets
./pkmgradegap --list-sets

# Analyze a recent set
./pkmgradegap --set "Surging Sparks"

# Start web interface
./pkmgradegap server --auto-open
```

### 2. Web Interface (NEW)

The web interface provides a user-friendly way to analyze cards:

**Starting the Server:**
```bash
# Default port 8080
./pkmgradegap --web

# Custom port
./pkmgradegap --web --port 9090
```

**Features:**
- **Interactive Dashboard**: Real-time analysis with progress tracking
- **Chart.js Visualizations**: Price trends, ROI analysis, population charts
- **Advanced Filtering**: Search and filter results across all columns
- **Batch Analysis**: Analyze multiple sets simultaneously
- **Export Options**: CSV, JSON, and PDF export
- **Theme Support**: Auto/Light/Dark modes
- **Live Updates**: Server-sent events for real-time progress

**API Endpoints:**
- `GET /api/health` - System health and provider status
- `GET /api/sets` - List available sets
- `POST /api/analyze` - Start analysis job
- `GET /api/jobs/{id}` - Get job status
- `GET /api/jobs/{id}/stream` - SSE stream for updates
- `DELETE /api/cache` - Clear cache
- `GET /api/metrics` - Prometheus metrics

### 3. Basic Analysis Workflow

**Step 1: Find Current Opportunities**
```bash
# Basic analysis
./pkmgradegap --set "Surging Sparks" --top 10

# Enhanced with all data sources (recommended)
./pkmgradegap --set "Surging Sparks" \
  --with-pop \       # Population data
  --with-sales \     # Sales history
  --with-gamestop \  # Trade-in values
  --fusion-mode \    # Multi-source fusion
  --top 10
```

**Step 2: Save Results for Later**
```bash
./pkmgradegap --set "Surging Sparks" \
  --snapshot-out surging_sparks_$(date +%Y%m%d).json \
  --history data/grading_targets.csv
```

**Step 3: Monitor Price Changes**
```bash
# Take another snapshot later
./pkmgradegap --set "Surging Sparks" \
  --snapshot-out surging_sparks_$(date +%Y%m%d).json

# Compare snapshots
./pkmgradegap --analysis alerts \
  --compare-snapshots surging_sparks_20250916.json,surging_sparks_20250930.json \
  --alert-csv alerts_report.csv
```

## New Features

### GameStop Integration
**What:** Trade-in values and buylist pricing from GameStop
**Why:** Identify arbitrage opportunities between GameStop and graded market
**How:** Enable with `--with-gamestop` flag

```bash
# Use GameStop web scraper
./pkmgradegap --set "Surging Sparks" --with-gamestop

# Note: No API key required, but may fail if website changes
```

**GameStop-Specific Analysis:**
- Trade-in values compared to raw prices
- Buylist opportunities for graded cards
- Arbitrage potential scoring
- Store credit vs cash calculations

### Data Fusion System
**What:** Combines prices from multiple sources with confidence scoring
**Why:** More accurate pricing by reconciling different data sources
**How:** Available in web interface mode only

```bash
# Fusion available in web mode
./pkmgradegap --web
# Then use the web interface to enable fusion analysis
```

**Fusion Features:**
- Weighted averaging based on source reliability
- Confidence intervals for price estimates
- Conflict resolution for discrepancies
- Source type differentiation (SALE vs LISTING vs ESTIMATE)

### Population Data Integration
**What:** PSA grading population statistics
**Why:** Low-population cards are more valuable
**How:** Enable with `--with-pop` flag

```bash
# Include population scarcity in scoring
./pkmgradegap --set "Surging Sparks" --with-pop --top 10
```

**Scarcity Bonuses:**
- 15 points: ≤10 PSA10 population
- 10 points: ≤50 PSA10 population
- 5 points: ≤100 PSA10 population
- 2 points: ≤500 PSA10 population

### Sales Data Integration
**What:** Real sales transaction data from marketplaces
**Why:** Shows actual selling prices, not just listings
**How:** Enable with `--with-sales` flag

```bash
# Use actual sales data for pricing
./pkmgradegap --set "Surging Sparks" --with-sales --top 10
```

### Advanced Caching
**What:** Multi-layer caching with predictive loading
**Why:** Faster performance and reduced API calls
**How:** Automatic with configurable TTL

```bash
# Custom cache location
./pkmgradegap --set "Surging Sparks" \
  --cache data/custom_cache.json
```

## Analysis Modes Explained

### Rank Mode (Default)
**Best for:** Finding profitable grading opportunities

**Scoring Factors:**
- Base profit margin (PSA10 - Raw - Costs)
- Premium lift bonus (steep PSA10 premiums)
- Japanese card multiplier
- Population scarcity bonus
- Volatility penalty
- GameStop trade bonus (if applicable)

```bash
./pkmgradegap --set "Surging Sparks" --analysis rank --why
```

### Market Timing
**Best for:** Seasonal buying/selling decisions

**Output:** Recommendations with confidence scores

```bash
./pkmgradegap --set "Surging Sparks" --analysis market-timing
```

### Bulk Optimization
**Best for:** PSA submission planning

**Features:**
- Service level recommendations (Regular/Express/Super Express)
- Batch optimization for volume discounts
- Submission timing suggestions

```bash
./pkmgradegap --set "Surging Sparks" --analysis bulk-optimize
```

### Price Alerts
**Best for:** Monitoring market changes

**Alert Types:**
- Price drops (buying opportunities)
- Price spikes (selling signals)
- Volatility alerts (risk warnings)
- New opportunities (newly profitable cards)

```bash
./pkmgradegap --analysis alerts \
  --compare-snapshots old.json,new.json \
  --alert-threshold-pct 15 \
  --alert-csv alerts.csv
```

## Real-World Examples

### GameStop Arbitrage Strategy

**Scenario:** Find cards to buy from GameStop and grade for profit

```bash
# Identify GameStop arbitrage opportunities
./pkmgradegap --set "Surging Sparks" \
  --with-gamestop \
  --analysis rank \
  --min-delta-usd 30 \
  --why
```

**Look for:**
- Cards where GameStop trade-in < raw market price
- High PSA10 premiums over GameStop buylist
- Store credit opportunities (usually 20% bonus)

### Multi-Source Validation

**Scenario:** Validate opportunities with all available data

```bash
# Maximum accuracy mode (CLI)
./pkmgradegap --set "Surging Sparks" \
  --with-pop \        # Population data
  --with-sales \      # Sales history
  --with-gamestop \   # Trade values
  --with-ebay \       # Current listings
  --with-volatility \ # Price stability
  --why \            # Show scoring breakdown
  --top 20

# For data fusion, use web interface:
./pkmgradegap --web
```

### Web Interface Workflow

**Scenario:** Using the web interface for analysis

1. **Start Server:**
```bash
./pkmgradegap --web
```

2. **In Browser:**
- Select sets to analyze
- Configure analysis parameters
- Enable data sources (GameStop, eBay, etc.)
- Start analysis

3. **Review Results:**
- Sort by different columns
- Use filters to find specific cards
- View charts for visual analysis
- Export results in preferred format

4. **Batch Analysis:**
- Select multiple sets
- Choose combined or separate view
- Export all results at once

## Performance Optimization

### Load Testing

Test server performance before deployment:

```bash
# Note: Load testing scripts not yet implemented
# Monitor performance through web interface metrics at /api/metrics
curl http://localhost:8080/api/metrics
```

**Performance Tips:**
- Use caching for frequently accessed data
- Enable compression for API responses
- Implement virtual scrolling for large datasets
- Use SSE batching for real-time updates

### Caching Strategy

```bash
# Pre-warm cache for common sets
for set in "Surging Sparks" "Stellar Crown" "Twilight Masquerade"; do
  ./pkmgradegap --set "$set" --snapshot-out cache/"$set".json
done

# Use cached data for faster analysis
./pkmgradegap --snapshot-in cache/"Surging Sparks".json
```

## Troubleshooting

### GameStop Integration Issues

**No GameStop data returned:**
```bash
# GameStop uses web scraping - check if their website structure changed
./pkmgradegap --set "Surging Sparks" --with-gamestop --verbose

# Common causes:
# - GameStop changed their website HTML structure
# - Rate limiting/IP blocking
# - Website temporarily down
```

### Web Interface Issues

**Server won't start:**
```bash
# Check port availability
lsof -i :8080

# Try different port
./pkmgradegap server --port 9090
```

**Slow performance:**
```bash
# Enable metrics endpoint
./pkmgradegap server --port 8080

# Check metrics
curl http://localhost:8080/api/metrics
```

### Data Fusion Conflicts

**Price discrepancies:**
```bash
# View fusion details
./pkmgradegap --set "Surging Sparks" \
  --fusion-mode \
  --verbose \
  --debug
```

## Pro Tips

### Finding the Best Opportunities

1. **GameStop Arbitrage:** Look for undervalued cards at GameStop
2. **Population Scarcity:** Focus on low-pop cards with --with-pop
3. **Data Fusion:** Use --fusion-mode for most accurate pricing
4. **Volatility Check:** Avoid highly volatile cards
5. **Japanese Focus:** Higher centering quality = better grades

### Risk Management

1. **Start Small:** Test with 1-3 cards first
2. **Diversify Sources:** Don't rely on single data provider
3. **Monitor Volatility:** Use --with-volatility flag
4. **Track History:** Build long-term data with --history
5. **Set Alerts:** Regular monitoring with price alerts

### Advanced Strategies

1. **GameStop Flips:** Buy underpriced cards, grade, and sell
2. **Bulk Submissions:** Optimize for PSA volume discounts
3. **Seasonal Timing:** Use market-timing analysis
4. **Cross-Source Validation:** Fusion mode for confidence
5. **Web Dashboard:** Monitor multiple sets in real-time

## API Configuration Details

### Required: PriceCharting
- Get token: https://www.pricecharting.com/api
- Rate limit: 20,000 requests/day
- Coverage: All graded prices

### GameStop (Web Scraping)
- No API available - uses web scraping
- May break if website structure changes
- Provides trade-in and buylist data
- Rate limited to avoid being blocked

### Optional: Pokemon Price Tracker
- Subscription required for API access
- Provides actual sales data
- Better accuracy than listings

### Optional: eBay
- Free developer account needed
- 5,000 calls/day on free tier
- Live market validation

## Getting Help

- **Documentation:** Check docs/ directory
- **Web Interface:** Access at http://localhost:8080 (start with `./pkmgradegap --web`)
- **API Status:** Check /api/health endpoint
- **Metrics:** Monitor at /api/metrics

Remember: This tool provides analysis based on available data. Market conditions, grading outcomes, and timing can vary. Always do additional research before making investments.