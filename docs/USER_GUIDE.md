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

# Optional: Set up eBay App ID (see docs/EBAY_SETUP_GUIDE.md)
export EBAY_APP_ID="your_app_id_here"

# NEW: Pokemon Price Tracker API for sales data
export POKEMON_PRICE_TRACKER_API_KEY="your_key_here"
```

**Quick Test:**
```bash
# List available sets
./pkmgradegap --list-sets

# Analyze a recent set
./pkmgradegap --set "Surging Sparks"
```

### 2. Basic Analysis Workflow

**Step 1: Find Current Opportunities**
```bash
# Basic analysis
./pkmgradegap --set "Surging Sparks" --top 10

# Enhanced with all data sources (recommended)
./pkmgradegap --set "Surging Sparks" --with-pop --with-sales --top 10
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
  --compare-snapshots surging_sparks_20250916.json,surging_sparks_20250930.json
```

### 3. Advanced Features

**Cost Optimization:**
```bash
./pkmgradegap --set "Surging Sparks" \
  --grading-cost 20 \        # Lower cost tier
  --shipping 15 \            # Bulk shipping discount
  --fee-pct 0.10 \          # Lower platform fees
  --japanese-weight 1.2      # Bonus for Japanese cards
```

**Market Validation:**
```bash
# Include live eBay listings for price verification
./pkmgradegap --set "Surging Sparks" \
  --with-ebay \
  --ebay-max 5 \
  --top 15
```

**Volatility Analysis:**
```bash
# Track price volatility for timing decisions
./pkmgradegap --set "Surging Sparks" \
  --with-volatility \
  --why                     # Show detailed scoring breakdown
```

## New Features (Sprint 3A)

### Sales Data Integration
**What:** Real sales transaction data from eBay and other marketplaces
**Why:** More accurate than listing prices; shows what buyers actually pay
**How:** Enable with `--with-sales` flag

```bash
# Use actual sales data for pricing
./pkmgradegap --set "Surging Sparks" --with-sales --top 10
```

### Population Data Integration
**What:** PSA grading population statistics
**Why:** Low-population cards are more valuable; affects investment decisions
**How:** Enable with `--with-pop` flag

```bash
# Include population scarcity in scoring
./pkmgradegap --set "Surging Sparks" --with-pop --top 10
```

### Combined Data Sources
**What:** Use all available data for maximum accuracy
**Why:** Better investment decisions with complete information
**How:** Combine all flags

```bash
# Maximum accuracy mode
./pkmgradegap --set "Surging Sparks" \
  --with-pop \       # Population data
  --with-sales \     # Sales data
  --with-ebay \      # Current listings
  --with-volatility \ # Price stability
  --top 20
```

## Analysis Modes Explained

### Rank Mode (Default)
**Best for:** Finding profitable grading opportunities

**Output:** Scored list of cards ranked by profitability

**Key Metrics:**
- **Score:** Combined profitability metric (higher = better)
- **BreakEvenUSD:** Minimum sale price needed for profit
- **CostUSD:** Total investment including all fees

**When to use:** Daily analysis for new grading opportunities

**Example:**
```bash
./pkmgradegap --set "Surging Sparks" --analysis rank
```

### Raw vs PSA 10 Analysis
**Best for:** Simple price gap identification

**Output:** Basic price differences without cost analysis

**When to use:** Quick market research or data collection

**Example:**
```bash
./pkmgradegap --set "Surging Sparks" --analysis raw-vs-psa10
```

### Crossgrade Analysis
**Best for:** Upgrading existing graded cards

**Output:** ROI for sending CGC/BGS 9.5 cards to PSA for PSA 10

**When to use:** You have graded cards to potentially upgrade

**Example:**
```bash
./pkmgradegap --set "Surging Sparks" --analysis crossgrade
```

### Price Alerts
**Best for:** Monitoring market changes

**Output:** Alert report showing significant price movements

**When to use:** Regular monitoring of your tracked cards

**Example:**
```bash
./pkmgradegap --analysis alerts \
  --compare-snapshots old.json,new.json \
  --alert-threshold-pct 15
```

### Bulk Optimization
**Best for:** PSA submission planning

**Output:** Optimized batches for bulk submission

**When to use:** Planning large grading submissions

**Example:**
```bash
./pkmgradegap --set "Surging Sparks" --analysis bulk-optimize
```

## Real-World Examples

### Finding Your First Grading Opportunity

**Scenario:** New to grading, want to start with low-risk opportunities

**Strategy:**
```bash
# Look for recent sets with good print quality
./pkmgradegap --set "Surging Sparks" \
  --max-age-years 2 \
  --min-raw-usd 20 \        # Higher starting price = lower risk
  --min-delta-usd 40 \      # Good profit margin
  --top 5                   # Focus on best opportunities
```

**Analysis:**
1. Focus on cards with **Score > 30** for strong opportunities
2. Check **BreakEvenUSD** - ensure it's reasonable for the card
3. Verify with **--with-ebay** to see actual market listings
4. Start with 1-2 cards to test the process

### Setting Up Price Monitoring

**Scenario:** Track opportunities over time for optimal timing

**Monthly Routine:**
```bash
# Take monthly snapshots
./pkmgradegap --set "Surging Sparks" \
  --snapshot-out snapshots/surging_sparks_$(date +%Y%m).json \
  --history data/tracking.csv

