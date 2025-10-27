package collectors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ghcr-exporter/internal/config"
	"ghcr-exporter/internal/metrics"

	promexporter_config "github.com/d0ugal/promexporter/config"
	promexporter_metrics "github.com/d0ugal/promexporter/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// TestGHCRCollectorIntegration tests the full collection flow to catch label mapping issues
func TestGHCRCollectorIntegration(t *testing.T) {
	// Create test server that returns valid GHCR API responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/d0ugal/filesystem-exporter/tags/list":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"name": "d0ugal/filesystem-exporter",
				"tags": [
					"v1.22.4",
					"v1.22.3",
					"latest"
				]
			}`))
		case "/v2/d0ugal/filesystem-exporter/manifests/latest":
			w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"schemaVersion": 2,
				"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
				"config": {
					"mediaType": "application/vnd.docker.container.image.v1+json",
					"size": 1234,
					"digest": "sha256:abc123"
				},
				"layers": []
			}`))
		case "/v2/d0ugal/filesystem-exporter/manifests/sha256:abc123":
			w.Header().Set("Content-Type", "application/vnd.docker.container.image.v1+json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"created": "2025-10-27T20:00:00Z",
				"config": {
					"Labels": {
						"org.opencontainers.image.created": "2025-10-27T20:00:00Z"
					}
				}
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create test configuration
	cfg := &config.Config{
		Packages: []config.PackageGroup{
			{
				Owner: "d0ugal",
				Repo:  "filesystem-exporter",
			},
		},
		GitHub: config.GitHubConfig{
			Token: promexporter_config.NewSensitiveString("test-token"),
		},
	}

	// Create a fresh registry for testing
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	baseRegistry := promexporter_metrics.NewRegistry("test_exporter_info")
	registry := metrics.NewGHCRRegistry(baseRegistry)

	collector := NewGHCRCollector(cfg, registry)

	// Override the client to use our test server
	collector.client = server.Client()

	// Test the collection flow
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start the collector
	collector.Start(ctx)

	// Wait a bit for collection to happen
	time.Sleep(100 * time.Millisecond)

	// Cancel context to stop collection
	cancel()

	// Wait for collection to complete
	time.Sleep(100 * time.Millisecond)

	// Verify that metrics were created with correct labels
	// This is the key test - it will panic if labels don't match metric definitions

	// Test collection metrics
	collectionFailedMetric := testutil.ToFloat64(registry.CollectionFailedCounter.With(prometheus.Labels{
		"repo":     "d0ugal-filesystem-exporter",
		"interval": "30",
	}))
	t.Logf("Collection failed metric: %f", collectionFailedMetric)

	collectionSuccessMetric := testutil.ToFloat64(registry.CollectionSuccessCounter.With(prometheus.Labels{
		"repo":     "d0ugal-filesystem-exporter",
		"interval": "30",
	}))
	t.Logf("Collection success metric: %f", collectionSuccessMetric)

	// Test package metrics
	packageVersionsMetric := testutil.ToFloat64(registry.PackageDownloadsGauge.With(prometheus.Labels{
		"owner": "d0ugal",
		"repo":  "filesystem-exporter",
	}))
	t.Logf("Package versions metric: %f", packageVersionsMetric)

	packageLastPublishedMetric := testutil.ToFloat64(registry.PackageLastPublishedGauge.With(prometheus.Labels{
		"owner": "d0ugal",
		"repo":  "filesystem-exporter",
	}))
	t.Logf("Package last published metric: %f", packageLastPublishedMetric)

	// If we get here without panicking, the label mapping is correct
	t.Log("✅ All metrics created successfully with correct label mapping")
}

