package population

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/guarzo/pkmgradegap/internal/model"
)

// TargetingEngine determines which cards are worth fetching population data for
type TargetingEngine struct {
	minRawValue     float64
	minPredictedROI float64
	rarityFilter    []string
	patternMatcher  *PatternMatcher
	historicalData  map[string]float64 // card key -> historical ROI
	alwaysFetch     []string           // patterns to always fetch
	neverFetch      []string           // patterns to never fetch
}

// TargetingConfig holds configuration for the targeting engine
type TargetingConfig struct {
	MinRawValue      float64  // Minimum raw card value to consider
	MinPredictedROI  float64  // Minimum predicted ROI to fetch population
	RarityFilter     []string // Rarities to always fetch (e.g., "Secret Rare", "Ultra Rare")
	AlwaysFetch      []string // Card name patterns to always fetch
	NeverFetch       []string // Card name patterns to never fetch
	EnableHeuristics bool     // Use card name/type heuristics
}

// PatternMatcher contains regex patterns for identifying valuable cards
type PatternMatcher struct {
	chaseCards   *regexp.Regexp
	fullArts     *regexp.Regexp
	secretRares  *regexp.Regexp
	promos       *regexp.Regexp
	firstEdition *regexp.Regexp
	shadowless   *regexp.Regexp
	japanese     *regexp.Regexp
	energyCards  *regexp.Regexp
	commons      *regexp.Regexp
}

// NewTargetingEngine creates a new targeting engine
func NewTargetingEngine(config TargetingConfig) *TargetingEngine {
	engine := &TargetingEngine{
		minRawValue:     config.MinRawValue,
		minPredictedROI: config.MinPredictedROI,
		rarityFilter:    config.RarityFilter,
		historicalData:  make(map[string]float64),
		alwaysFetch:     config.AlwaysFetch,
		neverFetch:      config.NeverFetch,
	}

	if config.EnableHeuristics {
		engine.patternMatcher = newPatternMatcher()
	}

	// Set defaults if not provided
	if engine.minRawValue == 0 {
		engine.minRawValue = 1.0 // $1 minimum
	}
	if engine.minPredictedROI == 0 {
		engine.minPredictedROI = 0.2 // 20% ROI minimum
	}
	if len(engine.rarityFilter) == 0 {
		engine.rarityFilter = []string{
			"Secret Rare", "Ultra Rare", "Special Illustration Rare",
			"Alternate Art", "Full Art", "Rainbow Rare", "Gold Rare",
		}
	}

	return engine
}

// ShouldFetchPopulation determines if population data should be fetched for a card
func (t *TargetingEngine) ShouldFetchPopulation(card model.Card) bool {
	// Always fetch patterns
	for _, pattern := range t.alwaysFetch {
		if matched, _ := regexp.MatchString(pattern, card.Name); matched {
			return true
		}
	}

	// Never fetch patterns
	for _, pattern := range t.neverFetch {
		if matched, _ := regexp.MatchString(pattern, card.Name); matched {
			return false
		}
	}

	// Quick filters first (no external calls needed)
	if !t.passesBasicFilters(card) {
		return false
	}

	// Pattern matching for valuable cards
	if t.patternMatcher != nil && t.isValuableByPattern(card) {
		return true
	}

	// Rarity-based filtering
	if t.isValuableByRarity(card) {
		return true
	}

	// Historical ROI check
	if t.hasGoodHistoricalROI(card) {
		return true
	}

	// Default: fetch for cards that might be valuable
	return t.couldBeValuable(card)
}

// passesBasicFilters applies quick filters that don't require external data
func (t *TargetingEngine) passesBasicFilters(card model.Card) bool {
	cardName := strings.ToLower(card.Name)

	// Skip basic energy cards (almost never worth grading)
	if t.isBasicEnergy(cardName) {
		return false
	}

	// Skip common trainer cards unless they're from vintage sets
	if t.isCommonTrainer(card) && !t.isVintageSet(card.SetName) {
		return false
	}

	// Skip cards with obviously low value keywords
	lowValueKeywords := []string{
		"energy", "potion", "switch", "professor", "bill",
		"pokemon center", "pokemon mart", "computer search",
	}

	for _, keyword := range lowValueKeywords {
		if strings.Contains(cardName, keyword) && !t.isVintageSet(card.SetName) {
			return false
		}
	}

	return true
}

// isValuableByPattern checks if card matches valuable patterns
func (t *TargetingEngine) isValuableByPattern(card model.Card) bool {
	cardName := strings.ToLower(card.Name)

	// Chase Pokémon (always valuable)
	if t.patternMatcher.chaseCards.MatchString(cardName) {
		return true
	}

	// Full Art cards
	if t.patternMatcher.fullArts.MatchString(cardName) {
		return true
	}

	// Secret Rares
	if t.patternMatcher.secretRares.MatchString(cardName) {
		return true
	}

	// Promo cards (can be very valuable)
	if t.patternMatcher.promos.MatchString(cardName) {
		return true
	}

	// First Edition
	if t.patternMatcher.firstEdition.MatchString(cardName) {
		return true
	}

	// Shadowless
	if t.patternMatcher.shadowless.MatchString(cardName) {
		return true
	}

	// Japanese cards (often have better centering)
	if t.patternMatcher.japanese.MatchString(cardName) {
		return true
	}

	return false
}

