# PokéPrice - Pokemon Card Grade Gap Analyzer

Analyze price differences between raw and graded Pokemon cards to identify profitable grading opportunities.

## What It Does

PokéPrice fetches current market prices for Pokemon cards and compares ungraded (raw) prices against various graded conditions (PSA 10, PSA 9, CGC 9.5, BGS 10). This helps collectors identify which cards offer the best potential return when professionally graded.

## Quick Start

```bash
# Clone and build
git clone https://github.com/guarzo/pkmgradegap.git
cd pkmgradegap
go mod tidy
go build -o pkmgradegap ./cmd/pokeprice

# Set API tokens
export PRICECHARTING_TOKEN="your_token_here"  # Required
export POKEMONTCGIO_API_KEY="optional_key"     # Optional, increases rate limits

# Run analysis
./pkmgradegap --set "Surging Sparks" --analysis raw-vs-psa10 > report.csv
```

## Commands

### List all Pokemon TCG sets
```bash
./pkmgradegap --list-sets
```

### Analyze raw vs PSA 10 prices
```bash
./pkmgradegap --set "Surging Sparks" --analysis raw-vs-psa10
```

### Compare multiple grading companies
```bash
./pkmgradegap --set "Surging Sparks" --analysis psa9-cgc95-bgs95-vs-psa10
```

## API Requirements

| API | Required | Purpose | Get Key |
|-----|----------|---------|---------|
| PriceCharting | ✅ Yes | Graded card prices | [pricecharting.com/api](https://www.pricecharting.com/api) |
| Pokemon TCG | ❌ Optional | Card data (works without key) | [pokemontcg.io](https://pokemontcg.io) |

## Output Examples

### Raw vs PSA 10 Analysis
```csv
Card,Number,RawUSD,RawSource,PSA10_USD,Delta_USD,Notes
Pikachu ex,238,$45.00,tcgplayer.market,$125.00,$80.00,USD
Charizard ex,199,$89.99,tcgplayer.market,$450.00,$360.01,USD
```

### Multi-Grade Comparison
```csv
Card,Number,PSA9_USD,CGC/BGS_9.5_USD,BGS10_USD,PSA10_USD,PSA9/10_%,9.5/10_%,BGS10/PSA10_%
Pikachu ex,238,$65.00,$85.00,$150.00,$125.00,52.0%,68.0%,120.0%
```

## How It Works

1. **Fetches card data** from Pokemon TCG API (v2)
   - Includes embedded TCGPlayer and Cardmarket prices for raw cards
   
2. **Looks up graded prices** via PriceCharting API
   - Maps specific price fields to grades:
     - PSA 10: `manual-only-price`
     - Grade 9: `graded-price`
     - Grade 9.5: `box-only-price`
     - BGS 10: `bgs-10-price`

3. **Calculates gaps** between raw and graded prices
   - Shows dollar differences and percentages
   - Helps identify best ROI opportunities

## Project Structure

```
pkmgradegap/
├── cmd/pokeprice/main.go        # CLI interface
├── internal/
│   ├── analysis/analysis.go     # Price comparison logic
│   ├── cards/poketcgio.go       # Pokemon TCG API client
│   ├── prices/pricechart.go     # PriceCharting API client
│   └── model/types.go           # Data structures
├── go.mod
└── README.md
```

## Analysis Types

### `raw-vs-psa10`
Best for identifying cards with the highest absolute price increase when graded PSA 10.

**Use when:** You want to find cards worth grading to PSA 10  
**Output:** Raw price, PSA 10 price, and dollar difference

### `psa9-cgc95-bgs95-vs-psa10`
Compares multiple grade levels as percentages of PSA 10 value.

**Use when:** You want to understand relative values across grading companies  
**Output:** Prices and percentages for PSA 9, CGC/BGS 9.5, and BGS 10 vs PSA 10

## Known Issues

- **Variant matching**: May not perfectly match promo cards or special editions
- **Currency**: Cardmarket prices shown in EUR without conversion
- **Rate limits**: Free tier limited to 20,000 requests/day for Pokemon TCG API

## Contributing

Pull requests welcome! Areas that need help:
- Improving card variant matching
- Adding new price providers
- Building a web interface
- Documentation improvements

## License

MIT

## Credits

Built using:
- [Pokemon TCG API](https://pokemontcg.io) for card data
- [PriceCharting](https://www.pricecharting.com) for graded prices
- Go standard library (no external dependencies)