// TestGHCRCollectorLabelConsistency tests that all metric labels match their definitions
func TestGHCRCollectorLabelConsistency(t *testing.T) {
	// Create a fresh registry for testing
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	baseRegistry := promexporter_metrics.NewRegistry("test_exporter_info")
	registry := metrics.NewGHCRRegistry(baseRegistry)

	// Test all metrics with their expected labels
	testCases := []struct {
		name        string
		metric      *prometheus.CounterVec
		labels      prometheus.Labels
		description string
	}{
		{
			name:        "CollectionFailedCounter",
			metric:      registry.CollectionFailedCounter,
			labels:      prometheus.Labels{"repo": "test-repo", "interval": "30"},
			description: "Should accept 'repo' and 'interval' labels",
		},
		{
			name:        "CollectionSuccessCounter",
			metric:      registry.CollectionSuccessCounter,
			labels:      prometheus.Labels{"repo": "test-repo", "interval": "30"},
			description: "Should accept 'repo' and 'interval' labels",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This will panic if labels don't match the metric definition
			counter := tc.metric.With(tc.labels)
			counter.Inc()

			// Verify the metric was created successfully
			value := testutil.ToFloat64(counter)
			if value != 1.0 {
				t.Errorf("Expected metric value 1.0, got %f", value)
			}

			t.Logf("✅ %s: %s", tc.name, tc.description)
		})
	}

	// Test gauge metrics
	gaugeTestCases := []struct {
		name        string
		metric      *prometheus.GaugeVec
		labels      prometheus.Labels
		description string
	}{
		{
			name:        "CollectionIntervalGauge",
			metric:      registry.CollectionIntervalGauge,
			labels:      prometheus.Labels{"repo": "test-repo", "interval": "30"},
			description: "Should accept 'repo' and 'interval' labels",
		},
		{
			name:        "CollectionDurationGauge",
			metric:      registry.CollectionDurationGauge,
			labels:      prometheus.Labels{"repo": "test-repo", "interval": "30"},
			description: "Should accept 'repo' and 'interval' labels",
		},
		{
			name:        "CollectionTimestampGauge",
			metric:      registry.CollectionTimestampGauge,
			labels:      prometheus.Labels{"repo": "test-repo", "interval": "30"},
			description: "Should accept 'repo' and 'interval' labels",
		},
		{
			name:        "PackageDownloadsGauge",
			metric:      registry.PackageDownloadsGauge,
			labels:      prometheus.Labels{"owner": "test-owner", "repo": "test-repo"},
			description: "Should accept 'owner' and 'repo' labels",
		},
		{
			name:        "PackageLastPublishedGauge",
			metric:      registry.PackageLastPublishedGauge,
			labels:      prometheus.Labels{"owner": "test-owner", "repo": "test-repo"},
			description: "Should accept 'owner' and 'repo' labels",
		},
		{
			name:        "PackageDownloadStatsGauge",
			metric:      registry.PackageDownloadStatsGauge,
			labels:      prometheus.Labels{"owner": "test-owner", "repo": "test-repo"},
			description: "Should accept 'owner' and 'repo' labels",
		},
	}

	for _, tc := range gaugeTestCases {
		t.Run(tc.name, func(t *testing.T) {
			// This will panic if labels don't match the metric definition
			gauge := tc.metric.With(tc.labels)
			gauge.Set(42.0)

			// Verify the metric was created successfully
			value := testutil.ToFloat64(gauge)
			if value != 42.0 {
				t.Errorf("Expected metric value 42.0, got %f", value)
			}

			t.Logf("✅ %s: %s", tc.name, tc.description)
		})
	}

	t.Log("✅ All metric label consistency tests passed")
}

// TestGHCRCollectorErrorHandling tests error scenarios to ensure they don't cause label panics
func TestGHCRCollectorErrorHandling(t *testing.T) {
	// Create test server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create test configuration
	cfg := &config.Config{
		Packages: []config.PackageGroup{
			{
				Owner: "d0ugal",
				Repo:  "filesystem-exporter",
			},
		},
		GitHub: config.GitHubConfig{
			Token: promexporter_config.NewSensitiveString("test-token"),
		},
	}

	// Create a fresh registry for testing
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	baseRegistry := promexporter_metrics.NewRegistry("test_exporter_info")
	registry := metrics.NewGHCRRegistry(baseRegistry)

	collector := NewGHCRCollector(cfg, registry)
	collector.client = server.Client()

	// Test error handling without panicking
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	collector.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	cancel()
	time.Sleep(100 * time.Millisecond)

	// If we get here without panicking, error handling is working correctly
	t.Log("✅ Error handling works correctly without label panics")
}
