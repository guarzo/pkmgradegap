# Pokemon TCG Grade Gap Analyzer - Comprehensive Functionality Documentation

## Executive Summary

The Pokemon TCG Grade Gap Analyzer is a sophisticated financial analysis tool that identifies profitable grading opportunities in the Pokemon Trading Card Game market. It analyzes price differentials between raw (ungraded) cards and professionally graded cards (primarily PSA 10) to help collectors and investors make data-driven decisions about which cards to submit for grading.

## Core Purpose & Business Value

### Primary Objective
Identify Pokemon cards where the price difference between raw and PSA 10 graded versions exceeds the total costs of grading (submission fees, shipping, marketplace fees), resulting in positive ROI opportunities.

### Key Value Propositions
1. **Risk Mitigation**: Avoid grading cards with thin profit margins or negative ROI
2. **Market Timing**: Track price volatility and market trends for optimal buying/selling
3. **Batch Optimization**: Maximize profitability when submitting bulk grading orders
4. **Cross-Grading Analysis**: Identify CGC/BGS graded cards worth re-grading to PSA
5. **Real-Time Market Data**: Aggregate pricing from multiple sources for accurate valuations

## System Architecture

### Technology Stack
- **Language**: Go 1.21+
- **Architecture Pattern**: Provider-based plugin architecture
- **Concurrency Model**: Worker pools with rate limiting
- **Caching Strategy**: Multi-layer (memory + disk) with predictive prefetching
- **Data Storage**: JSON files for snapshots, CSV for exports
- **Web Framework**: Native Go HTTP server with SSE support

### Core Components

#### 1. Data Providers Layer (`internal/`)

##### Card Data Provider (`cards/`)
- **PokeTCGIO**: Official Pokemon TCG API integration
  - Fetches comprehensive card metadata
  - Embedded TCGPlayer/Cardmarket pricing
  - Pagination support (250 cards/page)
  - Set and card number lookups

##### Price Providers (`prices/`)
- **PriceCharting**: Primary graded price source
  - Real-time PSA 10, PSA 9, BGS 10, CGC 9.5 prices
  - Historical price trends and volatility data
  - Recent sales transaction history
  - Field mapping:
    - `manual-only-price` → PSA 10
    - `graded-price` → PSA/BGS 9
    - `box-only-price` → CGC/BGS 9.5
    - `bgs-10-price` → BGS 10
    - `loose-price` → Raw/Ungraded

- **UPC Database**: Barcode-based lookups (Sprint 4)
  - Fast exact matching via UPC codes
  - Product variant identification
  - Cross-reference with price databases

- **Marketplace Integration**: Enhanced market data (Sprint 3)
  - Active listing counts
  - Lowest available prices
  - Sales velocity metrics
  - Competition analysis

##### GameStop Provider (`gamestop/`)
- Web scraping-based integration (no API required)
- Graded card inventory and pricing
- Trade-in value calculations
- SKU pattern: `pokemon-tcg-{normalized-set}-{card-name}-{number}`
- Features:
  - Real-time stock availability
  - Price comparison with market
  - Trade bonus opportunities
  - Bulk listing retrieval

##### eBay Provider (`ebay/`)
- Finding API for market validation
- Raw card listing searches
- Price distribution analysis
- Features:
  - Completed sales data
  - Active listing analysis
  - Seller reputation filtering
  - Condition-specific searches

##### Population Provider (`population/`)
- **PSA Population API**: Official grading statistics
  - Total graded population by grade
  - PSA 10/9 success rates
  - Scarcity level calculations:
    - ULTRA_RARE: ≤10 PSA 10s
    - RARE: ≤50 PSA 10s
    - UNCOMMON: ≤500 PSA 10s
    - COMMON: >500 PSA 10s
  - Trend analysis based on submission velocity

- **Web Scraper Fallback**: When API unavailable
  - Automated PSA website parsing
  - Cached results for efficiency
  - Grade distribution extraction

