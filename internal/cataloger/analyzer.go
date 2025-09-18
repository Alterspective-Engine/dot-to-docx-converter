package cataloger

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
)

// FieldCategory represents the type of field
type FieldCategory string

const (
	FieldCategoryBasic        FieldCategory = "BASIC"        // Simple replacement fields
	FieldCategoryCalculated   FieldCategory = "CALCULATED"   // Fields with formulas
	FieldCategoryConditional  FieldCategory = "CONDITIONAL"  // IF statements
	FieldCategoryNested       FieldCategory = "NESTED"       // Nested conditionals
	FieldCategoryLookup       FieldCategory = "LOOKUP"       // Reference/lookup fields
	FieldCategoryDate         FieldCategory = "DATE"         // Date fields
	FieldCategoryJurisdiction FieldCategory = "JURISDICTION" // Legal jurisdiction
	FieldCategoryMatterType   FieldCategory = "MATTER_TYPE"  // Type of legal matter
	FieldCategoryComplex      FieldCategory = "COMPLEX"      // Complex/unknown
)

// DocumentComplexity levels
type ComplexityLevel string

const (
	ComplexitySimple   ComplexityLevel = "SIMPLE"   // Fully automated
	ComplexityModerate ComplexityLevel = "MODERATE" // Mostly automated
	ComplexityComplex  ComplexityLevel = "COMPLEX"  // Needs review
	ComplexityCritical ComplexityLevel = "CRITICAL" // Manual intervention required
)

// ContentBlock represents reusable content sections
type ContentBlock struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"` // header, footer, letterhead, signature
	Content    string            `json:"content"`
	Hash       string            `json:"hash"`
	Frequency  int               `json:"frequency"`
	Documents  []string          `json:"documents"`
	Variables  map[string]string `json:"variables"`
	Confidence float64           `json:"confidence"`
}

// EnhancedField represents detailed field information
type EnhancedField struct {
	Name             string        `json:"name"`
	Category         FieldCategory `json:"category"`
	OriginalSyntax   string        `json:"originalSyntax"`
	StandardizedName string        `json:"standardizedName"`
	DataType         string        `json:"dataType"`
	SampleValues     []string      `json:"sampleValues"`
	Frequency        int           `json:"frequency"`
	Documents        []string      `json:"documents"`
	NestingLevel     int           `json:"nestingLevel"`
	Dependencies     []string      `json:"dependencies"`
	ValidationRules  []string      `json:"validationRules"`
	MergeCandidates  []string      `json:"mergeCandidates"`
	Confidence       float64       `json:"confidence"`
}

// DocumentCatalog represents comprehensive document analysis
type DocumentCatalog struct {
	AnalysisDate     time.Time                 `json:"analysisDate"`
	TotalDocuments   int                       `json:"totalDocuments"`
	ProcessingTime   time.Duration             `json:"processingTime"`
	Fields           map[string]*EnhancedField `json:"fields"`
	ContentBlocks    map[string]*ContentBlock  `json:"contentBlocks"`
	CommonPatterns   map[string]int            `json:"commonPatterns"`
	Jurisdictions    map[string]int            `json:"jurisdictions"`
	MatterTypes      map[string]int            `json:"matterTypes"`
	ComplexityDist   map[ComplexityLevel]int   `json:"complexityDistribution"`
	DocumentProfiles []DocumentProfile         `json:"documentProfiles"`
	MergeGroups      []FieldMergeGroup         `json:"mergeGroups"`
	Statistics       CatalogStatistics         `json:"statistics"`
	Recommendations  []string                  `json:"recommendations"`
	QualityMetrics   QualityMetrics            `json:"qualityMetrics"`
}

