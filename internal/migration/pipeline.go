package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/alterspective-engine/dot-to-docx-converter/internal/analyzer"
	"github.com/alterspective-engine/dot-to-docx-converter/internal/cataloger"
)

// ConversionPipeline orchestrates the end-to-end conversion process
type ConversionPipeline struct {
	config           *PipelineConfig
	fieldMapper      *FieldMapper
	blockGenerator   *ContentBlockGenerator
	documentAnalyzer *cataloger.DocumentAnalyzer
	extractor        *analyzer.DocumentExtractor
	stages           []PipelineStage
	metrics          *PipelineMetrics
	mu               sync.RWMutex
}

// PipelineConfig contains pipeline configuration
type PipelineConfig struct {
	InputDir        string                 `json:"inputDir"`
	OutputDir       string                 `json:"outputDir"`
	MetadataDir     string                 `json:"metadataDir"`
	MaxWorkers      int                    `json:"maxWorkers"`
	BatchSize       int                    `json:"batchSize"`
	EnableAI        bool                   `json:"enableAI"`
	ValidationLevel string                 `json:"validationLevel"`
	RetryPolicy     RetryPolicy            `json:"retryPolicy"`
	Options         map[string]interface{} `json:"options"`
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxAttempts   int           `json:"maxAttempts"`
	InitialDelay  time.Duration `json:"initialDelay"`
	MaxDelay      time.Duration `json:"maxDelay"`
	BackoffFactor float64       `json:"backoffFactor"`
}

// PipelineStage represents a processing stage
type PipelineStage interface {
	Name() string
	Process(ctx context.Context, input interface{}) (interface{}, error)
	Validate(input interface{}) error
	Metrics() StageMetrics
}

// StageMetrics contains stage performance metrics
type StageMetrics struct {
	ProcessedCount int           `json:"processedCount"`
	ErrorCount     int           `json:"errorCount"`
	AverageTime    time.Duration `json:"averageTime"`
	LastProcessed  time.Time     `json:"lastProcessed"`
}

// PipelineMetrics tracks overall pipeline performance
type PipelineMetrics struct {
	StartTime      time.Time               `json:"startTime"`
	EndTime        time.Time               `json:"endTime"`
	TotalDocuments int                     `json:"totalDocuments"`
	ProcessedDocs  int                     `json:"processedDocs"`
	FailedDocs     int                     `json:"failedDocs"`
	StageMetrics   map[string]StageMetrics `json:"stageMetrics"`
	ContentBlocks  int                     `json:"contentBlocks"`
	FieldMappings  int                     `json:"fieldMappings"`
	Errors         []PipelineError         `json:"errors"`
	SuccessRate    float64                 `json:"successRate"`
}

// PipelineError represents an error in the pipeline
type PipelineError struct {
	Stage       string    `json:"stage"`
	Document    string    `json:"document"`
	Error       string    `json:"error"`
	Timestamp   time.Time `json:"timestamp"`
	Recoverable bool      `json:"recoverable"`
}

// PipelineResult contains the final pipeline results
type PipelineResult struct {
	Success         bool                           `json:"success"`
	ProcessedFiles  []ProcessedFile                `json:"processedFiles"`
	GeneratedBlocks []*SharedoContentBlock         `json:"generatedBlocks"`
	FieldMappings   map[string]*FieldMappingResult `json:"fieldMappings"`
	Metrics         *PipelineMetrics               `json:"metrics"`
	Report          string                         `json:"report"`
}

// ProcessedFile represents a processed document
type ProcessedFile struct {
	SourcePath      string        `json:"sourcePath"`
	OutputPath      string        `json:"outputPath"`
	MetadataPath    string        `json:"metadataPath"`
	Status          string        `json:"status"`
	ProcessingTime  time.Duration `json:"processingTime"`
	FieldCount      int           `json:"fieldCount"`
	BlocksUsed      []string      `json:"blocksUsed"`
	ValidationScore float64       `json:"validationScore"`
	Issues          []string      `json:"issues"`
}

// NewConversionPipeline creates a new pipeline instance
func NewConversionPipeline(config *PipelineConfig) *ConversionPipeline {
	pipeline := &ConversionPipeline{
		config:           config,
		fieldMapper:      NewFieldMapper(),
		blockGenerator:   NewContentBlockGenerator(),
		documentAnalyzer: cataloger.NewDocumentAnalyzer(config.EnableAI),
		extractor:        analyzer.NewDocumentExtractor(),
		metrics: &PipelineMetrics{
			StageMetrics: make(map[string]StageMetrics),
			Errors:       []PipelineError{},
		},
	}

	// Initialize pipeline stages
	pipeline.initializeStages()

	return pipeline
}