##### Sales Provider (`sales/`)
- Historical transaction aggregation
- Market trend analysis
- Price stability metrics
- Volume-weighted average prices

#### 2. Analysis Engine (`internal/analysis/`)

##### Core Data Structures
```go
Row {
    Card: Card metadata
    RawUSD: Ungraded price
    Grades: {PSA10, PSA9, BGS10, CGC9.5}
    Population: PSA population data
    Volatility: 30-day variance (0-1)
    MarketplaceData: Active listings, velocity
    UPC: Universal product code
    MatchConfidence: Search accuracy (0-1)
}

ScoredRow extends Row {
    Score: Calculated profitability score
    NetProfitUSD: Expected profit after costs
    PSA10Rate: Historical PSA 10 rate
    ScoreBreakdown: Detailed scoring factors
}
```

##### Scoring Algorithm
The ranking algorithm considers multiple weighted factors:

1. **Base Score**: `(PSA10 - Raw - TotalCosts)`
   - TotalCosts = GradingFee + Shipping + (PSA10 * SellingFee%)

2. **Premium Lift Bonus**: Extra points for steep PSA10 premiums
   - If PSA10/Raw > 3.0: +20 points
   - If PSA10/Raw > 5.0: +40 points

3. **Japanese Card Multiplier**: Configurable boost (default 1.2x)
   - Detects Hiragana/Katakana/Kanji characters

4. **Population Scarcity Bonus**: 0-15 points based on rarity
   - ULTRA_RARE: +15 points
   - RARE: +10 points
   - UNCOMMON: +5 points

5. **Volatility Penalty**: Reduces score for unstable prices
   - High volatility (>30%): -10 points
   - Extreme volatility (>50%): -20 points

6. **GameStop Trade Bonus**: When trade-in value available
   - Adds potential trade-in profit to score

##### Analysis Modes

1. **rank** (Default): Comprehensive scoring for grading opportunities
2. **raw-vs-psa10**: Simple price differential analysis
3. **crossgrade**: CGC/BGS to PSA crossgrade ROI calculations
4. **alerts**: Snapshot comparison for price change detection
5. **trends**: Historical price trend analysis
6. **bulk-optimize**: PSA bulk submission optimization
7. **market-timing**: Seasonal and cyclical timing recommendations
8. **volatility**: Price stability analysis for risk assessment

#### 3. Monitoring System (`internal/monitoring/`)

##### Alert Engine
Generates actionable alerts based on market events:
- **Price Drop Alerts**: Raw card buying opportunities
- **Price Increase Alerts**: PSA 10 selling opportunities
- **New Opportunity Alerts**: Cards crossing profitability thresholds
- **Volatility Alerts**: Unusual price movements
- **Population Alerts**: Significant grading rate changes

Alert severity levels: HIGH, MEDIUM, LOW

##### Snapshot Management
- Point-in-time market state captures
- Differential analysis between snapshots
- Historical trend tracking
- Format: JSON with compression support

##### Optimization Engine
- Bulk submission composition optimization
- Service level recommendations (economy/standard/express)
- Risk/reward portfolio balancing
- Timing recommendations based on PSA turnaround times

#### 4. Caching System (`internal/cache/`)

##### Multi-Layer Architecture
- **L1 Cache** (Memory):
  - Hot data: 2000 items
  - TTL: 30 minutes
  - LRU eviction policy

- **L2 Cache** (Disk):
  - Persistent storage: 100MB
  - TTL: 24 hours
  - Compression enabled
  - Path: `./data/cache/`

##### Predictive Prefetching
- Analyzes access patterns
- Preloads likely next requests
- Reduces API call latency

##### Query Deduplication
- Prevents duplicate concurrent API calls
- Shares results across waiting requests
- Reduces rate limit pressure

#### 5. Web Interface (`cmd/pkmgradegap/server.go`)

##### API Endpoints