# Compare with previous month
./pkmgradegap --analysis alerts \
  --compare-snapshots snapshots/surging_sparks_202409.json,snapshots/surging_sparks_202410.json \
  --alert-threshold-pct 10 \
  --alert-threshold-usd 5
```

**Interpretation:**
- **Price Drops:** Good buying opportunities
- **Price Spikes:** Consider selling graded inventory
- **New Opportunities:** Cards that weren't profitable before

### Optimizing Bulk Submissions

**Scenario:** Submit 20+ cards to PSA for volume discounts

**Planning Process:**
```bash
# Gather candidates from multiple sets
./pkmgradegap --set "Surging Sparks" --snapshot-out candidates_ss.json
./pkmgradegap --set "Stellar Crown" --snapshot-out candidates_sc.json

# Combine and optimize
./pkmgradegap --analysis bulk-optimize \
  --grading-cost 18 \       # Bulk pricing
  --shipping 25             # Single shipment cost
```

**Optimization Tips:**
1. Mix high-value and medium-value cards
2. Include some Japanese cards for quality bonus
3. Avoid cards with thin PSA 9/10 premiums
4. Consider market timing recommendations

## Troubleshooting

### Common Issues

**"set not found" Error:**
```bash
# List available sets first
./pkmgradegap --list-sets | grep -i "partial_name"

# Use exact set name
./pkmgradegap --set "Sword & Shield"
```

**No Results Returned:**
```bash
# Lower the minimum thresholds
./pkmgradegap --set "Surging Sparks" \
  --min-delta-usd 10 \
  --min-raw-usd 1 \
  --max-age-years 20
```

**eBay Integration Not Working:**
```bash
# Test with mock mode first
EBAY_APP_ID="mock" ./pkmgradegap --set "Surging Sparks" --with-ebay

# Check your App ID configuration
echo $EBAY_APP_ID
```

**Cache Issues:**
```bash
# Clear cache if data seems stale
rm data/cache.json

# Specify different cache location
./pkmgradegap --set "Surging Sparks" --cache /tmp/temp_cache.json
```

### API Configuration

**PriceCharting API Issues:**
1. Verify token at https://www.pricecharting.com/api
2. Check rate limits (20,000 requests/day)
3. Ensure token is properly exported

**Pokemon TCG API Issues:**
1. Works without key (lower rate limits)
2. Key improves performance but isn't required
3. Check https://pokemontcg.io for status

**eBay API Issues:**
1. Use mock mode for testing: `EBAY_APP_ID="mock"`
2. See docs/EBAY_SETUP_GUIDE.md for full setup
3. Free tier provides 5,000 calls/day

### Performance Tips

**Improve Speed:**
```bash
# Use cached data when possible
./pkmgradegap --set "Surging Sparks" --cache data/cache.json

# Load from snapshot for offline analysis
./pkmgradegap --snapshot-in snapshot.json
```

**Reduce API Usage:**
```bash
# Analyze fewer cards
./pkmgradegap --set "Surging Sparks" --top 10

# Use longer cache TTL
# (Cache is automatically managed with reasonable TTLs)
```

**Batch Operations:**
```bash
# Process multiple sets efficiently
for set in "Surging Sparks" "Stellar Crown" "Twilight Masquerade"; do
  ./pkmgradegap --set "$set" --snapshot-out "$(echo $set | tr ' ' '_')_snapshot.json"
done
```

## Pro Tips

### Finding the Best Opportunities

1. **Recent Sets:** Cards from sets released in the last 2 years typically grade better
2. **Japanese Cards:** Often have superior print quality and centering
3. **Low Population:** Check PSA pop reports manually for scarce cards
4. **Market Timing:** Buy during off-season (spring/summer), sell during holidays
5. **Condition Matters:** Only grade Near Mint or better cards

### Managing Risk

1. **Start Small:** Test with 1-3 cards before bulk submissions
2. **Diversify:** Mix different price points and card types
3. **Track History:** Use --history flag to build long-term data
4. **Set Budgets:** Use --min-raw-usd and --top to control spending
5. **Monitor Markets:** Regular price alerts help time buying/selling

### Advanced Strategies

1. **Crossgrading:** Upgrade CGC/BGS 9.5 to PSA 10 for premium
2. **Japanese Focus:** Use --japanese-weight 1.2+ for quality bonus
3. **Volatility Trading:** Use --with-volatility to time market entries
4. **Bulk Optimization:** Plan submissions around PSA pricing tiers
5. **Set Rotation:** Focus on sets 6-18 months old for optimal timing

## Getting Help

- **Documentation:** Check docs/ directory for detailed guides
- **Issues:** Report bugs at the project repository
- **API Status:** Monitor provider status pages for outages
- **Community:** Share findings and strategies with other users

Remember: This tool provides analysis, but market conditions, grading outcomes, and timing can vary. Always do additional research before making significant investments.