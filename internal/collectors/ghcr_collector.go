package collectors

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	for _, group := range gc.config.Packages {
		interval := gc.config.GetPackageInterval(group)
		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		tickers[group.Name] = ticker

		// Initial collection for this package
		gc.collectSinglePackage(ctx, group.Name, group)

		// Start goroutine for this package
		go func(pkg config.PackageGroup) {
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					gc.collectSinglePackage(ctx, pkg.Name, pkg)
				}
			}
		}(group)
	}

	// Wait for context cancellation
	<-ctx.Done()
	slog.Info("GHCR collector stopped")
}

func (gc *GHCRCollector) collectSinglePackage(ctx context.Context, name string, pkg config.PackageGroup) {
	startTime := time.Now()
	interval := gc.config.GetPackageInterval(pkg)

	// If repo is not specified, discover all repos for the owner
	if pkg.Repo == "" {
		gc.collectOwnerPackages(ctx, name, pkg)
		return
	}

	slog.Info("Starting GHCR package metrics collection", "name", name, "owner", pkg.Owner, "repo", pkg.Repo)

	// Retry with exponential backoff
	err := gc.retryWithBackoff(func() error {
		return gc.collectPackageMetrics(ctx, pkg.Repo, pkg)
	}, 3, 2*time.Second)
	if err != nil {
		slog.Error("Failed to collect package metrics after retries", "name", name, "error", err)
		gc.metrics.CollectionFailedCounter.WithLabelValues(name, strconv.Itoa(interval)).Inc()

		return
	}

	gc.metrics.CollectionSuccessCounter.WithLabelValues(name, strconv.Itoa(interval)).Inc()
	// Expose configured interval as a numeric gauge for PromQL arithmetic
	gc.metrics.CollectionIntervalGauge.WithLabelValues(name).Set(float64(interval))

	duration := time.Since(startTime).Seconds()
	gc.metrics.CollectionDurationGauge.WithLabelValues(name, strconv.Itoa(interval)).Set(duration)
	gc.metrics.CollectionTimestampGauge.WithLabelValues(name, strconv.Itoa(interval)).Set(float64(time.Now().Unix()))

	slog.Info("GHCR package metrics collection completed", "name", name, "duration", duration)
}

// collectOwnerPackages discovers and collects metrics for all packages owned by the specified owner
func (gc *GHCRCollector) collectOwnerPackages(ctx context.Context, name string, pkg config.PackageGroup) {
	startTime := time.Now()
	interval := gc.config.GetPackageInterval(pkg)

	slog.Info("Starting GHCR owner package discovery", "name", name, "owner", pkg.Owner)

	// Get all packages for the owner
	packages, err := gc.getOwnerPackages(ctx, pkg.Owner)
	if err != nil {
		slog.Error("Failed to get owner packages", "name", name, "owner", pkg.Owner, "error", err)
		gc.metrics.CollectionFailedCounter.WithLabelValues(name, strconv.Itoa(interval)).Inc()

		return
	}

	slog.Info("Discovered packages for owner", "name", name, "owner", pkg.Owner, "package_count", len(packages))

	// Collect metrics for each discovered package
	successCount := 0

	for _, discoveredPkg := range packages {
		// Create a PackageGroup for the discovered package
		discoveredGroup := config.PackageGroup{
			Name:  fmt.Sprintf("%s-%s", name, discoveredPkg.Name),
			Owner: pkg.Owner,
			Repo:  discoveredPkg.Name,
		}

		err := gc.collectPackageMetrics(ctx, discoveredPkg.Name, discoveredGroup)
		if err != nil {
			slog.Warn("Failed to collect metrics for discovered package",
				"name", name,
				"owner", pkg.Owner,
				"package", discoveredPkg.Name,
				"error", err)
		} else {
			successCount++
		}
	}

	gc.metrics.CollectionSuccessCounter.WithLabelValues(name, strconv.Itoa(interval)).Inc()
	gc.metrics.CollectionIntervalGauge.WithLabelValues(name).Set(float64(interval))

	duration := time.Since(startTime).Seconds()
	gc.metrics.CollectionDurationGauge.WithLabelValues(name, strconv.Itoa(interval)).Set(duration)
	gc.metrics.CollectionTimestampGauge.WithLabelValues(name, strconv.Itoa(interval)).Set(float64(time.Now().Unix()))

	slog.Info("GHCR owner package discovery completed",
		"name", name,
		"owner", pkg.Owner,
		"total_packages", len(packages),
		"successful_collections", successCount,
		"duration", duration)
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
	gc.updatePackageMetrics(ctx, pkg, packageInfo, versions)

	return nil
}