###### Core Analysis
- `GET /api/health` - System health with provider status
- `GET /api/sets` - List available Pokemon TCG sets
- `POST /api/analyze` - Start analysis job (returns job ID)
- `GET /api/jobs/{id}` - Get job status and results
- `GET /api/jobs/{id}/stream` - SSE stream for real-time progress

###### Data Management
- `GET /api/cache/stats` - Cache statistics and hit rates
- `DELETE /api/cache` - Clear all cache layers
- `GET /api/snapshots` - List saved snapshots
- `POST /api/snapshots` - Save current market snapshot

###### eBay Integration
- `GET /api/ebay/search` - Search eBay listings
- `POST /api/ebay/listing` - Create eBay listing
- `PUT /api/ebay/listing/{id}` - Update listing price
- `DELETE /api/ebay/listing/{id}` - End listing
- `POST /api/ebay/reprice` - Bulk repricing

###### Monitoring
- `GET /api/metrics` - Prometheus metrics export
- `GET /api/alerts` - Active alerts
- `POST /api/alerts/config` - Configure alert thresholds

##### Frontend Features
- **Real-time Progress**: SSE-based live updates
- **Interactive Charts**: Chart.js price trends and ROI visualization
- **Advanced Filtering**: Multi-criteria result filtering
- **Preset Management**: Save and load filter configurations
- **Batch Analysis**: Process multiple sets concurrently
- **Export Options**: CSV, JSON, PDF formats
- **Theme Support**: Auto/Light/Dark modes
- **Virtual Scrolling**: Efficient rendering of large datasets

#### 6. Pipeline Processing (`internal/pipeline/`)

##### Concurrent Architecture
- Worker pool pattern for parallel processing
- Configurable concurrency levels (default: 5 workers)
- Rate limiting per provider
- Progress reporting with ETAs

##### Error Resilience
- Continues on individual card failures
- Retry logic with exponential backoff
- Circuit breaker for provider failures
- Detailed error logging and recovery

## Data Flow & Processing Pipeline

### Standard Analysis Flow

1. **Initialization Phase**
   - Parse command-line flags/API parameters
   - Initialize providers with API keys
   - Setup caching layers
   - Configure rate limiters

2. **Card Discovery Phase**
   - Query PokeTCGIO for set cards
   - Paginate through results (250/page)
   - Filter by age and availability
   - Build processing queue

3. **Price Enrichment Phase**
   - **Parallel Processing**: Distribute cards to worker pool
   - **For each card**:
     - Check L1 cache → L2 cache → API
     - Query PriceCharting for graded prices
     - Query TCGPlayer for raw prices
     - Optional: Query GameStop for trade values
     - Optional: Query eBay for market validation
     - Optional: Query PSA for population data
     - Cache results at both layers

4. **Analysis Phase**
   - Normalize prices to USD
   - Calculate profitability metrics
   - Apply scoring algorithm
   - Filter by thresholds
   - Sort by score/criteria

5. **Output Phase**
   - Format as CSV/JSON
   - Generate visualizations
   - Create alerts if configured
   - Save snapshot if requested

### Data Fusion Pipeline (Advanced Mode)

1. **Multi-Source Collection**
   - Query all available providers
   - Collect confidence scores
   - Track response times

2. **Confidence Scoring**
   - Match accuracy (exact/fuzzy/partial)
   - Source reliability weights
   - Recency factors
   - Volume indicators

3. **Price Resolution**
   - Weighted average by confidence
   - Outlier detection and removal
   - Variance calculation
   - Trend adjustment

4. **Result Synthesis**
   - Unified price with confidence interval
   - Source attribution
   - Quality metrics

## Configuration & Environment

### Required Environment Variables
```bash
PRICECHARTING_TOKEN      # Required for graded prices
EBAY_APP_ID              # Optional for eBay listings
POKEMONTCGIO_API_KEY     # Optional, increases rate limits
PSA_POPULATION_API_KEY   # Optional for PSA data
```

### Configuration Parameters

