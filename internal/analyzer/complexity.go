// Package analyzer provides comprehensive complexity analysis for Microsoft Word DOT template files.
// It detects nested conditionals, merge fields, formulas, macros, and other complexity indicators
// that may affect the conversion process from DOT to DOCX format.
//
// Example usage:
//
//	content, _ := os.ReadFile("template.dot")
//	report := analyzer.AnalyzeComplexity(content)
//	if report.NeedsReview {
//	    log.Printf("Document requires review: %s", report.Level)
//	}
package analyzer

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Constants for complexity thresholds - CALIBRATED BASED ON TESTING
const (
	// IF nesting thresholds
	NestedIfHighThreshold   = 3
	NestedIfMediumThreshold = 1

	// IF count thresholds
	IfCountHighThreshold = 10

	// Merge field thresholds
	MergeFieldHighCount = 15

	// Complexity score levels - ADJUSTED FOR BETTER DISTRIBUTION
	ComplexityScoreCritical = 120
	ComplexityScoreHigh     = 60
	ComplexityScoreMedium   = 30

	// Score weights for different features - CALIBRATED
	NestedIfHighWeight      = 15
	NestedIfMediumWeight    = 8
	MultipleIfWeight        = 3
	ComplexMergeFieldWeight = 6
	MacroDetectionWeight    = 40
	FormulaWeight           = 5
	NestedTableWeight       = 20
	MultipleTableWeight     = 8
	ActiveXControlWeight    = 35

	// Validation constants
	MinFormulaLength     = 10
	MaxNonPrintableRatio = 0.3

	// Default collection limits
	DefaultMaxStoredFormulas   = 10
	DefaultMaxStoredFieldCodes = 20
)

// PatternRegistry encapsulates all regex patterns used for analysis
type PatternRegistry struct {
	// IF statement patterns
	IFFieldStart *regexp.Regexp
	IFFieldFull  *regexp.Regexp

	// Merge field patterns
	MergeFields       []*regexp.Regexp
	ComplexMergeField *regexp.Regexp

	// Formula patterns
	Formulas []*regexp.Regexp

	// Macro patterns
	Macros []*regexp.Regexp

	// Table patterns
	Table       *regexp.Regexp
	NestedTable *regexp.Regexp

	// ActiveX patterns
	ActiveX []*regexp.Regexp

	// Field code patterns
	FieldCodes []*regexp.Regexp
}

