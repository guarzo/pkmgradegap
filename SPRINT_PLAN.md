# **Multi-Sprint Plan: Enhanced Pokemon Card Trading Intelligence**

## **Sprint 1: Auction Ending Alert System (2 weeks)**

### **User Stories**

**Epic**: As a Pokemon card trader, I want to be notified of ending auctions with high profit potential so I can capitalize on undervalued opportunities.

**Story 1.1**: Auction Discovery (5 points)
- **As a** user **I want** the system to monitor eBay auctions ending within 1 hour **so that** I can identify last-minute bidding opportunities
- **Acceptance Criteria**:
  - System polls eBay Finding API every 5 minutes for auctions ending <1 hour
  - Filters for Pokemon cards only
  - Captures current bid price, bid count, and estimated value
  - Stores auction data with expiration tracking

**Story 1.2**: Opportunity Scoring (3 points)
- **As a** user **I want** auctions scored by profit potential **so that** I can prioritize which ones to bid on
- **Acceptance Criteria**:
  - Score = (EstimatedValue - CurrentBid - Fees) / CurrentBid
  - Only show auctions with >30% profit potential
  - Factor in shipping costs and seller rating
  - Include grading potential assessment

**Story 1.3**: Alert Generation (3 points)
- **As a** user **I want** real-time notifications for high-value auction opportunities **so that** I don't miss profitable deals
- **Acceptance Criteria**:
  - Generate alerts for auctions with score >50% profit
  - Include auction details, current bid, time remaining
  - Support email, SMS, and web notifications
  - Rate limit alerts to prevent spam (max 5/hour)

**Story 1.4**: Web Dashboard Integration (2 points)
- **As a** user **I want** to see auction alerts in the web interface **so that** I can act on opportunities immediately
- **Acceptance Criteria**:
  - Real-time auction feed on dashboard
  - Click to open eBay auction in new tab
  - Mark auctions as "watching" or "ignored"
  - Historical alert performance tracking

### **Technical Tasks**

**Task 1.1**: eBay API Enhancement (8 hours)
```go
// Add to internal/ebay/ebay.go
func (c *Client) GetEndingAuctions(minutesRemaining int, category string) ([]Auction, error)
func (c *Client) GetAuctionDetails(itemId string) (*AuctionDetail, error)
```

**Task 1.2**: Auction Monitoring Service (12 hours)
```go
// Create internal/monitoring/auction_monitor.go
type AuctionMonitor struct {
    ebayClient *ebay.Client
    alertEngine *AlertEngine
    scheduler *cron.Cron
}
```

**Task 1.3**: Alert Type Extension (4 hours)
```go
// Add to internal/monitoring/alerts.go
const AlertAuctionOpportunity AlertType = "AUCTION_OPPORTUNITY"
```

**Task 1.4**: Web API Endpoints (6 hours)
```go
// Add to web server
GET /api/auctions/alerts - Get active auction alerts
POST /api/auctions/{id}/watch - Mark auction as watching
DELETE /api/auctions/{id}/ignore - Ignore auction
```

## **Sprint 2: Liquidity Scoring Enhancement (2 weeks)**

### **User Stories**

**Epic**: As a Pokemon card trader, I want to understand card liquidity so I can avoid buying cards that are difficult to sell.

**Story 2.1**: Market Data Collection (5 points)
- **As a** system **I need** to collect active listing counts and sales velocity data **so that** liquidity can be calculated
- **Acceptance Criteria**:
  - Fetch active listing counts from eBay, TCGPlayer, GameStop
  - Calculate 30-day sales velocity (sales per day)
  - Track listing duration averages
  - Store historical liquidity trends

**Story 2.2**: Liquidity Score Integration (3 points)
- **As a** user **I want** liquidity factored into opportunity scoring **so that** I prioritize cards that sell quickly
- **Acceptance Criteria**:
  - Add liquidity multiplier to scoring in `analysis.go:261`
  - Formula: `liquidityScore = (ListingVelocity * 10) / (ActiveListings + 1)`
  - High liquidity (>5) gets 1.2x score multiplier
  - Low liquidity (<1) gets 0.8x score multiplier
  - Display liquidity grade (A-F) in results

**Story 2.3**: Liquidity Reporting (2 points)
- **As a** user **I want** to see liquidity metrics in analysis reports **so that** I can make informed purchase decisions
- **Acceptance Criteria**:
  - Add liquidity columns to CSV output
  - Show "Days to Sell" estimate
  - Include liquidity trend (improving/declining)
  - Flag cards with poor liquidity (>30 days average)

