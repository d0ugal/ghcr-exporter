package collectors

import (
	"context"
	"testing"
	"time"

	"ghcr-exporter/internal/config"
	"ghcr-exporter/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewGHCRCollector(t *testing.T) {
	cfg := &config.Config{}
	registry := metrics.NewRegistry()

	collector := NewGHCRCollector(cfg, registry)

	if collector == nil {
		t.Fatal("Expected collector to be created, got nil")
	}

	if collector.config != cfg {
		t.Error("Expected config to be set correctly")
	}

	if collector.metrics != registry {
		t.Error("Expected metrics registry to be set correctly")
	}
}

func TestGHCRCollectorStart(t *testing.T) {
	cfg := &config.Config{
		Packages: map[string]config.PackageGroup{
			"test-package": {
				Owner: "test-owner",
				Repo:  "test-repo",
			},
		},
	}

	// Create a fresh registry for this test
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	registry := metrics.NewRegistry()

	collector := NewGHCRCollector(cfg, registry)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	collector.Start(ctx)

	// Wait for context to be cancelled
	<-ctx.Done()
}