func (gc *GHCRCollector) getPackageInfo(ctx context.Context, owner, repo, packageName string) (*GHCRPackageResponse, error) {
	resp, err := gc.makeGitHubAPIRequest(ctx, fmt.Sprintf("/users/%s/packages/container/%s", owner, packageName))
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("Error closing response body", "error", err)
		}
	}()

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

	userReq, err := http.NewRequestWithContext(ctx, http.MethodGet, userURL, nil)
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
		if err := userResp.Body.Close(); err != nil {
			slog.Error("Error closing user response body", "error", err)
		}

		// Replace /users/ with /orgs/ in the path
		orgPath := strings.Replace(path, "/users/", "/orgs/", 1)
		orgURL := fmt.Sprintf("https://api.github.com%s", orgPath)

		orgReq, err := http.NewRequestWithContext(ctx, http.MethodGet, orgURL, nil)
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
		if err := orgResp.Body.Close(); err != nil {
			slog.Error("Error closing org response body", "error", err)
		}

		return nil, fmt.Errorf("API request failed with status %d", orgResp.StatusCode)
	}

	// If user endpoint fails with something other than 404, return that error
	if err := userResp.Body.Close(); err != nil {
		slog.Error("Error closing user response body", "error", err)
	}

	return nil, fmt.Errorf("API request failed with status %d", userResp.StatusCode)
}

func (gc *GHCRCollector) getPackageVersions(ctx context.Context, owner, repo, packageName string) ([]GHCRVersionResponse, error) {
	resp, err := gc.makeGitHubAPIRequest(ctx, fmt.Sprintf("/users/%s/packages/container/%s/versions", owner, packageName))
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("Error closing response body", "error", err)
		}
	}()

	var versions []GHCRVersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, err
	}

	return versions, nil
}

