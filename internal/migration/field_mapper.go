package migration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FieldMapper provides intelligent field mapping between legacy and Sharedo formats
type FieldMapper struct {
	mappingRules        map[string]MappingRule
	learningCache       map[string]LearnedMapping
	confidenceThreshold float64
	mu                  sync.RWMutex
}

// MappingRule defines how to transform a legacy field to Sharedo format
type MappingRule struct {
	LegacyPattern   string                 `json:"legacyPattern"`
	SharedoTemplate string                 `json:"sharedoTemplate"`
	TransformFunc   string                 `json:"transformFunc"`
	Confidence      float64                `json:"confidence"`
	Examples        []MappingExample       `json:"examples"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// MappingExample provides concrete examples for learning
type MappingExample struct {
	Input    string    `json:"input"`
	Output   string    `json:"output"`
	Context  string    `json:"context"`
	Verified bool      `json:"verified"`
	Created  time.Time `json:"created"`
}

// LearnedMapping represents mappings learned from user corrections
type LearnedMapping struct {
	Pattern     string    `json:"pattern"`
	Mapping     string    `json:"mapping"`
	Occurrences int       `json:"occurrences"`
	LastUsed    time.Time `json:"lastUsed"`
	Confidence  float64   `json:"confidence"`
}

// FieldMappingResult contains the mapping result with metadata
type FieldMappingResult struct {
	Original        string                 `json:"original"`
	Mapped          string                 `json:"mapped"`
	Confidence      float64                `json:"confidence"`
	MappingType     string                 `json:"mappingType"`
	Alternatives    []AlternativeMapping   `json:"alternatives"`
	RequiresReview  bool                   `json:"requiresReview"`
	Transformations []string               `json:"transformations"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// AlternativeMapping provides alternative mapping suggestions
type AlternativeMapping struct {
	Suggestion string  `json:"suggestion"`
	Confidence float64 `json:"confidence"`
	Rationale  string  `json:"rationale"`
}

// NewFieldMapper creates an intelligent field mapper
func NewFieldMapper() *FieldMapper {
	mapper := &FieldMapper{
		mappingRules:        initializeMappingRules(),
		learningCache:       make(map[string]LearnedMapping),
		confidenceThreshold: 0.75,
	}

	// Load learned mappings if available
	mapper.loadLearnedMappings()

	return mapper
}

// MapField intelligently maps a legacy field to Sharedo format
func (fm *FieldMapper) MapField(legacyField string, context map[string]interface{}) *FieldMappingResult {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	result := &FieldMappingResult{
		Original:        legacyField,
		Alternatives:    []AlternativeMapping{},
		Transformations: []string{},
		Metadata:        make(map[string]interface{}),
	}

	// Check learned mappings first (highest priority)
	if learned, exists := fm.learningCache[legacyField]; exists {
		result.Mapped = learned.Mapping
		result.Confidence = learned.Confidence
		result.MappingType = "learned"

		// Update usage stats
		learned.Occurrences++
		learned.LastUsed = time.Now()
		fm.learningCache[legacyField] = learned

		if result.Confidence >= fm.confidenceThreshold {
			return result
		}
	}

	// Apply rule-based mapping
	bestMatch := fm.findBestRuleMatch(legacyField, context)
	if bestMatch != nil {
		result.Mapped = fm.applyMapping(legacyField, bestMatch)
		result.Confidence = bestMatch.Confidence
		result.MappingType = "rule-based"
		result.Transformations = append(result.Transformations, bestMatch.TransformFunc)
	}

	// Generate alternatives using different strategies
	result.Alternatives = fm.generateAlternatives(legacyField, context)

	// Determine if review is needed
	result.RequiresReview = result.Confidence < fm.confidenceThreshold

	// Add metadata
	result.Metadata["processedAt"] = time.Now()
	result.Metadata["mappingVersion"] = "2.1.0"

	return result
}

// LearnFromCorrection updates mapping knowledge from user corrections
func (fm *FieldMapper) LearnFromCorrection(original, corrected string, confidence float64) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if existing, exists := fm.learningCache[original]; exists {
		// Update existing learned mapping
		existing.Mapping = corrected
		existing.Confidence = (existing.Confidence + confidence) / 2
		existing.Occurrences++
		existing.LastUsed = time.Now()
		fm.learningCache[original] = existing
	} else {
		// Create new learned mapping
		fm.learningCache[original] = LearnedMapping{
			Pattern:     original,
			Mapping:     corrected,
			Occurrences: 1,
			LastUsed:    time.Now(),
			Confidence:  confidence,
		}
	}

	// Persist learned mappings
	fm.saveLearnedMappings()
}

