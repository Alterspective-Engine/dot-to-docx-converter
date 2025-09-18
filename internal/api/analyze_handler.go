package api

import (
	"io"
	"net/http"
	"path/filepath"

	"github.com/alterspective-engine/dot-to-docx-converter/internal/analyzer"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// AnalyzeHandler handles document complexity analysis without conversion
func AnalyzeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse multipart form
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
			return
		}
		defer file.Close()

		// Validate file extension
		ext := filepath.Ext(header.Filename)
		if ext != ".dot" && ext != ".DOT" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "only .dot files are supported"})
			return
		}

		// Read file data
		fileData, err := io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
			return
		}

		// Analyze document complexity
		complexityReport := analyzer.AnalyzeComplexity(fileData)

		log.Infof("Document analysis for %s: Level=%s, Score=%d, NeedsReview=%v",
			header.Filename, complexityReport.Level, complexityReport.Score, complexityReport.NeedsReview)

		// Return complexity analysis
		c.JSON(http.StatusOK, gin.H{
			"filename": header.Filename,
			"size": header.Size,
			"complexity_report": complexityReport,
		})
	}
}

// AnalyzeBatchRequest represents a batch analysis request
type AnalyzeBatchRequest struct {
	Files []string `json:"files" binding:"required"`
}

// AnalyzeBatchResponse represents the response for a single file analysis in a batch
type AnalyzeBatchResponse struct {
	Filename string                     `json:"filename"`
	Error    string                     `json:"error,omitempty"`
	Report   *analyzer.ComplexityReport `json:"complexity_report,omitempty"`
}

// AnalyzeBatchHandler analyzes multiple files for complexity
func AnalyzeBatchHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse the request
		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid multipart form"})
			return
		}

		files := form.File["files"]
		if len(files) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no files provided"})
			return
		}

		if len(files) > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "maximum 100 files per batch"})
			return
		}

		results := make([]AnalyzeBatchResponse, 0, len(files))

		// Process each file
		for _, fileHeader := range files {
			result := AnalyzeBatchResponse{
				Filename: fileHeader.Filename,
			}

			// Validate extension
			ext := filepath.Ext(fileHeader.Filename)
			if ext != ".dot" && ext != ".DOT" {
				result.Error = "only .dot files are supported"
				results = append(results, result)
				continue
			}

			// Open the file
			file, err := fileHeader.Open()
			if err != nil {
				result.Error = "failed to open file"
				results = append(results, result)
				continue
			}

			// Read file data
			fileData, err := io.ReadAll(file)
			file.Close()
			if err != nil {
				result.Error = "failed to read file"
				results = append(results, result)
				continue
			}

			// Analyze complexity
			result.Report = analyzer.AnalyzeComplexity(fileData)
			results = append(results, result)

			log.Infof("Batch analysis for %s: Level=%s, Score=%d, NeedsReview=%v",
				fileHeader.Filename, result.Report.Level, result.Report.Score, result.Report.NeedsReview)
		}

		// Summary statistics
		var totalFiles = len(results)
		var needsReview = 0
		var criticalCount = 0
		var highCount = 0
		var mediumCount = 0
		var lowCount = 0

		for _, result := range results {
			if result.Report != nil {
				if result.Report.NeedsReview {
					needsReview++
				}
				switch result.Report.Level {
				case "critical":
					criticalCount++
				case "high":
					highCount++
				case "medium":
					mediumCount++
				case "low":
					lowCount++
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"results": results,
			"summary": gin.H{
				"total_files": totalFiles,
				"needs_review": needsReview,
				"complexity_distribution": gin.H{
					"critical": criticalCount,
					"high": highCount,
					"medium": mediumCount,
					"low": lowCount,
				},
			},
		})
	}
}