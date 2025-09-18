package api

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/alterspective-engine/dot-to-docx-converter/internal/cataloger"
	"github.com/alterspective-engine/dot-to-docx-converter/internal/migration"
	"github.com/gin-gonic/gin"
)

// MigrationAnalyzeRequest represents the request for migration analysis
type MigrationAnalyzeRequest struct {
	Documents []DocumentInput `json:"documents"`
	Options   AnalysisOptions `json:"options,omitempty"`
}

// DocumentInput represents a document for analysis
type DocumentInput struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AnalysisOptions configures the analysis
type AnalysisOptions struct {
	EnableAI           bool    `json:"enable_ai,omitempty"`
	ConfidenceThreshold float64 `json:"confidence_threshold,omitempty"`
	IncludeSamples     bool    `json:"include_samples,omitempty"`
}

// MigrationAnalyzeResponse contains the analysis results
type MigrationAnalyzeResponse struct {
	Success   bool                      `json:"success"`
	Catalog   *cataloger.DocumentCatalog `json:"catalog,omitempty"`
	Report    string                     `json:"report"`
	Error     string                     `json:"error,omitempty"`
}

// FieldMappingRequest represents field mapping request
type FieldMappingRequest struct {
	Fields  []string               `json:"fields"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// FieldMappingResponse contains mapping results
type FieldMappingResponse struct {
	Success  bool                                       `json:"success"`
	Mappings map[string]*migration.FieldMappingResult `json:"mappings"`
	Stats    map[string]interface{}                   `json:"stats,omitempty"`
}

// ContentBlockRequest for generating content blocks
type ContentBlockRequest struct {
	Documents []DocumentInput `json:"documents"`
	Options   BlockOptions    `json:"options,omitempty"`
}

// BlockOptions configures block generation
type BlockOptions struct {
	MinFrequency int     `json:"min_frequency,omitempty"`
	MinSimilarity float64 `json:"min_similarity,omitempty"`
}

// ContentBlockResponse contains generated blocks
type ContentBlockResponse struct {
	Success bool                               `json:"success"`
	Blocks  []*migration.SharedoContentBlock `json:"blocks"`
	Stats   map[string]interface{}           `json:"stats,omitempty"`
}

// LearnMappingRequest for teaching the system
type LearnMappingRequest struct {
	Original   string  `json:"original"`
	Corrected  string  `json:"corrected"`
	Confidence float64 `json:"confidence,omitempty"`
}

// PipelineRequest for full migration pipeline
type PipelineRequest struct {
	InputPath   string                 `json:"input_path"`
	OutputPath  string                 `json:"output_path,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

// MigrationAnalyzeHandler analyzes documents for migration
func MigrationAnalyzeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MigrationAnalyzeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, MigrationAnalyzeResponse{
				Success: false,
				Error:   err.Error(),
			})
			return
		}

		// Convert input to cataloger documents
		documents := make([]cataloger.DocumentData, len(req.Documents))
		for i, doc := range req.Documents {
			// Convert metadata from interface{} to string
			metadata := make(map[string]string)
			for k, v := range doc.Metadata {
				if str, ok := v.(string); ok {
					metadata[k] = str
				}
			}

			documents[i] = cataloger.DocumentData{
				Filename:      doc.Filename,
				ExtractedText: doc.Content,
				Metadata:      metadata,
			}
		}

		// Analyze documents
		analyzer := cataloger.NewDocumentAnalyzer(req.Options.EnableAI)

		catalog, err := analyzer.AnalyzeDocuments(documents)
		if err != nil {
			c.JSON(http.StatusInternalServerError, MigrationAnalyzeResponse{
				Success: false,
				Error:   err.Error(),
			})
			return
		}

		// Generate report
		report := generateCatalogReport(catalog)

		c.JSON(http.StatusOK, MigrationAnalyzeResponse{
			Success: true,
			Catalog: catalog,
			Report:  report,
		})
	}
}

// FieldMappingHandler maps fields to Sharedo format
func FieldMappingHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req FieldMappingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, FieldMappingResponse{
				Success: false,
			})
			return
		}

		mapper := migration.NewFieldMapper()
		mappings := mapper.BatchMapFields(req.Fields, req.Context)
		stats := mapper.GetMappingStatistics()

		c.JSON(http.StatusOK, FieldMappingResponse{
			Success:  true,
			Mappings: mappings,
			Stats:    stats,
		})
	}
}

// ContentBlockHandler generates content blocks
func ContentBlockHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ContentBlockRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, ContentBlockResponse{
				Success: false,
			})
			return
		}

		// Convert to generator format
		documents := make([]migration.DocumentContent, len(req.Documents))
		for i, doc := range req.Documents {
			documents[i] = migration.DocumentContent{
				Filename: doc.Filename,
				Content:  doc.Content,
				Metadata: doc.Metadata,
			}
		}

		generator := migration.NewContentBlockGenerator()
		// Note: MinFrequency configuration would be added to generator if needed

		// Generate blocks
		analysis := generator.AnalyzeContent(documents)
		blocks := make([]*migration.SharedoContentBlock, 0)

		for _, common := range analysis.CommonBlocks {
			blockName := filepath.Base(common.Content)
			if blockName == "" {
				blockName = "Block"
			}
			block := generator.GenerateContentBlock(common, blockName)
			blocks = append(blocks, block)
		}

		c.JSON(http.StatusOK, ContentBlockResponse{
			Success: true,
			Blocks:  blocks,
			Stats:   generator.GetStatistics(),
		})
	}
}

