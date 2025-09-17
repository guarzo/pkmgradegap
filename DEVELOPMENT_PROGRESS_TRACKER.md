# Development Progress Tracker

## Current Sprint
**Sprint 3A: Data Integration Completion**
- Status: Planning
- Start: 2025-03-01
- End: 2025-03-14
- Progress: 0/28 points
- Focus: Complete sales and population data integration with pragmatic approach

## Previous Sprint Status
**Sprint 3: Real Sales Data & Complete Workflows**
- Status: ~70% Complete
- Validation Date: 2025-09-17
- Completion: 22/32 points
- Key Pivot: API integration over web scraping

## Project Overview

### Architecture Strengths ✅
- **Clean Provider Pattern**: Well-defined boundaries in `internal/` packages
- **Interface-based design**: Providers communicate through implicit interfaces
- **Separation of concerns**: Each package has single responsibility
- **Good test coverage**: 73% overall with table-driven tests
- **Benchmark tests**: Performance testing for analysis algorithms
- **Clear data flow**: CLI → Cards → Prices → Analysis → CSV

### Completed Features ✅

#### Sprint 1 (Complete)
- ✅ Data Sanitization: Outlier detection and filtering
- ✅ CSV Security: Formula injection protection
- ✅ Provider Testing: Core API client test coverage (>90%)
- ✅ Integration Tests: End-to-end flow validation
- ✅ Main.go Refactoring: Broken down from 575 lines

#### Sprint 2 (85% Complete)
- ✅ Provider Test Coverage: PokeTCGIO and PriceCharting comprehensive tests
- ✅ Progress Indicators: ETA calculations for all data operations
- ✅ PSA Population Integration: Mock provider with scarcity scoring
- ⚠️ Price Alerts System: Infrastructure complete, needs snapshot workflow
- ⚠️ Trend Analysis Module: Basic structure, needs statistical features
- ⚠️ Market Timing Engine: Core engine built, needs historical data

#### Core Features
- ✅ Rank Mode: Deterministic scoring algorithm for grading opportunities
- ✅ Cost Analysis: Grading fees, shipping, marketplace costs calculation
- ✅ Set Age Filtering: Focus on recent sets with better print quality
- ✅ Japanese Card Weighting: Configurable multiplier for Japanese cards
- ✅ Smart Caching: Local JSON cache with TTL management
- ✅ Reproducible Analysis: Save/load price snapshots
- ✅ History Tracking: CSV tracking for trend analysis
- ✅ eBay Integration (Basic): Live listings with mock mode
- ✅ Volatility Tracking: 30-day price volatility calculation

### Sprint 3A Objectives 🚧
- 🚧 Complete Sales Integration: PokemonPriceTracker + PriceCharting enhancement
- 🚧 PSA Population Access: CSV imports + targeted data fetching
- 🚧 Integration Testing: E2E tests for all workflows
- 🚧 Documentation: User guides and examples
- 🚧 Performance Optimization: Concurrent fetching and caching

### Planned Features 📋

#### Sprint 3 (70% Complete)
- ✅ Snapshot Workflow: Automated save/compare functionality
- ✅ Historical Analysis: Moving averages and seasonal patterns
- ✅ Alert Integration: Threshold-based alerts from snapshots
- ⚠️ Sales Data: PokemonPriceTracker API (partial)
- ⚠️ PSA Population: Interface only, no data source

#### Sprint 3A (In Planning)
- 📋 Complete Sales Data: Finish API integration with fallbacks
- 📋 PSA Data Access: CSV imports + targeted fetching
- 📋 Integration Testing: Comprehensive E2E test suite
- 📋 Documentation: User guides for all features
- 📋 Performance: Optimize data fetching and caching

#### Future Sprints
- 📋 Web Interface: Simple localhost dashboard (Sprint 4+)
- 📋 Enhanced eBay: Better raw card filtering
- 📋 Additional Grading Companies: BGS, CGC, SGC support
- 📋 ML Grade Prediction: Machine learning for grade estimation
- 📋 PriceCharting Enhanced: Historical price trends