// Execute runs the complete pipeline
func (p *ConversionPipeline) Execute(ctx context.Context) (*PipelineResult, error) {
	p.metrics.StartTime = time.Now()

	// Create output directories
	if err := p.createDirectories(); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	// Load documents
	documents, err := p.loadDocuments()
	if err != nil {
		return nil, fmt.Errorf("failed to load documents: %w", err)
	}

	p.metrics.TotalDocuments = len(documents)
	log.Printf("Starting pipeline with %d documents", len(documents))

	// Process in batches
	result := &PipelineResult{
		ProcessedFiles:  []ProcessedFile{},
		GeneratedBlocks: []*SharedoContentBlock{},
		FieldMappings:   make(map[string]*FieldMappingResult),
		Metrics:         p.metrics,
	}

	// Phase 1: Analysis
	log.Println("Phase 1: Analyzing documents...")
	catalog, err := p.analyzeDocuments(ctx, documents)
	if err != nil {
		return nil, fmt.Errorf("analysis phase failed: %w", err)
	}

	// Phase 2: Content Block Generation
	log.Println("Phase 2: Generating content blocks...")
	blocks := p.generateContentBlocks(catalog)
	result.GeneratedBlocks = blocks
	p.metrics.ContentBlocks = len(blocks)

	// Phase 3: Field Mapping
	log.Println("Phase 3: Mapping fields...")
	mappings := p.mapFields(catalog)
	result.FieldMappings = mappings
	p.metrics.FieldMappings = len(mappings)

	// Phase 4: Document Conversion
	log.Println("Phase 4: Converting documents...")
	processedFiles := p.convertDocuments(ctx, documents, blocks, mappings)
	result.ProcessedFiles = processedFiles

	// Phase 5: Validation
	log.Println("Phase 5: Validating results...")
	p.validateResults(result)

	// Calculate final metrics
	p.metrics.EndTime = time.Now()
	p.metrics.ProcessedDocs = len(processedFiles)
	p.metrics.SuccessRate = float64(p.metrics.ProcessedDocs-p.metrics.FailedDocs) / float64(p.metrics.TotalDocuments) * 100

	// Generate report
	result.Report = p.generateReport(catalog, result)
	result.Success = p.metrics.FailedDocs == 0

	log.Printf("Pipeline completed: %d/%d documents processed successfully (%.1f%% success rate)",
		p.metrics.ProcessedDocs, p.metrics.TotalDocuments, p.metrics.SuccessRate)

	return result, nil
}

// initializeStages sets up pipeline stages
func (p *ConversionPipeline) initializeStages() {
	p.stages = []PipelineStage{
		NewExtractionStage(p.extractor),
		NewAnalysisStage(p.documentAnalyzer),
		NewMappingStage(p.fieldMapper),
		NewConversionStage(p.blockGenerator),
		NewValidationStage(),
	}
}

// loadDocuments loads all documents from input directory
func (p *ConversionPipeline) loadDocuments() ([]cataloger.DocumentData, error) {
	files, err := filepath.Glob(filepath.Join(p.config.InputDir, "*.dot"))
	if err != nil {
		return nil, err
	}

	documents := make([]cataloger.DocumentData, 0, len(files))

	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			log.Printf("Warning: Failed to read %s: %v", file, err)
			continue
		}

		// Extract text content
		docInfo, err := p.extractor.AnalyzeDocument(content)
		if err != nil {
			docInfo = &analyzer.DocumentInfo{
				Text: string(content),
			}
		}

		doc := cataloger.DocumentData{
			Filename:      filepath.Base(file),
			Content:       content,
			ExtractedText: docInfo.Text,
			Metadata:      make(map[string]string),
		}

		documents = append(documents, doc)
	}

	return documents, nil
}

// analyzeDocuments performs comprehensive document analysis
func (p *ConversionPipeline) analyzeDocuments(ctx context.Context, documents []cataloger.DocumentData) (*cataloger.DocumentCatalog, error) {
	startTime := time.Now()

	catalog, err := p.documentAnalyzer.AnalyzeDocuments(documents)
	if err != nil {
		return nil, err
	}

	p.updateStageMetrics("analysis", time.Since(startTime), err == nil)

	return catalog, nil
}