// LearnMappingHandler teaches the system field mappings
func LearnMappingHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LearnMappingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		mapper := migration.NewFieldMapper()
		confidence := req.Confidence
		if confidence == 0 {
			confidence = 0.9
		}

		mapper.LearnFromCorrection(req.Original, req.Corrected, confidence)

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Mapping learned successfully",
		})
	}
}

// MigrationPipelineHandler runs the full migration pipeline
func MigrationPipelineHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req PipelineRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		// Create pipeline config
		config := &migration.PipelineConfig{
			InputDir:  req.InputPath,
			OutputDir: req.OutputPath,
			Options:   req.Options,
		}

		if config.OutputDir == "" {
			config.OutputDir = "./output"
		}

		// Note: This would run the pipeline in production
		// For now, return configuration acknowledgment
		c.JSON(http.StatusAccepted, gin.H{
			"success": true,
			"message": "Migration pipeline configured",
			"config":  config,
			"note":    "Pipeline execution would start in production mode",
		})
	}
}

// MigrationPlanHandler returns the implementation plan
func MigrationPlanHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		plan := migration.CreateImplementationPlan()
		quality, recommendations := plan.EvaluateQuality()
		report := plan.GenerateReport()

		c.JSON(http.StatusOK, gin.H{
			"success":         true,
			"quality_score":   quality,
			"recommendations": recommendations,
			"report":          report,
			"rubrics":         plan.Rubrics,
			"priorities":      plan.Priorities,
			"timeline":        plan.Timeline,
		})
	}
}

// MigrationStatsHandler returns migration system statistics
func MigrationStatsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		mapper := migration.NewFieldMapper()
		generator := migration.NewContentBlockGenerator()

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"stats": gin.H{
				"field_mapper":    mapper.GetMappingStatistics(),
				"content_blocks":  generator.GetStatistics(),
				"system_version":  "2.0.0",
				"capabilities": []string{
					"field_extraction",
					"pattern_recognition",
					"content_block_generation",
					"intelligent_mapping",
					"learning_system",
					"pipeline_orchestration",
				},
			},
		})
	}
}

// generateCatalogReport creates a human-readable report from the document catalog
func generateCatalogReport(catalog *cataloger.DocumentCatalog) string {
	if catalog == nil {
		return "No catalog data available"
	}

	report := fmt.Sprintf("Document Analysis Report\n")
	report += fmt.Sprintf("========================\n\n")
	report += fmt.Sprintf("Analysis Date: %s\n", catalog.AnalysisDate.Format("2006-01-02 15:04:05"))
	report += fmt.Sprintf("Total Documents Analyzed: %d\n", catalog.TotalDocuments)
	report += fmt.Sprintf("Processing Time: %v\n\n", catalog.ProcessingTime)

	// Fields summary
	report += fmt.Sprintf("Fields Analysis:\n")
	report += fmt.Sprintf("- Total Unique Fields: %d\n", len(catalog.Fields))

	// Field categories
	categoryCount := make(map[cataloger.FieldCategory]int)
	for _, field := range catalog.Fields {
		categoryCount[field.Category]++
	}

	for category, count := range categoryCount {
		report += fmt.Sprintf("- %s Fields: %d\n", string(category), count)
	}
	report += "\n"

	// Content blocks summary
	report += fmt.Sprintf("Content Blocks:\n")
	report += fmt.Sprintf("- Total Content Blocks: %d\n", len(catalog.ContentBlocks))
	for blockType, block := range catalog.ContentBlocks {
		report += fmt.Sprintf("- %s (Frequency: %d)\n", blockType, block.Frequency)
	}
	report += "\n"

	// Complexity distribution
	report += fmt.Sprintf("Complexity Distribution:\n")
	for complexity, count := range catalog.ComplexityDist {
		report += fmt.Sprintf("- %s: %d documents\n", string(complexity), count)
	}
	report += "\n"

	// Jurisdictions
	if len(catalog.Jurisdictions) > 0 {
		report += fmt.Sprintf("Jurisdictions Found:\n")
		for jurisdiction, count := range catalog.Jurisdictions {
			report += fmt.Sprintf("- %s: %d documents\n", jurisdiction, count)
		}
		report += "\n"
	}

	// Matter types
	if len(catalog.MatterTypes) > 0 {
		report += fmt.Sprintf("Matter Types:\n")
		for matterType, count := range catalog.MatterTypes {
			report += fmt.Sprintf("- %s: %d documents\n", matterType, count)
		}
		report += "\n"
	}

	// Recommendations
	if len(catalog.Recommendations) > 0 {
		report += fmt.Sprintf("Recommendations:\n")
		for i, rec := range catalog.Recommendations {
			report += fmt.Sprintf("%d. %s\n", i+1, rec)
		}
	}

	return report
}