#### Removed from Scope (Not Needed)
- ❌ Multi-user support (single user application)
- ❌ Authentication system (local only)
- ❌ Scheduled analysis (on-demand only)
- ❌ Reporting dashboards (CSV export sufficient)
- ❌ Bulk optimization features (over-engineered)

## Technical Debt 🔧

### Resolved Issues ✅
- ✅ **Data Quality**: Outlier values sanitized with rarity-based caps
- ✅ **CSV Security**: Formula injection protection implemented
- ✅ **Cache Locking**: Deadlock prevention with proper locking
- ✅ **Main Function**: Refactored into manageable functions

### Remaining Gaps (Sprint 3 Focus)
- **Snapshot Workflow**: Automated snapshot generation for alerts
- **Historical Data**: Integration for trend/timing analysis
- **Statistical Analysis**: Regression and moving averages
- **Real Sales Data**: No actual sale price source yet
- **PSA Population**: Only mock data, need real API/scraping

### Test Coverage Status ✅
- ✅ `cards/poketcgio.go`: >90% coverage achieved
- ✅ `prices/pricechart.go`: Comprehensive test coverage
- ✅ Integration tests for complete flow implemented
- ✅ Error handling tests for API failures
- ✅ Golden tests for CSV header stability
- ✅ Tests for vendor selection logic

### Sprint 3 Testing Priorities
- Web scraping reliability tests (130point, PSA)
- Snapshot comparison workflows
- Historical data analysis validation
- Performance tests with real sales data

### Code Quality
- No structured logging framework (uses basic `log`)
- No graceful shutdown for long operations
- No context propagation from CLI to providers
- No shared HTTP clients per host
- Binary size at 21MB needs optimization (`-ldflags "-s -w"`)
- Inconsistent error wrapping (need `fmt.Errorf("context: %w", err)`)
- No operation timing metrics

### Implementation Gaps
- **Quantile-based outlier detection**: Using fixed caps instead
- **Release date parsing**: Needs tolerant parsing for multiple formats
- **Progress indicators**: No user feedback during long operations
- **HTTP client reuse**: No `internal/httpx` package for shared clients
- **Price selection logic**: Needs formalized priority (TCG market > mid > Cardmarket trend > avg30)
- **eBay filters**: Weak graded card detection (need psa|bgs|cgc|slab filter)
- **Snapshot metadata**: Missing run parameters in snapshots

## Quick Wins (1-2 hour fixes) 🎯

### Immediate CLI Improvements
- Add `--why` flag for scoring rationale per row
- Set better defaults: `--min-raw-usd 0.50`, `--min-delta-usd 25`
- Exit early with message if set age > `--max-age-years`
- Add progress bar for long operations

### Data Sanity Implementation
- Create `internal/analysis/sanitize.go` with ChooseRawUSD function
- Implement vendor priority: TCGPlayer market > mid > Cardmarket trend > avg30
- Add rarity-based price caps
- Filter penny cards and outliers
- Return median with source label

### Documentation Updates
- Add copy-paste example to README
- Create column descriptions table
- Document environment variable setup
- Add troubleshooting section

## Sprint History

### Sprint 1: Data Quality & Core Stability ✅
- **Duration**: Jan 16-30, 2025
- **Goal**: Fix critical data issues and establish robust testing
- **Result**: COMPLETED (34/34 points)
- **Key Achievements**:
  - Fixed outlier data sanitization
  - Implemented CSV security
  - Resolved cache locking issues
  - Added comprehensive provider tests
  - Refactored main.go

### Sprint 2: Advanced Analytics & Market Intelligence (~85%)
- **Duration**: Jan 31 - Feb 13, 2025
- **Goal**: Deliver monitoring and analytics features
- **Result**: PARTIALLY COMPLETE (27/32 points)
- **Key Achievements**:
  - Provider test coverage >90%
  - Progress indicators with ETA
  - PSA population mock integration
  - Alert/trend/timing infrastructure built
- **Gaps**: Snapshot workflows, historical data integration

