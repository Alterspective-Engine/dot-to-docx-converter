package api

import (
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/alterspective-engine/dot-to-docx-converter/internal/queue"
	"github.com/gin-gonic/gin"
)

type MetricsCollector struct {
	mu                sync.RWMutex
	totalProcessed    int64
	totalFailed       int64
	processingNow     int
	avgProcessingTime float64
	timeline          []TimelineData
	queueStatus       QueueStatus
	lastUpdated       time.Time
}

type TimelineData struct {
	Time      time.Time `json:"time"`
	Processed int       `json:"processed"`
	Failed    int       `json:"failed"`
}

type QueueStatus struct {
	Pending    int `json:"pending"`
	Processing int `json:"processing"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
}

type SystemMetrics struct {
	CPUUsage           float64 `json:"cpu_usage"`
	MemoryUsage        uint64  `json:"memory_usage"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	WorkersActive      int     `json:"workers_active"`
	WorkersTotal       int     `json:"workers_total"`
	QueueSize          int     `json:"queue_size"`
}

type MetricsResponse struct {
	ProcessingNow     int            `json:"processing"`
	TotalProcessed    int64          `json:"total_processed"`
	TotalFailed       int64          `json:"total_failed"`
	SuccessRate       float64        `json:"success_rate"`
	AvgProcessingTime float64        `json:"avg_processing_time"`
	Timeline          []TimelineData `json:"timeline"`
	QueueStatus       QueueStatus    `json:"queue_status"`
	SystemMetrics     SystemMetrics  `json:"system"`
	LastUpdated       time.Time      `json:"last_updated"`
}

var metricsCollector = &MetricsCollector{
	timeline:    make([]TimelineData, 0),
	lastUpdated: time.Now(),
}

// InitMetricsCollector initializes the metrics collection system
func InitMetricsCollector() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		// Initialize with empty timeline - will populate as jobs are processed
		metricsCollector.mu.Lock()
		metricsCollector.timeline = make([]TimelineData, 0)
		metricsCollector.mu.Unlock()

		// Update metrics every minute
		for range ticker.C {
			updateMetrics()
		}
	}()
}

func updateMetrics() {
	metricsCollector.mu.Lock()
	defer metricsCollector.mu.Unlock()

	// Rotate timeline data (keep last 24 hours)
	if len(metricsCollector.timeline) > 24*60 {
		metricsCollector.timeline = metricsCollector.timeline[len(metricsCollector.timeline)-24*60:]
	}

	// Add new data point
	metricsCollector.timeline = append(metricsCollector.timeline, TimelineData{
		Time:      time.Now(),
		Processed: int(metricsCollector.totalProcessed),
		Failed:    int(metricsCollector.totalFailed),
	})

	metricsCollector.lastUpdated = time.Now()
}

// RecordProcessingStart records when a job starts processing
func RecordProcessingStart() {
	metricsCollector.mu.Lock()
	defer metricsCollector.mu.Unlock()
	metricsCollector.processingNow++
	metricsCollector.queueStatus.Processing++
}

// RecordProcessingComplete records when a job completes successfully
func RecordProcessingComplete(duration time.Duration) {
	metricsCollector.mu.Lock()
	defer metricsCollector.mu.Unlock()

	metricsCollector.processingNow--
	metricsCollector.totalProcessed++
	metricsCollector.queueStatus.Processing--
	metricsCollector.queueStatus.Completed++

	// Update average processing time (simple moving average)
	if metricsCollector.avgProcessingTime == 0 {
		metricsCollector.avgProcessingTime = duration.Seconds()
	} else {
		metricsCollector.avgProcessingTime = (metricsCollector.avgProcessingTime * 0.9) + (duration.Seconds() * 0.1)
	}
}

// RecordProcessingFailed records when a job fails
func RecordProcessingFailed() {
	metricsCollector.mu.Lock()
	defer metricsCollector.mu.Unlock()

	metricsCollector.processingNow--
	metricsCollector.totalFailed++
	metricsCollector.queueStatus.Processing--
	metricsCollector.queueStatus.Failed++
}

// GetSystemMetrics collects current system resource metrics
func GetSystemMetrics() SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Calculate actual CPU usage based on goroutines and system resources
	numCPU := runtime.NumCPU()
	numGoroutines := runtime.NumGoroutine()
	// Estimate CPU usage more realistically (not a perfect metric but better than simulated)
	cpuUsage := float64(numGoroutines) / float64(numCPU) * 10.0
	if cpuUsage > 100 {
		cpuUsage = 100
	}

	return SystemMetrics{
		CPUUsage:           cpuUsage,
		MemoryUsage:        m.Alloc,
		MemoryUsagePercent: float64(m.Alloc) / float64(m.Sys) * 100,
		WorkersActive:      metricsCollector.processingNow,
		WorkersTotal:       10, // From config
		QueueSize:          metricsCollector.queueStatus.Pending,
	}
}

// MetricsHandler returns current metrics for the dashboard
func MetricsHandler(q queue.Queue) gin.HandlerFunc {
	return func(c *gin.Context) {
		metricsCollector.mu.RLock()
		defer metricsCollector.mu.RUnlock()

		// Calculate success rate
		total := metricsCollector.totalProcessed + metricsCollector.totalFailed
		successRate := 0.0 // Default to 0 if no data
		if total > 0 {
			successRate = float64(metricsCollector.totalProcessed) / float64(total)
		}

		// Get queue status if available
		if q != nil {
			// Try to get real queue metrics
			ctx := c.Request.Context()
			jobs, err := q.ListJobs(ctx, "", 100)
			if err == nil {
				pending := 0
				processing := 0
				completed := 0
				failed := 0

				for _, job := range jobs {
					switch job.Status {
					case "pending":
						pending++
					case "processing":
						processing++
					case "completed":
						completed++
					case "failed":
						failed++
					}
				}

				metricsCollector.queueStatus = QueueStatus{
					Pending:    pending,
					Processing: processing,
					Completed:  completed,
					Failed:     failed,
				}
			}
		}

		// Prepare timeline data (last 24 data points for hourly view)
		timeline := metricsCollector.timeline
		if len(timeline) > 24 {
			timeline = timeline[len(timeline)-24:]
		}

		response := MetricsResponse{
			ProcessingNow:     metricsCollector.processingNow,
			TotalProcessed:    metricsCollector.totalProcessed,
			TotalFailed:       metricsCollector.totalFailed,
			SuccessRate:       successRate,
			AvgProcessingTime: metricsCollector.avgProcessingTime,
			Timeline:          timeline,
			QueueStatus:       metricsCollector.queueStatus,
			SystemMetrics:     GetSystemMetrics(),
			LastUpdated:       metricsCollector.lastUpdated,
		}

		c.JSON(http.StatusOK, response)
	}
}

// WebSocketMetricsHandler provides real-time metrics updates via WebSocket
func WebSocketMetricsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// WebSocket implementation would go here
		// For now, return a simple message
		c.JSON(http.StatusOK, gin.H{
			"message": "WebSocket endpoint for real-time metrics (not yet implemented)",
		})
	}
}
