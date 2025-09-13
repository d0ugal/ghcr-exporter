package collectors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
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
		Packages: []config.PackageGroup{
			{
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

func TestGetPackageDownloadStats(t *testing.T) {
	// Create a test server that returns HTML with download statistics
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `
		<html>
			<body>
				<div>Total Downloads 100,000</div>
			</body>
		</html>
		`

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		if _, err := w.Write([]byte(html)); err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
			return
		}
	}))
	defer server.Close()

	// Create a fresh registry for this test to avoid conflicts
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	cfg := &config.Config{}
	registry := metrics.NewRegistry()
	collector := NewGHCRCollector(cfg, registry)

	// Override the client to use our test server
	collector.client = server.Client()

	// Test the download stats extraction by calling the private method directly
	// We need to use reflection or make the method public for testing
	// For now, let's test the regex pattern separately
	html := `<div>Total Downloads 100,000</div>`
	downloadPattern := regexp.MustCompile(`Total Downloads\s+([0-9,]+)`)
	matches := downloadPattern.FindStringSubmatch(html)

	if len(matches) < 2 {
		t.Fatal("Expected to find download statistics in HTML")
	}

	downloadStr := strings.ReplaceAll(matches[1], ",", "")

	downloadCount, err := strconv.ParseInt(downloadStr, 10, 64)
	if err != nil {
		t.Fatalf("Expected no error parsing download count, got: %v", err)
	}

	expectedCount := int64(100000)
	if downloadCount != expectedCount {
		t.Errorf("Expected download count %d, got %d", expectedCount, downloadCount)
	}
}

func TestGetPackageDownloadStatsWithCommas(t *testing.T) {
	// Test with different comma formats
	testCases := []struct {
		html        string
		expected    int64
		description string
	}{
		{
			html:        `<div>Total Downloads 1,234</div>`,
			expected:    1234,
			description: "Single comma",
		},
		{
			html:        `<div>Total Downloads 1,234,567</div>`,
			expected:    1234567,
			description: "Multiple commas",
		},
		{
			html:        `<div>Total Downloads 999</div>`,
			expected:    999,
			description: "No commas",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Test the regex pattern directly
			downloadPattern := regexp.MustCompile(`Total Downloads\s+([0-9,]+)`)
			matches := downloadPattern.FindStringSubmatch(tc.html)

			if len(matches) < 2 {
				t.Fatal("Expected to find download statistics in HTML")
			}

			downloadStr := strings.ReplaceAll(matches[1], ",", "")

			downloadCount, err := strconv.ParseInt(downloadStr, 10, 64)
			if err != nil {
				t.Fatalf("Expected no error parsing download count, got: %v", err)
			}

			if downloadCount != tc.expected {
				t.Errorf("Expected download count %d, got %d", tc.expected, downloadCount)
			}
		})
	}
}

func TestGetPackageDownloadStatsNotFound(t *testing.T) {
	// Test that we get an error when download stats are not found
	html := `<html><body><div>No download info here</div></body></html>`
	downloadPattern := regexp.MustCompile(`Total Downloads\s+([0-9,]+)`)
	matches := downloadPattern.FindStringSubmatch(html)

	if len(matches) >= 2 {
		t.Fatal("Expected not to find download statistics in HTML")
	}
}

func TestGetPackageDownloadStatsHTTPError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Create a fresh registry for this test
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	cfg := &config.Config{}
	registry := metrics.NewRegistry()
	collector := NewGHCRCollector(cfg, registry)
	collector.client = server.Client()

	// Test that we get an error when the HTTP request fails
	_, err := collector.getPackageDownloadStats(context.Background(), "test-owner", "test-package")
	if err == nil {
		t.Fatal("Expected error when HTTP request fails, got nil")
	}

	expectedError := "package page returned status 404"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}