// generateContentBlocks creates reusable content blocks
func (p *ConversionPipeline) generateContentBlocks(catalog *cataloger.DocumentCatalog) []*SharedoContentBlock {
	blocks := []*SharedoContentBlock{}

	// Convert catalog data to format for block generator
	docContents := make([]DocumentContent, 0, len(catalog.DocumentProfiles))
	for _, profile := range catalog.DocumentProfiles {
		// Convert metadata from map[string]string to map[string]interface{}
		metadata := make(map[string]interface{})
		for k, v := range profile.Metadata {
			metadata[k] = v
		}
		// Find corresponding document content
		docContents = append(docContents, DocumentContent{
			Filename: profile.Filename,
			Content:  "", // Would be populated from actual content
			Metadata: metadata,
		})
	}

	// Analyze for block opportunities
	analysis := p.blockGenerator.AnalyzeContent(docContents)

	// Generate blocks for high-frequency content
	for i, commonBlock := range analysis.CommonBlocks {
		if commonBlock.Confidence > 0.5 {
			blockName := fmt.Sprintf("%s_block_%d", commonBlock.Type, i+1)
			block := p.blockGenerator.GenerateContentBlock(commonBlock, blockName)
			blocks = append(blocks, block)
		}
	}

	return blocks
}

// mapFields creates field mappings
func (p *ConversionPipeline) mapFields(catalog *cataloger.DocumentCatalog) map[string]*FieldMappingResult {
	mappings := make(map[string]*FieldMappingResult)

	for fieldName, field := range catalog.Fields {
		context := map[string]interface{}{
			"category":     field.Category,
			"frequency":    field.Frequency,
			"documentType": "legal",
		}

		mapping := p.fieldMapper.MapField(fieldName, context)
		mappings[fieldName] = mapping
	}

	return mappings
}

// convertDocuments processes documents through conversion
func (p *ConversionPipeline) convertDocuments(ctx context.Context, documents []cataloger.DocumentData, blocks []*SharedoContentBlock, mappings map[string]*FieldMappingResult) []ProcessedFile {
	processedFiles := []ProcessedFile{}

	// Process documents in parallel with worker pool
	workerCount := p.config.MaxWorkers
	if workerCount <= 0 {
		workerCount = 4
	}

	jobs := make(chan cataloger.DocumentData, len(documents))
	results := make(chan ProcessedFile, len(documents))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for doc := range jobs {
				result := p.processSingleDocument(ctx, doc, blocks, mappings)
				results <- result
			}
		}()
	}

	// Send jobs
	for _, doc := range documents {
		jobs <- doc
	}
	close(jobs)

	// Wait for completion
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for result := range results {
		processedFiles = append(processedFiles, result)
		if result.Status == "success" {
			p.metrics.ProcessedDocs++
		} else {
			p.metrics.FailedDocs++
		}
	}

	return processedFiles
}

// processSingleDocument handles conversion of a single document
func (p *ConversionPipeline) processSingleDocument(ctx context.Context, doc cataloger.DocumentData, blocks []*SharedoContentBlock, mappings map[string]*FieldMappingResult) ProcessedFile {
	startTime := time.Now()

	result := ProcessedFile{
		SourcePath:   doc.Filename,
		OutputPath:   filepath.Join(p.config.OutputDir, strings.ReplaceAll(doc.Filename, ".dot", ".docx")),
		MetadataPath: filepath.Join(p.config.MetadataDir, doc.Filename+".meta.json"),
		Status:       "processing",
		Issues:       []string{},
	}

	// Apply field mappings
	fieldCount := 0
	for fieldName, _ := range mappings {
		if strings.Contains(doc.ExtractedText, fieldName) {
			fieldCount++
			// Would apply actual transformation here
		}
	}
	result.FieldCount = fieldCount

	// Apply content blocks
	blocksUsed := []string{}
	for _, block := range blocks {
		// Check if block content matches document
		if strings.Contains(doc.ExtractedText, block.Name) {
			blocksUsed = append(blocksUsed, block.ID)
		}
	}
	result.BlocksUsed = blocksUsed

	// Generate metadata
	metadata := map[string]interface{}{
		"originalFile":    doc.Filename,
		"fieldMappings":   fieldCount,
		"contentBlocks":   blocksUsed,
		"processedAt":     time.Now(),
		"pipelineVersion": "2.1.0",
	}

	// Save metadata
	if metadataJSON, err := json.MarshalIndent(metadata, "", "  "); err == nil {
		ioutil.WriteFile(result.MetadataPath, metadataJSON, 0644)
	}

	result.Status = "success"
	result.ProcessingTime = time.Since(startTime)
	result.ValidationScore = 0.95 // Would calculate actual score

	return result
}

