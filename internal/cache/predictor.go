package cache

import (
	"encoding/json"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// CachePredictor uses machine learning techniques to predict cache access patterns
type CachePredictor struct {
	accessHistory []AccessRecord
	patterns      map[string]*AccessPattern
	sequences     map[string]*SequencePattern
	timePatterns  map[int]*TimePattern // hour of day -> pattern
	correlations  map[string][]string  // key -> related keys
	config        PredictorConfig
	mu            sync.RWMutex
}

// PredictorConfig holds configuration for the cache predictor
type PredictorConfig struct {
	MaxHistorySize    int           // Maximum access records to keep
	MinPatternSupport int           // Minimum occurrences for a pattern
	CorrelationWindow time.Duration // Time window for correlations
	PredictionCount   int           // Number of predictions to return
	PatternDecay      float64       // Decay rate for old patterns
}

// AccessRecord represents a single cache access
type AccessRecord struct {
	Key       string    `json:"key"`
	Timestamp time.Time `json:"timestamp"`
	Hit       bool      `json:"hit"`
	Size      int64     `json:"size"`
	Context   string    `json:"context"` // e.g., "analysis", "population", "price"
}

// AccessPattern represents a learned access pattern
type AccessPattern struct {
	Key           string         `json:"key"`
	Frequency     float64        `json:"frequency"` // Access frequency per hour
	LastAccessed  time.Time      `json:"last_accessed"`
	AccessCount   int            `json:"access_count"`
	HitRate       float64        `json:"hit_rate"`
	AverageSize   int64          `json:"average_size"`
	Contexts      map[string]int `json:"contexts"` // context -> count
	PredictedNext []string       `json:"predicted_next"`
	Confidence    float64        `json:"confidence"`
}

// SequencePattern represents sequential access patterns
type SequencePattern struct {
	Sequence   []string           `json:"sequence"`
	Support    int                `json:"support"`    // Number of times seen
	Confidence float64            `json:"confidence"` // Likelihood of completion
	LastSeen   time.Time          `json:"last_seen"`
	NextItems  map[string]float64 `json:"next_items"` // item -> probability
}

// TimePattern represents time-based access patterns
type TimePattern struct {
	Hour          int                `json:"hour"`
	PopularKeys   map[string]float64 `json:"popular_keys"` // key -> access probability
	PopularSets   map[string]float64 `json:"popular_sets"` // set -> access probability
	TotalAccesses int                `json:"total_accesses"`
}

// PredictionResult represents a cache prediction
type PredictionResult struct {
	Key           string        `json:"key"`
	Probability   float64       `json:"probability"`
	Reason        string        `json:"reason"`
	Context       string        `json:"context"`
	EstimatedSize int64         `json:"estimated_size"`
	TTL           time.Duration `json:"ttl"`
}

// NewCachePredictor creates a new cache predictor
func NewCachePredictor() *CachePredictor {
	config := PredictorConfig{
		MaxHistorySize:    10000,
		MinPatternSupport: 3,
		CorrelationWindow: 1 * time.Hour,
		PredictionCount:   20,
		PatternDecay:      0.95,
	}

	return &CachePredictor{
		accessHistory: make([]AccessRecord, 0, config.MaxHistorySize),
		patterns:      make(map[string]*AccessPattern),
		sequences:     make(map[string]*SequencePattern),
		timePatterns:  make(map[int]*TimePattern),
		correlations:  make(map[string][]string),
		config:        config,
	}
}

// RecordAccess records a cache access for learning
func (p *CachePredictor) RecordAccess(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	record := AccessRecord{
		Key:       key,
		Timestamp: time.Now(),
		Hit:       true, // Assume hit for now
		Context:   p.extractContext(key),
	}

	p.addAccessRecord(record)
	p.updatePatterns(record)
	p.updateSequences(record)
	p.updateTimePatterns(record)
	p.updateCorrelations(record)
}

// RecordSet records a cache set operation
func (p *CachePredictor) RecordSet(key string, data interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()

	size := p.estimateSize(data)

	record := AccessRecord{
		Key:       key,
		Timestamp: time.Now(),
		Hit:       false, // Set operation means it wasn't cached
		Size:      size,
		Context:   p.extractContext(key),
	}

	p.addAccessRecord(record)
}

// GetPredictions returns predicted cache targets
func (p *CachePredictor) GetPredictions() []PrefetchTarget {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var predictions []PredictionResult

	// Frequency-based predictions
	predictions = append(predictions, p.getFrequencyPredictions()...)

	// Sequence-based predictions
	predictions = append(predictions, p.getSequencePredictions()...)

	// Time-based predictions
	predictions = append(predictions, p.getTimePredictions()...)

	// Correlation-based predictions
	predictions = append(predictions, p.getCorrelationPredictions()...)

	// Sort by probability and convert to prefetch targets
	sort.Slice(predictions, func(i, j int) bool {
		return predictions[i].Probability > predictions[j].Probability
	})

	// Take top N predictions
	maxPredictions := p.config.PredictionCount
	if len(predictions) > maxPredictions {
		predictions = predictions[:maxPredictions]
	}

	targets := make([]PrefetchTarget, len(predictions))
	for i, pred := range predictions {
		targets[i] = PrefetchTarget{
			Key:         pred.Key,
			Priority:    len(predictions) - i, // Higher priority for higher probability
			TTL:         pred.TTL,
			Probability: pred.Probability,
		}
	}

	return targets
}

// Optimize performs maintenance on the prediction model
func (p *CachePredictor) Optimize() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()

	// Trim old access history
	if len(p.accessHistory) > p.config.MaxHistorySize {
		cutoff := len(p.accessHistory) - p.config.MaxHistorySize
		p.accessHistory = p.accessHistory[cutoff:]
	}

	// Decay old patterns
	for key, pattern := range p.patterns {
		age := now.Sub(pattern.LastAccessed)
		decayFactor := math.Pow(p.config.PatternDecay, age.Hours()/24.0)
		pattern.Frequency *= decayFactor
		pattern.Confidence *= decayFactor

		// Remove patterns with very low confidence
		if pattern.Confidence < 0.1 {
			delete(p.patterns, key)
		}
	}

	// Clean old sequences
	for seqKey, sequence := range p.sequences {
		age := now.Sub(sequence.LastSeen)
		if age > 7*24*time.Hour { // Remove sequences older than 7 days
			delete(p.sequences, seqKey)
		}
	}
}