**Story 2.4**: Market Depth Analysis (3 points)
- **As a** user **I want** to understand market competition levels **so that** I can assess selling difficulty
- **Acceptance Criteria**:
  - Calculate price distribution of active listings
  - Identify "price walls" (many listings at same price)
  - Show market depth score (0-100)
  - Warn about oversaturated markets

### **Technical Tasks**

**Task 2.1**: Market Data Provider Enhancement (10 hours)
```go
// Extend internal/analysis/analysis.go Row struct
type Row struct {
    // ... existing fields
    LiquidityScore    float64
    SalesVelocity     float64  // sales per day
    AvgListingDays    int      // average days to sell
    MarketDepth       int      // number of active listings
    PriceSpread       float64  // high/low listing ratio
}
```

**Task 2.2**: eBay Sales History API (8 hours)
```go
// Add to internal/ebay/ebay.go
func (c *Client) GetCompletedSales(query string, days int) ([]CompletedSale, error)
func (c *Client) GetActiveListingCount(query string) (int, error)
```

**Task 2.3**: Scoring Algorithm Update (6 hours)
```go
// Update ReportRankWithEbay in analysis.go
func calculateLiquidityMultiplier(row Row) float64 {
    liquidityScore := (row.SalesVelocity * 10) / (float64(row.MarketDepth) + 1)
    if liquidityScore >= 5 { return 1.2 }
    if liquidityScore <= 1 { return 0.8 }
    return 1.0
}
```

**Task 2.4**: Report Enhancement (4 hours)
```go
// Add liquidity columns to CSV output
header = append(header, "LiquidityGrade", "AvgDaysToSell", "MarketDepth", "LiquidityTrend")
```

## **Sprint 3: Smart Repricing Engine (2 weeks)**

### **User Stories**

**Epic**: As a Pokemon card seller, I want automated pricing adjustments so I can maximize sales velocity and profit margins.

**Story 3.1**: Velocity-Based Pricing (5 points)
- **As a** seller **I want** prices to decrease automatically for stale listings **so that** I don't miss sales opportunities
- **Acceptance Criteria**:
  - Decrease price by 5% every 7 days without watchers
  - Decrease by 3% every 7 days with watchers but no sales
  - Minimum price floor at 80% of original listing price
  - Email notification before each price adjustment

**Story 3.2**: Competitive Pricing (5 points)
- **As a** seller **I want** prices adjusted based on competitor changes **so that** I remain competitively positioned
- **Acceptance Criteria**:
  - Monitor competitor prices daily
  - Auto-adjust to match lowest price if within 5% margin
  - Increase price by 3% when velocity score >50%
  - Never adjust more than 10% per day

**Story 3.3**: Market Momentum Pricing (3 points)
- **As a** seller **I want** prices to respond to market trends **so that** I capitalize on bullish markets
- **Acceptance Criteria**:
  - Increase prices 2-5% during bullish trends
  - Decrease prices 3-7% during bearish trends
  - Use 7-day price change momentum indicators
  - Factor in tournament results and meta changes

**Story 3.4**: Repricing Dashboard (2 points)
- **As a** seller **I want** a dashboard to monitor and control repricing **so that** I can override automatic adjustments
- **Acceptance Criteria**:
  - Show all active listings with repricing status
  - Manual override controls for individual items
  - Repricing performance analytics
  - Profit/loss tracking per adjustment

### **Technical Tasks**

**Task 3.1**: Enhanced Repricer Core (12 hours)
```go
// Update internal/ebay/repricer.go
type RepricingEngine struct {
    strategies []RepricingStrategy
    priceFloor float64
    maxDailyChange float64
}

type RepricingStrategy interface {
    CalculateAdjustment(listing UserListing, marketData MarketData) PriceAdjustment
}
```

**Task 3.2**: Market Momentum Calculator (8 hours)
```go
// Create internal/monitoring/momentum.go
func CalculateMarketMomentum(priceHistory []float64, days int) MomentumScore
func DetectTournamentImpact(cardName string, recentEvents []TournamentResult) float64
```

**Task 3.3**: Automated Pricing Scheduler (6 hours)
```go
// Create internal/ebay/pricing_scheduler.go
type PricingScheduler struct {
    repricer *Repricer
    cron *cron.Cron
}
func (ps *PricingScheduler) ScheduleDailyRepricing()
```

