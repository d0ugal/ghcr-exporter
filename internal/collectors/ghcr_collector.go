package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ghcr-exporter/internal/config"
	"ghcr-exporter/internal/metrics"
)

type GHCRCollector struct {
	config  *config.Config
	metrics *metrics.Registry
	client  *http.Client
	token   string
}

// GHCRPackageResponse represents the response from GHCR API
type GHCRPackageResponse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	PackageType string `json:"package_type"`
	Owner       struct {
		Login string `json:"login"`
	} `json:"owner"`
	Repository struct {
		ID       int    `json:"id"`
		NodeID   string `json:"node_id"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		Private  bool   `json:"private"`
	} `json:"repository"`
	VersionCount int    `json:"version_count"`
	Visibility   string `json:"visibility"`
	URL          string `json:"url"`
}

// GHCRVersionResponse represents the response for package versions
type GHCRVersionResponse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	PackageHTML string `json:"package_html"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	HTMLURL     string `json:"html_url"`
	Metadata    struct {
		PackageType string `json:"package_type"`
		Container   struct {
			Tags []string `json:"tags"`
		} `json:"container"`
	} `json:"metadata"`
	PackageFiles []struct {
		DownloadURL string `json:"download_url"`
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Size        int    `json:"size"`
		ContentType string `json:"content_type"`
		State       string `json:"state"`
	} `json:"package_files"`
}

func NewGHCRCollector(cfg *config.Config, registry *metrics.Registry) *GHCRCollector {
	return &GHCRCollector{
		config:  cfg,
		metrics: registry,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		token: cfg.GitHub.Token,
	}
}

func (gc *GHCRCollector) Start(ctx context.Context) {
	go gc.run(ctx)
}

func (gc *GHCRCollector) run(ctx context.Context) {
	// Create individual tickers for each package
	tickers := make(map[string]*time.Ticker)

	defer func() {
		for _, ticker := range tickers {
			ticker.Stop()
		}
	}()

	// Start individual tickers for each package
	for groupName, group := range gc.config.Packages {
		interval := gc.config.GetPackageInterval(group)
		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		tickers[groupName] = ticker

		// Initial collection for this package
		gc.collectSinglePackage(ctx, group.Repo, group)

		// Start goroutine for this package
		go func(name string, pkg config.PackageGroup) {
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					gc.collectSinglePackage(ctx, pkg.Repo, pkg)
				}
			}
		}(groupName, group)
	}

	// Wait for context cancellation
	<-ctx.Done()
	slog.Info("GHCR collector stopped")
}

func (gc *GHCRCollector) collectSinglePackage(ctx context.Context, repo string, pkg config.PackageGroup) {
	startTime := time.Now()
	interval := gc.config.GetPackageInterval(pkg)

	slog.Info("Starting GHCR package metrics collection", "repo", repo, "package", pkg.Repo)

	// Retry with exponential backoff
	err := gc.retryWithBackoff(func() error {
		return gc.collectPackageMetrics(ctx, repo, pkg)
	}, 3, 2*time.Second)
	if err != nil {
		slog.Error("Failed to collect package metrics after retries", "repo", repo, "error", err)
		gc.metrics.CollectionFailedCounter.WithLabelValues(repo, strconv.Itoa(interval)).Inc()
		return
	}

	gc.metrics.CollectionSuccessCounter.WithLabelValues(repo, strconv.Itoa(interval)).Inc()
	// Expose configured interval as a numeric gauge for PromQL arithmetic
	gc.metrics.CollectionIntervalGauge.WithLabelValues(repo).Set(float64(interval))

	duration := time.Since(startTime).Seconds()
	gc.metrics.CollectionDurationGauge.WithLabelValues(repo, strconv.Itoa(interval)).Set(duration)
	gc.metrics.CollectionTimestampGauge.WithLabelValues(repo, strconv.Itoa(interval)).Set(float64(time.Now().Unix()))

	slog.Info("GHCR package metrics collection completed", "repo", repo, "duration", duration)
}

func (gc *GHCRCollector) collectPackageMetrics(ctx context.Context, repo string, pkg config.PackageGroup) error {
	slog.Info("Collecting metrics for package",
		"owner", pkg.Owner,
		"repo", pkg.Repo,
		"package", pkg.Repo)

	// Check if we have a GitHub token
	if gc.token == "" {
		return fmt.Errorf("GitHub token required to access package information")
	}

	// Get package information from GitHub API
	packageInfo, err := gc.getPackageInfo(ctx, pkg.Owner, pkg.Repo, pkg.Repo)
	if err != nil {
		return fmt.Errorf("failed to get package info: %w", err)
	}

	// Get package versions for more detailed metrics
	versions, err := gc.getPackageVersions(ctx, pkg.Owner, pkg.Repo, pkg.Repo)
	if err != nil {
		slog.Warn("Failed to get package versions", "error", err)
		// Continue with basic metrics even if versions fail
	}

	// Update metrics
	gc.updatePackageMetrics(pkg, packageInfo, versions)

	return nil
}

