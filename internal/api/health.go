package api

import (
	"net/http"

	"github.com/alterspective-engine/dot-to-docx-converter/internal/queue"
	"github.com/gin-gonic/gin"
)

// HealthCheck returns a basic health check handler
func HealthCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "dot-to-docx-converter",
			"version": "1.0.0",
		})
	}
}

// LivenessCheck checks if the service is alive
func LivenessCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "alive",
		})
	}
}

// ReadinessCheck checks if the service is ready to accept requests
func ReadinessCheck(q queue.Queue) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check queue connectivity
		if _, err := q.Size(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"error":  "queue unavailable",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
		})
	}
}
