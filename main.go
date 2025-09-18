package main

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alterspective-engine/dot-to-docx-converter/internal/api"
	"github.com/alterspective-engine/dot-to-docx-converter/internal/config"
	"github.com/alterspective-engine/dot-to-docx-converter/internal/converter"
	"github.com/alterspective-engine/dot-to-docx-converter/internal/queue"
	"github.com/alterspective-engine/dot-to-docx-converter/internal/storage"
	"github.com/alterspective-engine/dot-to-docx-converter/internal/worker"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

//go:embed web/*
var webFS embed.FS


func main() {
	// Initialize configuration
	cfg := config.Load()

	// Setup logging
	setupLogging(cfg.LogLevel)

	log.Info("Starting DOT to DOCX Converter Service")
	log.Infof("Configuration: Workers=%d, Port=%s", cfg.WorkerCount, cfg.Port)

	// Initialize components
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize storage
	var storageClient storage.Storage
	if cfg.AzureStorageConnectionString != "" {
		azStorage, err := storage.NewAzureStorage(cfg.AzureStorageConnectionString, cfg.AzureStorageContainer)
		if err != nil {
			log.Warnf("Failed to initialize Azure Storage: %v, falling back to local storage", err)
			storageClient = storage.NewLocalStorage("/tmp/conversions")
		} else {
			storageClient = azStorage
		}
	} else {
		storageClient = storage.NewLocalStorage("/tmp/conversions")
	}

	// Initialize queue
	var queueClient queue.Queue
	if cfg.RedisURL != "" {
		redisQueue, err := queue.NewRedisQueue(cfg.RedisURL)
		if err != nil {
			log.Warnf("Failed to initialize Redis queue, using in-memory queue: %v", err)
			queueClient = queue.NewMemoryQueue()
		} else {
			queueClient = redisQueue
		}
	} else {
		queueClient = queue.NewMemoryQueue()
	}

	// Initialize converter
	conv := converter.NewLibreOfficeConverter(cfg.ConversionTimeout)

	// Start worker pool
	workerPool := worker.NewPool(cfg.WorkerCount, queueClient, conv, storageClient)
	go workerPool.Start(ctx)

	// Setup HTTP server with converter for sync endpoints
	router := setupRouter(cfg, queueClient, storageClient, conv)

	// Setup graceful shutdown
	srv := &api.Server{
		Router: router,
		Port:   cfg.Port,
	}

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Errorf("Server forced to shutdown: %v", err)
	}

	cancel() // Cancel worker context
	log.Info("Service stopped")
}

func setupRouter(cfg *config.Config, queue queue.Queue, storage storage.Storage, conv converter.Converter) *gin.Engine {
	if cfg.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Initialize metrics collection
	api.InitMetricsCollector()

	// Serve static files for dashboard using embedded filesystem
	// Try embedded files first, fallback to disk for development
	webSubFS, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Warn("Failed to create sub filesystem for web assets, using disk files")
		router.Static("/static", "./web")
	} else {
		router.StaticFS("/static", http.FS(webSubFS))
	}

	// Dashboard page - try embedded first, fallback to disk
	router.GET("/dashboard", func(c *gin.Context) {
		dashboardContent, err := webFS.ReadFile("web/dashboard.html")
		if err != nil {
			// Fallback to disk file for development
			c.File("./web/dashboard.html")
			return
		}
		c.Data(200, "text/html; charset=utf-8", dashboardContent)
	})

	// Static pages and documentation
	router.GET("/", api.ServeLandingPage())
	router.GET("/swagger", api.ServeSwaggerUI())
	router.GET("/api/v1/openapi.yaml", api.ServeOpenAPISpec())

	// Health checks
	router.GET("/health", api.HealthCheck())
	router.GET("/health/live", api.LivenessCheck())
	router.GET("/health/ready", api.ReadinessCheck(queue))

	// Metrics
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	router.GET("/api/v1/metrics", api.MetricsHandler(queue))
	router.GET("/ws/metrics", api.WebSocketMetricsHandler())

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Asynchronous conversion (queue-based)
		v1.POST("/convert", api.ConvertHandler(queue, storage))
		v1.POST("/batch", api.BatchConvertHandler(queue, storage))

		// Synchronous conversion (immediate response)
		v1.POST("/convert/sync", api.ConvertSyncHandler(conv, cfg.SyncMaxFileSize, cfg.SyncTimeout))
		v1.POST("/convert/sync/json", api.ConvertSyncJSONHandler(conv, cfg.SyncMaxFileSize, cfg.SyncTimeout))

		// Complexity analysis endpoints (no conversion)
		v1.POST("/analyze", api.AnalyzeHandler())
		v1.POST("/analyze/batch", api.AnalyzeBatchHandler())

		// Job management (for async)
		v1.GET("/jobs/:id", api.GetJobStatus(queue))
		v1.GET("/jobs", api.ListJobs(queue))
		v1.DELETE("/jobs/:id", api.CancelJob(queue))

		// Download converted file
		v1.GET("/download/:id", api.DownloadFile(storage))
	}

	return router
}

func setupLogging(level string) {
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	log.SetOutput(os.Stdout)

	switch level {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
}