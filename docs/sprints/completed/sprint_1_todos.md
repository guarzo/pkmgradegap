# Sprint 1 Todo List

## 🔴 Critical Bugs (Must Fix First)

### BUG-1: Data Sanitizer for Outlier Prices
- [ ] Create `internal/analysis/sanitize.go` with outlier detection
- [ ] Implement rarity-based price caps (e.g., $500 for commons, $5k for SIR/HR)
- [ ] Add vendor selection priority (TCGPlayer market > mid > Cardmarket trend)
- [ ] Filter out penny cards (<$0.05) unless flag enabled
- [ ] Add comprehensive tests with 69420.xx values
- [ ] Integrate sanitizer into main analysis flow

### BUG-2: CSV Formula Injection Protection
- [ ] Create `internal/report/csvsafe.go` for escaping formulas
- [ ] Escape cells starting with =, +, -, @ with single quote
- [ ] Add golden test for CSV header stability
- [ ] Ensure consistent column ordering across modes
- [ ] Test with Excel and Google Sheets

### BUG-3: Cache Locking Fix
- [ ] Review `internal/cache/cache.go` for deadlock potential
- [ ] Refactor to use `putLocked` and `saveLocked` pattern
- [ ] Add concurrent access tests
- [ ] Verify no deadlocks under load
- [ ] Add mutex contention benchmarks

## 🧪 Test Coverage (Core Providers)

### TEST-1: PokeTCGIO Provider Tests
- [ ] Create `internal/cards/poketcgio_test.go`
- [ ] Mock Pokemon TCG API responses
- [ ] Test ListSets() with pagination
- [ ] Test CardsBySetID() with various set sizes
- [ ] Test error handling for API failures
- [ ] Test rate limiting behavior
- [ ] Achieve >80% coverage

### TEST-2: PriceCharting Provider Tests
- [ ] Create `internal/prices/pricechart_test.go`
- [ ] Mock PriceCharting API responses
- [ ] Test LookupCard() with various queries
- [ ] Test price field mapping (PSA10, Grade9, etc.)
- [ ] Test error handling and retries
- [ ] Test cache integration
- [ ] Achieve >80% coverage

### TEST-3: Integration Tests
- [ ] Create `internal/integration/flow_test.go`
- [ ] Test complete flow: CLI → Cards → Prices → Analysis → CSV
- [ ] Test with mock data providers
- [ ] Test snapshot save/load cycle
- [ ] Test history tracking append
- [ ] Test eBay integration path
- [ ] Test error propagation
- [ ] Test context cancellation

## 🔧 Code Quality Improvements

### REFACTOR-1: Break Down main.go
- [ ] Extract flag parsing to separate function
- [ ] Extract provider initialization
- [ ] Extract analysis mode selection
- [ ] Extract CSV output logic
- [ ] Create helper functions for common patterns
- [ ] Ensure no function >50 lines
- [ ] Add unit tests for extracted functions

## ✨ Features

### FEATURE-1: Progress Indicators
- [ ] Add progress bar library or simple spinner
- [ ] Show progress during set listing
- [ ] Show progress during card fetching (X/Y cards)
- [ ] Show progress during price lookups
- [ ] Add --quiet flag to disable progress
- [ ] Test with various terminal types

## 📝 Documentation Updates

### Documentation Tasks
- [ ] Update README with data quality notes
- [ ] Document outlier handling behavior
- [ ] Add troubleshooting section
- [ ] Document CSV security measures
- [ ] Update API requirements section
- [ ] Add performance tuning guide

## 🔍 Validation & Testing

### Manual Validation Checklist
- [ ] Test with "Surging Sparks" set
- [ ] Verify outliers are filtered
- [ ] Verify CSV is Excel-safe
- [ ] Test concurrent cache access
- [ ] Test with missing API keys
- [ ] Test with rate limit scenarios
- [ ] Verify progress indicators work
- [ ] Test snapshot save/load
- [ ] Test history append
- [ ] Performance benchmark before/after

## 📊 Sprint Metrics to Track

### Before Sprint
- [ ] Record current test coverage (73%)
- [ ] Count warnings/errors in build
- [ ] Benchmark current performance
- [ ] Document known failing scenarios

### After Sprint
- [ ] Measure new test coverage (target: 80%+)
- [ ] Verify zero warnings in build
- [ ] Compare performance benchmarks
- [ ] Verify all scenarios fixed

## 🚀 Definition of Done

### Each Story Must Have
- [ ] Code implementation complete
- [ ] Unit tests written and passing
- [ ] Integration tests where applicable
- [ ] Documentation updated
- [ ] Manual testing performed
- [ ] Code review completed
- [ ] No new warnings or errors
- [ ] Performance acceptable

### Sprint Complete When
- [ ] All critical bugs fixed
- [ ] Test coverage >80% for core providers
- [ ] Integration tests passing
- [ ] main.go refactored
- [ ] Progress indicators working
- [ ] Documentation updated
- [ ] Manual validation complete
- [ ] Sprint retrospective done

---

*Sprint 1: Data Quality & Core Stability*
*Duration: Jan 16-30, 2025*
*Total Points: 34*