### Sprint 3: Real Sales & Complete Workflows (~70%)
- **Duration**: Feb 14-28, 2025
- **Goal**: Complete monitoring features and add real data sources
- **Result**: PARTIALLY COMPLETE (22/32 points)
- **Key Achievements**:
  - Monitoring workflows completed
  - Provider architecture established
  - UX improvements delivered
- **Key Pivot**: API integration chosen over web scraping

### Sprint 3A: Data Integration Completion (Planning)
- **Duration**: Mar 1-14, 2025
- **Goal**: Complete data integration with pragmatic approach
- **Stories** (28 points total):
  - Complete sales data integration (8 pts)
  - PSA population data access (8 pts)
  - Integration testing suite (5 pts)
  - User documentation (3 pts)
  - Performance optimization (4 pts)

## Metrics & Quality Gates

### Test Coverage
- Current: ~85% overall (Sprint 2 achievement)
- Target: Maintain >80%
- Strong Coverage:
  - Core providers (>90% after Sprint 2)
  - Analysis, cache, ebay (mocks)
  - Monitoring, ratelimit, volatility
  - Progress package (78.9%)
- Has Benchmarks: analysis algorithms (`analysis_bench_test.go`)

### Code Quality
- Static Analysis: ✅ Clean
- Compiler Warnings: ✅ 0
- Runtime Warnings: ✅ 0
- Placeholder Code: ⚠️ 31 potential stubs detected

### Performance
- API Response Time: <2s average
- Cache Hit Rate: Not tracked
- Binary Size: 21MB (needs optimization)

## Scoring Guardrails & Business Logic 📊

### Default Thresholds (Reduce Noise)
- Skip if PSA10 <= Raw (inverted market)
- Skip if PSA9/PSA10 >= 0.8 (thin premium) unless `--allow-thin-premium`
- Default `--min-raw-usd 0.50` to avoid penny cards
- Include grading cost ($25) + shipping ($20) in ROI calculations
- Marketplace fees: 13% (eBay final value + payment processing)

### Price Selection Priority
1. TCGPlayer.market (if present and not outlier)
2. TCGPlayer.mid (fallback if market missing)
3. Cardmarket.trendPrice (EUR conversion)
4. Cardmarket.avg30 (if trend missing)
5. Take median of valid candidates after sanitation

### eBay Listing Filters
- Drop titles with: `psa|bgs|cgc|slab|graded|sgc|beckett` (case-insensitive)
- Add max shipping filter when parsing price
- Enforce 2s timeout per query

## API Rate Limiting & Resilience

### Current Implementation
- Built-in rate limiter (`internal/ratelimit/`)
- Properly prevents API bans

### Enhancements Needed
- Per-host rate buckets (pokemontcg.io, pricecharting.com, svcs.ebay.com)
- On 429/5xx: jittered backoff (cap 4-8s) with single retry
- Deterministic burst blocking in tests

## Risk Register

### High Priority
1. **Outlier Data**: Invalid prices (69420.xx) causing incorrect recommendations
2. **API Rate Limits**: Risk of being banned from external APIs
3. **CSV Injection**: Security vulnerability in Excel/Sheets
4. **Untested Core**: 0% coverage on critical providers

### Medium Priority
1. **Test Coverage**: Overall 73% but missing critical paths
2. **Technical Debt**: 575-line main function hard to maintain
3. **No Progress Feedback**: Users unsure if tool is working
4. **Snapshot Integrity**: Missing run parameters in snapshots

### Low Priority
1. **Binary Size**: 21MB is large but functional
2. **No Logging Framework**: Basic logging works for now
3. **No Web UI**: CLI is sufficient for target users
4. **Cache Permissions**: Need to verify file permissions

## Test Coverage Priorities 🧪

### Sprint 1 Test Focus
1. **Outlier handling**: Test 69420.77 market price gets clamped/ignored
2. **CSV header stability**: Assert exact header sequence (golden file)
3. **Vendor selection**: Table-driven tests for TCG/Cardmarket fallback logic
4. **Rate limiter**: Assert N+1 requests in window are delayed
5. **History writer**: Verify single header, subsequent runs add rows only