**Task 3.4**: Trading API Integration (8 hours)
```go
// Extend internal/ebay/trading_api.go
func (t *TradingAPI) UpdateItemPrice(itemID string, newPrice float64) error
func (t *TradingAPI) GetMyActiveListings() ([]UserListing, error)
```

## **Integration Points & Dependencies**

### **Existing Code Integration**
- **eBay Client** (`internal/ebay/ebay.go`): Extend Finding API with auction-specific methods
- **Alert Engine** (`internal/monitoring/alerts.go:54`): Add auction alert types
- **Analysis Scoring** (`internal/analysis/analysis.go:261`): Inject liquidity multiplier
- **Repricer Foundation** (`internal/ebay/repricer.go:140`): Enhance with time-based strategies

### **Cross-Sprint Dependencies**
1. **Sprint 1 → Sprint 2**: Auction data feeds into liquidity calculations
2. **Sprint 2 → Sprint 3**: Liquidity scores inform repricing decisions
3. **All Sprints**: Web dashboard integration requires coordinated API development

### **External API Dependencies**
- **eBay Finding API**: Rate limits (5000 calls/day)
- **eBay Trading API**: OAuth token management
- **TCGPlayer API**: For price comparison data
- **Tournament results feeds**: For meta impact analysis

## **Testing Strategy & Risk Mitigation**

### **Testing Strategy**

**Unit Testing** (Each Sprint)
- 90%+ code coverage for new modules
- Mock eBay API responses for consistent testing
- Property-based testing for scoring algorithms
- Edge case testing (empty results, API failures)

**Integration Testing**
- End-to-end auction monitoring workflow
- Cross-provider data consistency validation
- Performance testing with 1000+ concurrent auctions
- Load testing web dashboard with real-time updates

**User Acceptance Testing**
- Beta testing with 5-10 active Pokemon traders
- A/B testing different repricing strategies
- Performance benchmarking against manual trading
- Financial impact measurement over 30-day periods

### **Risk Mitigation**

**Technical Risks**
- **eBay API Rate Limits**: Implement exponential backoff, request batching
- **Data Quality**: Validation layers, anomaly detection, fallback data sources
- **Performance**: Database indexing, caching layers, async processing

**Business Risks**
- **Repricing Errors**: Price floors, maximum change limits, manual override controls
- **Market Manipulation**: Detection algorithms, human review queues
- **Regulatory Compliance**: Terms of service adherence, rate limiting respect

**Operational Risks**
- **Alert Fatigue**: Smart filtering, severity levels, user preferences
- **False Positives**: Machine learning model refinement, user feedback loops
- **Maintenance Windows**: Graceful degradation, status page updates

## **Sprint Timeline & Deliverables**

**Sprint 1 (Weeks 1-2)**: Auction Alert MVP
- ✅ Basic auction monitoring (ending <1 hour)
- ✅ Simple profit score calculation
- ✅ Email/web notifications
- ✅ Web dashboard integration

**Sprint 2 (Weeks 3-4)**: Liquidity Intelligence
- ✅ Market data collection pipeline
- ✅ Liquidity score integration
- ✅ Enhanced reporting with liquidity metrics
- ✅ Market depth analysis

**Sprint 3 (Weeks 5-6)**: Smart Repricing Engine
- ✅ Velocity-based pricing adjustments
- ✅ Competitive monitoring and response
- ✅ Market momentum integration
- ✅ Repricing dashboard and controls

## **Success Metrics**

**Sprint 1**: Identify 20+ profitable auctions per week
**Sprint 2**: Improve card selection accuracy by 30%
**Sprint 3**: Increase sales velocity by 25%, reduce manual repricing by 80%

## **Implementation Notes**

This comprehensive sprint plan builds incrementally on the existing codebase architecture, leveraging the provider pattern and maintaining backward compatibility. Each sprint delivers standalone value while setting up the foundation for subsequent features.

The plan prioritizes high-impact, low-risk implementations first (auction alerts) before moving to more complex algorithmic enhancements (liquidity scoring) and finally automated trading features (smart repricing) that require careful risk management.

### **Key File Locations**
- Main analysis logic: `/home/tng/workspace/pkmgradegap/internal/analysis/analysis.go`
- eBay integration: `/home/tng/workspace/pkmgradegap/internal/ebay/`
- Monitoring & alerts: `/home/tng/workspace/pkmgradegap/internal/monitoring/`
- Web server endpoints: Referenced in existing web server implementation