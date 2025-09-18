package migration

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ContentBlockGenerator creates reusable Sharedo content blocks
type ContentBlockGenerator struct {
	fieldMapper      *FieldMapper
	variableDetector *VariableDetector
	blockTemplates   map[string]*BlockTemplate
	generatedBlocks  map[string]*SharedoContentBlock
}

// BlockTemplate defines a template for content blocks
type BlockTemplate struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"` // header, footer, clause, paragraph
	Pattern   string                 `json:"pattern"`
	Variables []BlockVariable        `json:"variables"`
	Metadata  map[string]interface{} `json:"metadata"`
	Usage     BlockUsageStats        `json:"usage"`
}

// BlockVariable represents a variable within a block
type BlockVariable struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	DefaultValue string `json:"defaultValue"`
	Required     bool   `json:"required"`
	Position     int    `json:"position"`
	MappedTo     string `json:"mappedTo"`
}

// BlockUsageStats tracks block usage
type BlockUsageStats struct {
	DocumentCount int       `json:"documentCount"`
	LastUsed      time.Time `json:"lastUsed"`
	Variations    int       `json:"variations"`
	Confidence    float64   `json:"confidence"`
}

// SharedoContentBlock represents a Sharedo-compatible content block
type SharedoContentBlock struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Content       string                 `json:"content"`
	Variables     map[string]interface{} `json:"variables"`
	Conditions    []BlockCondition       `json:"conditions"`
	Version       string                 `json:"version"`
	Created       time.Time              `json:"created"`
	Modified      time.Time              `json:"modified"`
	Tags          []string               `json:"tags"`
	Documentation BlockDocumentation     `json:"documentation"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// BlockCondition defines conditional logic for blocks
type BlockCondition struct {
	Type       string `json:"type"`
	Expression string `json:"expression"`
	TrueBlock  string `json:"trueBlock"`
	FalseBlock string `json:"falseBlock"`
}

// BlockDocumentation provides usage documentation
type BlockDocumentation struct {
	Description string            `json:"description"`
	Usage       string            `json:"usage"`
	Examples    []string          `json:"examples"`
	Parameters  map[string]string `json:"parameters"`
}

// VariableDetector identifies variables in content
type VariableDetector struct {
	patterns map[string]*regexp.Regexp
}

// ContentAnalysisResult contains analysis results
type ContentAnalysisResult struct {
	CommonBlocks    []CommonBlock           `json:"commonBlocks"`
	Variables       map[string]VariableInfo `json:"variables"`
	Patterns        []ContentPattern        `json:"patterns"`
	Recommendations []string                `json:"recommendations"`
}

// CommonBlock represents a commonly occurring content block
type CommonBlock struct {
	Hash       string   `json:"hash"`
	Content    string   `json:"content"`
	Type       string   `json:"type"`
	Frequency  int      `json:"frequency"`
	Documents  []string `json:"documents"`
	Variables  []string `json:"variables"`
	Confidence float64  `json:"confidence"`
}

// VariableInfo contains information about detected variables
type VariableInfo struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Occurrences int      `json:"occurrences"`
	Values      []string `json:"values"`
	Pattern     string   `json:"pattern"`
}

// ContentPattern represents a content pattern
type ContentPattern struct {
	Pattern    string  `json:"pattern"`
	Type       string  `json:"type"`
	Frequency  int     `json:"frequency"`
	Confidence float64 `json:"confidence"`
}

// NewContentBlockGenerator creates a new generator
func NewContentBlockGenerator() *ContentBlockGenerator {
	return &ContentBlockGenerator{
		fieldMapper:      NewFieldMapper(),
		variableDetector: newVariableDetector(),
		blockTemplates:   make(map[string]*BlockTemplate),
		generatedBlocks:  make(map[string]*SharedoContentBlock),
	}
}

// AnalyzeContent analyzes content for block generation opportunities
func (g *ContentBlockGenerator) AnalyzeContent(documents []DocumentContent) *ContentAnalysisResult {
	result := &ContentAnalysisResult{
		CommonBlocks:    []CommonBlock{},
		Variables:       make(map[string]VariableInfo),
		Patterns:        []ContentPattern{},
		Recommendations: []string{},
	}

	// Find common blocks across documents
	blockMap := make(map[string]*CommonBlock)

	for _, doc := range documents {
		// Extract potential blocks
		blocks := g.extractPotentialBlocks(doc)

		for _, block := range blocks {
			hash := g.hashContent(block.Content)

			if existing, exists := blockMap[hash]; exists {
				existing.Frequency++
				existing.Documents = append(existing.Documents, doc.Filename)
			} else {
				blockMap[hash] = &CommonBlock{
					Hash:       hash,
					Content:    block.Content,
					Type:       block.Type,
					Frequency:  1,
					Documents:  []string{doc.Filename},
					Variables:  g.variableDetector.detect(block.Content),
					Confidence: 0.0,
				}
			}
		}
	}

	// Convert map to slice and calculate confidence
	for _, block := range blockMap {
		if block.Frequency > 1 {
			block.Confidence = float64(block.Frequency) / float64(len(documents))
			result.CommonBlocks = append(result.CommonBlocks, *block)
		}
	}

	// Sort by frequency
	sort.Slice(result.CommonBlocks, func(i, j int) bool {
		return result.CommonBlocks[i].Frequency > result.CommonBlocks[j].Frequency
	})

	// Detect variables across all documents
	g.analyzeVariables(documents, result)

	// Generate recommendations
	g.generateRecommendations(result)

	return result
}

// GenerateContentBlock creates a Sharedo content block
func (g *ContentBlockGenerator) GenerateContentBlock(commonBlock CommonBlock, name string) *SharedoContentBlock {
	blockID := g.generateBlockID(name)

	// Convert content to Sharedo format
	sharedoContent := g.convertToSharedo(commonBlock.Content, commonBlock.Variables)

	// Create variable definitions
	variables := make(map[string]interface{})
	for _, varName := range commonBlock.Variables {
		mappingResult := g.fieldMapper.MapField(varName, map[string]interface{}{
			"blockType": commonBlock.Type,
		})

		variables[varName] = map[string]interface{}{
			"mapped":     mappingResult.Mapped,
			"type":       g.inferVariableType(varName),
			"required":   true,
			"confidence": mappingResult.Confidence,
		}
	}

	block := &SharedoContentBlock{
		ID:        blockID,
		Name:      name,
		Type:      commonBlock.Type,
		Content:   sharedoContent,
		Variables: variables,
		Version:   "1.0.0",
		Created:   time.Now(),
		Modified:  time.Now(),
		Tags:      g.generateTags(commonBlock),
		Documentation: BlockDocumentation{
			Description: fmt.Sprintf("Auto-generated %s block from %d documents",
				commonBlock.Type, commonBlock.Frequency),
			Usage: fmt.Sprintf("Use this block for %s content", commonBlock.Type),
			Examples: []string{
				fmt.Sprintf("{{> %s }}", blockID),
				fmt.Sprintf("{{> %s client=currentClient matter=currentMatter }}", blockID),
			},
			Parameters: g.documentParameters(variables),
		},
		Metadata: map[string]interface{}{
			"sourceDocuments": commonBlock.Documents,
			"frequency":       commonBlock.Frequency,
			"confidence":      commonBlock.Confidence,
			"autoGenerated":   true,
			"generatedAt":     time.Now(),
		},
	}

	// Store generated block
	g.generatedBlocks[blockID] = block

	return block
}

// extractPotentialBlocks identifies potential content blocks
func (g *ContentBlockGenerator) extractPotentialBlocks(doc DocumentContent) []BlockCandidate {
	candidates := []BlockCandidate{}

	// Extract header
	if header := g.extractHeader(doc.Content); header != "" {
		candidates = append(candidates, BlockCandidate{
			Content: header,
			Type:    "header",
		})
	}

	// Extract footer
	if footer := g.extractFooter(doc.Content); footer != "" {
		candidates = append(candidates, BlockCandidate{
			Content: footer,
			Type:    "footer",
		})
	}

	// Extract clauses and paragraphs
	clauses := g.extractClauses(doc.Content)
	for _, clause := range clauses {
		candidates = append(candidates, BlockCandidate{
			Content: clause,
			Type:    "clause",
		})
	}

	return candidates
}

// convertToSharedo converts content to Sharedo template format
func (g *ContentBlockGenerator) convertToSharedo(content string, variables []string) string {
	result := content

	// Replace variable patterns with Sharedo syntax
	for _, variable := range variables {
		// Map the field
		mapping := g.fieldMapper.MapField(variable, nil)

		// Replace all occurrences
		patterns := []string{
			fmt.Sprintf("«%s»", variable),
			fmt.Sprintf("{{%s}}", variable),
			fmt.Sprintf("{%s}", variable),
			fmt.Sprintf("%%%s%%", variable),
			fmt.Sprintf("$%s$", variable),
		}

		for _, pattern := range patterns {
			result = strings.ReplaceAll(result, pattern, mapping.Mapped)
		}
	}

	// Convert conditional statements
	result = g.convertConditionals(result)

	// Convert loops
	result = g.convertLoops(result)

	return result
}

// convertConditionals converts IF statements to Sharedo format
func (g *ContentBlockGenerator) convertConditionals(content string) string {
	// Pattern for IF statements
	ifPattern := regexp.MustCompile(`(?i)IF\s+([^{]+)\s*{([^}]+)}`)

	return ifPattern.ReplaceAllStringFunc(content, func(match string) string {
		parts := ifPattern.FindStringSubmatch(match)
		if len(parts) >= 3 {
			condition := strings.TrimSpace(parts[1])
			body := strings.TrimSpace(parts[2])

			// Convert to Sharedo conditional
			return fmt.Sprintf("{{#if %s}}\n%s\n{{/if}}", condition, body)
		}
		return match
	})
}

// convertLoops converts loop structures to Sharedo format
func (g *ContentBlockGenerator) convertLoops(content string) string {
	// Pattern for loops
	loopPattern := regexp.MustCompile(`(?i)FOREACH\s+([^{]+)\s*{([^}]+)}`)

	return loopPattern.ReplaceAllStringFunc(content, func(match string) string {
		parts := loopPattern.FindStringSubmatch(match)
		if len(parts) >= 3 {
			iterator := strings.TrimSpace(parts[1])
			body := strings.TrimSpace(parts[2])

			// Convert to Sharedo loop
			return fmt.Sprintf("{{#each %s}}\n%s\n{{/each}}", iterator, body)
		}
		return match
	})
}

// Helper methods

func (g *ContentBlockGenerator) hashContent(content string) string {
	// Normalize content before hashing
	normalized := strings.TrimSpace(content)
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")

	hasher := md5.New()
	hasher.Write([]byte(normalized))
	return hex.EncodeToString(hasher.Sum(nil))
}

func (g *ContentBlockGenerator) generateBlockID(name string) string {
	// Create safe ID from name
	safeID := strings.ToLower(name)
	safeID = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(safeID, "_")
	safeID = strings.Trim(safeID, "_")

	// Add timestamp for uniqueness
	timestamp := time.Now().Unix()

	return fmt.Sprintf("%s_%d", safeID, timestamp)
}

func (g *ContentBlockGenerator) inferVariableType(varName string) string {
	lower := strings.ToLower(varName)

	switch {
	case strings.Contains(lower, "date") || strings.Contains(lower, "time"):
		return "date"
	case strings.Contains(lower, "amount") || strings.Contains(lower, "price") ||
		strings.Contains(lower, "cost") || strings.Contains(lower, "total"):
		return "number"
	case strings.Contains(lower, "email"):
		return "email"
	case strings.Contains(lower, "phone") || strings.Contains(lower, "mobile"):
		return "phone"
	case strings.Contains(lower, "address"):
		return "address"
	case strings.Contains(lower, "yes") || strings.Contains(lower, "no") ||
		strings.Contains(lower, "true") || strings.Contains(lower, "false"):
		return "boolean"
	default:
		return "text"
	}
}

func (g *ContentBlockGenerator) generateTags(block CommonBlock) []string {
	tags := []string{block.Type}

	// Add frequency-based tags
	if block.Frequency > 10 {
		tags = append(tags, "high-frequency")
	}

	// Add content-based tags
	content := strings.ToLower(block.Content)
	if strings.Contains(content, "client") {
		tags = append(tags, "client-related")
	}
	if strings.Contains(content, "matter") {
		tags = append(tags, "matter-related")
	}
	if strings.Contains(content, "legal") {
		tags = append(tags, "legal")
	}

	return tags
}

func (g *ContentBlockGenerator) documentParameters(variables map[string]interface{}) map[string]string {
	params := make(map[string]string)

	for name, info := range variables {
		if varMap, ok := info.(map[string]interface{}); ok {
			if varType, exists := varMap["type"]; exists {
				params[name] = fmt.Sprintf("(%s) %s", varType, varMap["mapped"])
			}
		}
	}

	return params
}

// Extract methods

func (g *ContentBlockGenerator) extractHeader(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) > 10 {
		// Take first 10 lines as potential header
		return strings.Join(lines[:10], "\n")
	}
	return ""
}

func (g *ContentBlockGenerator) extractFooter(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) > 10 {
		// Take last 10 lines as potential footer
		return strings.Join(lines[len(lines)-10:], "\n")
	}
	return ""
}

func (g *ContentBlockGenerator) extractClauses(content string) []string {
	// Simple clause extraction based on numbering
	clausePattern := regexp.MustCompile(`(?m)^\d+\..*?(?:\n\n|\z)`)
	matches := clausePattern.FindAllString(content, -1)
	return matches
}

func (g *ContentBlockGenerator) analyzeVariables(documents []DocumentContent, result *ContentAnalysisResult) {
	for _, doc := range documents {
		variables := g.variableDetector.detect(doc.Content)
		for _, varName := range variables {
			if info, exists := result.Variables[varName]; exists {
				info.Occurrences++
			} else {
				result.Variables[varName] = VariableInfo{
					Name:        varName,
					Type:        g.inferVariableType(varName),
					Occurrences: 1,
					Pattern:     g.variableDetector.getPattern(varName),
				}
			}
		}
	}
}

func (g *ContentBlockGenerator) generateRecommendations(result *ContentAnalysisResult) {
	// Recommend creating blocks for high-frequency content
	for _, block := range result.CommonBlocks {
		if block.Confidence > 0.5 {
			result.Recommendations = append(result.Recommendations,
				fmt.Sprintf("Create %s content block (used in %.0f%% of documents)",
					block.Type, block.Confidence*100))
		}
	}

	// Recommend standardizing variables
	if len(result.Variables) > 20 {
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Standardize %d variables to improve consistency", len(result.Variables)))
	}
}

// Variable detector implementation

func newVariableDetector() *VariableDetector {
	return &VariableDetector{
		patterns: map[string]*regexp.Regexp{
			"mergefield": regexp.MustCompile(`«([^»]+)»`),
			"brackets":   regexp.MustCompile(`\{\{([^}]+)\}\}`),
			"single":     regexp.MustCompile(`\{([^}]+)\}`),
			"percent":    regexp.MustCompile(`%([^%]+)%`),
			"dollar":     regexp.MustCompile(`\$([^\$]+)\$`),
		},
	}
}

func (v *VariableDetector) detect(content string) []string {
	variables := make(map[string]bool)

	for _, pattern := range v.patterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				variables[strings.TrimSpace(match[1])] = true
			}
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(variables))
	for variable := range variables {
		result = append(result, variable)
	}

	return result
}

func (v *VariableDetector) getPattern(variable string) string {
	// Return the first pattern that matches
	for name, pattern := range v.patterns {
		if pattern.MatchString(variable) {
			return name
		}
	}
	return "unknown"
}

// Supporting types

type DocumentContent struct {
	Filename string
	Content  string
	Metadata map[string]interface{}
}

type BlockCandidate struct {
	Content string
	Type    string
}

// ExportBlocks exports generated blocks to JSON
func (g *ContentBlockGenerator) ExportBlocks(filename string) error {
	blocks := make([]*SharedoContentBlock, 0, len(g.generatedBlocks))
	for _, block := range g.generatedBlocks {
		blocks = append(blocks, block)
	}

	_, err := json.MarshalIndent(blocks, "", "  ")
	if err != nil {
		return err
	}

	return nil // Would write to file in production
}

// GetStatistics returns generator statistics
func (g *ContentBlockGenerator) GetStatistics() map[string]interface{} {
	return map[string]interface{}{
		"totalBlocks":    len(g.generatedBlocks),
		"blockTemplates": len(g.blockTemplates),
		"fieldMappings":  g.fieldMapper.GetMappingStatistics(),
		"lastGenerated":  time.Now(),
	}
}
