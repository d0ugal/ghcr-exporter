package collectors

import (
	"context"
	"encoding/json"
	"fmt"
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
	collector.retryDelay = 0

	return collector, registry
}

// Owners with more packages than one page holds are discovered in full.
func TestGetOwnerPackagesPaginates(t *testing.T) {
	const total = ownerPackagesPerPage + 50

	mux := http.NewServeMux()
	mux.HandleFunc("/users/acme/packages", func(w http.ResponseWriter, r *http.Request) {
		perPage := ownerPackagesPerPage
		if got := r.URL.Query().Get("per_page"); got != fmt.Sprint(perPage) {
			t.Errorf("per_page = %q, want %d", got, perPage)
		}

		var page int
		if _, err := fmt.Sscan(r.URL.Query().Get("page"), &page); err != nil || page < 1 {
			t.Errorf("page = %q, want a positive number", r.URL.Query().Get("page"))
			return
		}

		packages := []GHCRPackageResponse{}
		for i := (page - 1) * perPage; i < min(page*perPage, total); i++ {
			packages = append(packages, GHCRPackageResponse{Name: fmt.Sprintf("images/app-%d", i)})
		}

		_ = json.NewEncoder(w).Encode(packages)
	})

	collector, _ := newFakeGitHubCollector(t, mux)

	packages, err := collector.getOwnerPackages(context.Background(), "acme")
	if err != nil {
		t.Fatalf("getOwnerPackages: %v", err)
	}

	if len(packages) != total {
		t.Errorf("got %d packages, want %d", len(packages), total)
	}
}

// A discovered package that fails to collect, after retries, fails the owner collection.
func TestCollectOwnerPackagesCountsFailedPackages(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		brokenFailures        int // requests answered 500 before the package is served
		wantFailed, wantSuccs float64
	}{
		{name: "all packages collected", brokenFailures: 0, wantSuccs: 1},
		{name: "one package failing once", brokenFailures: 1, wantSuccs: 1},
		{name: "one package always failing", brokenFailures: packageRetries + 1, wantFailed: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			brokenRequests := 0

			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.EscapedPath() {
				case "/users/acme/packages":
					_, _ = w.Write([]byte(`[{"name": "images/ok"}, {"name": "images/broken"}]`))
				case "/users/acme/packages/container/images%2Fok":
					_, _ = w.Write([]byte(`{"name": "images/ok"}`))
				case "/users/acme/packages/container/images%2Fbroken":
					if brokenRequests++; brokenRequests <= tc.brokenFailures {
						w.WriteHeader(http.StatusInternalServerError)
					}

					_, _ = w.Write([]byte(`{"name": "images/broken"}`))
				default:
					_, _ = w.Write([]byte(`[]`))
				}
			})

			collector, registry := newFakeGitHubCollector(t, mux)
			pkg := config.PackageGroup{Owner: "acme"}
			interval := fmt.Sprint(collector.config.GetPackageInterval(pkg))

			collector.collectOwnerPackages(context.Background(), pkg.GetName(), pkg)

			if got := testutil.ToFloat64(registry.CollectionFailedCounter.WithLabelValues(pkg.GetName(), interval)); got != tc.wantFailed {
				t.Errorf("ghcr_collection_failed_total = %v, want %v", got, tc.wantFailed)
			}

			if got := testutil.ToFloat64(registry.CollectionSuccessCounter.WithLabelValues(pkg.GetName(), interval)); got != tc.wantSuccs {
				t.Errorf("ghcr_collection_success_total = %v, want %v", got, tc.wantSuccs)
			}
		})
	}
}

// A nested package name ("charts/app") is one escaped path segment in the API,
// and its page is the html_url GitHub returns, not a URL built from the name.
func TestCollectPackageMetricsNestedName(t *testing.T) {
	var baseURL string // the fake server, known once it runs

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.EscapedPath() {
		case "/users/acme/packages/container/charts%2Fapp":
			_, _ = w.Write([]byte(`{"name": "charts/app", "version_count": 7, "html_url": "` + baseURL + `/users/acme/packages/container/package/charts%2Fapp"}`))
		case "/users/acme/packages/container/charts%2Fapp/versions":
			_, _ = w.Write([]byte(`[{"created_at": "2026-01-02T03:04:05Z"}]`))
		case "/users/acme/packages/container/package/charts%2Fapp":
			_, _ = w.Write([]byte("<span>Total downloads</span>\n<h3 title=\"1234\">1.2K</h3>\n"))
		default:
			http.NotFound(w, r)
		}
	})

	collector, registry := newFakeGitHubCollector(t, mux)
	baseURL = collector.apiBaseURL

	pkg := config.PackageGroup{Owner: "acme", Repo: "charts/app"}
	if err := collector.collectPackageMetrics(context.Background(), pkg.Repo, pkg); err != nil {
		t.Fatalf("collectPackageMetrics: %v", err)
	}

	if got := testutil.ToFloat64(registry.PackageDownloadsGauge.WithLabelValues("acme", "charts/app")); got != 7 {
		t.Errorf("ghcr_package_versions = %v, want 7", got)
	}

	if got := testutil.ToFloat64(registry.PackageDownloadStatsGauge.WithLabelValues("acme", "charts/app")); got != 1234 {
		t.Errorf("ghcr_package_downloads = %v, want 1234", got)
	}

	if got := testutil.ToFloat64(registry.PackageLastPublishedGauge.WithLabelValues("acme", "charts/app")); got != 1767323045 {
		t.Errorf("ghcr_package_last_published_timestamp = %v, want 1767323045", got)
	}
}
