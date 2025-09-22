package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"ghcr-exporter/internal/config"
	"ghcr-exporter/internal/metrics"
	"ghcr-exporter/internal/version"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	config  *config.Config
	metrics *metrics.Registry
	server  *http.Server
	router  *gin.Engine
}

func New(cfg *config.Config, metricsRegistry *metrics.Registry) *Server {
	router := gin.New()
	router.Use(gin.Recovery())

	server := &Server{
		config:  cfg,
		metrics: metricsRegistry,
		router:  router,
	}

	server.setupRoutes()

	return server
}

func (s *Server) setupRoutes() {
	// Root endpoint with HTML dashboard
	s.router.GET("/", s.handleRoot)

	// Metrics endpoint
	s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Health endpoint
	s.router.GET("/health", s.handleHealth)
}

func (s *Server) handleRoot(c *gin.Context) {
	versionInfo := version.Get()
	metricsInfo := s.metrics.GetMetricsInfo()

	// Convert metrics to template data
	metrics := make([]MetricData, 0, len(metricsInfo))
	for _, metric := range metricsInfo {
		metrics = append(metrics, MetricData{
			Name:         metric.Name,
			Help:         metric.Help,
			Labels:       metric.Labels,
			ExampleValue: metric.ExampleValue,
		})
	}

	// Determine token status
	tokenStatus := "Configured"
	if s.config.GitHub.Token == "" {
		tokenStatus = "Not configured"
	}

	data := TemplateData{
		Version:   versionInfo.Version,
		Commit:    versionInfo.Commit,
		BuildDate: versionInfo.BuildDate,
		Metrics:   metrics,
		Config: ConfigData{
			TokenConfigured:    tokenStatus,
			PackageCount:       len(s.config.Packages),
			CollectionInterval: s.config.Metrics.Collection.DefaultInterval.String(),
		},
	}

	c.Header("Content-Type", "text/html")

	if err := mainTemplate.Execute(c.Writer, data); err != nil {
		c.String(http.StatusInternalServerError, "Error rendering template: %v", err)
	}
}

func (s *Server) handleHealth(c *gin.Context) {
	versionInfo := version.Get()
	c.JSON(http.StatusOK, gin.H{
		"status":     "healthy",
		"timestamp":  time.Now().Unix(),
		"service":    "ghcr-exporter",
		"version":    versionInfo.Version,
		"commit":     versionInfo.Commit,
		"build_date": versionInfo.BuildDate,
	})
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)

	s.server = &http.Server{
		Addr:              addr,
		Handler:           s.router,
		ReadHeaderTimeout: 30 * time.Second,
	}

	return s.server.ListenAndServe()
}

func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}