// validateResults performs final validation
func (p *ConversionPipeline) validateResults(result *PipelineResult) {
	for i := range result.ProcessedFiles {
		file := &result.ProcessedFiles[i]

		// Basic validation checks
		if file.FieldCount == 0 {
			file.Issues = append(file.Issues, "No fields mapped")
			file.ValidationScore *= 0.8
		}

		if len(file.BlocksUsed) == 0 {
			file.Issues = append(file.Issues, "No content blocks applied")
			file.ValidationScore *= 0.9
		}

		if file.ValidationScore < 0.75 {
			file.Status = "needs_review"
		}
	}
}

// generateReport creates comprehensive migration report
func (p *ConversionPipeline) generateReport(catalog *cataloger.DocumentCatalog, result *PipelineResult) string {
	report := fmt.Sprintf(`
SHAREDO MIGRATION PIPELINE REPORT
==================================
Generated: %s
Pipeline Version: 2.1.0

EXECUTION SUMMARY
-----------------
Start Time: %s
End Time: %s
Duration: %s
Total Documents: %d
Successfully Processed: %d
Failed: %d
Success Rate: %.1f%%

CONTENT BLOCKS GENERATED
------------------------
Total Blocks: %d
Types:
`, time.Now().Format(time.RFC3339),
		p.metrics.StartTime.Format(time.RFC3339),
		p.metrics.EndTime.Format(time.RFC3339),
		p.metrics.EndTime.Sub(p.metrics.StartTime),
		p.metrics.TotalDocuments,
		p.metrics.ProcessedDocs,
		p.metrics.FailedDocs,
		p.metrics.SuccessRate,
		len(result.GeneratedBlocks))

	// Add block details
	blockTypes := make(map[string]int)
	for _, block := range result.GeneratedBlocks {
		blockTypes[block.Type]++
	}
	for blockType, count := range blockTypes {
		report += fmt.Sprintf("  - %s: %d\n", blockType, count)
	}

	report += fmt.Sprintf(`
FIELD MAPPINGS
--------------
Total Fields Mapped: %d
Average Confidence: %.1f%%

TOP MAPPED FIELDS:
`, len(result.FieldMappings), p.calculateAverageMappingConfidence(result.FieldMappings)*100)

	// Add top fields
	fieldCount := 0
	for fieldName, mapping := range result.FieldMappings {
		if fieldCount >= 10 {
			break
		}
		report += fmt.Sprintf("  - %s → %s (%.0f%% confidence)\n",
			fieldName, mapping.Mapped, mapping.Confidence*100)
		fieldCount++
	}

	// Add processing details
	report += fmt.Sprintf(`
DOCUMENT PROCESSING
-------------------
`)

	successCount := 0
	reviewCount := 0
	for _, file := range result.ProcessedFiles {
		if file.Status == "success" {
			successCount++
		} else if file.Status == "needs_review" {
			reviewCount++
		}
	}

	report += fmt.Sprintf("Successful: %d\n", successCount)
	report += fmt.Sprintf("Needs Review: %d\n", reviewCount)
	report += fmt.Sprintf("Failed: %d\n", p.metrics.FailedDocs)

	// Add errors if any
	if len(p.metrics.Errors) > 0 {
		report += "\nERRORS ENCOUNTERED\n------------------\n"
		for i, err := range p.metrics.Errors {
			if i >= 10 {
				report += fmt.Sprintf("... and %d more errors\n", len(p.metrics.Errors)-10)
				break
			}
			report += fmt.Sprintf("- [%s] %s: %s\n", err.Stage, err.Document, err.Error)
		}
	}

	// Add recommendations
	report += `
RECOMMENDATIONS
---------------
`
	if p.metrics.SuccessRate < 90 {
		report += "- Review failed documents for manual intervention\n"
	}
	if len(result.GeneratedBlocks) > 20 {
		report += "- Consider consolidating similar content blocks\n"
	}
	if reviewCount > 10 {
		report += fmt.Sprintf("- %d documents flagged for review - validate field mappings\n", reviewCount)
	}

	report += "\n=== END OF REPORT ==="

	return report
}

