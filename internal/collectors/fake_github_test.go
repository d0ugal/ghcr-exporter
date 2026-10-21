package collectors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"ghcr-exporter/internal/config"
	"ghcr-exporter/internal/metrics"
	"github.com/d0ugal/promexporter/app"
	promexporter_config "github.com/d0ugal/promexporter/config"
	promexporter_metrics "github.com/d0ugal/promexporter/metrics"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// newFakeGitHubCollector returns a collector whose API calls go to handler.
func newFakeGitHubCollector(t *testing.T, handler http.Handler) (*GHCRCollector, *metrics.GHCRRegistry) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	cfg := &config.Config{
		GitHub: config.GitHubConfig{Token: promexporter_config.NewSensitiveString("test-token")},
	}
	baseRegistry := promexporter_metrics.NewRegistry("test_exporter_info")
	registry := metrics.NewGHCRRegistry(baseRegistry)
	testApp := app.New("Test Exporter").WithConfig(&cfg.BaseConfig).WithMetrics(baseRegistry).Build()

	collector := NewGHCRCollector(cfg, registry, testApp)
	collector.client = server.Client()
	collector.apiBaseURL = server.URL

	return collector, registry
}

// A nested package name ("charts/app") is one escaped path segment in the API.
func TestCollectPackageMetricsNestedName(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.EscapedPath() {
		case "/users/acme/packages/container/charts%2Fapp":
			_, _ = w.Write([]byte(`{"name": "charts/app", "version_count": 7}`))
		case "/users/acme/packages/container/charts%2Fapp/versions":
			_, _ = w.Write([]byte(`[{"created_at": "2026-01-02T03:04:05Z"}]`))
		default:
			http.NotFound(w, r)
		}
	})

	collector, registry := newFakeGitHubCollector(t, mux)

	pkg := config.PackageGroup{Owner: "acme", Repo: "charts/app"}
	if err := collector.collectPackageMetrics(context.Background(), pkg.Repo, pkg); err != nil {
		t.Fatalf("collectPackageMetrics: %v", err)
	}

	if got := testutil.ToFloat64(registry.PackageDownloadsGauge.WithLabelValues("acme", "charts/app")); got != 7 {
		t.Errorf("ghcr_package_versions = %v, want 7", got)
	}

	if got := testutil.ToFloat64(registry.PackageLastPublishedGauge.WithLabelValues("acme", "charts/app")); got != 1767323045 {
		t.Errorf("ghcr_package_last_published_timestamp = %v, want 1767323045", got)
	}
}