func (gc *GHCRCollector) getPackageInfo(ctx context.Context, owner, repo, packageName string) (*GHCRPackageResponse, error) {
	resp, err := gc.makeGitHubAPIRequest(ctx, fmt.Sprintf("/users/%s/packages/container/%s", owner, packageName))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var packageInfo GHCRPackageResponse
	if err := json.NewDecoder(resp.Body).Decode(&packageInfo); err != nil {
		return nil, err
	}
	return &packageInfo, nil
}

// makeGitHubAPIRequest makes a request to GitHub API, trying user endpoint first, then org endpoint
func (gc *GHCRCollector) makeGitHubAPIRequest(ctx context.Context, path string) (*http.Response, error) {
	// Try user endpoint first
	userURL := fmt.Sprintf("https://api.github.com%s", path)
	userReq, err := http.NewRequestWithContext(ctx, "GET", userURL, nil)
	if err != nil {
		return nil, err
	}

	userReq.Header.Set("Accept", "application/vnd.github.v3+json")
	if gc.token != "" {
		userReq.Header.Set("Authorization", "Bearer "+gc.token)
	}

	userResp, err := gc.client.Do(userReq)
	if err != nil {
		return nil, err
	}

	// If user endpoint succeeds, return the response
	if userResp.StatusCode == http.StatusOK {
		return userResp, nil
	}

	// If user endpoint returns 404, try org endpoint
	if userResp.StatusCode == http.StatusNotFound {
		userResp.Body.Close()

		// Replace /users/ with /orgs/ in the path
		orgPath := strings.Replace(path, "/users/", "/orgs/", 1)
		orgURL := fmt.Sprintf("https://api.github.com%s", orgPath)

		orgReq, err := http.NewRequestWithContext(ctx, "GET", orgURL, nil)
		if err != nil {
			return nil, err
		}

		orgReq.Header.Set("Accept", "application/vnd.github.v3+json")
		if gc.token != "" {
			orgReq.Header.Set("Authorization", "Bearer "+gc.token)
		}

		orgResp, err := gc.client.Do(orgReq)
		if err != nil {
			return nil, err
		}

		if orgResp.StatusCode == http.StatusOK {
			return orgResp, nil
		}

		// If both fail, return the org endpoint error
		orgResp.Body.Close()
		return nil, fmt.Errorf("API request failed with status %d", orgResp.StatusCode)
	}

	// If user endpoint fails with something other than 404, return that error
	userResp.Body.Close()
	return nil, fmt.Errorf("API request failed with status %d", userResp.StatusCode)
}

func (gc *GHCRCollector) getPackageVersions(ctx context.Context, owner, repo, packageName string) ([]GHCRVersionResponse, error) {
	resp, err := gc.makeGitHubAPIRequest(ctx, fmt.Sprintf("/users/%s/packages/container/%s/versions", owner, packageName))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var versions []GHCRVersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, err
	}
	return versions, nil
}

func (gc *GHCRCollector) setFallbackMetrics(pkg config.PackageGroup) {
	// Set fallback metrics when API calls fail
	// This ensures we always have some metrics available
	gc.metrics.PackageDownloadsGauge.WithLabelValues(pkg.Owner, pkg.Repo).Set(0)
	gc.metrics.PackageLastPublishedGauge.WithLabelValues(pkg.Owner, pkg.Repo).Set(float64(time.Now().Unix()))

	slog.Info("Set fallback metrics for package", "package", pkg.Repo)
}

func (gc *GHCRCollector) updatePackageMetrics(pkg config.PackageGroup, packageInfo *GHCRPackageResponse, versions []GHCRVersionResponse) {
	// Update package-level metrics with real data
	// Note: GitHub API doesn't provide download statistics for packages
	// We'll use version count as a proxy metric and track last published time

	var lastPublished time.Time

	// Find the most recent version
	for _, version := range versions {
		// Parse the created_at timestamp
		if created, err := time.Parse(time.RFC3339, version.CreatedAt); err == nil {
			if created.After(lastPublished) {
				lastPublished = created
			}
		}
	}

	// Update package-level metrics
	// Use version count as a proxy for activity (more versions = more activity)
	gc.metrics.PackageDownloadsGauge.WithLabelValues(pkg.Owner, pkg.Repo).Set(float64(packageInfo.VersionCount))

	if !lastPublished.IsZero() {
		gc.metrics.PackageLastPublishedGauge.WithLabelValues(pkg.Owner, pkg.Repo).Set(float64(lastPublished.Unix()))
	}

	slog.Info("Updated package metrics",
		"package", pkg.Repo,
		"version_count", packageInfo.VersionCount,
		"last_published", lastPublished.Format(time.RFC3339))
}

func (gc *GHCRCollector) retryWithBackoff(operation func() error, maxRetries int, initialDelay time.Duration) error {
	var lastErr error
	delay := initialDelay

	for i := 0; i <= maxRetries; i++ {
		if err := operation(); err != nil {
			lastErr = err
			if i < maxRetries {
				slog.Warn("Operation failed, retrying", "attempt", i+1, "error", err, "delay", delay)
				time.Sleep(delay)
				delay *= 2 // Exponential backoff
			}
		} else {
			return nil
		}
	}

	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, lastErr)
}
