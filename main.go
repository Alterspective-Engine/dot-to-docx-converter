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

// Embed static files and documentation
//go:embed static/* docs/*
var embeddedFiles embed.FS

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
		azStorage, err := storage.NewAzureStorage(cfg.AzureStorageConnectionString)
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

	// Setup HTTP server
	router := setupRouter(cfg, queueClient, storageClient)

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

func setupRouter(cfg *config.Config, queue queue.Queue, storage storage.Storage) *gin.Engine {
	if cfg.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Static files and documentation using embedded files
	// Create a sub filesystem for static files
	staticFS, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		log.Warnf("Failed to create static filesystem: %v", err)
	} else {
		// Serve static files
		router.StaticFS("/static", http.FS(staticFS))

		// Serve landing page at root
		router.GET("/", func(c *gin.Context) {
			data, err := embeddedFiles.ReadFile("static/index.html")
			if err != nil {
				c.JSON(404, gin.H{"error": "Page not found"})
				return
			}
			c.Data(200, "text/html; charset=utf-8", data)
		})

		// Serve swagger page
		router.GET("/swagger", func(c *gin.Context) {
			data, err := embeddedFiles.ReadFile("static/swagger.html")
			if err != nil {
				c.JSON(404, gin.H{"error": "Page not found"})
				return
			}
			c.Data(200, "text/html; charset=utf-8", data)
		})
	}

	// Serve OpenAPI spec
	router.GET("/docs/openapi.yaml", func(c *gin.Context) {
		data, err := embeddedFiles.ReadFile("docs/openapi.yaml")
		if err != nil {
			c.JSON(404, gin.H{"error": "OpenAPI spec not found"})
			return
		}
		c.Data(200, "application/x-yaml", data)
	})

	// Health checks
	router.GET("/health", api.HealthCheck())
	router.GET("/health/live", api.LivenessCheck())
	router.GET("/health/ready", api.ReadinessCheck(queue))

	// Metrics
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Single file conversion
		v1.POST("/convert", api.ConvertHandler(queue, storage))

		// Batch conversion
		v1.POST("/batch", api.BatchConvertHandler(queue, storage))

		// Job management
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