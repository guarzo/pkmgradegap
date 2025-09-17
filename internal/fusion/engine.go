package fusion

import (
	"math"
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
)

type SourceType string

const (
	SourceTypeSale     SourceType = "SALE"
	SourceTypeListing  SourceType = "LISTING"
	SourceTypeEstimate SourceType = "ESTIMATE"
)

type DataSource struct {
	Name       string
	Type       SourceType
	Freshness  time.Duration
	Volume     int
	Confidence float64
	Timestamp  time.Time
}

type PriceData struct {
	Value    float64
	Currency string
	Source   DataSource
	Raw      bool
	Grade    string
}

type FusedPrice struct {
	Value           float64
	Currency        string
	Confidence      float64
	Sources         []DataSource
	ConfidenceRange struct {
		Low  float64
		High float64
	}
	Warnings []string
}

type FusionEngine struct {
	weights            map[SourceType]float64
	rules              []FusionRule
	freshnessDecayRate float64
	volumeWeight       float64
	varianceThreshold  float64
}

type FusionRule interface {
	Apply(prices []PriceData) *FusedPrice
}

func NewFusionEngine() *FusionEngine {
	return &FusionEngine{
		weights: map[SourceType]float64{
			SourceTypeSale:     1.0,
			SourceTypeListing:  0.7,
			SourceTypeEstimate: 0.5,
		},
		freshnessDecayRate: 0.95,
		volumeWeight:       0.2,
		varianceThreshold:  0.3,
	}
}

func (e *FusionEngine) FusePrice(prices []PriceData) FusedPrice {
	if len(prices) == 0 {
		return FusedPrice{
			Confidence: 0,
			Warnings:   []string{"No price data available"},
		}
	}

	if len(prices) == 1 {
		return e.singleSourcePrice(prices[0])
	}

	weighted := e.calculateWeightedAverage(prices)
	confidence := e.calculateConfidence(prices)
	e.addConfidenceInterval(&weighted, prices)
	e.detectAnomalies(&weighted, prices)

	weighted.Confidence = confidence
	return weighted
}

func (e *FusionEngine) calculateWeightedAverage(prices []PriceData) FusedPrice {
	var totalWeight float64
	var weightedSum float64
	sources := make([]DataSource, 0, len(prices))

	for _, price := range prices {
		weight := e.calculateWeight(price)
		weightedSum += price.Value * weight
		totalWeight += weight
		sources = append(sources, price.Source)
	}

	if totalWeight == 0 {
		return FusedPrice{
			Value:    0,
			Warnings: []string{"No valid weights calculated"},
		}
	}

	return FusedPrice{
		Value:    weightedSum / totalWeight,
		Currency: prices[0].Currency,
		Sources:  sources,
	}
}

func (e *FusionEngine) calculateWeight(price PriceData) float64 {
	typeWeight := e.weights[price.Source.Type]

	freshnessWeight := math.Pow(e.freshnessDecayRate, float64(price.Source.Freshness.Hours())/24.0)

	volumeWeight := 1.0
	if price.Source.Volume > 0 {
		volumeWeight = 1.0 + e.volumeWeight*math.Log10(float64(price.Source.Volume))
	}

	confidenceWeight := price.Source.Confidence
	if confidenceWeight == 0 {
		confidenceWeight = 0.5
	}

	return typeWeight * freshnessWeight * volumeWeight * confidenceWeight
}

func (e *FusionEngine) calculateConfidence(prices []PriceData) float64 {
	if len(prices) == 0 {
		return 0
	}

	var avgFreshness float64
	var totalVolume int
	var avgSourceConfidence float64

	for _, price := range prices {
		avgFreshness += float64(price.Source.Freshness.Hours())
		totalVolume += price.Source.Volume
		avgSourceConfidence += price.Source.Confidence
	}

	avgFreshness /= float64(len(prices))
	avgSourceConfidence /= float64(len(prices))

	freshnessScore := math.Max(0, 1.0-avgFreshness/720.0)

	volumeScore := math.Min(1.0, math.Log10(float64(totalVolume+1))/3.0)

	variance := e.calculateVariance(prices)
	varianceScore := math.Max(0, 1.0-variance/e.varianceThreshold)

	sourceTypeScore := 0.0
	for _, price := range prices {
		if price.Source.Type == SourceTypeSale {
			sourceTypeScore = 1.0
			break
		} else if price.Source.Type == SourceTypeListing {
			sourceTypeScore = math.Max(sourceTypeScore, 0.7)
		} else {
			sourceTypeScore = math.Max(sourceTypeScore, 0.5)
		}
	}

	confidence := (freshnessScore*0.2 + volumeScore*0.2 +
		varianceScore*0.3 + sourceTypeScore*0.2 + avgSourceConfidence*0.1)

	return math.Min(1.0, math.Max(0.0, confidence))
}