// DocumentProfile represents individual document analysis
type DocumentProfile struct {
	Filename        string            `json:"filename"`
	Hash            string            `json:"hash"`
	PageCount       int               `json:"pageCount"`
	WordCount       int               `json:"wordCount"`
	Complexity      ComplexityLevel   `json:"complexity"`
	ComplexityScore float64           `json:"complexityScore"`
	Fields          []string          `json:"fields"`
	HasMacros       bool              `json:"hasMacros"`
	HasTables       bool              `json:"hasTables"`
	HasImages       bool              `json:"hasImages"`
	HeaderHash      string            `json:"headerHash"`
	FooterHash      string            `json:"footerHash"`
	Jurisdiction    string            `json:"jurisdiction"`
	MatterType      string            `json:"matterType"`
	ReviewRequired  bool              `json:"reviewRequired"`
	ReviewReasons   []string          `json:"reviewReasons"`
	Metadata        map[string]string `json:"metadata"`
	Warnings        []string          `json:"warnings"`
}

// FieldMergeGroup represents fields that could be merged
type FieldMergeGroup struct {
	PrimaryField    string   `json:"primaryField"`
	Variants        []string `json:"variants"`
	Similarity      float64  `json:"similarity"`
	TotalOccurrence int      `json:"totalOccurrence"`
	Recommendation  string   `json:"recommendation"`
}

// CatalogStatistics provides overview metrics
type CatalogStatistics struct {
	UniqueFields        int            `json:"uniqueFields"`
	MergeableFields     int            `json:"mergeableFields"`
	ContentBlocksFound  int            `json:"contentBlocksFound"`
	AverageComplexity   float64        `json:"averageComplexity"`
	AutomationPotential float64        `json:"automationPotential"`
	EstimatedSavings    string         `json:"estimatedSavings"`
	MostCommonFields    []string       `json:"mostCommonFields"`
	ComplexityBreakdown map[string]int `json:"complexityBreakdown"`
}

// QualityMetrics tracks extraction quality
type QualityMetrics struct {
	ExtractionConfidence float64            `json:"extractionConfidence"`
	FieldCoverage        float64            `json:"fieldCoverage"`
	PatternConsistency   float64            `json:"patternConsistency"`
	ValidationErrors     int                `json:"validationErrors"`
	ProcessingErrors     int                `json:"processingErrors"`
	ConfidenceByCategory map[string]float64 `json:"confidenceByCategory"`
}

// DocumentAnalyzer performs comprehensive document analysis
type DocumentAnalyzer struct {
	patterns         map[string]*regexp.Regexp
	fieldNormalizer  *FieldNormalizer
	complexityScorer *ComplexityScorer
	contentDetector  *ContentBlockDetector
	aiEnhancer       *AIEnhancer
}

// NewDocumentAnalyzer creates analyzer instance
func NewDocumentAnalyzer(useAI bool) *DocumentAnalyzer {
	analyzer := &DocumentAnalyzer{
		patterns:         initializePatterns(),
		fieldNormalizer:  NewFieldNormalizer(),
		complexityScorer: NewComplexityScorer(),
		contentDetector:  NewContentBlockDetector(),
	}

	if useAI {
		analyzer.aiEnhancer = NewAIEnhancer()
	}

	return analyzer
}

// AnalyzeDocuments performs comprehensive analysis on document set
func (a *DocumentAnalyzer) AnalyzeDocuments(documents []DocumentData) (*DocumentCatalog, error) {
	startTime := time.Now()
	catalog := &DocumentCatalog{
		AnalysisDate:     startTime,
		TotalDocuments:   len(documents),
		Fields:           make(map[string]*EnhancedField),
		ContentBlocks:    make(map[string]*ContentBlock),
		CommonPatterns:   make(map[string]int),
		Jurisdictions:    make(map[string]int),
		MatterTypes:      make(map[string]int),
		ComplexityDist:   make(map[ComplexityLevel]int),
		DocumentProfiles: make([]DocumentProfile, 0, len(documents)),
		MergeGroups:      make([]FieldMergeGroup, 0),
	}

	// Process each document
	for _, doc := range documents {
		profile := a.analyzeDocument(doc, catalog)
		catalog.DocumentProfiles = append(catalog.DocumentProfiles, profile)
	}

	// Post-processing analysis
	a.detectContentBlocks(catalog)
	a.identifyFieldMergeGroups(catalog)
	a.calculateStatistics(catalog)
	a.generateRecommendations(catalog)
	a.assessQuality(catalog)

	// AI enhancement if enabled
	if a.aiEnhancer != nil {
		a.aiEnhancer.EnhanceCatalog(catalog)
	}

	catalog.ProcessingTime = time.Since(startTime)
	return catalog, nil
}