// isValuableByRarity checks if the card's rarity makes it worth fetching
func (t *TargetingEngine) isValuableByRarity(card model.Card) bool {
	for _, rarity := range t.rarityFilter {
		if strings.EqualFold(card.Rarity, rarity) {
			return true
		}
	}

	// Also check for rarity keywords in the card name or type
	rarityKeywords := []string{
		"ex", "gx", "v", "vmax", "vstar", "prime", "legend",
		"break", "tag team", "ultra", "secret", "rainbow",
		"gold", "full art", "alternate art", "special",
	}

	cardNameLower := strings.ToLower(card.Name)
	for _, keyword := range rarityKeywords {
		if strings.Contains(cardNameLower, keyword) {
			return true
		}
	}

	return false
}

// hasGoodHistoricalROI checks historical performance
func (t *TargetingEngine) hasGoodHistoricalROI(card model.Card) bool {
	cardKey := fmt.Sprintf("%s-%s", card.Number, card.Name)
	if roi, exists := t.historicalData[cardKey]; exists {
		return roi >= t.minPredictedROI
	}

	// If no specific data, check similar cards
	similarKey := strings.ToLower(card.Name)
	for key, roi := range t.historicalData {
		if strings.Contains(strings.ToLower(key), similarKey) {
			if roi >= t.minPredictedROI {
				return true
			}
		}
	}

	return false
}

// couldBeValuable makes a final determination for edge cases
func (t *TargetingEngine) couldBeValuable(card model.Card) bool {
	// Holofoil cards from popular sets
	if t.isHoloCard(card) && t.isPopularSet(card.SetName) {
		return true
	}

	// Number-based heuristics (first/last cards often valuable)
	if t.isSpecialNumber(card.Number) {
		return true
	}

	// Default to false for unknown cards to save API calls
	return false
}

// Helper methods for pattern matching

func (t *TargetingEngine) isBasicEnergy(cardName string) bool {
	energyTypes := []string{
		"fire energy", "water energy", "grass energy", "lightning energy",
		"psychic energy", "fighting energy", "darkness energy", "metal energy",
		"fairy energy", "dragon energy", "colorless energy",
	}

	for _, energyType := range energyTypes {
		if strings.Contains(cardName, energyType) {
			return true
		}
	}

	return false
}

func (t *TargetingEngine) isCommonTrainer(card model.Card) bool {
	if !strings.Contains(strings.ToLower(card.Name), "trainer") &&
		!strings.Contains(strings.ToLower(card.Type), "trainer") {
		return false
	}

	// These are typically common and low-value
	commonTrainers := []string{
		"pokemon center", "potion", "super potion", "full heal",
		"switch", "pokemon flute", "pokemon trader",
	}

	cardNameLower := strings.ToLower(card.Name)
	for _, trainer := range commonTrainers {
		if strings.Contains(cardNameLower, trainer) {
			return true
		}
	}

	return false
}

func (t *TargetingEngine) isVintageSet(setName string) bool {
	vintageKeywords := []string{
		"base", "jungle", "fossil", "rocket", "gym",
		"neo", "discovery", "revelation", "destiny",
		"expedition", "aquapolis", "skyridge",
	}

	setNameLower := strings.ToLower(setName)
	for _, keyword := range vintageKeywords {
		if strings.Contains(setNameLower, keyword) {
			return true
		}
	}

	// Year-based check (pre-2003 sets are generally vintage)
	yearPatterns := []string{
		"1998", "1999", "2000", "2001", "2002",
	}

	for _, year := range yearPatterns {
		if strings.Contains(setName, year) {
			return true
		}
	}

	return false
}

func (t *TargetingEngine) isHoloCard(card model.Card) bool {
	holoKeywords := []string{
		"holo", "holographic", "holofoil", "shiny",
	}

	cardNameLower := strings.ToLower(card.Name)
	for _, keyword := range holoKeywords {
		if strings.Contains(cardNameLower, keyword) {
			return true
		}
	}

	return false
}

func (t *TargetingEngine) isPopularSet(setName string) bool {
	popularSets := []string{
		"base set", "jungle", "fossil", "shadowless",
		"charizard", "pikachu", "evolving skies",
		"brilliant stars", "astral radiance", "lost origin",
		"silver tempest", "crown zenith", "paldea evolved",
		"obsidian flames", "paradox rift", "paldean fates",
		"surging sparks",
	}

	setNameLower := strings.ToLower(setName)
	for _, popular := range popularSets {
		if strings.Contains(setNameLower, popular) {
			return true
		}
	}

	return false
}

