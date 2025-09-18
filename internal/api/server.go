package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// Server represents the HTTP server
type Server struct {
	Router *gin.Engine
	Port   string
	server *http.Server
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%s", s.Port),
		Handler: s.Router,
	}

	log.Infof("Starting server on port %s", s.Port)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Info("Shutting down server...")
	return s.server.Shutdown(ctx)
}