// analyzeDocument performs detailed analysis of single document
func (a *DocumentAnalyzer) analyzeDocument(doc DocumentData, catalog *DocumentCatalog) DocumentProfile {
	profile := DocumentProfile{
		Filename: doc.Filename,
		Hash:     a.hashContent(doc.Content),
		Metadata: make(map[string]string),
		Warnings: make([]string, 0),
	}

	// Extract basic metrics
	profile.PageCount = a.estimatePageCount(doc.Content)
	profile.WordCount = len(strings.Fields(doc.ExtractedText))

	// Detect structural elements
	profile.HasMacros = a.detectMacros(doc.Content)
	profile.HasTables = a.detectTables(doc.ExtractedText)
	profile.HasImages = a.detectImages(doc.Content)

	// Extract header/footer
	header, footer := a.extractHeaderFooter(doc.ExtractedText)
	profile.HeaderHash = a.hashContent([]byte(header))
	profile.FooterHash = a.hashContent([]byte(footer))

	// Extract and categorize fields
	fields := a.extractFields(doc.ExtractedText)
	for _, field := range fields {
		a.catalogField(field, doc.Filename, catalog)
		profile.Fields = append(profile.Fields, field.Name)
	}

	// Detect jurisdiction and matter type
	profile.Jurisdiction = a.detectJurisdiction(doc.ExtractedText)
	profile.MatterType = a.detectMatterType(doc.ExtractedText)

	// Calculate complexity
	complexityScore := a.complexityScorer.Calculate(doc, fields)
	profile.ComplexityScore = complexityScore
	profile.Complexity = a.getComplexityLevel(complexityScore)

	// Determine review requirements
	profile.ReviewRequired, profile.ReviewReasons = a.assessReviewNeeds(profile, fields)

	// Update catalog counters
	catalog.ComplexityDist[profile.Complexity]++
	if profile.Jurisdiction != "" {
		catalog.Jurisdictions[profile.Jurisdiction]++
	}
	if profile.MatterType != "" {
		catalog.MatterTypes[profile.MatterType]++
	}

	return profile
}

// extractFields identifies and categorizes all fields
func (a *DocumentAnalyzer) extractFields(text string) []*EnhancedField {
	fields := make([]*EnhancedField, 0)
	seen := make(map[string]bool)

	// Extract different field types
	basicFields := a.extractBasicFields(text)
	calculatedFields := a.extractCalculatedFields(text)
	conditionalFields := a.extractConditionalFields(text)

	// Combine and deduplicate
	allFields := append(append(basicFields, calculatedFields...), conditionalFields...)

	for _, field := range allFields {
		if !seen[field.Name] {
			fields = append(fields, field)
			seen[field.Name] = true
		}
	}

	return fields
}

// catalogField adds field to catalog with deduplication
func (a *DocumentAnalyzer) catalogField(field *EnhancedField, filename string, catalog *DocumentCatalog) {
	standardized := a.fieldNormalizer.Standardize(field.Name)
	field.StandardizedName = standardized

	if existing, exists := catalog.Fields[standardized]; exists {
		existing.Frequency++
		existing.Documents = append(existing.Documents, filename)
		if field.Category > existing.Category { // Use most complex category
			existing.Category = field.Category
		}
	} else {
		field.Frequency = 1
		field.Documents = []string{filename}
		catalog.Fields[standardized] = field
	}
}

