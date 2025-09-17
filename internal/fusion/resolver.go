package fusion

import (
	"math"
	"sort"
)

type ResolutionStrategy string

const (
	StrategyWeightedAverage   ResolutionStrategy = "weighted_average"
	StrategyMostRecent        ResolutionStrategy = "most_recent"
	StrategyHighestConfidence ResolutionStrategy = "highest_confidence"
	StrategyConservative      ResolutionStrategy = "conservative"
	StrategyAggressive        ResolutionStrategy = "aggressive"
	StrategyMedian            ResolutionStrategy = "median"
)

type PriceConflict struct {
	Sources    []PriceData
	Resolution ResolutionStrategy
}

type Resolver struct {
	defaultStrategy   ResolutionStrategy
	strategyThreshold float64
}

func NewResolver(defaultStrategy ResolutionStrategy) *Resolver {
	return &Resolver{
		defaultStrategy:   defaultStrategy,
		strategyThreshold: 0.25,
	}
}

func (r *Resolver) ResolveConflict(conflict PriceConflict) FusedPrice {
	if len(conflict.Sources) == 0 {
		return FusedPrice{
			Confidence: 0,
			Warnings:   []string{"No price sources to resolve"},
		}
	}

	strategy := r.selectStrategy(conflict)

	switch strategy {
	case StrategyWeightedAverage:
		return r.resolveWeightedAverage(conflict.Sources)
	case StrategyMostRecent:
		return r.resolveMostRecent(conflict.Sources)
	case StrategyHighestConfidence:
		return r.resolveHighestConfidence(conflict.Sources)
	case StrategyConservative:
		return r.resolveConservative(conflict.Sources)
	case StrategyAggressive:
		return r.resolveAggressive(conflict.Sources)
	case StrategyMedian:
		return r.resolveMedian(conflict.Sources)
	default:
		return r.resolveWeightedAverage(conflict.Sources)
	}
}

func (r *Resolver) selectStrategy(conflict PriceConflict) ResolutionStrategy {
	if conflict.Resolution != "" {
		return conflict.Resolution
	}

	variance := r.calculatePriceVariance(conflict.Sources)

	if variance > r.strategyThreshold {
		hasSales := false
		for _, source := range conflict.Sources {
			if source.Source.Type == SourceTypeSale {
				hasSales = true
				break
			}
		}

		if hasSales {
			return StrategyHighestConfidence
		}
		return StrategyConservative
	}

	return r.defaultStrategy
}

func (r *Resolver) resolveWeightedAverage(sources []PriceData) FusedPrice {
	engine := NewFusionEngine()
	return engine.FusePrice(sources)
}

func (r *Resolver) resolveMostRecent(sources []PriceData) FusedPrice {
	if len(sources) == 0 {
		return FusedPrice{}
	}

	mostRecent := sources[0]
	for _, source := range sources[1:] {
		if source.Source.Timestamp.After(mostRecent.Source.Timestamp) {
			mostRecent = source
		}
	}

	return FusedPrice{
		Value:      mostRecent.Value,
		Currency:   mostRecent.Currency,
		Confidence: mostRecent.Source.Confidence,
		Sources:    []DataSource{mostRecent.Source},
		ConfidenceRange: struct {
			Low  float64
			High float64
		}{
			Low:  mostRecent.Value * 0.95,
			High: mostRecent.Value * 1.05,
		},
	}
}

func (r *Resolver) resolveHighestConfidence(sources []PriceData) FusedPrice {
	if len(sources) == 0 {
		return FusedPrice{}
	}

	highest := sources[0]
	for _, source := range sources[1:] {
		if source.Source.Confidence > highest.Source.Confidence {
			highest = source
		}
	}

	dataSources := make([]DataSource, len(sources))
	for i, s := range sources {
		dataSources[i] = s.Source
	}

	return FusedPrice{
		Value:      highest.Value,
		Currency:   highest.Currency,
		Confidence: highest.Source.Confidence,
		Sources:    dataSources,
		ConfidenceRange: struct {
			Low  float64
			High float64
		}{
			Low:  highest.Value * 0.9,
			High: highest.Value * 1.1,
		},
	}
}