// Helper methods

func (p *CachePredictor) addAccessRecord(record AccessRecord) {
	p.accessHistory = append(p.accessHistory, record)

	// Maintain size limit
	if len(p.accessHistory) > p.config.MaxHistorySize {
		p.accessHistory = p.accessHistory[1:]
	}
}

func (p *CachePredictor) updatePatterns(record AccessRecord) {
	pattern, exists := p.patterns[record.Key]
	if !exists {
		pattern = &AccessPattern{
			Key:           record.Key,
			Contexts:      make(map[string]int),
			PredictedNext: make([]string, 0),
		}
		p.patterns[record.Key] = pattern
	}

	pattern.AccessCount++
	pattern.LastAccessed = record.Timestamp
	pattern.Contexts[record.Context]++

	// Update frequency (accesses per hour)
	if pattern.AccessCount > 1 {
		timeDiff := record.Timestamp.Sub(pattern.LastAccessed).Hours()
		if timeDiff > 0 {
			pattern.Frequency = float64(pattern.AccessCount) / timeDiff
		}
	}

	// Update hit rate
	if record.Hit {
		pattern.HitRate = (pattern.HitRate*float64(pattern.AccessCount-1) + 1.0) / float64(pattern.AccessCount)
	} else {
		pattern.HitRate = (pattern.HitRate * float64(pattern.AccessCount-1)) / float64(pattern.AccessCount)
	}

	// Update average size
	if record.Size > 0 {
		pattern.AverageSize = (pattern.AverageSize*int64(pattern.AccessCount-1) + record.Size) / int64(pattern.AccessCount)
	}

	// Calculate confidence based on access count and recency
	recencyBonus := math.Exp(-time.Since(record.Timestamp).Hours() / 24.0)
	pattern.Confidence = math.Min(1.0, float64(pattern.AccessCount)/10.0) * recencyBonus
}

func (p *CachePredictor) updateSequences(record AccessRecord) {
	// Look at recent access history to find sequences
	recentSize := 5
	if len(p.accessHistory) < recentSize {
		return
	}

	recent := p.accessHistory[len(p.accessHistory)-recentSize:]
	sequence := make([]string, len(recent))
	for i, r := range recent {
		sequence[i] = r.Key
	}

	// Create sequence key
	seqKey := strings.Join(sequence, "->")

	seqPattern, exists := p.sequences[seqKey]
	if !exists {
		seqPattern = &SequencePattern{
			Sequence:  sequence,
			NextItems: make(map[string]float64),
		}
		p.sequences[seqKey] = seqPattern
	}

	seqPattern.Support++
	seqPattern.LastSeen = record.Timestamp

	// Look ahead to see what commonly comes next
	if len(p.accessHistory) > len(recent) {
		nextKey := p.accessHistory[len(p.accessHistory)-len(recent)-1].Key
		seqPattern.NextItems[nextKey]++
	}

	// Calculate confidence
	seqPattern.Confidence = math.Min(1.0, float64(seqPattern.Support)/float64(p.config.MinPatternSupport))
}

func (p *CachePredictor) updateTimePatterns(record AccessRecord) {
	hour := record.Timestamp.Hour()

	timePattern, exists := p.timePatterns[hour]
	if !exists {
		timePattern = &TimePattern{
			Hour:        hour,
			PopularKeys: make(map[string]float64),
			PopularSets: make(map[string]float64),
		}
		p.timePatterns[hour] = timePattern
	}

	timePattern.TotalAccesses++
	timePattern.PopularKeys[record.Key]++

	// Extract set name from key
	if setName := p.extractSetName(record.Key); setName != "" {
		timePattern.PopularSets[setName]++
	}
}