func (t *TargetingEngine) isSpecialNumber(number string) bool {
	// First cards in set (often starters or popular Pokémon)
	if number == "1" || number == "001" {
		return true
	}

	// Last cards are often secret rares
	if strings.Contains(number, "/") {
		// Parse number/total format
		parts := strings.Split(number, "/")
		if len(parts) == 2 && parts[0] == parts[1] {
			return true // Last card in set
		}
	}

	return false
}

func (t *TargetingEngine) UpdateHistoricalData(cardKey string, roi float64) {
	t.historicalData[cardKey] = roi
}

func (t *TargetingEngine) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"min_raw_value":     t.minRawValue,
		"min_predicted_roi": t.minPredictedROI,
		"historical_cards":  len(t.historicalData),
		"rarity_filters":    t.rarityFilter,
		"always_fetch":      len(t.alwaysFetch),
		"never_fetch":       len(t.neverFetch),
	}
}

// newPatternMatcher creates regex patterns for identifying valuable cards
func newPatternMatcher() *PatternMatcher {
	return &PatternMatcher{
		chaseCards:   regexp.MustCompile(`(?i)(charizard|pikachu|lugia|mewtwo|mew|rayquaza|arceus|dialga|palkia|giratina|reshiram|zekrom|kyurem|xerneas|yveltal|zygarde|solgaleo|lunala|necrozma|zacian|zamazenta|eternatus|calyrex|koraidon|miraidon|eevee|umbreon|espeon|vaporeon|jolteon|flareon|leafeon|glaceon|sylveon)`),
		fullArts:     regexp.MustCompile(`(?i)(full art|fa|alternate art|alt art|special illustration|sir)`),
		secretRares:  regexp.MustCompile(`(?i)(secret rare|rainbow rare|gold rare|hyper rare|sr|hr)`),
		promos:       regexp.MustCompile(`(?i)(promo|promotional|staff|winner|champion|tournament|world|nationals)`),
		firstEdition: regexp.MustCompile(`(?i)(1st edition|first edition)`),
		shadowless:   regexp.MustCompile(`(?i)(shadowless)`),
		japanese:     regexp.MustCompile(`(?i)(japanese|jp|japan)`),
		energyCards:  regexp.MustCompile(`(?i)(energy)$`),
		commons:      regexp.MustCompile(`(?i)(common|trainer|supporter|item|stadium)$`),
	}
}

// BatchTargeting applies targeting to multiple cards efficiently
type BatchTargeting struct {
	engine  *TargetingEngine
	batches [][]model.Card
}

// NewBatchTargeting creates a new batch targeting system
func NewBatchTargeting(engine *TargetingEngine, batchSize int) *BatchTargeting {
	return &BatchTargeting{
		engine: engine,
	}
}

// ProcessCards processes multiple cards and returns those worth fetching
func (b *BatchTargeting) ProcessCards(cards []model.Card) []model.Card {
	var worthFetching []model.Card

	for _, card := range cards {
		if b.engine.ShouldFetchPopulation(card) {
			worthFetching = append(worthFetching, card)
		}
	}

	return worthFetching
}

// GetTargetingReport generates a report of targeting decisions
func (b *BatchTargeting) GetTargetingReport(cards []model.Card) *TargetingReport {
	report := &TargetingReport{
		TotalCards:  len(cards),
		ProcessedAt: time.Now(),
		ByReason:    make(map[string]int),
	}

	for _, card := range cards {
		reason := b.getTargetingReason(card)
		report.ByReason[reason]++

		if reason != "skipped" {
			report.TargetedCards++
		}
	}

	report.SkippedCards = report.TotalCards - report.TargetedCards
	if report.TotalCards > 0 {
		report.TargetingRate = float64(report.TargetedCards) / float64(report.TotalCards)
	}

	return report
}

func (b *BatchTargeting) getTargetingReason(card model.Card) string {
	// Check in order of priority
	for _, pattern := range b.engine.alwaysFetch {
		if matched, _ := regexp.MatchString(pattern, card.Name); matched {
			return "always_fetch_pattern"
		}
	}

	for _, pattern := range b.engine.neverFetch {
		if matched, _ := regexp.MatchString(pattern, card.Name); matched {
			return "skipped"
		}
	}

	if !b.engine.passesBasicFilters(card) {
		return "skipped"
	}

	if b.engine.isValuableByRarity(card) {
		return "valuable_rarity"
	}

	if b.engine.patternMatcher != nil && b.engine.isValuableByPattern(card) {
		return "valuable_pattern"
	}

	if b.engine.hasGoodHistoricalROI(card) {
		return "good_historical_roi"
	}

	if b.engine.couldBeValuable(card) {
		return "potentially_valuable"
	}

	return "skipped"
}

// TargetingReport contains statistics about targeting decisions
type TargetingReport struct {
	TotalCards    int            `json:"total_cards"`
	TargetedCards int            `json:"targeted_cards"`
	SkippedCards  int            `json:"skipped_cards"`
	TargetingRate float64        `json:"targeting_rate"`
	ProcessedAt   time.Time      `json:"processed_at"`
	ByReason      map[string]int `json:"by_reason"`
}
