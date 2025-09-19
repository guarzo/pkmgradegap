package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/guarzo/pkmgradegap/internal/analysis"
	// "github.com/guarzo/pkmgradegap/internal/fusion" // TODO: Update when fusion package is refactored
	"github.com/guarzo/pkmgradegap/internal/model"
)

// Pipeline processes cards through multiple stages concurrently
type Pipeline struct {
	stages       []Stage
	bufferSize   int
	errorHandler ErrorHandler
	metrics      *PipelineMetrics
	progressChan chan StageProgress
}

// Stage represents a processing stage in the pipeline
type Stage interface {
	Name() string
	Process(ctx context.Context, input <-chan StageData) <-chan StageData
	Parallel() bool // Whether this stage can run in parallel
}

// StageData carries data between pipeline stages
type StageData struct {
	Card      model.Card
	CardData  interface{}
	PriceData interface{}
	PopData   interface{}
	// FusedData   *fusion.FusedData // TODO: Update when fusion package is refactored
	AnalysisRow *analysis.Row
	Error       error
	Metadata    map[string]interface{}
}

// ErrorHandler defines how to handle stage errors
type ErrorHandler func(stage string, data StageData, err error) StageData

// PipelineConfig holds pipeline configuration
type PipelineConfig struct {
	BufferSize   int
	ErrorHandler ErrorHandler
	Stages       []Stage
}

// PipelineMetrics tracks pipeline performance
type PipelineMetrics struct {
	StartTime      time.Time
	EndTime        time.Time
	TotalItems     int
	ProcessedItems int
	ErrorCount     int
	StageMetrics   map[string]*StageMetrics
	Throughput     float64 // items per second
	mu             sync.RWMutex
}

// StageMetrics tracks individual stage performance
type StageMetrics struct {
	Name           string
	ItemsProcessed int
	ItemsErrored   int
	AverageLatency time.Duration
	TotalLatency   time.Duration
	StartTime      time.Time
	EndTime        time.Time
}

// StageProgress represents progress through pipeline stages
type StageProgress struct {
	StageName     string
	Completed     int
	Total         int
	CurrentItem   string
	StageNumber   int
	TotalStages   int
	ElapsedTime   time.Duration
	EstimatedLeft time.Duration
}

// NewPipeline creates a new processing pipeline
func NewPipeline(config PipelineConfig) *Pipeline {
	bufferSize := config.BufferSize
	if bufferSize == 0 {
		bufferSize = 100 // Default buffer size
	}

	pipeline := &Pipeline{
		stages:       config.Stages,
		bufferSize:   bufferSize,
		errorHandler: config.ErrorHandler,
		progressChan: make(chan StageProgress, 100),
		metrics: &PipelineMetrics{
			StageMetrics: make(map[string]*StageMetrics),
		},
	}

	// Initialize stage metrics
	for i, stage := range pipeline.stages {
		pipeline.metrics.StageMetrics[stage.Name()] = &StageMetrics{
			Name: stage.Name(),
		}
		_ = i // Use i if needed for stage numbering
	}

	return pipeline
}

