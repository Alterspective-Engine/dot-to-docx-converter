package api

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/alterspective-engine/dot-to-docx-converter/internal/queue"
	"github.com/alterspective-engine/dot-to-docx-converter/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// ConvertRequest represents a single file conversion request
type ConvertRequest struct {
	Priority int               `json:"priority" form:"priority"`
	Metadata map[string]string `json:"metadata" form:"metadata"`
}

// BatchConvertRequest represents a batch conversion request
type BatchConvertRequest struct {
	Source      string   `json:"source" binding:"required"`
	Destination string   `json:"destination" binding:"required"`
	Files       []string `json:"files" binding:"required"`
	Priority    int      `json:"priority"`
}

// JobResponse represents a job response
type JobResponse struct {
	JobID       string    `json:"job_id"`
	Status      string    `json:"status"`
	InputPath   string    `json:"input_path"`
	OutputPath  string    `json:"output_path,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Duration    string    `json:"duration,omitempty"`
	Error       string    `json:"error,omitempty"`
	DownloadURL string    `json:"download_url,omitempty"`
}

// ConvertHandler handles single file conversion
func ConvertHandler(q queue.Queue, s storage.Storage) gin.HandlerFunc {
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

		// Parse request options
		var req ConvertRequest
		if err := c.ShouldBind(&req); err != nil {
			log.Warnf("Failed to parse request options: %v", err)
		}

		// Generate job ID
		jobID := uuid.New().String()

		// Save uploaded file
		inputPath := fmt.Sprintf("uploads/%s/%s", jobID, header.Filename)

		// Save file to storage (handles both local and Azure)
		data, err := io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
			return
		}

		// WriteFile handles upload to Azure automatically if Azure Storage is configured
		if err := s.WriteFile(inputPath, data); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
			log.Errorf("Failed to save file for job %s: %v", jobID, err)
			return
		}

		// Create output path
		outputFilename := header.Filename[:len(header.Filename)-len(ext)] + ".docx"
		outputPath := fmt.Sprintf("outputs/%s/%s", jobID, outputFilename)

		// Create job
		job := &queue.Job{
			ID:         jobID,
			InputPath:  inputPath,
			OutputPath: outputPath,
			Status:     queue.StatusPending,
			Priority:   req.Priority,
			CreatedAt:  time.Now(),
			Metadata:   req.Metadata,
		}

		// Add to queue
		if err := q.Enqueue(c, job); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to queue job"})
			return
		}

		// Return job response
		response := JobResponse{
			JobID:      job.ID,
			Status:     job.Status,
			InputPath:  job.InputPath,
			OutputPath: job.OutputPath,
			CreatedAt:  job.CreatedAt,
		}

		c.JSON(http.StatusAccepted, response)
	}
}

// BatchConvertHandler handles batch conversion requests
func BatchConvertHandler(q queue.Queue, s storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req BatchConvertRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate files
		if len(req.Files) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no files specified"})
			return
		}

		if len(req.Files) > 1000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "maximum 1000 files per batch"})
			return
		}

		// Create batch ID
		batchID := uuid.New().String()
		jobs := make([]JobResponse, 0, len(req.Files))

		// Create jobs for each file
		for _, file := range req.Files {
			// Validate extension
			ext := filepath.Ext(file)
			if ext != ".dot" && ext != ".DOT" {
				log.Warnf("Skipping non-.dot file: %s", file)
				continue
			}

			// Generate job ID
			jobID := uuid.New().String()

			// Create paths
			inputPath := filepath.Join(req.Source, file)
			outputFile := file[:len(file)-len(ext)] + ".docx"
			outputPath := filepath.Join(req.Destination, outputFile)

			// Create job
			job := &queue.Job{
				ID:        jobID,
				InputPath: inputPath,
				OutputPath: outputPath,
				Status:    queue.StatusPending,
				Priority:  req.Priority,
				CreatedAt: time.Now(),
				Metadata: map[string]string{
					"batch_id": batchID,
					"filename": file,
				},
			}

			// Add to queue
			if err := q.Enqueue(c, job); err != nil {
				log.Errorf("Failed to queue job for %s: %v", file, err)
				continue
			}

			jobs = append(jobs, JobResponse{
				JobID:      job.ID,
				Status:     job.Status,
				InputPath:  job.InputPath,
				OutputPath: job.OutputPath,
				CreatedAt:  job.CreatedAt,
			})
		}

		c.JSON(http.StatusAccepted, gin.H{
			"batch_id": batchID,
			"jobs":     jobs,
			"count":    len(jobs),
		})
	}
}

// GetJobStatus retrieves job status
func GetJobStatus(q queue.Queue) gin.HandlerFunc {
	return func(c *gin.Context) {
		jobID := c.Param("id")

		job, err := q.GetJob(c, jobID)
		if err != nil {
			if err == queue.ErrJobNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		response := JobResponse{
			JobID:       job.ID,
			Status:      job.Status,
			InputPath:   job.InputPath,
			OutputPath:  job.OutputPath,
			CreatedAt:   job.CreatedAt,
			StartedAt:   job.StartedAt,
			CompletedAt: job.CompletedAt,
			Error:       job.Error,
		}

		if job.Duration > 0 {
			response.Duration = job.Duration.String()
		}

		if job.Status == queue.StatusCompleted {
			response.DownloadURL = fmt.Sprintf("/api/v1/download/%s", job.ID)
		}

		c.JSON(http.StatusOK, response)
	}
}

// ListJobs lists all jobs
func ListJobs(q queue.Queue) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.Query("status")
		limit := 100 // Default limit

		jobs, err := q.ListJobs(c, status, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		responses := make([]JobResponse, 0, len(jobs))
		for _, job := range jobs {
			response := JobResponse{
				JobID:       job.ID,
				Status:      job.Status,
				InputPath:   job.InputPath,
				OutputPath:  job.OutputPath,
				CreatedAt:   job.CreatedAt,
				StartedAt:   job.StartedAt,
				CompletedAt: job.CompletedAt,
				Error:       job.Error,
			}

			if job.Duration > 0 {
				response.Duration = job.Duration.String()
			}

			responses = append(responses, response)
		}

		c.JSON(http.StatusOK, gin.H{
			"jobs":  responses,
			"count": len(responses),
		})
	}
}

// CancelJob cancels a pending job
func CancelJob(q queue.Queue) gin.HandlerFunc {
	return func(c *gin.Context) {
		jobID := c.Param("id")

		if err := q.CancelJob(c, jobID); err != nil {
			if err == queue.ErrJobNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "job cancelled"})
	}
}

// DownloadFile downloads the converted file
func DownloadFile(s storage.Storage) gin.HandlerFunc {
	return func(c *gin.Context) {
		jobID := c.Param("id")

		// For now, construct the path based on job ID
		// In production, you'd look up the job to get the actual output path
		outputPath := fmt.Sprintf("outputs/%s/", jobID)

		// Find the file
		files, err := s.List(c, outputPath)
		if err != nil || len(files) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}

		// Download the first DOCX file found
		for _, file := range files {
			if filepath.Ext(file) == ".docx" {
				data, err := s.ReadFile(c, file)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
					return
				}

				filename := filepath.Base(file)
				c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
				c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", data)
				return
			}
		}

		c.JSON(http.StatusNotFound, gin.H{"error": "converted file not found"})
	}
}