// findBestRuleMatch finds the best matching rule for a field
func (fm *FieldMapper) findBestRuleMatch(field string, context map[string]interface{}) *MappingRule {
	var bestMatch *MappingRule
	highestConfidence := 0.0

	fieldLower := strings.ToLower(field)

	for pattern, rule := range fm.mappingRules {
		if matches := fm.patternMatches(fieldLower, pattern); matches {
			// Adjust confidence based on context
			adjustedConfidence := fm.adjustConfidenceByContext(rule.Confidence, context)

			if adjustedConfidence > highestConfidence {
				highestConfidence = adjustedConfidence
				bestMatch = &rule
			}
		}
	}

	return bestMatch
}

// patternMatches checks if a field matches a pattern
func (fm *FieldMapper) patternMatches(field, pattern string) bool {
	// Simple pattern matching - can be enhanced with regex
	return strings.Contains(field, pattern) ||
		strings.HasPrefix(field, pattern) ||
		strings.HasSuffix(field, pattern)
}

// adjustConfidenceByContext adjusts confidence based on context
func (fm *FieldMapper) adjustConfidenceByContext(baseConfidence float64, context map[string]interface{}) float64 {
	adjusted := baseConfidence

	// Boost confidence if context matches expectations
	if docType, exists := context["documentType"]; exists {
		if docType == "legal" {
			adjusted *= 1.1
		}
	}

	if jurisdiction, exists := context["jurisdiction"]; exists {
		if jurisdiction != "" {
			adjusted *= 1.05
		}
	}

	// Cap at 1.0
	if adjusted > 1.0 {
		adjusted = 1.0
	}

	return adjusted
}

// applyMapping applies a mapping rule to transform the field
func (fm *FieldMapper) applyMapping(field string, rule *MappingRule) string {
	// Apply transformation based on template
	mapped := rule.SharedoTemplate

	// Replace placeholders
	mapped = strings.ReplaceAll(mapped, "{field}", field)
	mapped = strings.ReplaceAll(mapped, "{camelCase}", fm.toCamelCase(field))
	mapped = strings.ReplaceAll(mapped, "{snakeCase}", fm.toSnakeCase(field))

	return mapped
}

// generateAlternatives generates alternative mapping suggestions
func (fm *FieldMapper) generateAlternatives(field string, context map[string]interface{}) []AlternativeMapping {
	alternatives := []AlternativeMapping{}

	// Strategy 1: Direct template variable
	alternatives = append(alternatives, AlternativeMapping{
		Suggestion: fmt.Sprintf("{{%s}}", fm.toCamelCase(field)),
		Confidence: 0.6,
		Rationale:  "Direct field mapping",
	})

	// Strategy 2: Categorized field
	category := fm.inferCategory(field)
	if category != "" {
		alternatives = append(alternatives, AlternativeMapping{
			Suggestion: fmt.Sprintf("{{%s.%s}}", category, fm.toCamelCase(field)),
			Confidence: 0.7,
			Rationale:  fmt.Sprintf("Categorized under %s", category),
		})
	}

	// Strategy 3: Context-aware mapping
	if docType, exists := context["documentType"]; exists {
		alternatives = append(alternatives, AlternativeMapping{
			Suggestion: fmt.Sprintf("{{%s.%s}}", docType, fm.toCamelCase(field)),
			Confidence: 0.65,
			Rationale:  fmt.Sprintf("Document type specific: %s", docType),
		})
	}

	return alternatives
}

// inferCategory attempts to infer field category
func (fm *FieldMapper) inferCategory(field string) string {
	fieldLower := strings.ToLower(field)

	categories := map[string][]string{
		"client":   {"name", "firstname", "lastname", "email", "phone", "address"},
		"matter":   {"number", "reference", "type", "status", "date"},
		"document": {"title", "date", "author", "version", "template"},
		"finance":  {"amount", "total", "cost", "price", "fee", "payment"},
		"date":     {"date", "time", "deadline", "created", "modified"},
	}

	for category, keywords := range categories {
		for _, keyword := range keywords {
			if strings.Contains(fieldLower, keyword) {
				return category
			}
		}
	}

	return ""
}

