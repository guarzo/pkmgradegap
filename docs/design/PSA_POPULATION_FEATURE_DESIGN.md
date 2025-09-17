# PSA Population Feature Design Document

## Executive Summary

This document outlines the design for implementing PSA Population Report integration into the pkmgradegap CLI tool. The feature will enable scarcity-based scoring by incorporating real PSA grading population data to improve grading opportunity recommendations.

## Background & Business Case

From the development tracker and sprint planning documents, PSA population data has been identified as a critical enhancement for more accurate grading opportunity analysis. Currently, the scoring algorithm relies solely on price differentials, missing the crucial scarcity factor that significantly impacts card values.

### Current State
- Population structure exists in `model.PSAPopulation` but no data source
- Scoring algorithm has placeholder for population but no implementation
- Comments indicate "Population multipliers removed - no public PSA API available"

### Business Value
- **Scarcity-based scoring**: Low population PSA 10s command higher premiums
- **Risk assessment**: High population cards have more predictable outcomes
- **Market timing**: Population growth rates indicate market saturation
- **ROI optimization**: Better prediction of grading success rates

## Requirements Analysis

### Functional Requirements
1. **Population Data Integration**: Support both PSA API and scraping fallback
2. **Scarcity Scoring**: Enhance scoring algorithm with population-based multipliers
3. **Cache Management**: Efficient storage and TTL for population data
4. **User Interface**: Optional population display in analysis output
5. **Rate Limiting**: Respect PSA API limits (100 calls/day free tier)

### Non-Functional Requirements
- **Performance**: Population lookups must not significantly slow analysis
- **Reliability**: Graceful degradation when population data unavailable
- **Scalability**: Support both single-card and bulk set analysis
- **Maintainability**: Clean provider pattern integration

## Technical Architecture

### Provider Pattern Integration

The solution follows the existing provider pattern with a new `PopulationProvider` interface:

```go
type PopulationProvider interface {
    Available() bool
    GetPopulation(setName, cardName, number string) (*model.PSAPopulation, error)
    GetPopulationBySpecID(specID int) (*model.PSAPopulation, error)
}
```

### Implementation Options

#### Option 1: PSA API Provider (Recommended)
**Advantages:**
- Official data source
- Structured JSON response
- Real-time data

**Implementation:**
```go
type PSAAPIProvider struct {
    client   *http.Client
    cache    *cache.Store
    limiter  *ratelimit.Limiter
    token    string
}
```

**Workflow:**
1. Resolve card name/number to PSA SpecID
2. Call `/pop/GetPSASpecPopulation/{specID}`
3. Parse grade breakdown (1-10, qualifiers)
4. Cache with appropriate TTL

#### Option 2: Scraping Provider (Fallback)
**Advantages:**
- No API key required
- Access to complete population data
- Backup when API unavailable

**Implementation:**
- Use existing scraper patterns from ChrisMuir/psa-scrape
- Parse HTML tables for population data
- More resilient to PSA website changes

#### Option 3: CSV Provider (Manual Data)
**Current Implementation:**
- Already exists in `internal/population/psa.go`
- Manual data import from exported CSV
- Useful for offline analysis

### Enhanced Data Model

Extend `model.PSAPopulation` to support full grade breakdown:

```go
type PSAPopulation struct {
    SpecID      int       `json:"specId"`
    Description string    `json:"description"`
    Total       int       `json:"total"`
    Auth        int       `json:"auth"`
    Grade1      int       `json:"grade1"`
    Grade1Q     int       `json:"grade1Q"`
    // ... through Grade10/Grade10Q
    Grade10     int       `json:"grade10"`
    Grade10Q    int       `json:"grade10Q"`
    LastUpdated time.Time `json:"lastUpdated"`
}
```

### Scoring Algorithm Enhancement

Integrate population data into the ranking algorithm:

```go
// Population scarcity multiplier
if r.Population != nil && r.Population.Grade10 > 0 {
    scarcityMultiplier := calculateScarcityMultiplier(r.Population.Grade10)
    score *= scarcityMultiplier

    // Calculate grade success rates
    psa10Rate = float64(r.Population.Grade10) / float64(r.Population.Total)
    psa9Rate = float64(r.Population.Grade9) / float64(r.Population.Total)
}

func calculateScarcityMultiplier(psa10Count int) float64 {
    switch {
    case psa10Count < 10:    return 1.5  // Ultra rare
    case psa10Count < 50:    return 1.3  // Very rare
    case psa10Count < 200:   return 1.1  // Rare
    case psa10Count < 1000:  return 1.0  // Common
    default:                 return 0.9  // Very common
    }
}
```

## Implementation Plan

### Phase 1: PSA API Integration
**Sprint 2 Priority: HIGH (5 points)**

1. **Create PSA API Provider** (`internal/population/psa_api.go`)
   - OAuth2 authentication
   - SpecID resolution logic
   - Population data fetching
   - Error handling and retries

2. **Enhance Cache System** (`internal/cache/`)
   - Population-specific cache keys
   - Configurable TTL (daily updates)
   - Bulk cache warming