### Missing Critical Tests
- PokeTCGIO provider: ListSets, CardsBySetID, pagination
- PriceCharting provider: LookupCard, field mapping, error handling
- Integration tests: Full CLI → Cards → Prices → Analysis → CSV flow
- Error scenarios: API failures, timeouts, rate limits
- Concurrent access: Cache locking under load

## Definition of Done

### Code Standards
- [ ] No placeholder or stub implementations
- [ ] No magic numbers or hardcoded values
- [ ] No random/demo data in production
- [ ] All tests passing
- [ ] Static analysis clean
- [ ] Documentation updated
- [ ] Error wrapping consistent (`fmt.Errorf("context: %w", err)`)

### Testing Requirements
- [ ] Unit tests for new code (>80% coverage)
- [ ] Integration tests for critical paths
- [ ] Manual validation completed
- [ ] Test coverage maintained/improved
- [ ] Performance benchmarks collected
- [ ] Golden tests for CSV headers

### Documentation
- [ ] README updated with examples
- [ ] API documentation current
- [ ] Sprint documentation complete
- [ ] Progress tracker updated

## Potential Data Source Integrations 🔍

### To Investigate (Future Enhancement)

#### 1. PriceCharting Enhanced Integration
- **URL**: https://www.pricecharting.com/
- **Current Use**: Already integrated for graded prices via API
- **Potential Enhancement**:
  - Historic price trends for better timing recommendations
  - Raw card pricing as backup/validation source
  - eBay sales data they track for more accurate market prices
- **Priority**: Medium - Already have API integration

#### 2. 130point.com - Real Sales Data
- **URL**: https://130point.com/sales/
- **Current Use**: Not integrated
- **Potential Value**:
  - Exact eBay sales including bidding history
  - Best offer accepted prices (usually hidden)
  - Real-time market validation
  - More accurate than listings for actual sale prices
- **Priority**: HIGH - Would significantly improve price accuracy
- **Integration Path**: Web scraping or API if available

#### 3. PSA Population Reports
- **URL**: https://www.psacard.com/pop
- **Current Use**: Population structure exists but no data source
- **Potential Value**:
  - Total graded copies for scarcity scoring
  - Compare populations between sets
  - Identify undervalued cards with low populations
  - Better ROI predictions based on supply
- **Priority**: HIGH - Critical for scarcity-based scoring
- **Integration Path**: Web scraping (no public API)

#### 4. Pokellector - Set Index
- **URL**: https://www.pokellector.com/
- **Current Use**: Not integrated
- **Potential Value**:
  - Complete card index with images
  - Accurate release dates for all sets
  - Set print runs and rarity information
  - Card variations and promos tracking
- **Priority**: Medium - Good for data validation
- **Integration Path**: Web scraping or manual data import

### Implementation Considerations
- **Rate Limiting**: All sources need respectful scraping
- **Caching Strategy**: Heavy caching for population/set data
- **Legal**: Review ToS for each service
- **Fallback**: Ensure tool works without these sources
- **Priority Order**:
  1. 130point for real sales
  2. PSA pop for scarcity scoring
  3. Enhanced PriceCharting historic data
  4. Pokellector for set metadata

## Next Actions

### Immediate (Sprint 3A - Week 1)
1. Complete PokemonPriceTracker integration
2. Enhance PriceCharting sales data extraction
3. Implement CSV population provider
4. Research PSA API availability
5. Design data fusion strategy

### Sprint 3A - Week 2
1. Complete integration testing suite
2. Write user documentation
3. Optimize performance (concurrent fetching)
4. Final validation and testing
5. Prepare for Sprint 4 (web interface)

### Future Priorities (Sprint 4+)
1. Simple web interface (localhost dashboard)
2. Enhanced eBay filtering
3. Additional grading companies (BGS, CGC)
4. Machine learning grade prediction

### Removed from Roadmap
- ❌ Multi-user features
- ❌ Authentication systems
- ❌ Scheduled/automated analysis
- ❌ Enterprise dashboards
- ❌ Mobile app development

---

*Last Updated: 2025-09-17*
*Sprint 3 Validation Complete, Sprint 3A Planning*