// NewPatternRegistry creates and initializes all regex patterns
func NewPatternRegistry() *PatternRegistry {
	return &PatternRegistry{
		// IF statement patterns
		IFFieldStart: regexp.MustCompile(`(?i)\{[\s]*IF\b`),
		IFFieldFull:  regexp.MustCompile(`(?i)\{[\s]*IF\b[^}]*\}`),

		// Merge field patterns for DOT files
		MergeFields: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\{[\s]*MERGEFIELD\s+([^}]+)\}`),
			regexp.MustCompile(`(?i)«([^»]+)»`), // Alternative merge field format
			regexp.MustCompile(`(?i)\{[\s]*DOCVARIABLE\s+([^}]+)\}`),
			regexp.MustCompile(`(?i)\{[\s]*DOCPROPERTY\s+([^}]+)\}`),
			regexp.MustCompile(`(?i)\{[\s]*ASK\s+([^}]+)\}`),
			regexp.MustCompile(`(?i)\{[\s]*FILLIN\s+([^}]+)\}`),
			regexp.MustCompile(`(?i)\{[\s]*REF\s+([^}]+)\}`),
		},
		ComplexMergeField: regexp.MustCompile(`(?i)\{[\s]*MERGEFIELD\s+[^}]*(\*|\\[\w]+|MERGEFORMAT)[^}]*\}`),

		// Formula patterns
		Formulas: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\{[\s]*=\s*[^}]+\}`),       // Field calculations
			regexp.MustCompile(`(?i)\{[\s]*FORMULA\s+[^}]+\}`), // FORMULA fields
			regexp.MustCompile(`(?i)\{[\s]*EQ\s+[^}]+\}`),      // Equation fields
			regexp.MustCompile(`(?i)\{[\s]*CALC\s+[^}]+\}`),    // Calculation fields
			regexp.MustCompile(`(?i)\{[\s]*SYMBOL\s+[^}]+\}`),  // Symbol fields
		},

		// Macro patterns
		Macros: []*regexp.Regexp{
			regexp.MustCompile(`(?i)Sub\s+\w+\s*\(`),
			regexp.MustCompile(`(?i)Function\s+\w+\s*\(`),
			regexp.MustCompile(`(?i)Private\s+Sub`),
			regexp.MustCompile(`(?i)Public\s+Sub`),
			regexp.MustCompile(`(?i)\.VBProject`),
			regexp.MustCompile(`(?i)Macro\d+`),
			regexp.MustCompile(`(?i)Auto(Open|Close|New|Exit)`),
			regexp.MustCompile(`(?i)Document_Open`),
		},

		// Table patterns
		Table:       regexp.MustCompile(`(?i)<table[^>]*>`),
		NestedTable: regexp.MustCompile(`(?i)<table[^>]*>.*?<table[^>]*>`),

		// ActiveX patterns
		ActiveX: []*regexp.Regexp{
			regexp.MustCompile(`(?i)ACTIVEX`),
			regexp.MustCompile(`(?i)\.OCX`),
			regexp.MustCompile(`(?i)CLSID:`),
			regexp.MustCompile(`(?i)ComboBox\d+`),
			regexp.MustCompile(`(?i)CheckBox\d+`),
			regexp.MustCompile(`(?i)CommandButton\d+`),
			regexp.MustCompile(`(?i)Forms\.`),
		},

		// Field code patterns
		FieldCodes: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\{[\s]*AUTOTEXT\s+[^}]+\}`),
			regexp.MustCompile(`(?i)\{[\s]*INCLUDETEXT\s+[^}]+\}`),
			regexp.MustCompile(`(?i)\{[\s]*LINK\s+[^}]+\}`),
			regexp.MustCompile(`(?i)\{[\s]*EMBED\s+[^}]+\}`),
		},
	}
}

// Global pattern registry (lazy initialization)
var defaultPatterns *PatternRegistry

func getPatterns() *PatternRegistry {
	if defaultPatterns == nil {
		defaultPatterns = NewPatternRegistry()
	}
	return defaultPatterns
}

// ContentValidator provides methods to validate content
type ContentValidator struct {
	MinLength            int
	MaxNonPrintableRatio float64
}

// NewContentValidator creates a validator with default settings
func NewContentValidator() *ContentValidator {
	return &ContentValidator{
		MinLength:            MinFormulaLength,
		MaxNonPrintableRatio: MaxNonPrintableRatio,
	}
}

// IsValid checks if content is likely text and not binary data
func (v *ContentValidator) IsValid(content string) bool {
	if len(content) < v.MinLength {
		return false
	}

	nonPrintable := 0
	replacementChars := 0
	totalChars := 0

	for _, r := range content {
		totalChars++
		if r == '\ufffd' { // Unicode replacement character
			replacementChars++
		} else if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
			nonPrintable++
		}
	}

	// If more than 20% replacement characters, it's likely binary
	if float64(replacementChars)/float64(totalChars) > 0.2 {
		return false
	}

	// If more than MaxNonPrintableRatio non-printable, likely binary
	if float64(nonPrintable)/float64(totalChars) > v.MaxNonPrintableRatio {
		return false
	}

	return true
}

// ExtractClean extracts readable text from potentially binary content
func (v *ContentValidator) ExtractClean(content string) string {
	var result strings.Builder
	for _, r := range content {
		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// PatternMatcher provides generic pattern matching functionality
type PatternMatcher struct {
	validator *ContentValidator
}

// NewPatternMatcher creates a new pattern matcher
func NewPatternMatcher() *PatternMatcher {
	return &PatternMatcher{
		validator: NewContentValidator(),
	}
}

// MatchPatterns extracts matches from content using provided patterns
func (m *PatternMatcher) MatchPatterns(
	content string,
	patterns []*regexp.Regexp,
	limit int,
	validate bool,
) (matches []string, validCount, invalidCount int) {
	matches = make([]string, 0, limit)
	seen := make(map[string]bool)

	for _, pattern := range patterns {
		found := pattern.FindAllString(content, -1)
		for _, match := range found {
			// Skip duplicates
			if seen[match] {
				continue
			}
			seen[match] = true

			if validate {
				if m.validator.IsValid(match) {
					validCount++
					if len(matches) < limit {
						cleanMatch := m.validator.ExtractClean(match)
						if len(cleanMatch) > 100 {
							cleanMatch = cleanMatch[:100] + "..."
						}
						matches = append(matches, cleanMatch)
					}
				} else {
					invalidCount++
				}
			} else {
				if len(matches) < limit {
					if len(match) > 100 {
						match = match[:100] + "..."
					}
					matches = append(matches, match)
				}
			}
		}
	}

	return matches, validCount, invalidCount
}

// ComplexityReport contains metrics about document complexity
type ComplexityReport struct {
	Score              int               `json:"complexity_score"`
	Level              string            `json:"complexity_level"` // low, medium, high, critical
	NeedsReview        bool              `json:"needs_human_review"`
	NestedIfDepth      int               `json:"nested_if_depth"`
	TotalIfStatements  int               `json:"total_if_statements"`
	TotalMergeFields   int               `json:"total_merge_fields"`
	ComplexMergeFields []string          `json:"complex_merge_fields"`
	Macros             []string          `json:"macros_found"`
	Formulas           []string          `json:"formulas_found"`
	Issues             []ComplexityIssue `json:"potential_issues"`
	Recommendations    []string          `json:"recommendations"`
	ParseErrors        []string          `json:"parse_errors,omitempty"`
	FieldCodes         []string          `json:"field_codes,omitempty"`
	ValidFormulas      int               `json:"valid_formulas_count"`
	InvalidFormulas    int               `json:"invalid_formulas_count"`
}

// ComplexityIssue represents a specific complexity concern
type ComplexityIssue struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Location    string `json:"location,omitempty"`
	Severity    string `json:"severity"` // low, medium, high
}

// ComplexityConfig allows customization of thresholds and limits
type ComplexityConfig struct {
	// IF statement thresholds
	NestedIfHighThreshold   int
	NestedIfMediumThreshold int
	IfCountHighThreshold    int

	// Merge field thresholds
	MergeFieldHighCount int

	// Score thresholds
	CriticalScore int
	HighScore     int
	MediumScore   int

	// Collection limits
	MaxStoredFormulas   int
	MaxStoredFieldCodes int

	// Feature flags
	ValidateFormulas  bool
	ExtractFieldCodes bool

	// Pattern registry (optional custom patterns)
	Patterns *PatternRegistry
}

// DefaultConfig returns the default configuration
func DefaultConfig() *ComplexityConfig {
	return &ComplexityConfig{
		NestedIfHighThreshold:   NestedIfHighThreshold,
		NestedIfMediumThreshold: NestedIfMediumThreshold,
		IfCountHighThreshold:    IfCountHighThreshold,
		MergeFieldHighCount:     MergeFieldHighCount,
		CriticalScore:           ComplexityScoreCritical,
		HighScore:               ComplexityScoreHigh,
		MediumScore:             ComplexityScoreMedium,
		MaxStoredFormulas:       DefaultMaxStoredFormulas,
		MaxStoredFieldCodes:     DefaultMaxStoredFieldCodes,
		ValidateFormulas:        true,
		ExtractFieldCodes:       true,
		Patterns:                nil, // Use default patterns
	}
}

// AnalyzeComplexity analyzes a DOT file content for complexity indicators
//
// Example:
//
//	content, err := os.ReadFile("template.dot")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	report := analyzer.AnalyzeComplexity(content)
func AnalyzeComplexity(content []byte) *ComplexityReport {
	return AnalyzeComplexityWithConfig(context.Background(), content, DefaultConfig())
}

// AnalyzeComplexityWithContext analyzes with context support for cancellation
func AnalyzeComplexityWithContext(ctx context.Context, content []byte) *ComplexityReport {
	return AnalyzeComplexityWithConfig(ctx, content, DefaultConfig())
}

// AnalyzeComplexityWithConfig analyzes with custom configuration and context
func AnalyzeComplexityWithConfig(ctx context.Context, content []byte, config *ComplexityConfig) *ComplexityReport {
	// Use custom patterns if provided, otherwise use defaults
	patterns := config.Patterns
	if patterns == nil {
		patterns = getPatterns()
	}

	report := &ComplexityReport{
		Score:              0,
		Level:              "low",
		NeedsReview:        false,
		ComplexMergeFields: []string{},
		Macros:             []string{},
		Formulas:           []string{},
		Issues:             []ComplexityIssue{},
		Recommendations:    []string{},
		ParseErrors:        []string{},
		FieldCodes:         []string{},
		ValidFormulas:      0,
		InvalidFormulas:    0,
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		report.ParseErrors = append(report.ParseErrors, "Analysis cancelled")
		return report
	default:
	}

	// Extract text from binary formats if needed
	extractor := NewDocumentExtractor()
	docInfo, err := extractor.AnalyzeDocument(content)

	var contentStr string
	if err != nil {
		// If extraction fails, try to use raw content
		contentStr = string(content)
		report.ParseErrors = append(report.ParseErrors, fmt.Sprintf("Document extraction warning: %v", err))
	} else {
		// Use extracted text for analysis
		contentStr = docInfo.Text

		// Add format information
		formatName := "Unknown"
		switch docInfo.Format {
		case FormatZipBased:
			formatName = "Modern Word (ZIP-based)"
		case FormatOLEBased:
			formatName = "Legacy Word (OLE-based)"
		case FormatRTF:
			formatName = "Rich Text Format"
		case FormatPlainText:
			formatName = "Plain Text"
		}

		// Check if we extracted meaningful content
		if len(contentStr) < 100 && docInfo.Format != FormatPlainText {
			report.ParseErrors = append(report.ParseErrors, fmt.Sprintf("Limited text extraction from %s format (only %d chars)", formatName, len(contentStr)))
			// Try fallback extraction
			fallbackText := extractor.extractReadableText(content)
			if len(fallbackText) > len(contentStr) {
				contentStr = fallbackText
			}
		}

		// Add macro detection from document info
		if docInfo.HasMacros {
			report.Macros = append(report.Macros, "VBA Project detected in document")
			report.NeedsReview = true
		}

		// Add extracted field codes
		if len(docInfo.FieldCodes) > 0 {
			report.FieldCodes = append(report.FieldCodes, docInfo.FieldCodes...)
		}
	}

	analyzer := &complexityAnalyzer{
		patterns:       patterns,
		config:         config,
		patternMatcher: NewPatternMatcher(),
		validator:      NewContentValidator(),
	}

	// Run all analyzers with error handling
	if err := analyzer.analyzeNestedIfs(ctx, contentStr, report); err != nil {
		report.ParseErrors = append(report.ParseErrors, fmt.Sprintf("IF analysis error: %v", err))
	}

	if err := analyzer.analyzeMergeFields(ctx, contentStr, report); err != nil {
		report.ParseErrors = append(report.ParseErrors, fmt.Sprintf("Merge field analysis error: %v", err))
	}

	if err := analyzer.detectMacros(ctx, contentStr, report); err != nil {
		report.ParseErrors = append(report.ParseErrors, fmt.Sprintf("Macro detection error: %v", err))
	}

	if err := analyzer.detectFormulas(ctx, contentStr, report); err != nil {
		report.ParseErrors = append(report.ParseErrors, fmt.Sprintf("Formula detection error: %v", err))
	}

	if err := analyzer.detectTables(ctx, contentStr, report); err != nil {
		report.ParseErrors = append(report.ParseErrors, fmt.Sprintf("Table detection error: %v", err))
	}

	if err := analyzer.detectActiveX(ctx, contentStr, report); err != nil {
		report.ParseErrors = append(report.ParseErrors, fmt.Sprintf("ActiveX detection error: %v", err))
	}

	if config.ExtractFieldCodes {
		analyzer.detectFieldCodes(ctx, contentStr, report)
	}

	// Calculate final score and determine review needs
	analyzer.calculateScore(report)
	analyzer.generateRecommendations(report)

	return report
}

// complexityAnalyzer encapsulates the analysis logic
type complexityAnalyzer struct {
	patterns       *PatternRegistry
	config         *ComplexityConfig
	patternMatcher *PatternMatcher
	validator      *ContentValidator
}

// analyzeNestedIfs detects and measures nested IF statement depth
func (a *complexityAnalyzer) analyzeNestedIfs(ctx context.Context, content string, report *ComplexityReport) error {
	// Check context
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Find all IF field occurrences
	ifMatches := a.patterns.IFFieldFull.FindAllString(content, -1)
	report.TotalIfStatements = len(ifMatches)

	// Analyze nesting depth with improved algorithm
	maxDepth, err := a.calculateIfNestingDepth(content)
	if err != nil {
		return fmt.Errorf("failed to calculate IF nesting depth: %w", err)
	}

	report.NestedIfDepth = maxDepth

	// Add issue for nested conditionals
	if maxDepth > a.config.NestedIfHighThreshold {
		a.addIssue(report, "nested_conditionals",
			fmt.Sprintf("Deep nesting of IF statements detected (depth: %d)", maxDepth),
			"high", maxDepth*NestedIfHighWeight)
	} else if maxDepth > a.config.NestedIfMediumThreshold {
		a.addIssue(report, "nested_conditionals",
			fmt.Sprintf("Moderate nesting of IF statements detected (depth: %d)", maxDepth),
			"medium", maxDepth*NestedIfMediumWeight)
	}

	// Check for high number of IF statements
	if report.TotalIfStatements > a.config.IfCountHighThreshold {
		a.addIssue(report, "multiple_conditionals",
			fmt.Sprintf("High number of conditional statements (%d)", report.TotalIfStatements),
			"medium", report.TotalIfStatements*MultipleIfWeight)
	}

	return nil
}

// calculateIfNestingDepth calculates IF nesting with improved accuracy
func (a *complexityAnalyzer) calculateIfNestingDepth(content string) (int, error) {
	// Find all IF field start positions using proper regex
	ifStarts := a.patterns.IFFieldStart.FindAllStringIndex(content, -1)
	if len(ifStarts) == 0 {
		return 0, nil
	}

	maxDepth := 0
	ifNestPattern := a.patterns.IFFieldStart

	for _, ifStart := range ifStarts {
		depth := 1
		braceCount := 1
		position := ifStart[1]

		// Track brace depth to find the end of this IF field
		for position < len(content) && braceCount > 0 {
			switch content[position] {
			case '{':
				braceCount++
				// Check if this is another IF field using regex
				if position+10 < len(content) {
					testStr := content[position:minInt(position+20, len(content))]
					if ifNestPattern.MatchString(testStr) {
						depth++
					}
				}
			case '}':
				braceCount--
			}
			position++
		}

		if depth > maxDepth {
			maxDepth = depth
		}
	}

	return maxDepth, nil
}

// analyzeMergeFields detects and analyzes merge fields
func (a *complexityAnalyzer) analyzeMergeFields(ctx context.Context, content string, report *ComplexityReport) error {
	// Check context
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	totalMergeFields := 0

	// Use pattern matcher for merge fields
	for _, pattern := range a.patterns.MergeFields {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			totalMergeFields++
			if len(match) > 1 && a.validator.IsValid(match[1]) {
				// Field name is valid
			}
		}
	}

	report.TotalMergeFields = totalMergeFields

	// Find complex merge fields
	complexMatches, _, _ := a.patternMatcher.MatchPatterns(
		content,
		[]*regexp.Regexp{a.patterns.ComplexMergeField},
		a.config.MaxStoredFormulas,
		true,
	)
	report.ComplexMergeFields = complexMatches

	if len(report.ComplexMergeFields) > 0 {
		a.addIssue(report, "complex_merge_fields",
			fmt.Sprintf("Complex merge fields with formatting detected (%d)", len(report.ComplexMergeFields)),
			"medium", len(report.ComplexMergeFields)*ComplexMergeFieldWeight)
	}

	if report.TotalMergeFields > a.config.MergeFieldHighCount {
		a.addIssue(report, "numerous_merge_fields",
			fmt.Sprintf("Large number of merge fields detected (%d)", report.TotalMergeFields),
			"low", report.TotalMergeFields)
	}

	return nil
}

// detectMacros detects VBA macros
func (a *complexityAnalyzer) detectMacros(ctx context.Context, content string, report *ComplexityReport) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	macros, _, _ := a.patternMatcher.MatchPatterns(
		content, a.patterns.Macros, a.config.MaxStoredFormulas, true,
	)
	report.Macros = macros

	if len(report.Macros) > 0 {
		a.addIssue(report, "vba_macros",
			fmt.Sprintf("VBA macros detected in document (%d unique)", len(report.Macros)),
			"high", MacroDetectionWeight)
		report.NeedsReview = true
	}

	return nil
}

// detectFormulas detects formulas with validation
func (a *complexityAnalyzer) detectFormulas(ctx context.Context, content string, report *ComplexityReport) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	formulas, validCount, invalidCount := a.patternMatcher.MatchPatterns(
		content, a.patterns.Formulas, a.config.MaxStoredFormulas, a.config.ValidateFormulas,
	)

	report.Formulas = formulas
	report.ValidFormulas = validCount
	report.InvalidFormulas = invalidCount

	if validCount > 0 {
		a.addIssue(report, "formulas",
			fmt.Sprintf("Valid formulas and calculations detected (%d valid, %d invalid)", validCount, invalidCount),
			"medium", validCount*FormulaWeight)
	}

	return nil
}

// detectTables detects complex table structures
func (a *complexityAnalyzer) detectTables(ctx context.Context, content string, report *ComplexityReport) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	tables := a.patterns.Table.FindAllString(content, -1)
	nestedTables := a.patterns.NestedTable.FindAllString(content, -1)

	if len(nestedTables) > 0 {
		a.addIssue(report, "nested_tables",
			fmt.Sprintf("Nested table structures detected (%d)", len(nestedTables)),
			"medium", NestedTableWeight)
	}

	if len(tables) > 10 {
		a.addIssue(report, "multiple_tables",
			fmt.Sprintf("Multiple table structures detected (%d)", len(tables)),
			"low", MultipleTableWeight)
	}

	return nil
}

// detectActiveX detects ActiveX controls
func (a *complexityAnalyzer) detectActiveX(ctx context.Context, content string, report *ComplexityReport) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	hasActiveX := false
	for _, pattern := range a.patterns.ActiveX {
		if matches := pattern.FindAllString(content, 1); len(matches) > 0 {
			if a.validator.IsValid(matches[0]) {
				hasActiveX = true
				break
			}
		}
	}

	if hasActiveX {
		a.addIssue(report, "activex_controls",
			"ActiveX controls detected",
			"high", ActiveXControlWeight)
		report.NeedsReview = true
	}

	return nil
}

// detectFieldCodes detects special field codes
func (a *complexityAnalyzer) detectFieldCodes(ctx context.Context, content string, report *ComplexityReport) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	fieldCodes, _, _ := a.patternMatcher.MatchPatterns(
		content, a.patterns.FieldCodes, a.config.MaxStoredFieldCodes, true,
	)
	report.FieldCodes = fieldCodes

	if len(report.FieldCodes) > 0 {
		a.addIssue(report, "field_codes",
			fmt.Sprintf("Special field codes detected (%d)", len(report.FieldCodes)),
			"low", len(report.FieldCodes)*2)
	}
}

// addIssue adds an issue to the report and updates the score
func (a *complexityAnalyzer) addIssue(report *ComplexityReport, issueType, description, severity string, scoreIncrease int) {
	report.Issues = append(report.Issues, ComplexityIssue{
		Type:        issueType,
		Description: description,
		Severity:    severity,
	})
	report.Score += scoreIncrease
}

// calculateScore determines final score and complexity level
func (a *complexityAnalyzer) calculateScore(report *ComplexityReport) {
	// Apply penalty for invalid formulas
	if report.InvalidFormulas > report.ValidFormulas*2 {
		if divisor := report.ValidFormulas + report.InvalidFormulas + 1; divisor > 0 {
			report.Score = report.Score * report.ValidFormulas / divisor
		}
	}

	// Determine complexity level
	switch {
	case report.Score >= a.config.CriticalScore:
		report.Level = "critical"
		report.NeedsReview = true
	case report.Score >= a.config.HighScore:
		report.Level = "high"
		report.NeedsReview = true
	case report.Score >= a.config.MediumScore:
		report.Level = "medium"
		for _, issue := range report.Issues {
			if issue.Severity == "high" {
				report.NeedsReview = true
				break
			}
		}
	default:
		report.Level = "low"
	}

	// Force review for certain conditions
	if report.NestedIfDepth > a.config.NestedIfHighThreshold || len(report.Macros) > 0 {
		report.NeedsReview = true
	}
}

// generateRecommendations creates actionable recommendations
func (a *complexityAnalyzer) generateRecommendations(report *ComplexityReport) {
	if report.NeedsReview {
		report.Recommendations = append(report.Recommendations,
			"This document requires human review after conversion")
	}

	if report.NestedIfDepth > a.config.NestedIfMediumThreshold {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("Review nested conditional logic for accuracy (depth: %d)", report.NestedIfDepth))
	}

	if len(report.ComplexMergeFields) > 0 {
		report.Recommendations = append(report.Recommendations,
			"Verify complex merge field formatting after conversion")
	}

	if len(report.Macros) > 0 {
		report.Recommendations = append(report.Recommendations,
			"VBA macros will not be converted - manual recreation may be needed")
	}

	if report.ValidFormulas > 0 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("Test all formulas and calculations for accuracy (%d valid formulas found)", report.ValidFormulas))
	}

	if report.InvalidFormulas > 5 {
		report.Recommendations = append(report.Recommendations,
			"Document contains binary or corrupted formula data - verify source file integrity")
	}

	// Check for ActiveX controls
	for _, issue := range report.Issues {
		if issue.Type == "activex_controls" {
			report.Recommendations = append(report.Recommendations,
				"ActiveX controls require manual replacement or removal")
			break
		}
	}

	if len(report.FieldCodes) > 0 {
		report.Recommendations = append(report.Recommendations,
			"Special field codes detected - verify they convert correctly")
	}

	if report.Level == "critical" {
		report.Recommendations = append(report.Recommendations,
			"Consider manual conversion or specialized tools for this complex document")
	}

	if len(report.ParseErrors) > 0 {
		report.Recommendations = append(report.Recommendations,
			"Some analysis features encountered errors - manual review recommended")
	}
}

// Helper function for minimum integer
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
