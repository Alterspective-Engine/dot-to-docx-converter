package analyzer

import (
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
	MergeFieldHighCount = 15  // Reduced from 20

	// Complexity score levels - ADJUSTED FOR BETTER DISTRIBUTION
	ComplexityScoreCritical = 120  // Increased from 100
	ComplexityScoreHigh     = 60   // Increased from 50
	ComplexityScoreMedium   = 30   // Increased from 25

	// Score weights for different features - CALIBRATED
	NestedIfHighWeight      = 15   // Increased from 10
	NestedIfMediumWeight    = 8    // Increased from 5
	MultipleIfWeight        = 3    // Increased from 2
	ComplexMergeFieldWeight = 6    // Increased from 5
	MacroDetectionWeight    = 40   // Increased from 30
	FormulaWeight           = 5    // Reduced from 8 (was causing inflation)
	NestedTableWeight       = 20   // Increased from 15
	MultipleTableWeight     = 8    // Increased from 5
	ActiveXControlWeight    = 35   // Increased from 25

	// Validation constants
	MinFormulaLength        = 10   // Minimum length for valid formula
	MaxNonPrintableRatio    = 0.3  // Maximum ratio of non-printable chars
)

// Pre-compiled regex patterns for better performance
var (
	// IF statement patterns
	ifFieldStartPattern = regexp.MustCompile(`(?i)\{[\s]*IF\b`)
	ifFieldFullPattern  = regexp.MustCompile(`(?i)\{[\s]*IF\b[^}]*\}`)

	// Enhanced merge field patterns for DOT files
	mergeFieldPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\{[\s]*MERGEFIELD\s+([^}]+)\}`),
		regexp.MustCompile(`(?i)«([^»]+)»`),  // Alternative merge field format
		regexp.MustCompile(`(?i)\{[\s]*DOCVARIABLE\s+([^}]+)\}`),
		regexp.MustCompile(`(?i)\{[\s]*DOCPROPERTY\s+([^}]+)\}`),
		regexp.MustCompile(`(?i)\{[\s]*ASK\s+([^}]+)\}`),
		regexp.MustCompile(`(?i)\{[\s]*FILLIN\s+([^}]+)\}`),
		regexp.MustCompile(`(?i)\{[\s]*REF\s+([^}]+)\}`),
	}

	complexMergeFieldPattern = regexp.MustCompile(`(?i)\{[\s]*MERGEFIELD\s+[^}]*(\*|\\[\w]+|MERGEFORMAT)[^}]*\}`)

	// Formula patterns with validation
	formulaPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\{[\s]*=\s*[^}]+\}`),       // Field calculations
		regexp.MustCompile(`(?i)\{[\s]*FORMULA\s+[^}]+\}`), // FORMULA fields
		regexp.MustCompile(`(?i)\{[\s]*EQ\s+[^}]+\}`),      // Equation fields
		regexp.MustCompile(`(?i)\{[\s]*CALC\s+[^}]+\}`),    // Calculation fields
		regexp.MustCompile(`(?i)\{[\s]*SYMBOL\s+[^}]+\}`),  // Symbol fields
	}

	// Macro patterns
	macroPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)Sub\s+\w+\s*\(`),
		regexp.MustCompile(`(?i)Function\s+\w+\s*\(`),
		regexp.MustCompile(`(?i)Private\s+Sub`),
		regexp.MustCompile(`(?i)Public\s+Sub`),
		regexp.MustCompile(`(?i)\.VBProject`),
		regexp.MustCompile(`(?i)Macro\d+`),
		regexp.MustCompile(`(?i)Auto(Open|Close|New|Exit)`),
		regexp.MustCompile(`(?i)Document_Open`),
	}

	// Table patterns
	tablePattern       = regexp.MustCompile(`(?i)<table[^>]*>`)
	nestedTablePattern = regexp.MustCompile(`(?i)<table[^>]*>.*?<table[^>]*>`) // Non-greedy match

	// ActiveX patterns
	activeXPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)ACTIVEX`),
		regexp.MustCompile(`(?i)\.OCX`),
		regexp.MustCompile(`(?i)CLSID:`),
		regexp.MustCompile(`(?i)ComboBox\d+`),
		regexp.MustCompile(`(?i)CheckBox\d+`),
		regexp.MustCompile(`(?i)CommandButton\d+`),
		regexp.MustCompile(`(?i)Forms\.`),
	}

	// Field code patterns for DOT files
	fieldCodePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\{[\s]*AUTOTEXT\s+[^}]+\}`),
		regexp.MustCompile(`(?i)\{[\s]*INCLUDETEXT\s+[^}]+\}`),
		regexp.MustCompile(`(?i)\{[\s]*LINK\s+[^}]+\}`),
		regexp.MustCompile(`(?i)\{[\s]*EMBED\s+[^}]+\}`),
	}
)

// isValidContent checks if content is likely text and not binary data
func isValidContent(content string) bool {
	if len(content) < MinFormulaLength {
		return false
	}

	nonPrintable := 0
	replacementChars := 0
	totalChars := 0

	for _, r := range content {
		totalChars++
		if r == '\ufffd' {  // Unicode replacement character
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
	if float64(nonPrintable)/float64(totalChars) > MaxNonPrintableRatio {
		return false
	}

	return true
}

// extractCleanText attempts to extract readable text from potentially binary content
func extractCleanText(content string) string {
	var result strings.Builder
	for _, r := range content {
		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ComplexityReport contains metrics about document complexity
type ComplexityReport struct {
	Score              int                 `json:"complexity_score"`
	Level              string              `json:"complexity_level"` // low, medium, high, critical
	NeedsReview        bool                `json:"needs_human_review"`
	NestedIfDepth      int                 `json:"nested_if_depth"`
	TotalIfStatements  int                 `json:"total_if_statements"`
	TotalMergeFields   int                 `json:"total_merge_fields"`
	ComplexMergeFields []string            `json:"complex_merge_fields"`
	Macros             []string            `json:"macros_found"`
	Formulas           []string            `json:"formulas_found"`
	Issues             []ComplexityIssue   `json:"potential_issues"`
	Recommendations    []string            `json:"recommendations"`
	ParseErrors        []string            `json:"parse_errors,omitempty"`
	FieldCodes         []string            `json:"field_codes,omitempty"`
	ValidFormulas      int                 `json:"valid_formulas_count"`
	InvalidFormulas    int                 `json:"invalid_formulas_count"`
}

// ComplexityIssue represents a specific complexity concern
type ComplexityIssue struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Location    string `json:"location,omitempty"`
	Severity    string `json:"severity"` // low, medium, high
}

// ComplexityConfig allows customization of thresholds
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

	// Feature flags
	ValidateFormulas bool
	ExtractFieldCodes bool
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
		ValidateFormulas:        true,
		ExtractFieldCodes:       true,
	}
}

// AnalyzeComplexity analyzes a DOT file content for complexity indicators
func AnalyzeComplexity(content []byte) *ComplexityReport {
	return AnalyzeComplexityWithConfig(content, DefaultConfig())
}

// AnalyzeComplexityWithConfig analyzes with custom configuration
func AnalyzeComplexityWithConfig(content []byte, config *ComplexityConfig) *ComplexityReport {
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

	contentStr := string(content)

	// Run all analyzers with error handling
	if err := analyzeNestedIfs(contentStr, report, config); err != nil {
		report.ParseErrors = append(report.ParseErrors, fmt.Sprintf("IF analysis error: %v", err))
	}

	if err := analyzeMergeFieldsEnhanced(contentStr, report, config); err != nil {
		report.ParseErrors = append(report.ParseErrors, fmt.Sprintf("Merge field analysis error: %v", err))
	}

	if err := detectMacros(contentStr, report); err != nil {
		report.ParseErrors = append(report.ParseErrors, fmt.Sprintf("Macro detection error: %v", err))
	}

	if err := detectFormulasWithValidation(contentStr, report, config); err != nil {
		report.ParseErrors = append(report.ParseErrors, fmt.Sprintf("Formula detection error: %v", err))
	}

	if err := detectComplexTables(contentStr, report); err != nil {
		report.ParseErrors = append(report.ParseErrors, fmt.Sprintf("Table detection error: %v", err))
	}

	if err := detectActiveXControls(contentStr, report); err != nil {
		report.ParseErrors = append(report.ParseErrors, fmt.Sprintf("ActiveX detection error: %v", err))
	}

	if config.ExtractFieldCodes {
		detectFieldCodes(contentStr, report)
	}

	// Calculate final score and determine review needs
	calculateComplexityScore(report, config)

	// Generate recommendations
	generateRecommendations(report, config)

	return report
}

// analyzeNestedIfs detects and measures nested IF statement depth using proper parsing
func analyzeNestedIfs(content string, report *ComplexityReport, config *ComplexityConfig) error {
	// Find all IF field occurrences
	ifMatches := ifFieldFullPattern.FindAllString(content, -1)
	report.TotalIfStatements = len(ifMatches)

	// Properly analyze nesting depth using a more accurate algorithm
	maxDepth, err := calculateIfNestingDepth(content)
	if err != nil {
		return fmt.Errorf("failed to calculate IF nesting depth: %w", err)
	}

	report.NestedIfDepth = maxDepth

	// Evaluate complexity based on nesting depth
	if maxDepth > config.NestedIfHighThreshold {
		report.Issues = append(report.Issues, ComplexityIssue{
			Type:        "nested_conditionals",
			Description: fmt.Sprintf("Deep nesting of IF statements detected (depth: %d)", maxDepth),
			Severity:    "high",
		})
		report.Score += maxDepth * NestedIfHighWeight
	} else if maxDepth > config.NestedIfMediumThreshold {
		report.Issues = append(report.Issues, ComplexityIssue{
			Type:        "nested_conditionals",
			Description: fmt.Sprintf("Moderate nesting of IF statements detected (depth: %d)", maxDepth),
			Severity:    "medium",
		})
		report.Score += maxDepth * NestedIfMediumWeight
	}

	// Check for high number of IF statements
	if report.TotalIfStatements > config.IfCountHighThreshold {
		report.Score += report.TotalIfStatements * MultipleIfWeight
		report.Issues = append(report.Issues, ComplexityIssue{
			Type:        "multiple_conditionals",
			Description: fmt.Sprintf("High number of conditional statements (%d)", report.TotalIfStatements),
			Severity:    "medium",
		})
	}

	return nil
}

// calculateIfNestingDepth properly calculates the nesting depth of IF fields
func calculateIfNestingDepth(content string) (int, error) {
	// Find all IF field start positions
	ifStarts := ifFieldStartPattern.FindAllStringIndex(content, -1)
	if len(ifStarts) == 0 {
		return 0, nil
	}

	maxDepth := 0

	// For each IF field, calculate its nesting level
	for _, ifStart := range ifStarts {
		depth := 1
		braceCount := 1
		position := ifStart[1] // Start after the IF field opening

		// Track brace depth to find the end of this IF field
		for position < len(content) && braceCount > 0 {
			switch content[position] {
			case '{':
				braceCount++
				// Check if this is another IF field
				if position+5 < len(content) {
					nextPart := content[position:min(position+10, len(content))]
					if strings.Contains(strings.ToUpper(nextPart), "IF") {
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

// analyzeMergeFieldsEnhanced detects and analyzes merge fields with DOT-specific patterns
func analyzeMergeFieldsEnhanced(content string, report *ComplexityReport, config *ComplexityConfig) error {
	totalMergeFields := 0
	mergeFieldNames := make(map[string]bool)

	// Check all merge field patterns
	for _, pattern := range mergeFieldPatterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			totalMergeFields++
			if len(match) > 1 && isValidContent(match[1]) {
				mergeFieldNames[match[1]] = true
			}
		}
	}

	report.TotalMergeFields = totalMergeFields

	// Find complex merge fields (with formatting or switches)
	complexMatches := complexMergeFieldPattern.FindAllString(content, -1)
	for _, match := range complexMatches {
		if isValidContent(match) {
			// Store only first 100 chars to avoid memory issues
			if len(match) > 100 {
				match = match[:100] + "..."
			}
			report.ComplexMergeFields = append(report.ComplexMergeFields, match)
		}
	}

	if len(report.ComplexMergeFields) > 0 {
		report.Issues = append(report.Issues, ComplexityIssue{
			Type:        "complex_merge_fields",
			Description: fmt.Sprintf("Complex merge fields with formatting detected (%d)", len(report.ComplexMergeFields)),
			Severity:    "medium",
		})
		report.Score += len(report.ComplexMergeFields) * ComplexMergeFieldWeight
	}

	if report.TotalMergeFields > config.MergeFieldHighCount {
		report.Score += report.TotalMergeFields
		report.Issues = append(report.Issues, ComplexityIssue{
			Type:        "numerous_merge_fields",
			Description: fmt.Sprintf("Large number of merge fields detected (%d)", report.TotalMergeFields),
			Severity:    "low",
		})
	}

	return nil
}

// detectFormulasWithValidation detects formulas and validates they're not binary data
func detectFormulasWithValidation(content string, report *ComplexityReport, config *ComplexityConfig) error {
	formulasFound := []string{}
	invalidCount := 0

	for _, pattern := range formulaPatterns {
		matches := pattern.FindAllString(content, -1)
		for _, match := range matches {
			if config.ValidateFormulas {
				if isValidContent(match) {
					// Store only first 10 valid formulas to prevent memory issues
					if len(formulasFound) < 10 {
						// Clean and truncate formula for storage
						cleanFormula := extractCleanText(match)
						if len(cleanFormula) > 100 {
							cleanFormula = cleanFormula[:100] + "..."
						}
						formulasFound = append(formulasFound, cleanFormula)
					}
					report.ValidFormulas++
				} else {
					invalidCount++
					report.InvalidFormulas++
				}
			} else {
				// No validation, store as-is (limited)
				if len(formulasFound) < 10 {
					formulasFound = append(formulasFound, match)
				}
			}
		}
	}

	report.Formulas = formulasFound

	// Only count valid formulas for scoring
	validFormulaCount := report.ValidFormulas
	if validFormulaCount > 0 {
		report.Issues = append(report.Issues, ComplexityIssue{
			Type:        "formulas",
			Description: fmt.Sprintf("Valid formulas and calculations detected (%d valid, %d invalid)", validFormulaCount, invalidCount),
			Severity:    "medium",
		})
		// Use calibrated weight
		report.Score += validFormulaCount * FormulaWeight
	}

	return nil
}

// detectFieldCodes detects special field codes in DOT files
func detectFieldCodes(content string, report *ComplexityReport) {
	for _, pattern := range fieldCodePatterns {
		matches := pattern.FindAllString(content, -1)
		for _, match := range matches {
			if isValidContent(match) && len(report.FieldCodes) < 20 {
				report.FieldCodes = append(report.FieldCodes, match)
			}
		}
	}

	if len(report.FieldCodes) > 0 {
		report.Issues = append(report.Issues, ComplexityIssue{
			Type:        "field_codes",
			Description: fmt.Sprintf("Special field codes detected (%d)", len(report.FieldCodes)),
			Severity:    "low",
		})
		report.Score += len(report.FieldCodes) * 2
	}
}

// detectMacros detects VBA macros with error handling
func detectMacros(content string, report *ComplexityReport) error {
	macrosFound := []string{}

	for _, pattern := range macroPatterns {
		matches := pattern.FindAllString(content, -1)
		if len(matches) > 0 {
			macrosFound = append(macrosFound, matches...)
		}
	}

	// Deduplicate macros
	seen := make(map[string]bool)
	for _, macro := range macrosFound {
		if !seen[macro] && isValidContent(macro) {
			seen[macro] = true
			report.Macros = append(report.Macros, macro)
		}
	}

	if len(report.Macros) > 0 {
		report.Issues = append(report.Issues, ComplexityIssue{
			Type:        "vba_macros",
			Description: fmt.Sprintf("VBA macros detected in document (%d unique)", len(report.Macros)),
			Severity:    "high",
		})
		report.Score += MacroDetectionWeight
		report.NeedsReview = true
	}

	return nil
}

// detectComplexTables looks for complex table structures
func detectComplexTables(content string, report *ComplexityReport) error {
	tables := tablePattern.FindAllString(content, -1)
	nestedTables := nestedTablePattern.FindAllString(content, -1)

	if len(nestedTables) > 0 {
		report.Issues = append(report.Issues, ComplexityIssue{
			Type:        "nested_tables",
			Description: fmt.Sprintf("Nested table structures detected (%d)", len(nestedTables)),
			Severity:    "medium",
		})
		report.Score += NestedTableWeight
	}

	if len(tables) > 10 {
		report.Issues = append(report.Issues, ComplexityIssue{
			Type:        "multiple_tables",
			Description: fmt.Sprintf("Multiple table structures detected (%d)", len(tables)),
			Severity:    "low",
		})
		report.Score += MultipleTableWeight
	}

	return nil
}

// detectActiveXControls looks for ActiveX controls
func detectActiveXControls(content string, report *ComplexityReport) error {
	hasActiveX := false
	detectedControls := []string{}

	for _, pattern := range activeXPatterns {
		if matches := pattern.FindAllString(content, 1); len(matches) > 0 {
			if isValidContent(matches[0]) {
				hasActiveX = true
				detectedControls = append(detectedControls, matches[0])
			}
		}
	}

	if hasActiveX {
		report.Issues = append(report.Issues, ComplexityIssue{
			Type:        "activex_controls",
			Description: fmt.Sprintf("ActiveX controls detected (%d types)", len(detectedControls)),
			Severity:    "high",
		})
		report.Score += ActiveXControlWeight
		report.NeedsReview = true
	}

	return nil
}

// calculateComplexityScore determines final score and complexity level
func calculateComplexityScore(report *ComplexityReport, config *ComplexityConfig) {
	// Apply bonus/penalty based on validation results
	if report.InvalidFormulas > report.ValidFormulas*2 {
		// Reduce score if mostly invalid formulas (likely binary data)
		report.Score = report.Score * report.ValidFormulas / (report.ValidFormulas + report.InvalidFormulas + 1)
	}

	// Determine complexity level based on score
	switch {
	case report.Score >= config.CriticalScore:
		report.Level = "critical"
		report.NeedsReview = true
	case report.Score >= config.HighScore:
		report.Level = "high"
		report.NeedsReview = true
	case report.Score >= config.MediumScore:
		report.Level = "medium"
		// Check for high severity issues
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
	if report.NestedIfDepth > config.NestedIfHighThreshold || len(report.Macros) > 0 {
		report.NeedsReview = true
	}
}

// generateRecommendations creates actionable recommendations
func generateRecommendations(report *ComplexityReport, config *ComplexityConfig) {
	if report.NeedsReview {
		report.Recommendations = append(report.Recommendations,
			"This document requires human review after conversion")
	}

	if report.NestedIfDepth > config.NestedIfMediumThreshold {
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

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}