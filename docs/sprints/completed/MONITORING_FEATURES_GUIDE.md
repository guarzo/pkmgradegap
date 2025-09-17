# Monitoring Features User Guide

This guide covers the advanced monitoring and analytics features available in pkmgradegap.

## Table of Contents

- [Price Alerts](#price-alerts)
- [Trend Analysis](#trend-analysis)
- [Market Timing](#market-timing)
- [Bulk Optimization](#bulk-optimization)
- [Population Data Integration](#population-data-integration)
- [Automatic Snapshots](#automatic-snapshots)

## Price Alerts

The alerts feature compares two price snapshots to identify significant changes and opportunities.

### Basic Usage

```bash
# Compare two snapshots for price alerts
./pkmgradegap --analysis alerts \
  --compare-snapshots "old_snapshot.json,new_snapshot.json"
```

### Alert Types

- **PRICE_DROP**: Raw or PSA10 price decreased significantly
- **PRICE_INCREASE**: Raw or PSA10 price increased significantly
- **NEW_OPPORTUNITY**: Card became profitable for grading
- **LOST_OPPORTUNITY**: Card no longer profitable
- **VOLATILITY_SPIKE**: High price volatility detected
- **VOLATILITY_LOW**: Low volatility (stable prices)

### Configuration Options

```bash
# Customize alert thresholds
./pkmgradegap --analysis alerts \
  --compare-snapshots "snap1.json,snap2.json" \
  --alert-threshold-pct 15 \    # Alert on 15% changes
  --alert-threshold-usd 10 \     # Alert on $10 changes
  --alert-csv "alerts.csv"       # Export to CSV
```

### Example Output

```
PRICE ALERTS REPORT
==================
Generated: 2025-02-01 10:00:00
Total Alerts: 5
Severity Breakdown: 2 High, 3 Medium, 0 Low

[HIGH] PRICE_INCREASE
Card: Latias ex (#189)
Message: PSA10 price increased 54.5% ($150.00)
Recommended Actions:
  1. Consider selling if you own this card graded
  2. New PSA10 price: $425.00
  3. Monitor for sustained price level
```

## Trend Analysis

Analyze historical price data to identify market trends and patterns.

### Basic Usage

```bash
# Analyze trends from historical tracking
./pkmgradegap --analysis trends \
  --history "data/targets.csv"
```

### Export to CSV

```bash
# Export detailed trend analysis
./pkmgradegap --analysis trends \
  --history "data/targets.csv" \
  --trends-csv "trends_report.csv"
```

### Features

- **Linear Regression**: Identifies overall market direction
- **Moving Averages**: 7-day, 14-day, and 30-day MAs
- **Momentum Analysis**: Detects acceleration and support/resistance levels
- **Seasonal Patterns**: Identifies best/worst months for trading
- **Performance Tracking**: Shows top historical picks

### Example Output

```
HISTORICAL TREND ANALYSIS
=========================
Total Recommendations: 19
Average Score: 261.87

TREND ANALYSIS:
Overall Trend: BEARISH
Trend Strength: 41.9/100
Confidence: 0.6%

MOVING AVERAGES:
7-Day MA: $375.29
Signal: HOLD
```

## Market Timing

Get timing recommendations for when to buy, sell, or submit cards for grading.

### Basic Usage

```bash
# Get market timing recommendations for a set
./pkmgradegap --set "Surging Sparks" \
  --analysis market-timing
```

### Recommendations Types

- **BUY**: Raw prices low, good entry point
- **SELL**: PSA10 prices high, consider selling
- **SUBMIT**: Good time to submit for grading
- **HOLD**: No clear market signal

### Seasonal Analysis

The market timing engine analyzes seasonal patterns:

- **Holiday Season** (Nov-Dec): Higher prices, good for selling
- **Post-Holiday** (Jan-Feb): Lower prices, good for buying
- **Spring** (Mar-May): Stable prices
- **Summer** (Jun-Aug): Lower activity
- **Fall** (Sep-Oct): Increasing activity

## Bulk Optimization

Optimize card selection for PSA bulk submission tiers.

### Basic Usage

```bash
# Optimize for bulk submission
./pkmgradegap --set "Surging Sparks" \
  --analysis bulk-optimize
```

### Service Levels

The optimizer considers PSA service levels:
- **Regular**: $25/card, best for cards $50-$199
- **Express**: $50/card, best for cards $200-$499
- **Super Express**: $100/card, best for cards $500+

### Batch Recommendations

```
BULK SUBMISSION OPTIMIZATION
============================

SERVICE LEVEL: Regular ($25/card)
Target Value: $50 - $199
Batch Size: 20 cards

RECOMMENDED BATCH:
1. Pikachu ex #001 - Score: 82.0
2. Milotic ex #150 - Score: 25.0
...

Total Investment: $530.00
Expected Return: $1,650.00
Expected ROI: 211.3%
```

## Population Data Integration

Include PSA population data for more accurate scoring.

### Basic Usage

```bash
# Include population data in analysis
./pkmgradegap --set "Surging Sparks" \
  --analysis rank \
  --with-pop
```

### Scarcity Scoring

Population data adds bonus points based on PSA10 scarcity:
- **Ultra Rare** (≤10 PSA10s): +15 points
- **Very Rare** (≤50 PSA10s): +10 points
- **Rare** (≤100 PSA10s): +5 points
- **Uncommon** (≤500 PSA10s): +2 points

### Mock Mode

When real PSA API is unavailable, mock data is used:
```
Using mock population data for enhanced scoring
```

## Automatic Snapshots

Snapshots are automatically generated for monitoring features.

### Automatic Generation

When running rank or raw-vs-psa10 analysis:
```bash
./pkmgradegap --set "Surging Sparks" --analysis rank
```

Automatically saves to: `data/snapshots/Surging_Sparks_20250201_100000.json`

### Manual Snapshots

```bash
# Save snapshot to specific location
./pkmgradegap --set "Surging Sparks" \
  --analysis rank \
  --snapshot-out "my_snapshot.json"
```

### Loading Snapshots

```bash
# Load and analyze from snapshot
./pkmgradegap --set "Surging Sparks" \
  --snapshot-in "my_snapshot.json" \
  --analysis rank
```

## Workflow Examples

### Weekly Price Monitoring

1. Run analysis weekly with automatic snapshots:
```bash
./pkmgradegap --set "Surging Sparks" --analysis rank
```

2. Compare with previous week:
```bash
./pkmgradegap --analysis alerts \
  --compare-snapshots "data/snapshots/last_week.json,data/snapshots/this_week.json"
```

### Historical Performance Tracking

1. Enable history tracking:
```bash
./pkmgradegap --set "Surging Sparks" \
  --analysis rank \
  --history "data/targets.csv"
```

2. Analyze trends monthly:
```bash
./pkmgradegap --analysis trends \
  --history "data/targets.csv" \
  --trends-csv "monthly_trends.csv"
```

### Complete Monitoring Pipeline

```bash
# 1. Initial analysis with all features
./pkmgradegap --set "Surging Sparks" \
  --analysis rank \
  --with-pop \
  --with-ebay \
  --with-volatility \
  --top 20 \
  --snapshot-out "baseline.json" \
  --history "data/tracking.csv"

# 2. Weekly monitoring
./pkmgradegap --analysis alerts \
  --compare-snapshots "baseline.json,current.json" \
  --alert-csv "weekly_alerts.csv"

# 3. Monthly trend analysis
./pkmgradegap --analysis trends \
  --history "data/tracking.csv" \
  --trends-csv "monthly_trends.csv"

# 4. Quarterly market timing
./pkmgradegap --set "Surging Sparks" \
  --analysis market-timing
```

## Tips and Best Practices

1. **Regular Snapshots**: Take snapshots weekly for consistent monitoring
2. **Historical Data**: Build history over time for better trend analysis
3. **Alert Thresholds**: Adjust based on your risk tolerance
4. **Population Data**: Always use `--with-pop` for better accuracy
5. **Combine Features**: Use multiple analysis modes for comprehensive insights

## Troubleshooting

### No Alerts Generated
- Ensure snapshots have overlapping cards
- Check threshold settings aren't too high
- Verify snapshot format is correct

### Trends Not Showing
- Need at least 5 historical data points
- Check CSV format matches expected structure
- Ensure dates are properly formatted

### Market Timing No Recommendations
- Requires sufficient price data
- May need historical snapshots for better analysis
- Check grading cost parameters are realistic

## Sample Data

Example files are provided in `data/samples/`:
- `surging_sparks_snapshot_old.json` - Sample old snapshot
- `surging_sparks_snapshot_new.json` - Sample new snapshot
- `historical_tracking.csv` - Sample historical data

Use these for testing and learning the features.