// detectContentBlocks identifies reusable content sections
func (a *DocumentAnalyzer) detectContentBlocks(catalog *DocumentCatalog) {
	headerHashes := make(map[string][]string)
	footerHashes := make(map[string][]string)

	for _, profile := range catalog.DocumentProfiles {
		if profile.HeaderHash != "" {
			headerHashes[profile.HeaderHash] = append(headerHashes[profile.HeaderHash], profile.Filename)
		}
		if profile.FooterHash != "" {
			footerHashes[profile.FooterHash] = append(footerHashes[profile.FooterHash], profile.Filename)
		}
	}

	// Create content blocks for common headers
	for hash, docs := range headerHashes {
		if len(docs) > 1 {
			block := &ContentBlock{
				ID:         fmt.Sprintf("header_%s", hash[:8]),
				Type:       "header",
				Hash:       hash,
				Frequency:  len(docs),
				Documents:  docs,
				Confidence: float64(len(docs)) / float64(catalog.TotalDocuments),
			}
			catalog.ContentBlocks[block.ID] = block
		}
	}

	// Create content blocks for common footers
	for hash, docs := range footerHashes {
		if len(docs) > 1 {
			block := &ContentBlock{
				ID:         fmt.Sprintf("footer_%s", hash[:8]),
				Type:       "footer",
				Hash:       hash,
				Frequency:  len(docs),
				Documents:  docs,
				Confidence: float64(len(docs)) / float64(catalog.TotalDocuments),
			}
			catalog.ContentBlocks[block.ID] = block
		}
	}
}

// identifyFieldMergeGroups finds similar fields that could be merged
func (a *DocumentAnalyzer) identifyFieldMergeGroups(catalog *DocumentCatalog) {
	fields := make([]string, 0, len(catalog.Fields))
	for name := range catalog.Fields {
		fields = append(fields, name)
	}

	// Find similar field groups
	groups := a.fieldNormalizer.FindSimilarGroups(fields)

	for primary, variants := range groups {
		if len(variants) > 0 {
			totalOccurrence := catalog.Fields[primary].Frequency
			for _, variant := range variants {
				if f, exists := catalog.Fields[variant]; exists {
					totalOccurrence += f.Frequency
				}
			}

			group := FieldMergeGroup{
				PrimaryField:    primary,
				Variants:        variants,
				Similarity:      a.fieldNormalizer.CalculateSimilarity(primary, variants[0]),
				TotalOccurrence: totalOccurrence,
				Recommendation:  fmt.Sprintf("Merge %d variants into '%s'", len(variants), primary),
			}
			catalog.MergeGroups = append(catalog.MergeGroups, group)
		}
	}
}

// Helper methods

func (a *DocumentAnalyzer) hashContent(content []byte) string {
	hasher := md5.New()
	hasher.Write(content)
	return hex.EncodeToString(hasher.Sum(nil))
}

func (a *DocumentAnalyzer) estimatePageCount(content []byte) int {
	// Rough estimation: 3000 bytes per page
	return int(math.Ceil(float64(len(content)) / 3000))
}

func (a *DocumentAnalyzer) detectMacros(content []byte) bool {
	macroPatterns := [][]byte{
		[]byte("vbaProject"),
		[]byte("VBA"),
		[]byte("Macros"),
		[]byte("Sub "),
		[]byte("Function "),
		[]byte("End Sub"),
	}

	for _, pattern := range macroPatterns {
		if strings.Contains(string(content), string(pattern)) {
			return true
		}
	}
	return false
}

func (a *DocumentAnalyzer) detectTables(text string) bool {
	// Simple table detection - look for tab/pipe separated content
	lines := strings.Split(text, "\n")
	tabCount := 0
	for _, line := range lines {
		if strings.Count(line, "\t") > 2 || strings.Count(line, "|") > 2 {
			tabCount++
		}
	}
	return tabCount > 3
}

func (a *DocumentAnalyzer) detectImages(content []byte) bool {
	imagePatterns := [][]byte{
		[]byte("PNG"),
		[]byte("JFIF"),
		[]byte("JPEG"),
		[]byte("GIF89"),
		[]byte("BM"), // BMP
	}

	for _, pattern := range imagePatterns {
		if strings.Contains(string(content), string(pattern)) {
			return true
		}
	}
	return false
}