3. **Update Analysis Integration** (`internal/analysis/analysis.go`)
   - Population provider injection
   - Scoring algorithm enhancement
   - Optional population display

### Phase 2: Scraping Fallback
**Sprint 3 Priority: MEDIUM (3 points)**

1. **Web Scraper Provider** (`internal/population/psa_scraper.go`)
   - HTML parsing for population tables
   - Rate limiting and retries
   - Graceful error handling

2. **Provider Chain** (`internal/population/provider.go`)
   - Try API first, fallback to scraper
   - Configurable provider selection
   - Health checking

### Phase 3: Advanced Features
**Sprint 4 Priority: LOW (2 points)**

1. **Population Trending** (`internal/population/trends.go`)
   - Historical population tracking
   - Growth rate calculations
   - Market saturation indicators

2. **Population Analysis Reports**
   - Population-focused analysis modes
   - Scarcity reports by set
   - Grade distribution analysis

## Configuration & Usage

### Environment Variables
```bash
export PSA_API_TOKEN="your_psa_bearer_token"     # PSA API access
export PSA_RATE_LIMIT="100"                     # Daily API limit
export PSA_CACHE_TTL="24h"                      # Population cache TTL
```

### CLI Flags
```bash
# Enable population data
./pkmgradegap --set "Surging Sparks" --with-pop

# Population analysis mode
./pkmgradegap --set "Surging Sparks" --analysis population

# Show population in output
./pkmgradegap --set "Surging Sparks" --analysis rank --show-pop

# Force population provider
./pkmgradegap --set "Surging Sparks" --pop-provider api|scraper|csv
```

### Example Output Enhancement
```csv
Card,No,RawUSD,PSA10USD,DeltaUSD,Score,PSA10Pop,PSA10Rate,ScarcityMultiplier,Notes
Charizard ex,223/197,$45.00,$180.00,$135.00,87.3,23,8.5%,1.3x,Ultra rare pop
```

## Risk Assessment & Mitigation

### Technical Risks

1. **PSA API Rate Limits**
   - *Risk*: 100 calls/day insufficient for large sets
   - *Mitigation*: Aggressive caching, batch processing, scraper fallback

2. **API Authentication Changes**
   - *Risk*: PSA modifies OAuth flow
   - *Mitigation*: Scraper fallback, manual CSV import option

3. **Data Quality Issues**
   - *Risk*: Stale or incorrect population data
   - *Mitigation*: Cache TTL, data validation, user feedback mechanism

4. **Performance Impact**
   - *Risk*: Population lookups slow analysis
   - *Mitigation*: Concurrent fetching, cache warming, optional feature

### Operational Risks

1. **PSA Website Changes**
   - *Risk*: Scraper breaks with website updates
   - *Mitigation*: Robust parsing, multiple fallback strategies

2. **Legal/ToS Concerns**
   - *Risk*: PSA prohibits scraping
   - *Mitigation*: Respectful scraping, API-first approach, user-provided data

## Success Metrics

### Phase 1 Success Criteria
- [ ] PSA API provider successfully fetches population data
- [ ] Scoring algorithm integrates population multipliers
- [ ] Cache system handles population data efficiently
- [ ] No performance regression in analysis speed
- [ ] Graceful degradation when population unavailable

### User Experience Metrics
- Population data coverage: >80% of analyzed cards
- Analysis accuracy improvement: Better correlation with market premiums
- Performance impact: <20% increase in analysis time
- Error rate: <5% failed population lookups

## Testing Strategy

### Unit Tests
- PSA API provider response parsing
- Scoring algorithm with population data
- Cache behavior for population data
- Error handling scenarios

### Integration Tests
- End-to-end analysis with population data
- Provider fallback mechanisms
- Rate limiting behavior
- Cache persistence across sessions

### Performance Tests
- Population lookup latency
- Bulk analysis with population data
- Memory usage with large population datasets
- Cache warming efficiency

## Future Enhancements

### Advanced Population Analytics
- Historical population tracking and trends
- Cross-grading company population comparison (BGS, CGC)
- Set-level population analysis and recommendations
- Population-based market timing predictions

### Machine Learning Integration
- Grade prediction models using population patterns
- Anomaly detection for unusual population data
- Dynamic scarcity scoring based on market conditions
- Predictive population growth modeling

### Enterprise Features
- Population data API for third-party integrations
- Real-time population alerts and notifications
- Population-focused investment portfolio tracking
- Advanced visualization and reporting dashboards

## Conclusion

The PSA Population feature represents a significant enhancement to the pkmgradegap tool's analytical capabilities. By integrating official PSA population data, the tool will provide more accurate, scarcity-aware grading recommendations that better reflect real market dynamics.

The phased implementation approach ensures reliable delivery while maintaining the tool's performance and user experience. The provider pattern integration maintains architectural consistency and allows for future enhancements and alternative data sources.

Success of this feature will establish pkmgradegap as the premier tool for data-driven Pokemon card grading analysis, setting the foundation for advanced market intelligence features in future releases.