func (e *FusionEngine) calculateVariance(prices []PriceData) float64 {
	if len(prices) < 2 {
		return 0
	}

	var sum float64
	for _, price := range prices {
		sum += price.Value
	}
	mean := sum / float64(len(prices))

	var variance float64
	for _, price := range prices {
		diff := price.Value - mean
		variance += diff * diff
	}

	return math.Sqrt(variance/float64(len(prices))) / mean
}

func (e *FusionEngine) addConfidenceInterval(fused *FusedPrice, prices []PriceData) {
	if len(prices) < 2 {
		fused.ConfidenceRange.Low = fused.Value * 0.95
		fused.ConfidenceRange.High = fused.Value * 1.05
		return
	}

	variance := e.calculateVariance(prices)
	margin := fused.Value * variance

	fused.ConfidenceRange.Low = fused.Value - margin
	fused.ConfidenceRange.High = fused.Value + margin
}

func (e *FusionEngine) detectAnomalies(fused *FusedPrice, prices []PriceData) {
	if len(prices) < 3 {
		return
	}

	for _, price := range prices {
		deviation := math.Abs(price.Value-fused.Value) / fused.Value
		if deviation > e.varianceThreshold {
			warning := "Anomaly detected: " + price.Source.Name +
				" deviates significantly from consensus"
			fused.Warnings = append(fused.Warnings, warning)
		}
	}
}

func (e *FusionEngine) singleSourcePrice(price PriceData) FusedPrice {
	confidence := e.calculateSingleSourceConfidence(price)

	return FusedPrice{
		Value:      price.Value,
		Currency:   price.Currency,
		Confidence: confidence,
		Sources:    []DataSource{price.Source},
		ConfidenceRange: struct {
			Low  float64
			High float64
		}{
			Low:  price.Value * 0.9,
			High: price.Value * 1.1,
		},
	}
}

func (e *FusionEngine) calculateSingleSourceConfidence(price PriceData) float64 {
	typeScore := e.weights[price.Source.Type]
	freshnessScore := math.Max(0, 1.0-float64(price.Source.Freshness.Hours())/720.0)
	volumeScore := 0.5
	if price.Source.Volume > 0 {
		volumeScore = math.Min(1.0, math.Log10(float64(price.Source.Volume+1))/2.0)
	}

	return (typeScore*0.4 + freshnessScore*0.3 + volumeScore*0.3) * price.Source.Confidence
}

type FusedData struct {
	Card       model.Card
	RawPrice   FusedPrice
	PSA10Price FusedPrice
	PSA9Price  FusedPrice
	CGC95Price FusedPrice
	BGS10Price FusedPrice
	Population *model.PSAPopulation
	Sales      []SaleData
	Confidence ConfidenceScore
}

type SaleData struct {
	Price     float64
	Date      time.Time
	Platform  string
	Condition string
}

type ConfidenceScore struct {
	Overall  float64
	Factors  map[string]float64
	Warnings []string
}

func (e *FusionEngine) FuseCardData(card model.Card, prices map[string][]PriceData,
	population *model.PSAPopulation, sales []SaleData) FusedData {

	result := FusedData{
		Card:       card,
		Population: population,
		Sales:      sales,
	}

	if rawPrices, ok := prices["raw"]; ok {
		result.RawPrice = e.FusePrice(rawPrices)
	}

	if psa10Prices, ok := prices["psa10"]; ok {
		result.PSA10Price = e.FusePrice(psa10Prices)
	}

	if psa9Prices, ok := prices["psa9"]; ok {
		result.PSA9Price = e.FusePrice(psa9Prices)
	}

	if cgc95Prices, ok := prices["cgc95"]; ok {
		result.CGC95Price = e.FusePrice(cgc95Prices)
	}

	if bgs10Prices, ok := prices["bgs10"]; ok {
		result.BGS10Price = e.FusePrice(bgs10Prices)
	}

	result.Confidence = e.calculateOverallConfidence(result)

	return result
}

func (e *FusionEngine) calculateOverallConfidence(data FusedData) ConfidenceScore {
	factors := make(map[string]float64)
	warnings := []string{}

	factors["raw_price"] = data.RawPrice.Confidence
	factors["psa10_price"] = data.PSA10Price.Confidence

	if data.Population != nil && data.Population.TotalGraded > 0 {
		factors["population"] = 1.0
	} else {
		factors["population"] = 0.0
		warnings = append(warnings, "No population data available")
	}

	if len(data.Sales) > 0 {
		factors["sales"] = math.Min(1.0, float64(len(data.Sales))/10.0)
	} else {
		factors["sales"] = 0.0
		warnings = append(warnings, "No recent sales data")
	}

	var totalWeight float64
	var weightedSum float64
	weights := map[string]float64{
		"raw_price":   0.25,
		"psa10_price": 0.35,
		"population":  0.20,
		"sales":       0.20,
	}

	for factor, value := range factors {
		weight := weights[factor]
		weightedSum += value * weight
		totalWeight += weight
	}

	overall := weightedSum / totalWeight

	return ConfidenceScore{
		Overall:  overall,
		Factors:  factors,
		Warnings: append(warnings, data.RawPrice.Warnings...),
	}
}