// Helper functions for case conversion
func (fm *FieldMapper) toCamelCase(s string) string {
	words := strings.Fields(strings.ReplaceAll(s, "_", " "))
	if len(words) == 0 {
		return ""
	}

	result := strings.ToLower(words[0])
	for i := 1; i < len(words); i++ {
		if len(words[i]) > 0 {
			result += strings.ToUpper(string(words[i][0])) + strings.ToLower(words[i][1:])
		}
	}

	return result
}

func (fm *FieldMapper) toSnakeCase(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, " ", "_"))
}

// Persistence methods
func (fm *FieldMapper) loadLearnedMappings() {
	dataFile := filepath.Join("data", "learned_mappings.json")
	if data, err := os.ReadFile(dataFile); err == nil {
		json.Unmarshal(data, &fm.learningCache)
	}
}

func (fm *FieldMapper) saveLearnedMappings() {
	dataDir := "data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return
	}

	dataFile := filepath.Join(dataDir, "learned_mappings.json")
	if data, err := json.MarshalIndent(fm.learningCache, "", "  "); err == nil {
		os.WriteFile(dataFile, data, 0644)
	}
}

// initializeMappingRules creates default mapping rules
func initializeMappingRules() map[string]MappingRule {
	return map[string]MappingRule{
		"clientname": {
			LegacyPattern:   "clientname",
			SharedoTemplate: "{{client.fullName}}",
			TransformFunc:   "direct",
			Confidence:      0.95,
		},
		"firstname": {
			LegacyPattern:   "firstname",
			SharedoTemplate: "{{client.firstName}}",
			TransformFunc:   "direct",
			Confidence:      0.95,
		},
		"lastname": {
			LegacyPattern:   "lastname",
			SharedoTemplate: "{{client.lastName}}",
			TransformFunc:   "direct",
			Confidence:      0.95,
		},
		"matter": {
			LegacyPattern:   "matter",
			SharedoTemplate: "{{matter.reference}}",
			TransformFunc:   "direct",
			Confidence:      0.90,
		},
		"date": {
			LegacyPattern:   "date",
			SharedoTemplate: "{{document.date | date: 'DD/MM/YYYY'}}",
			TransformFunc:   "dateFormat",
			Confidence:      0.85,
		},
		"amount": {
			LegacyPattern:   "amount",
			SharedoTemplate: "{{finance.amount | currency}}",
			TransformFunc:   "currencyFormat",
			Confidence:      0.85,
		},
		"address": {
			LegacyPattern:   "address",
			SharedoTemplate: "{{client.address.full}}",
			TransformFunc:   "addressFormat",
			Confidence:      0.80,
		},
		"email": {
			LegacyPattern:   "email",
			SharedoTemplate: "{{client.email}}",
			TransformFunc:   "direct",
			Confidence:      0.95,
		},
		"phone": {
			LegacyPattern:   "phone",
			SharedoTemplate: "{{client.phone}}",
			TransformFunc:   "phoneFormat",
			Confidence:      0.90,
		},
		"jurisdiction": {
			LegacyPattern:   "jurisdiction",
			SharedoTemplate: "{{matter.jurisdiction}}",
			TransformFunc:   "direct",
			Confidence:      0.85,
		},
	}
}

// BatchMapFields maps multiple fields efficiently
func (fm *FieldMapper) BatchMapFields(fields []string, context map[string]interface{}) map[string]*FieldMappingResult {
	results := make(map[string]*FieldMappingResult)

	for _, field := range fields {
		results[field] = fm.MapField(field, context)
	}

	return results
}

// GetMappingStatistics returns statistics about mappings
func (fm *FieldMapper) GetMappingStatistics() map[string]interface{} {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	stats := map[string]interface{}{
		"totalRules":          len(fm.mappingRules),
		"learnedMappings":     len(fm.learningCache),
		"confidenceThreshold": fm.confidenceThreshold,
		"avgConfidence":       fm.calculateAverageConfidence(),
	}

	return stats
}

func (fm *FieldMapper) calculateAverageConfidence() float64 {
	if len(fm.learningCache) == 0 {
		return 0.0
	}

	total := 0.0
	for _, mapping := range fm.learningCache {
		total += mapping.Confidence
	}

	return total / float64(len(fm.learningCache))
}