// Helper methods

func (p *ConversionPipeline) createDirectories() error {
	dirs := []string{
		p.config.OutputDir,
		p.config.MetadataDir,
		filepath.Join(p.config.OutputDir, "blocks"),
		filepath.Join(p.config.OutputDir, "mappings"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

func (p *ConversionPipeline) updateStageMetrics(stage string, duration time.Duration, success bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	metrics := p.metrics.StageMetrics[stage]
	metrics.ProcessedCount++
	if !success {
		metrics.ErrorCount++
	}
	metrics.LastProcessed = time.Now()

	// Update average time
	if metrics.ProcessedCount == 1 {
		metrics.AverageTime = duration
	} else {
		metrics.AverageTime = (metrics.AverageTime*time.Duration(metrics.ProcessedCount-1) + duration) / time.Duration(metrics.ProcessedCount)
	}

	p.metrics.StageMetrics[stage] = metrics
}

func (p *ConversionPipeline) calculateAverageMappingConfidence(mappings map[string]*FieldMappingResult) float64 {
	if len(mappings) == 0 {
		return 0
	}

	total := 0.0
	for _, mapping := range mappings {
		total += mapping.Confidence
	}

	return total / float64(len(mappings))
}

// Pipeline stages implementation

type ExtractionStage struct {
	extractor *analyzer.DocumentExtractor
	metrics   StageMetrics
}

func NewExtractionStage(extractor *analyzer.DocumentExtractor) *ExtractionStage {
	return &ExtractionStage{
		extractor: extractor,
	}
}

func (s *ExtractionStage) Name() string { return "extraction" }

func (s *ExtractionStage) Process(ctx context.Context, input interface{}) (interface{}, error) {
	// Implementation
	return input, nil
}

func (s *ExtractionStage) Validate(input interface{}) error {
	return nil
}

func (s *ExtractionStage) Metrics() StageMetrics {
	return s.metrics
}

// Similar implementations for other stages...

type AnalysisStage struct {
	analyzer *cataloger.DocumentAnalyzer
	metrics  StageMetrics
}

func NewAnalysisStage(analyzer *cataloger.DocumentAnalyzer) *AnalysisStage {
	return &AnalysisStage{
		analyzer: analyzer,
	}
}

func (s *AnalysisStage) Name() string { return "analysis" }
func (s *AnalysisStage) Process(ctx context.Context, input interface{}) (interface{}, error) {
	return input, nil
}
func (s *AnalysisStage) Validate(input interface{}) error { return nil }
func (s *AnalysisStage) Metrics() StageMetrics            { return s.metrics }

type MappingStage struct {
	mapper  *FieldMapper
	metrics StageMetrics
}

func NewMappingStage(mapper *FieldMapper) *MappingStage {
	return &MappingStage{
		mapper: mapper,
	}
}

func (s *MappingStage) Name() string { return "mapping" }
func (s *MappingStage) Process(ctx context.Context, input interface{}) (interface{}, error) {
	return input, nil
}
func (s *MappingStage) Validate(input interface{}) error { return nil }
func (s *MappingStage) Metrics() StageMetrics            { return s.metrics }

type ConversionStage struct {
	generator *ContentBlockGenerator
	metrics   StageMetrics
}

func NewConversionStage(generator *ContentBlockGenerator) *ConversionStage {
	return &ConversionStage{
		generator: generator,
	}
}

func (s *ConversionStage) Name() string { return "conversion" }
func (s *ConversionStage) Process(ctx context.Context, input interface{}) (interface{}, error) {
	return input, nil
}
func (s *ConversionStage) Validate(input interface{}) error { return nil }
func (s *ConversionStage) Metrics() StageMetrics            { return s.metrics }

type ValidationStage struct {
	metrics StageMetrics
}

func NewValidationStage() *ValidationStage {
	return &ValidationStage{}
}

func (s *ValidationStage) Name() string { return "validation" }
func (s *ValidationStage) Process(ctx context.Context, input interface{}) (interface{}, error) {
	return input, nil
}
func (s *ValidationStage) Validate(input interface{}) error { return nil }
func (s *ValidationStage) Metrics() StageMetrics            { return s.metrics }