func (p *CachePredictor) updateCorrelations(record AccessRecord) {
	// Find keys accessed within the correlation window
	cutoff := record.Timestamp.Add(-p.config.CorrelationWindow)
	var recentKeys []string

	for i := len(p.accessHistory) - 1; i >= 0; i-- {
		access := p.accessHistory[i]
		if access.Timestamp.Before(cutoff) {
			break
		}
		if access.Key != record.Key {
			recentKeys = append(recentKeys, access.Key)
		}
	}

	// Update correlations
	if len(recentKeys) > 0 {
		p.correlations[record.Key] = recentKeys
	}
}

func (p *CachePredictor) getFrequencyPredictions() []PredictionResult {
	var predictions []PredictionResult

	for key, pattern := range p.patterns {
		if pattern.Frequency > 0.1 { // At least 0.1 accesses per hour
			probability := math.Min(1.0, pattern.Frequency/10.0) * pattern.Confidence

			predictions = append(predictions, PredictionResult{
				Key:           key,
				Probability:   probability,
				Reason:        "frequency_pattern",
				Context:       p.extractContext(key),
				EstimatedSize: pattern.AverageSize,
				TTL:           1 * time.Hour,
			})
		}
	}

	return predictions
}

func (p *CachePredictor) getSequencePredictions() []PredictionResult {
	var predictions []PredictionResult

	// Get recent access sequence
	if len(p.accessHistory) < 3 {
		return predictions
	}

	recentKeys := make([]string, 3)
	for i := 0; i < 3; i++ {
		recentKeys[i] = p.accessHistory[len(p.accessHistory)-1-i].Key
	}

	// Look for matching sequence patterns
	for _, sequence := range p.sequences {
		if p.matchesSequence(recentKeys, sequence.Sequence) {
			for nextKey, count := range sequence.NextItems {
				probability := (count / float64(sequence.Support)) * sequence.Confidence

				predictions = append(predictions, PredictionResult{
					Key:         nextKey,
					Probability: probability,
					Reason:      "sequence_pattern",
					Context:     p.extractContext(nextKey),
					TTL:         30 * time.Minute,
				})
			}
		}
	}

	return predictions
}

func (p *CachePredictor) getTimePredictions() []PredictionResult {
	var predictions []PredictionResult

	currentHour := time.Now().Hour()
	timePattern, exists := p.timePatterns[currentHour]
	if !exists {
		return predictions
	}

	// Predict based on popular keys at this time
	for key, count := range timePattern.PopularKeys {
		probability := count / float64(timePattern.TotalAccesses)
		if probability > 0.1 { // At least 10% probability

			predictions = append(predictions, PredictionResult{
				Key:         key,
				Probability: probability,
				Reason:      "time_pattern",
				Context:     p.extractContext(key),
				TTL:         2 * time.Hour,
			})
		}
	}

	return predictions
}

func (p *CachePredictor) getCorrelationPredictions() []PredictionResult {
	var predictions []PredictionResult

	// Get recently accessed keys
	if len(p.accessHistory) == 0 {
		return predictions
	}

	recentKey := p.accessHistory[len(p.accessHistory)-1].Key
	correlatedKeys, exists := p.correlations[recentKey]
	if !exists {
		return predictions
	}

	// Predict correlated keys
	for _, correlatedKey := range correlatedKeys {
		probability := 0.3 // Base correlation probability

		// Boost probability if the correlated key has its own patterns
		if pattern, exists := p.patterns[correlatedKey]; exists {
			probability *= pattern.Confidence
		}

		predictions = append(predictions, PredictionResult{
			Key:         correlatedKey,
			Probability: probability,
			Reason:      "correlation",
			Context:     p.extractContext(correlatedKey),
			TTL:         1 * time.Hour,
		})
	}

	return predictions
}

func (p *CachePredictor) extractContext(key string) string {
	key = strings.ToLower(key)
	if strings.Contains(key, "price") {
		return "price"
	}
	if strings.Contains(key, "pop") || strings.Contains(key, "population") {
		return "population"
	}
	if strings.Contains(key, "card") {
		return "card"
	}
	if strings.Contains(key, "sales") {
		return "sales"
	}
	return "unknown"
}

func (p *CachePredictor) extractSetName(key string) string {
	// Try to extract set name from cache key
	// This is heuristic-based and may need adjustment based on actual key format
	parts := strings.Split(key, "_")
	if len(parts) > 1 {
		return parts[0]
	}
	return ""
}

func (p *CachePredictor) estimateSize(data interface{}) int64 {
	// Simple size estimation
	if data == nil {
		return 0
	}

	// Try to serialize to JSON to estimate size
	if jsonData, err := json.Marshal(data); err == nil {
		return int64(len(jsonData))
	}

	return 1024 // Default estimate
}

func (p *CachePredictor) matchesSequence(recent []string, pattern []string) bool {
	if len(recent) < len(pattern) {
		return false
	}

	// Check if recent keys match the end of the pattern
	for i := 0; i < len(pattern); i++ {
		if recent[i] != pattern[len(pattern)-1-i] {
			return false
		}
	}

	return true
}