func (a *DocumentAnalyzer) extractHeaderFooter(text string) (string, string) {
	lines := strings.Split(text, "\n")
	if len(lines) < 10 {
		return "", ""
	}

	// Simple heuristic: first 5 lines = header, last 5 = footer
	header := strings.Join(lines[:5], "\n")
	footer := strings.Join(lines[len(lines)-5:], "\n")

	return header, footer
}

func (a *DocumentAnalyzer) detectJurisdiction(text string) string {
	jurisdictions := map[string][]string{
		"NSW":     {"New South Wales", "NSW", "Sydney"},
		"VIC":     {"Victoria", "VIC", "Melbourne"},
		"QLD":     {"Queensland", "QLD", "Brisbane"},
		"WA":      {"Western Australia", "WA", "Perth"},
		"SA":      {"South Australia", "SA", "Adelaide"},
		"TAS":     {"Tasmania", "TAS", "Hobart"},
		"NT":      {"Northern Territory", "NT", "Darwin"},
		"ACT":     {"Australian Capital Territory", "ACT", "Canberra"},
		"Federal": {"Federal", "Commonwealth", "Australia"},
	}

	textLower := strings.ToLower(text)
	for code, patterns := range jurisdictions {
		for _, pattern := range patterns {
			if strings.Contains(textLower, strings.ToLower(pattern)) {
				return code
			}
		}
	}

	return ""
}

func (a *DocumentAnalyzer) detectMatterType(text string) string {
	matterTypes := map[string][]string{
		"Personal Injury": {"personal injury", "injury claim", "compensation claim"},
		"Family Law":      {"family law", "divorce", "custody", "separation"},
		"Criminal":        {"criminal", "prosecution", "defence", "charges"},
		"Commercial":      {"commercial", "business", "contract", "agreement"},
		"Property":        {"property", "real estate", "conveyancing", "lease"},
		"Employment":      {"employment", "workplace", "unfair dismissal", "wages"},
		"Wills & Estates": {"will", "estate", "probate", "inheritance"},
		"Immigration":     {"immigration", "visa", "citizenship", "migration"},
	}

	textLower := strings.ToLower(text)
	for matterType, patterns := range matterTypes {
		for _, pattern := range patterns {
			if strings.Contains(textLower, pattern) {
				return matterType
			}
		}
	}

	return "General"
}

func (a *DocumentAnalyzer) getComplexityLevel(score float64) ComplexityLevel {
	switch {
	case score < 25:
		return ComplexitySimple
	case score < 50:
		return ComplexityModerate
	case score < 75:
		return ComplexityComplex
	default:
		return ComplexityCritical
	}
}

func (a *DocumentAnalyzer) assessReviewNeeds(profile DocumentProfile, fields []*EnhancedField) (bool, []string) {
	reasons := make([]string, 0)

	if profile.HasMacros {
		reasons = append(reasons, "Contains macros that need manual conversion")
	}

	if profile.ComplexityScore > 70 {
		reasons = append(reasons, "High complexity score requires review")
	}

	nestedCount := 0
	for _, field := range fields {
		if field.NestingLevel > 3 {
			nestedCount++
		}
	}
	if nestedCount > 5 {
		reasons = append(reasons, fmt.Sprintf("%d deeply nested fields detected", nestedCount))
	}

	return len(reasons) > 0, reasons
}

// Initialize regex patterns
func initializePatterns() map[string]*regexp.Regexp {
	return map[string]*regexp.Regexp{
		"mergefield":  regexp.MustCompile(`(?i)«([^»]+)»|{{\s*([^}]+)\s*}}|\bMERGEFIELD\s+(\w+)`),
		"formula":     regexp.MustCompile(`(?i)=\s*(?:SUM|AVG|COUNT|MAX|MIN|IF)\s*\([^)]+\)`),
		"ifstatement": regexp.MustCompile(`(?i)(?:IF\s+[^{]+\s*{|{{\s*#if\s+[^}]+}})`),
		"date":        regexp.MustCompile(`(?i)\b(?:DATE|TODAY|NOW)\b`),
		"lookup":      regexp.MustCompile(`(?i)\b(?:VLOOKUP|HLOOKUP|INDEX|MATCH|REF)\b`),
	}
}