// Process runs cards through the entire pipeline
func (p *Pipeline) Process(ctx context.Context, cards []model.Card) <-chan analysis.Row {
	p.metrics.StartTime = time.Now()
	p.metrics.TotalItems = len(cards)

	// Create input channel
	input := make(chan StageData, p.bufferSize)

	// Start pipeline stages
	current := p.startStages(ctx, input)

	// Convert final output to analysis.Row
	output := make(chan analysis.Row, p.bufferSize)
	go p.convertOutput(ctx, current, output)

	// Feed input cards
	go func() {
		defer close(input)
		for _, card := range cards {
			select {
			case input <- StageData{
				Card:     card,
				Metadata: make(map[string]interface{}),
			}:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Start progress tracking
	go p.trackProgress(ctx)

	return output
}

// startStages connects all pipeline stages
func (p *Pipeline) startStages(ctx context.Context, input <-chan StageData) <-chan StageData {
	current := input

	for i, stage := range p.stages {
		stageInput := current
		stageOutput := stage.Process(ctx, stageInput)

		// Add metrics tracking
		current = p.addMetricsTracking(ctx, stage.Name(), stageOutput, i)
	}

	return current
}

// addMetricsTracking wraps a stage output with metrics collection
func (p *Pipeline) addMetricsTracking(ctx context.Context, stageName string, input <-chan StageData, stageNum int) <-chan StageData {
	output := make(chan StageData, p.bufferSize)

	go func() {
		defer close(output)

		stageMetrics := p.metrics.StageMetrics[stageName]
		stageMetrics.StartTime = time.Now()

		for {
			select {
			case data, ok := <-input:
				if !ok {
					stageMetrics.EndTime = time.Now()
					return
				}

				start := time.Now()

				// Process with error handling
				if data.Error != nil && p.errorHandler != nil {
					data = p.errorHandler(stageName, data, data.Error)
				}

				latency := time.Since(start)

				// Update metrics
				p.updateStageMetrics(stageName, data, latency)

				// Send progress update
				p.sendProgressUpdate(stageName, stageNum, stageMetrics.ItemsProcessed)

				select {
				case output <- data:
				case <-ctx.Done():
					return
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return output
}

// convertOutput converts final pipeline output to analysis.Row
func (p *Pipeline) convertOutput(ctx context.Context, input <-chan StageData, output chan<- analysis.Row) {
	defer close(output)

	for {
		select {
		case data, ok := <-input:
			if !ok {
				p.metrics.EndTime = time.Now()
				p.calculateFinalMetrics()
				return
			}

			if data.AnalysisRow != nil {
				select {
				case output <- *data.AnalysisRow:
					p.updateProcessedCount()
				case <-ctx.Done():
					return
				}
			} else if data.Error != nil {
				p.updateErrorCount()
			}

		case <-ctx.Done():
			return
		}
	}
}

// Built-in pipeline stages

// CardFetchStage fetches basic card information
type CardFetchStage struct {
	cardProvider CardProvider
}

type CardProvider interface {
	GetCard(ctx context.Context, card model.Card) (interface{}, error)
}

func NewCardFetchStage(provider CardProvider) *CardFetchStage {
	return &CardFetchStage{cardProvider: provider}
}

func (s *CardFetchStage) Name() string   { return "card_fetch" }
func (s *CardFetchStage) Parallel() bool { return true }

func (s *CardFetchStage) Process(ctx context.Context, input <-chan StageData) <-chan StageData {
	output := make(chan StageData, 100)

	go func() {
		defer close(output)

		for {
			select {
			case data, ok := <-input:
				if !ok {
					return
				}

				if s.cardProvider != nil {
					cardData, err := s.cardProvider.GetCard(ctx, data.Card)
					if err != nil {
						data.Error = fmt.Errorf("card fetch failed: %w", err)
					} else {
						data.CardData = cardData
					}
				}

				select {
				case output <- data:
				case <-ctx.Done():
					return
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return output
}

// PriceFetchStage fetches price information
type PriceFetchStage struct {
	priceProvider PriceProvider
}

type PriceProvider interface {
	GetPrice(ctx context.Context, card model.Card) (interface{}, error)
}

func NewPriceFetchStage(provider PriceProvider) *PriceFetchStage {
	return &PriceFetchStage{priceProvider: provider}
}

func (s *PriceFetchStage) Name() string   { return "price_fetch" }
func (s *PriceFetchStage) Parallel() bool { return true }

func (s *PriceFetchStage) Process(ctx context.Context, input <-chan StageData) <-chan StageData {
	output := make(chan StageData, 100)

	go func() {
		defer close(output)

		for {
			select {
			case data, ok := <-input:
				if !ok {
					return
				}

				if s.priceProvider != nil {
					priceData, err := s.priceProvider.GetPrice(ctx, data.Card)
					if err != nil {
						data.Error = fmt.Errorf("price fetch failed: %w", err)
					} else {
						data.PriceData = priceData
					}
				}

				select {
				case output <- data:
				case <-ctx.Done():
					return
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return output
}

// PopulationFetchStage fetches population information
type PopulationFetchStage struct {
	populationProvider PopulationProvider
	targeting          TargetingEngine
}

type PopulationProvider interface {
	GetPopulation(ctx context.Context, card model.Card) (interface{}, error)
}

type TargetingEngine interface {
	ShouldFetchPopulation(card model.Card) bool
}

func NewPopulationFetchStage(provider PopulationProvider, targeting TargetingEngine) *PopulationFetchStage {
	return &PopulationFetchStage{
		populationProvider: provider,
		targeting:          targeting,
	}
}

func (s *PopulationFetchStage) Name() string   { return "population_fetch" }
func (s *PopulationFetchStage) Parallel() bool { return true }

func (s *PopulationFetchStage) Process(ctx context.Context, input <-chan StageData) <-chan StageData {
	output := make(chan StageData, 100)

	go func() {
		defer close(output)

		for {
			select {
			case data, ok := <-input:
				if !ok {
					return
				}

				// Use targeting to decide if we should fetch population
				if s.targeting != nil && !s.targeting.ShouldFetchPopulation(data.Card) {
					data.Metadata["population_skipped"] = true
				} else if s.populationProvider != nil {
					popData, err := s.populationProvider.GetPopulation(ctx, data.Card)
					if err != nil {
						data.Error = fmt.Errorf("population fetch failed: %w", err)
					} else {
						data.PopData = popData
					}
				}

				select {
				case output <- data:
				case <-ctx.Done():
					return
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return output
}

// DataFusionStage combines data from multiple sources
// TODO: Update when fusion package is refactored
type DataFusionStage struct {
	// fusionEngine *fusion.FusionEngine
}

// func NewDataFusionStage(engine *fusion.FusionEngine) *DataFusionStage {
// 	return &DataFusionStage{fusionEngine: engine}
// }

func (s *DataFusionStage) Name() string   { return "data_fusion" }
func (s *DataFusionStage) Parallel() bool { return true }

// TODO: Update when fusion package is refactored
func (s *DataFusionStage) Process(ctx context.Context, input <-chan StageData) <-chan StageData {
	output := make(chan StageData, 100)

	go func() {
		defer close(output)

		for {
			select {
			case data, ok := <-input:
				if !ok {
					return
				}

				// if s.fusionEngine != nil {
				// 	// Convert raw data to fusion format and fuse
				// 	// This is a simplified example - real implementation would convert
				// 	// the various data types to fusion.PriceData format
				// 	fusedData := &fusion.FusedData{
				// 		Card: data.Card,
				// 		// Would populate based on actual data types
				// 	}
				// 	data.FusedData = fusedData
				// }

				select {
				case output <- data:
				case <-ctx.Done():
					return
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return output
}

// AnalysisStage performs final analysis and scoring
type AnalysisStage struct {
	analyzer AnalysisEngine
}

type AnalysisEngine interface {
	// Analyze(ctx context.Context, fusedData *fusion.FusedData) (*analysis.Row, error) // TODO: Update when fusion package is refactored
	Analyze(ctx context.Context, data interface{}) (*analysis.Row, error)
}

func NewAnalysisStage(analyzer AnalysisEngine) *AnalysisStage {
	return &AnalysisStage{analyzer: analyzer}
}

func (s *AnalysisStage) Name() string   { return "analysis" }
func (s *AnalysisStage) Parallel() bool { return true }

func (s *AnalysisStage) Process(ctx context.Context, input <-chan StageData) <-chan StageData {
	output := make(chan StageData, 100)

	go func() {
		defer close(output)

		for {
			select {
			case data, ok := <-input:
				if !ok {
					return
				}

				// if s.analyzer != nil && data.FusedData != nil {
				// 	analysisRow, err := s.analyzer.Analyze(ctx, data.FusedData)
				if s.analyzer != nil {
					analysisRow, err := s.analyzer.Analyze(ctx, data.CardData)
					if err != nil {
						data.Error = fmt.Errorf("analysis failed: %w", err)
					} else {
						data.AnalysisRow = analysisRow
					}
				}

				select {
				case output <- data:
				case <-ctx.Done():
					return
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return output
}

// Helper methods for metrics and progress tracking

func (p *Pipeline) updateStageMetrics(stageName string, data StageData, latency time.Duration) {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()

	stageMetrics := p.metrics.StageMetrics[stageName]
	stageMetrics.ItemsProcessed++
	stageMetrics.TotalLatency += latency

	if data.Error != nil {
		stageMetrics.ItemsErrored++
	}

	if stageMetrics.ItemsProcessed > 0 {
		stageMetrics.AverageLatency = stageMetrics.TotalLatency / time.Duration(stageMetrics.ItemsProcessed)
	}
}

func (p *Pipeline) updateProcessedCount() {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()
	p.metrics.ProcessedItems++
}

func (p *Pipeline) updateErrorCount() {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()
	p.metrics.ErrorCount++
}

func (p *Pipeline) sendProgressUpdate(stageName string, stageNum, completed int) {
	elapsed := time.Since(p.metrics.StartTime)
	var estimated time.Duration
	if completed > 0 {
		rate := float64(completed) / elapsed.Seconds()
		remaining := p.metrics.TotalItems - completed
		estimated = time.Duration(float64(remaining)/rate) * time.Second
	}

	progress := StageProgress{
		StageName:     stageName,
		Completed:     completed,
		Total:         p.metrics.TotalItems,
		StageNumber:   stageNum + 1,
		TotalStages:   len(p.stages),
		ElapsedTime:   elapsed,
		EstimatedLeft: estimated,
	}

	select {
	case p.progressChan <- progress:
	default:
		// Don't block if channel is full
	}
}

func (p *Pipeline) trackProgress(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Progress updates are sent from individual stages
		case <-ctx.Done():
			return
		}
	}
}

func (p *Pipeline) calculateFinalMetrics() {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()

	if !p.metrics.EndTime.IsZero() {
		duration := p.metrics.EndTime.Sub(p.metrics.StartTime)
		if duration > 0 {
			p.metrics.Throughput = float64(p.metrics.ProcessedItems) / duration.Seconds()
		}
	}
}

// GetMetrics returns current pipeline metrics
func (p *Pipeline) GetMetrics() *PipelineMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	// Return a copy without copying the mutex
	metrics := PipelineMetrics{
		StartTime:      p.metrics.StartTime,
		EndTime:        p.metrics.EndTime,
		TotalItems:     p.metrics.TotalItems,
		ProcessedItems: p.metrics.ProcessedItems,
		ErrorCount:     p.metrics.ErrorCount,
		Throughput:     p.metrics.Throughput,
		StageMetrics:   make(map[string]*StageMetrics),
	}
	for k, v := range p.metrics.StageMetrics {
		stageCopy := *v
		metrics.StageMetrics[k] = &stageCopy
	}

	return &metrics
}

// GetProgressChannel returns the progress channel
func (p *Pipeline) GetProgressChannel() <-chan StageProgress {
	return p.progressChan
}