func (r *Resolver) resolveConservative(sources []PriceData) FusedPrice {
	if len(sources) == 0 {
		return FusedPrice{}
	}

	values := make([]float64, len(sources))
	for i, source := range sources {
		values[i] = source.Value
	}
	sort.Float64s(values)

	percentile25 := values[len(values)/4]

	dataSources := make([]DataSource, len(sources))
	var avgConfidence float64
	for i, s := range sources {
		dataSources[i] = s.Source
		avgConfidence += s.Source.Confidence
	}
	avgConfidence /= float64(len(sources))

	return FusedPrice{
		Value:      percentile25,
		Currency:   sources[0].Currency,
		Confidence: avgConfidence * 0.8,
		Sources:    dataSources,
		ConfidenceRange: struct {
			Low  float64
			High float64
		}{
			Low:  values[0],
			High: percentile25 * 1.1,
		},
		Warnings: []string{"Conservative estimate used due to high price variance"},
	}
}

func (r *Resolver) resolveAggressive(sources []PriceData) FusedPrice {
	if len(sources) == 0 {
		return FusedPrice{}
	}

	values := make([]float64, len(sources))
	for i, source := range sources {
		values[i] = source.Value
	}
	sort.Float64s(values)

	percentile75 := values[3*len(values)/4]

	dataSources := make([]DataSource, len(sources))
	var avgConfidence float64
	for i, s := range sources {
		dataSources[i] = s.Source
		avgConfidence += s.Source.Confidence
	}
	avgConfidence /= float64(len(sources))

	return FusedPrice{
		Value:      percentile75,
		Currency:   sources[0].Currency,
		Confidence: avgConfidence * 0.7,
		Sources:    dataSources,
		ConfidenceRange: struct {
			Low  float64
			High float64
		}{
			Low:  percentile75 * 0.9,
			High: values[len(values)-1],
		},
		Warnings: []string{"Aggressive estimate used for opportunity identification"},
	}
}

func (r *Resolver) resolveMedian(sources []PriceData) FusedPrice {
	if len(sources) == 0 {
		return FusedPrice{}
	}

	values := make([]float64, len(sources))
	for i, source := range sources {
		values[i] = source.Value
	}
	sort.Float64s(values)

	var median float64
	if len(values)%2 == 0 {
		median = (values[len(values)/2-1] + values[len(values)/2]) / 2
	} else {
		median = values[len(values)/2]
	}

	dataSources := make([]DataSource, len(sources))
	var avgConfidence float64
	for i, s := range sources {
		dataSources[i] = s.Source
		avgConfidence += s.Source.Confidence
	}
	avgConfidence /= float64(len(sources))

	mad := r.calculateMAD(values, median)

	return FusedPrice{
		Value:      median,
		Currency:   sources[0].Currency,
		Confidence: avgConfidence,
		Sources:    dataSources,
		ConfidenceRange: struct {
			Low  float64
			High float64
		}{
			Low:  median - mad,
			High: median + mad,
		},
	}
}

func (r *Resolver) calculatePriceVariance(sources []PriceData) float64 {
	if len(sources) < 2 {
		return 0
	}

	var sum float64
	for _, source := range sources {
		sum += source.Value
	}
	mean := sum / float64(len(sources))

	var variance float64
	for _, source := range sources {
		diff := source.Value - mean
		variance += diff * diff
	}

	return math.Sqrt(variance/float64(len(sources))) / mean
}

func (r *Resolver) calculateMAD(values []float64, median float64) float64 {
	deviations := make([]float64, len(values))
	for i, val := range values {
		deviations[i] = math.Abs(val - median)
	}
	sort.Float64s(deviations)

	if len(deviations)%2 == 0 {
		return (deviations[len(deviations)/2-1] + deviations[len(deviations)/2]) / 2
	}
	return deviations[len(deviations)/2]
}