// Extract basic fields
func (a *DocumentAnalyzer) extractBasicFields(text string) []*EnhancedField {
	fields := make([]*EnhancedField, 0)

	if pattern, exists := a.patterns["mergefield"]; exists {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			fieldName := ""
			for i := 1; i < len(match); i++ {
				if match[i] != "" {
					fieldName = match[i]
					break
				}
			}

			if fieldName != "" {
				field := &EnhancedField{
					Name:           fieldName,
					Category:       FieldCategoryBasic,
					OriginalSyntax: match[0],
					DataType:       a.inferDataType(fieldName),
					Confidence:     0.95,
				}
				fields = append(fields, field)
			}
		}
	}

	return fields
}

// Extract calculated fields
func (a *DocumentAnalyzer) extractCalculatedFields(text string) []*EnhancedField {
	fields := make([]*EnhancedField, 0)

	if pattern, exists := a.patterns["formula"]; exists {
		matches := pattern.FindAllString(text, -1)
		for _, match := range matches {
			field := &EnhancedField{
				Name:           fmt.Sprintf("formula_%d", len(fields)),
				Category:       FieldCategoryCalculated,
				OriginalSyntax: match,
				DataType:       "numeric",
				Confidence:     0.85,
			}
			fields = append(fields, field)
		}
	}

	return fields
}

// Extract conditional fields
func (a *DocumentAnalyzer) extractConditionalFields(text string) []*EnhancedField {
	fields := make([]*EnhancedField, 0)

	if pattern, exists := a.patterns["ifstatement"]; exists {
		matches := pattern.FindAllString(text, -1)
		for _, match := range matches {
			nestingLevel := a.calculateNestingLevel(match, text)
			category := FieldCategoryConditional
			if nestingLevel > 1 {
				category = FieldCategoryNested
			}

			field := &EnhancedField{
				Name:           fmt.Sprintf("condition_%d", len(fields)),
				Category:       category,
				OriginalSyntax: match,
				NestingLevel:   nestingLevel,
				DataType:       "boolean",
				Confidence:     0.80,
			}
			fields = append(fields, field)
		}
	}

	return fields
}

// Calculate nesting level of conditionals
func (a *DocumentAnalyzer) calculateNestingLevel(condition string, fullText string) int {
	// Simple nesting detection - count IF keywords
	level := 1
	index := strings.Index(fullText, condition)
	if index > 0 {
		// Look for nested IFs within this condition's scope
		endIndex := index + len(condition)
		scope := fullText[index:endIndex]
		level = strings.Count(scope, "IF ")
	}
	return level
}

// Infer data type from field name
func (a *DocumentAnalyzer) inferDataType(fieldName string) string {
	lower := strings.ToLower(fieldName)

	switch {
	case strings.Contains(lower, "date") || strings.Contains(lower, "time"):
		return "date"
	case strings.Contains(lower, "amount") || strings.Contains(lower, "total") ||
		strings.Contains(lower, "price") || strings.Contains(lower, "cost"):
		return "numeric"
	case strings.Contains(lower, "email"):
		return "email"
	case strings.Contains(lower, "phone") || strings.Contains(lower, "mobile"):
		return "phone"
	case strings.Contains(lower, "address") || strings.Contains(lower, "street"):
		return "address"
	case strings.Contains(lower, "yes") || strings.Contains(lower, "no") ||
		strings.Contains(lower, "true") || strings.Contains(lower, "false"):
		return "boolean"
	default:
		return "text"
	}
}

