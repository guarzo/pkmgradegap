# Population Data Integration (Future)

Currently, PSA does not provide a public API for population reports.
Future implementations could include:

1. **CSV Import**: Load population data from manually downloaded CSV files
2. **Web Scraping**: Automated scraping of PSA population reports
3. **Community Data**: Crowdsourced population tracking

The infrastructure exists via the CSV provider, but requires manual data input.

## Usage

If you have access to PSA population data in CSV format, you can use the CSV provider:

```go
popProvider := population.NewCSVPopulationProvider()
err := popProvider.LoadFromCSV("population_data.csv")
```

### Expected CSV Format

```csv
CardID,TotalGraded,PSA10,PSA9,PSA8
pokemon-base-set-charizard-4,15420,2847,4521,2892
pokemon-neo-genesis-lugia-9,8934,1247,2883,1654
```

The PSAPopulation struct in the Row data is available but currently unused due to the lack of public data sources.