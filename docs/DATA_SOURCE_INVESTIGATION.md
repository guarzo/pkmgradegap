# Data Source Investigation Plan

## Overview
Investigation of additional data sources to enhance price accuracy, population scoring, and market timing capabilities.

## Priority 1: Real Sales Data (130point.com)

### Current Gap
- Currently using eBay listings (asking prices)
- Missing actual sale prices, especially "Best Offer" accepted amounts
- No visibility into bidding patterns

### Investigation Tasks
- [ ] Review 130point.com structure and data availability
- [ ] Check for API documentation or endpoints
- [ ] Analyze HTML structure for scraping feasibility
- [ ] Test sample queries for Pokemon cards
- [ ] Estimate data freshness and update frequency
- [ ] Review Terms of Service for scraping permissions

### Potential Implementation
```go
// internal/sales/oneThirtyPoint.go
type SalesProvider interface {
    GetRecentSales(card string, days int) ([]Sale, error)
    GetAverageSalePrice(card string) (float64, error)
}

type Sale struct {
    Date     time.Time
    Price    float64
    Type     string // "auction", "buy_it_now", "best_offer"
    Graded   bool
    Grade    string
    URL      string
}
```

### Value Proposition
- Replace listing prices with actual sale prices
- Improve ROI calculations with real market data
- Identify pricing trends and volatility
- Better graded vs raw price comparisons

## Priority 2: PSA Population Reports

### Current Gap
- Have population structure but no data source
- Can't score based on scarcity
- Missing supply-side analysis

### Investigation Tasks
- [ ] Analyze PSA pop report URL structure
- [ ] Test queries for specific cards
- [ ] Check for bulk data export options
- [ ] Evaluate scraping complexity
- [ ] Determine update frequency needed
- [ ] Calculate storage requirements

### Potential Implementation
```go
// internal/population/psa.go
type PopulationProvider interface {
    GetPopulation(setName, cardName, number string) (*PopReport, error)
    GetSetPopulations(setName string) (map[string]*PopReport, error)
}

type PopReport struct {
    Total     int
    PSA10     int
    PSA9      int
    LastMonth int // Recent submissions
    Trend     string // "increasing", "stable", "decreasing"
}
```

### Scoring Enhancement
```go
// Add to scoring algorithm
func calculateScarcityBonus(pop *PopReport) float64 {
    if pop.PSA10 < 100 {
        return 20.0 // Ultra rare
    } else if pop.PSA10 < 500 {
        return 10.0 // Rare
    } else if pop.PSA10 < 1000 {
        return 5.0  // Scarce
    }
    return 0.0
}
```

## Priority 3: PriceCharting Historic Data

### Current Gap
- Only getting current prices
- No trend analysis
- Can't identify buying opportunities

### Investigation Tasks
- [ ] Review PriceCharting API for historic endpoints
- [ ] Check if historic data requires higher tier
- [ ] Analyze data granularity (daily/weekly/monthly)
- [ ] Test historic data retrieval
- [ ] Calculate additional API costs

### Potential Implementation
```go
// internal/prices/historic.go
type HistoricProvider interface {
    GetPriceHistory(card string, days int) ([]PricePoint, error)
    GetTrend(card string) (*Trend, error)
}

type PricePoint struct {
    Date  time.Time
    Raw   float64
    PSA10 float64
    PSA9  float64
}

type Trend struct {
    Direction   string  // "up", "down", "stable"
    Momentum    float64 // Rate of change
    Support     float64 // Price floor
    Resistance  float64 // Price ceiling
}
```

## Priority 4: Pokellector Set Metadata

### Current Gap
- Incomplete set information
- Missing card variations
- No promo tracking

### Investigation Tasks
- [ ] Review Pokellector data structure
- [ ] Check for API or data export
- [ ] Evaluate completeness vs Pokemon TCG API
- [ ] Test card variation detection
- [ ] Analyze update frequency

### Potential Implementation
```go
// internal/metadata/pokellector.go
type MetadataProvider interface {
    GetSetInfo(setName string) (*SetMetadata, error)
    GetCardVariations(setName, cardName string) ([]Variation, error)
}

type SetMetadata struct {
    ReleaseDate   time.Time
    PrintRun      string // "unlimited", "limited", etc.
    TotalCards    int
    SecretRares   int
    Availability  string // "in_print", "out_of_print"
}
```

## Integration Architecture

### Proposed Provider Registry
```go
// internal/providers/registry.go
type Registry struct {
    Cards       CardProvider
    Prices      PriceProvider
    Sales       SalesProvider      // NEW
    Population  PopulationProvider // NEW
    Historic    HistoricProvider   // NEW
    Metadata    MetadataProvider   // NEW
}
```

### Enhanced Analysis Flow
```
1. Fetch card data (existing)
2. Get current prices (existing)
3. Get recent sales (NEW - 130point)
4. Get population data (NEW - PSA)
5. Get price history (NEW - PriceCharting)
6. Get card metadata (NEW - Pokellector)
7. Calculate enhanced score with all factors
8. Output comprehensive analysis
```

## Implementation Phases

### Phase 1: Sales Data (Sprint 2)
- Integrate 130point.com
- Replace listing prices with sale prices
- Add sale price columns to CSV

### Phase 2: Population Scoring (Sprint 3)
- Integrate PSA pop reports
- Add scarcity scoring
- Include population columns in output

### Phase 3: Historic Analysis (Sprint 4)
- Add price history from PriceCharting
- Implement trend detection
- Add market timing recommendations

### Phase 4: Metadata Enhancement (Sprint 5)
- Integrate Pokellector
- Improve variation detection
- Add set metadata to analysis

## Success Metrics

### Accuracy Improvements
- Current: Using listing prices (often 10-20% high)
- Target: Actual sale prices (true market value)
- Measure: Compare recommendations before/after

### Scarcity Detection
- Current: No population consideration
- Target: Boost low-pop cards by 10-20 points
- Measure: Track performance of low-pop picks

### Market Timing
- Current: Snapshot in time
- Target: Buy/sell signals based on trends
- Measure: Backtest timing recommendations

## Risk Mitigation

### Technical Risks
- **Scraping blocks**: Implement respectful rate limiting
- **Data inconsistency**: Cross-validate between sources
- **API changes**: Abstract behind interfaces

### Legal Risks
- **ToS violations**: Review each service's terms
- **Data ownership**: Don't store/redistribute raw data
- **Rate limits**: Stay well below limits

### Operational Risks
- **Maintenance burden**: Automate monitoring
- **Cost increases**: Cache aggressively
- **Source unavailability**: Graceful degradation

## Next Steps

1. **Sprint 1 Focus**: Complete current data quality fixes
2. **Sprint 2 Planning**: Prototype 130point integration
3. **User Feedback**: Survey users on priority features
4. **Cost Analysis**: Calculate API/scraping costs
5. **Legal Review**: Confirm ToS compliance

---

*Created: 2025-01-16*
*Status: Investigation Pending*
*Owner: Development Team*