func (gc *GHCRCollector) updatePackageMetrics(ctx context.Context, pkg config.PackageGroup, packageInfo *GHCRPackageResponse, versions []GHCRVersionResponse) {
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

	// Try to get actual download statistics from the package page
	downloadCount, err := gc.getPackageDownloadStats(ctx, pkg.Owner, pkg.Repo)
	if err != nil {
		slog.Warn("Failed to get download statistics", "package", pkg.Repo, "error", err)
		// Set to -1 to indicate no data available
		gc.metrics.PackageDownloadStatsGauge.WithLabelValues(pkg.Owner, pkg.Repo).Set(-1)
	} else {
		gc.metrics.PackageDownloadStatsGauge.WithLabelValues(pkg.Owner, pkg.Repo).Set(float64(downloadCount))
	}

	if !lastPublished.IsZero() {
		gc.metrics.PackageLastPublishedGauge.WithLabelValues(pkg.Owner, pkg.Repo).Set(float64(lastPublished.Unix()))
	}

	slog.Info("Updated package metrics",
		"package", pkg.Repo,
		"version_count", packageInfo.VersionCount,
		"download_count", downloadCount,
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

// getPackageDownloadStats scrapes the package page to get actual download statistics
func (gc *GHCRCollector) getPackageDownloadStats(ctx context.Context, owner, packageName string) (int64, error) {
	slog.Info("Starting download statistics collection", "owner", owner, "package", packageName)

	// Construct the package page URL
	packageURL := fmt.Sprintf("https://github.com/%s/%s/pkgs/container/%s", owner, packageName, packageName)
	slog.Debug("Constructed package URL", "url", packageURL)

	// Create request to the package page
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, packageURL, nil)
	if err != nil {
		slog.Error("Failed to create HTTP request", "owner", owner, "package", packageName, "error", err)
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	slog.Debug("Created HTTP request successfully")

	// Set headers to mimic a browser request
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Cache-Control", "max-age=0")
	slog.Debug("Set browser-like headers", "user_agent", req.Header.Get("User-Agent"))

	// Make the request
	slog.Debug("Making HTTP request to package page")

	resp, err := gc.client.Do(req)
	if err != nil {
		slog.Error("Failed to fetch package page", "owner", owner, "package", packageName, "url", packageURL, "error", err)
		return 0, fmt.Errorf("failed to fetch package page: %w", err)
	}

	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			slog.Warn("Failed to close response body", "error", closeErr)
		}
	}()

	slog.Debug("Received HTTP response", "status_code", resp.StatusCode, "content_length", resp.ContentLength, "content_type", resp.Header.Get("Content-Type"))

	if resp.StatusCode != http.StatusOK {
		slog.Error("Package page returned non-OK status", "owner", owner, "package", packageName, "status_code", resp.StatusCode, "url", packageURL)
		return 0, fmt.Errorf("package page returned status %d", resp.StatusCode)
	}

	// Read the response body
	slog.Debug("Reading response body")

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Failed to read response body", "owner", owner, "package", packageName, "error", err)
		return 0, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle gzip decompression if needed
	if resp.Header.Get("Content-Encoding") == "gzip" {
		slog.Debug("Decompressing gzipped response")

		gzReader, err := gzip.NewReader(strings.NewReader(string(body)))
		if err != nil {
			slog.Error("Failed to create gzip reader", "owner", owner, "package", packageName, "error", err)
			return 0, fmt.Errorf("failed to create gzip reader: %w", err)
		}

		defer func() {
			if closeErr := gzReader.Close(); closeErr != nil {
				slog.Warn("Failed to close gzip reader", "error", closeErr)
			}
		}()

		// Read the decompressed content
		decompressedBody, err := io.ReadAll(gzReader)
		if err != nil {
			slog.Error("Failed to read decompressed body", "owner", owner, "package", packageName, "error", err)
			return 0, fmt.Errorf("failed to read decompressed body: %w", err)
		}

		body = decompressedBody
		slog.Debug("Gzip decompression successful", "original_size", len(body), "decompressed_size", len(decompressedBody))
	}

	bodySize := len(body)
	slog.Debug("Response body read successfully", "body_size_bytes", bodySize)

	if bodySize == 0 {
		slog.Error("Response body is empty", "owner", owner, "package", packageName, "url", packageURL)
		return 0, fmt.Errorf("response body is empty")
	}

	// Parse the HTML document
	slog.Debug("Parsing HTML document", "body_size_bytes", bodySize)

	// Simple grep-like approach: find "Total downloads" and get the next line
	htmlContent := string(body)
	lines := strings.Split(htmlContent, "\n")

	var downloadLine string

	for i, line := range lines {
		if strings.Contains(line, "Total downloads") {
			if i+1 < len(lines) {
				downloadLine = strings.TrimSpace(lines[i+1])
				slog.Debug("Found download line after 'Total downloads'", "line", downloadLine)

				break
			}
		}
	}

	if downloadLine == "" {
		slog.Error("Download statistics not found", "owner", owner, "package", packageName)

		// Log a few lines around where "Total downloads" should be for debugging
		for i, line := range lines {
			if strings.Contains(line, "download") {
				slog.Debug("Found line with 'download'", "line_number", i, "content", strings.TrimSpace(line))

				if i+1 < len(lines) {
					slog.Debug("Next line content", "line_number", i+1, "content", strings.TrimSpace(lines[i+1]))
				}
			}
		}

		return 0, fmt.Errorf("download statistics not found in package page")
	}

	slog.Debug("Found download line", "line", downloadLine)

	// Extract the title attribute which contains the full number
	// Look for title="123456" in the line (e.g., from <h3 title="123456">123K</h3>)
	titleStart := strings.Index(downloadLine, `title="`)
	if titleStart == -1 {
		slog.Error("Download count title attribute not found", "owner", owner, "package", packageName, "line", downloadLine)
		return 0, fmt.Errorf("download count title attribute not found")
	}

	titleStart += 7 // Skip 'title="'

	titleEnd := strings.Index(downloadLine[titleStart:], `"`)
	if titleEnd == -1 {
		slog.Error("Download count title attribute malformed", "owner", owner, "package", packageName, "line", downloadLine)
		return 0, fmt.Errorf("download count title attribute malformed")
	}

	title := downloadLine[titleStart : titleStart+titleEnd]
	slog.Debug("Extracted title attribute", "title", title)

	// Parse the download count from the title attribute
	downloadCount, err := strconv.ParseInt(title, 10, 64)
	if err != nil {
		slog.Error("Failed to parse download count", "owner", owner, "package", packageName, "title", title, "error", err)
		return 0, fmt.Errorf("failed to parse download count %s: %w", title, err)
	}

	slog.Info("Successfully extracted download statistics", "owner", owner, "package", packageName, "download_count", downloadCount, "raw_title", title)

	return downloadCount, nil
}

// getOwnerPackages retrieves all packages for a given owner
func (gc *GHCRCollector) getOwnerPackages(ctx context.Context, owner string) ([]GHCRPackageResponse, error) {
	slog.Info("Getting packages for owner", "owner", owner)

	// Try user endpoint first
	resp, err := gc.makeGitHubAPIRequest(ctx, fmt.Sprintf("/users/%s/packages?package_type=container", owner))
	if err != nil {
		// If user endpoint fails, try org endpoint
		slog.Debug("User endpoint failed, trying org endpoint", "owner", owner, "error", err)

		resp, err = gc.makeGitHubAPIRequest(ctx, fmt.Sprintf("/orgs/%s/packages?package_type=container", owner))
		if err != nil {
			return nil, fmt.Errorf("failed to get packages for owner %s: %w", owner, err)
		}
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("Error closing response body", "error", err)
		}
	}()

	var packages []GHCRPackageResponse
	if err := json.NewDecoder(resp.Body).Decode(&packages); err != nil {
		return nil, fmt.Errorf("failed to decode packages response: %w", err)
	}

	slog.Info("Retrieved packages for owner", "owner", owner, "package_count", len(packages))

	return packages, nil
}
