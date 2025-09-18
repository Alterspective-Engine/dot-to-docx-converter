package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/alterspective-engine/dot-to-docx-converter/internal/analyzer"
	"github.com/alterspective-engine/dot-to-docx-converter/internal/converter"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// SyncConvertRequest represents a synchronous conversion request
type SyncConvertRequest struct {
	WaitTimeout int `form:"timeout" json:"timeout"` // Max wait time in seconds (default: 30)
}

// ConvertSyncHandler handles synchronous file conversion
// This endpoint converts the file immediately and returns the result
// Suitable for smaller files and when immediate results are needed
func ConvertSyncHandler(conv converter.Converter, maxFileSize int64, syncTimeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

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

		// Check file size for sync processing
		if header.Size > maxFileSize {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("file too large for synchronous conversion (max: %d MB)", maxFileSize/(1024*1024)),
				"suggestion": "use async endpoint /api/v1/convert for large files",
			})
			return
		}

		// Parse request options
		var req SyncConvertRequest
		if err := c.ShouldBind(&req); err != nil {
			log.Warnf("Failed to parse sync request options: %v", err)
		}

		// Set default timeout if not specified
		timeout := syncTimeout
		if req.WaitTimeout > 0 && req.WaitTimeout <= 60 {
			timeout = time.Duration(req.WaitTimeout) * time.Second
		}

		// Generate unique ID for this conversion
		conversionID := uuid.New().String()

		// Create temporary directory for conversion
		tempDir := filepath.Join(os.TempDir(), "sync-conversion-"+conversionID)
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temp directory"})
			return
		}
		defer os.RemoveAll(tempDir)

		// Save uploaded file to temp location
		inputPath := filepath.Join(tempDir, header.Filename)
		inputFile, err := os.Create(inputPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temp file"})
			return
		}
		defer inputFile.Close()

		// Read file data for analysis
		fileData, err := io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read uploaded file"})
			return
		}

		// Write file data to temp location
		if _, err := inputFile.Write(fileData); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save uploaded file"})
			return
		}
		inputFile.Close()

		// Analyze document complexity
		complexityReport := analyzer.AnalyzeComplexity(fileData)
		log.Infof("Document complexity for sync conversion %s: Level=%s, Score=%d, NeedsReview=%v",
			conversionID, complexityReport.Level, complexityReport.Score, complexityReport.NeedsReview)

		// Set up output path
		outputFilename := header.Filename[:len(header.Filename)-len(ext)] + ".docx"
		outputPath := filepath.Join(tempDir, outputFilename)

		// Create context with timeout for conversion
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		// Perform synchronous conversion
		log.Infof("Starting synchronous conversion for file: %s (ID: %s)", header.Filename, conversionID)

		conversionErr := conv.Convert(ctx, inputPath, outputPath)
		if conversionErr != nil {
			// Check if it was a timeout
			if ctx.Err() == context.DeadlineExceeded {
				c.JSON(http.StatusRequestTimeout, gin.H{
					"error": "conversion timeout exceeded",
					"timeout": timeout.Seconds(),
					"suggestion": "use async endpoint /api/v1/convert for complex files",
				})
				return
			}

			log.Errorf("Synchronous conversion failed: %v", conversionErr)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "conversion failed",
				"details": conversionErr.Error(),
			})
			return
		}

		// Read the converted file
		convertedData, err := os.ReadFile(outputPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read converted file"})
			return
		}

		// Calculate processing time
		duration := time.Since(start)
		log.Infof("Synchronous conversion completed in %v: %s -> %s", duration, header.Filename, outputFilename)

		// Set response headers
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", outputFilename))
		c.Header("X-Conversion-Time", duration.String())
		c.Header("X-Conversion-ID", conversionID)
		c.Header("X-Complexity-Level", complexityReport.Level)
		c.Header("X-Complexity-Score", fmt.Sprintf("%d", complexityReport.Score))
		if complexityReport.NeedsReview {
			c.Header("X-Needs-Human-Review", "true")
		}

		// Return the converted file directly
		c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", convertedData)
	}
}

// ConvertSyncJSONHandler handles synchronous conversion with base64 encoded response
// This is useful for API clients that prefer JSON responses
func ConvertSyncJSONHandler(conv converter.Converter, maxFileSize int64, syncTimeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

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

		// Check file size
		if header.Size > maxFileSize {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("file too large for synchronous conversion (max: %d MB)", maxFileSize/(1024*1024)),
			})
			return
		}

		// Generate unique ID
		conversionID := uuid.New().String()

		// Create temp directory
		tempDir := filepath.Join(os.TempDir(), "sync-json-"+conversionID)
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temp directory"})
			return
		}
		defer os.RemoveAll(tempDir)

		// Save uploaded file
		inputPath := filepath.Join(tempDir, header.Filename)
		inputFile, err := os.Create(inputPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temp file"})
			return
		}

		// Read file data for analysis
		fileData, err := io.ReadAll(file)
		if err != nil {
			inputFile.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read uploaded file"})
			return
		}

		// Write file data to temp location
		if _, err := inputFile.Write(fileData); err != nil {
			inputFile.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save uploaded file"})
			return
		}
		inputFile.Close()

		// Analyze document complexity
		complexityReport := analyzer.AnalyzeComplexity(fileData)
		log.Infof("Document complexity for sync JSON conversion %s: Level=%s, Score=%d, NeedsReview=%v",
			conversionID, complexityReport.Level, complexityReport.Score, complexityReport.NeedsReview)

		// Set up output
		outputFilename := header.Filename[:len(header.Filename)-len(ext)] + ".docx"
		outputPath := filepath.Join(tempDir, outputFilename)

		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
		defer cancel()

		// Perform conversion
		if err := conv.Convert(ctx, inputPath, outputPath); err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				c.JSON(http.StatusRequestTimeout, gin.H{"error": "conversion timeout"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "conversion failed", "details": err.Error()})
			return
		}

		// Read converted file
		convertedData, err := os.ReadFile(outputPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read converted file"})
			return
		}

		duration := time.Since(start)

		// Return JSON response with file info and complexity report
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"conversion_id": conversionID,
			"filename": outputFilename,
			"size": len(convertedData),
			"duration": duration.String(),
			"download_url": fmt.Sprintf("/api/v1/sync/download/%s", conversionID),
			"complexity_report": complexityReport,
		})
	}
}