#### Cost Settings
- `--grading-cost`: PSA submission fee (default: $25)
- `--shipping-cost`: Round-trip shipping (default: $20)
- `--fee-pct`: Marketplace selling fee (default: 13%)

#### Filter Settings
- `--min-raw-usd`: Minimum raw card value
- `--min-delta-usd`: Minimum profit threshold
- `--max-age-years`: Maximum set age
- `--top`: Limit results count

#### Feature Toggles
- `--with-ebay`: Include eBay validation
- `--with-pop`: Include population data
- `--with-sales`: Include sales history
- `--with-volatility`: Include volatility analysis
- `--with-marketplace`: Include marketplace metrics

#### Analysis Modifiers
- `--japanese-weight`: Multiplier for Japanese cards
- `--allow-thin-premium`: Allow PSA9/PSA10 > 0.75
- `--show-why`: Display scoring breakdown

## Performance Characteristics

### Scalability Metrics
- **Cards per second**: 10-50 (varies by enabled features)
- **Concurrent workers**: 5 (configurable)
- **Cache hit rate**: 70-90% after warmup
- **Memory usage**: 100-500MB typical
- **Disk cache size**: 100MB maximum

### Optimization Strategies
1. **Batch Processing**: Groups API calls when possible
2. **Request Deduplication**: Prevents redundant API calls
3. **Progressive Loading**: Streams results as available
4. **Selective Features**: Only query needed providers
5. **Smart Caching**: Predictive prefetching

### Rate Limiting
- PriceCharting: 10 requests/second
- PokeTCGIO: 100 requests/minute (higher with API key)
- eBay: 5000 calls/day
- GameStop: 10 requests/minute (conservative scraping)
- PSA: Varies by subscription tier

## Security & Safety

### API Key Management
- Environment variable storage
- Never logged or displayed
- Mock mode for testing without keys

### Web Scraping Ethics
- Respects robots.txt
- Conservative rate limiting
- User-Agent rotation
- Graceful failure handling

### Data Privacy
- No personal data collection
- Local cache storage only
- No external analytics
- Optional snapshot encryption

## Testing & Quality Assurance

### Test Coverage
- Unit tests: >80% coverage target
- Integration tests: Provider interactions
- Mock providers: Isolated testing
- Load tests: Performance validation
- Regression tests: Price calculation accuracy

### Test Modes
```bash
# Run with mock providers
TEST_MODE=true go test ./...

# Integration tests with real APIs
PRICECHARTING_TOKEN="test" go test -v ./internal/integration/

# Load testing
go run scripts/load_test.go -users 50 -duration 60s
```

## Future Enhancements (Roadmap)

### Phase 5: Advanced Analytics
- Machine learning price predictions
- Seasonal trend analysis
- Collector sentiment analysis
- Portfolio optimization

### Phase 6: Mobile Support
- Progressive Web App
- Push notifications for alerts
- Camera-based card scanning
- Offline mode with sync

### Phase 7: Social Features
- Community price consensus
- Shared watchlists
- Trade matching
- Expert recommendations

## Maintenance & Operations

### Monitoring
- Prometheus metrics export
- Health check endpoints
- Provider status dashboard
- Performance profiling hooks

### Troubleshooting
- Detailed error logging
- Request/response tracing
- Cache inspection tools
- Debug mode with verbose output

### Backup & Recovery
- Snapshot versioning
- Cache persistence
- Configuration backup
- Automated recovery procedures

## Conclusion

The Pokemon TCG Grade Gap Analyzer represents a comprehensive solution for data-driven grading decisions in the Pokemon card market. Its modular architecture, sophisticated scoring algorithms, and multi-source data fusion capabilities provide users with actionable insights to maximize their investment returns while minimizing risk.

The system's emphasis on performance, reliability, and extensibility ensures it can adapt to changing market conditions and scale with user needs. Through careful attention to rate limiting, caching strategies, and error handling, it maintains stable operation even under heavy load or when dealing with unreliable external services.