// Calculate statistics
func (a *DocumentAnalyzer) calculateStatistics(catalog *DocumentCatalog) {
	stats := CatalogStatistics{
		UniqueFields:       len(catalog.Fields),
		MergeableFields:    len(catalog.MergeGroups),
		ContentBlocksFound: len(catalog.ContentBlocks),
	}

	// Calculate average complexity
	totalComplexity := 0.0
	for _, profile := range catalog.DocumentProfiles {
		totalComplexity += profile.ComplexityScore
	}
	stats.AverageComplexity = totalComplexity / float64(len(catalog.DocumentProfiles))

	// Calculate automation potential
	simpleCount := catalog.ComplexityDist[ComplexitySimple]
	moderateCount := catalog.ComplexityDist[ComplexityModerate]
	total := catalog.TotalDocuments
	stats.AutomationPotential = float64(simpleCount+moderateCount) / float64(total) * 100

	// Find most common fields
	type fieldFreq struct {
		name string
		freq int
	}
	fieldList := make([]fieldFreq, 0, len(catalog.Fields))
	for name, field := range catalog.Fields {
		fieldList = append(fieldList, fieldFreq{name, field.Frequency})
	}
	sort.Slice(fieldList, func(i, j int) bool {
		return fieldList[i].freq > fieldList[j].freq
	})

	for i := 0; i < 10 && i < len(fieldList); i++ {
		stats.MostCommonFields = append(stats.MostCommonFields, fieldList[i].name)
	}

	catalog.Statistics = stats
}

// Generate recommendations
func (a *DocumentAnalyzer) generateRecommendations(catalog *DocumentCatalog) {
	recommendations := make([]string, 0)

	// Content block recommendations
	if len(catalog.ContentBlocks) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Create %d reusable content blocks for headers/footers", len(catalog.ContentBlocks)))
	}

	// Field merge recommendations
	if len(catalog.MergeGroups) > 0 {
		totalVariants := 0
		for _, group := range catalog.MergeGroups {
			totalVariants += len(group.Variants)
		}
		recommendations = append(recommendations,
			fmt.Sprintf("Standardize %d field variants into %d primary fields", totalVariants, len(catalog.MergeGroups)))
	}

	// Complexity recommendations
	if catalog.Statistics.AutomationPotential < 80 {
		recommendations = append(recommendations,
			"Consider simplifying complex templates to increase automation potential")
	}

	// Review recommendations
	reviewCount := 0
	for _, profile := range catalog.DocumentProfiles {
		if profile.ReviewRequired {
			reviewCount++
		}
	}
	if reviewCount > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("%d documents require manual review before migration", reviewCount))
	}

	catalog.Recommendations = recommendations
}

// Assess extraction quality
func (a *DocumentAnalyzer) assessQuality(catalog *DocumentCatalog) {
	quality := QualityMetrics{
		ConfidenceByCategory: make(map[string]float64),
	}

	// Calculate average confidence by category
	categoryConfidence := make(map[FieldCategory][]float64)
	for _, field := range catalog.Fields {
		categoryConfidence[field.Category] = append(categoryConfidence[field.Category], field.Confidence)
	}

	for category, confidences := range categoryConfidence {
		sum := 0.0
		for _, conf := range confidences {
			sum += conf
		}
		quality.ConfidenceByCategory[string(category)] = sum / float64(len(confidences))
	}

	// Overall extraction confidence
	totalConfidence := 0.0
	fieldCount := 0
	for _, field := range catalog.Fields {
		totalConfidence += field.Confidence * float64(field.Frequency)
		fieldCount += field.Frequency
	}
	if fieldCount > 0 {
		quality.ExtractionConfidence = totalConfidence / float64(fieldCount)
	}

	// Field coverage (% of documents with fields extracted)
	docsWithFields := 0
	for _, profile := range catalog.DocumentProfiles {
		if len(profile.Fields) > 0 {
			docsWithFields++
		}
	}
	quality.FieldCoverage = float64(docsWithFields) / float64(catalog.TotalDocuments) * 100

	catalog.QualityMetrics = quality
}

// DocumentData represents input document
type DocumentData struct {
	Filename      string
	Content       []byte
	ExtractedText string
	Metadata      map[string]string
}
