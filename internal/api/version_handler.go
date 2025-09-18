package api

import (
	"net/http"
	"os"
	"strings"

	"github.com/alterspective-engine/dot-to-docx-converter/internal/version"
	"github.com/gin-gonic/gin"
)

// VersionHandler returns version information
func VersionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		info := version.GetInfo()
		c.JSON(http.StatusOK, info)
	}
}

// ChangelogHandler serves the changelog
func ChangelogHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read changelog file
		content, err := os.ReadFile("CHANGELOG.md")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Changelog not found",
			})
			return
		}

		// Check if client wants JSON or markdown
		acceptHeader := c.GetHeader("Accept")
		if strings.Contains(acceptHeader, "application/json") {
			c.JSON(http.StatusOK, gin.H{
				"version":   version.Version,
				"changelog": string(content),
			})
		} else {
			c.Data(http.StatusOK, "text/markdown; charset=utf-8", content)
		}
	}
}