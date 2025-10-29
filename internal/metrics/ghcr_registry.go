package metrics

import (
	promexporter_metrics "github.com/d0ugal/promexporter/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// GHCRRegistry wraps the promexporter registry with GHCR-specific metrics
type GHCRRegistry struct {
	*promexporter_metrics.Registry

	// GHCR package metrics
	PackageDownloadsGauge     *prometheus.GaugeVec
	PackageLastPublishedGauge *prometheus.GaugeVec
	PackageDownloadStatsGauge *prometheus.GaugeVec

	// Collection statistics
	CollectionFailedCounter  *prometheus.CounterVec
	CollectionSuccessCounter *prometheus.CounterVec
	CollectionIntervalGauge  *prometheus.GaugeVec
	CollectionDurationGauge  *prometheus.GaugeVec
	CollectionTimestampGauge *prometheus.GaugeVec
}

// NewGHCRRegistry creates a new GHCR metrics registry
func NewGHCRRegistry(baseRegistry *promexporter_metrics.Registry) *GHCRRegistry {
	// Get the underlying Prometheus registry
	promRegistry := baseRegistry.GetRegistry()
	factory := promauto.With(promRegistry)

	ghcr := &GHCRRegistry{
		Registry: baseRegistry,
	}

	// GHCR package metrics
	ghcr.PackageDownloadsGauge = factory.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ghcr_package_versions",
			Help: "Total number of versions for a GHCR package",
		},
		[]string{"owner", "repo"},
	)

	baseRegistry.AddMetricInfo("ghcr_package_versions", "Total number of versions for a GHCR package", []string{"owner", "repo"})

	ghcr.PackageDownloadStatsGauge = factory.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ghcr_package_downloads",
			Help: "Total number of downloads for a GHCR package (scraped from package page)",
		},
		[]string{"owner", "repo"},
	)

	baseRegistry.AddMetricInfo("ghcr_package_downloads", "Total downloads for a package from GitHub Container Registry", []string{"owner", "repo"})

	ghcr.PackageLastPublishedGauge = factory.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ghcr_package_last_published_timestamp",
			Help: "Timestamp of the last published version for a GHCR package",
		},
		[]string{"owner", "repo"},
	)

	baseRegistry.AddMetricInfo("ghcr_package_last_published_timestamp", "Timestamp of the last published version for a GHCR package", []string{"owner", "repo"})

	// Collection statistics
	ghcr.CollectionFailedCounter = factory.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ghcr_collection_failed_total",
			Help: "Total number of failed GHCR data collection attempts",
		},
		[]string{"repo", "interval"},
	)

	baseRegistry.AddMetricInfo("ghcr_collection_failed_total", "Total number of failed GHCR data collection attempts", []string{"repo", "interval"})

	ghcr.CollectionSuccessCounter = factory.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ghcr_collection_success_total",
			Help: "Total number of successful GHCR data collection attempts",
		},
		[]string{"repo", "interval"},
	)

	baseRegistry.AddMetricInfo("ghcr_collection_success_total", "Total number of successful GHCR data collection attempts", []string{"repo", "interval"})

	ghcr.CollectionIntervalGauge = factory.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ghcr_collection_interval_seconds",
			Help: "Collection interval in seconds for GHCR data",
		},
		[]string{"repo", "interval"},
	)

	baseRegistry.AddMetricInfo("ghcr_collection_interval_seconds", "Collection interval in seconds for GHCR data", []string{"repo", "interval"})

	ghcr.CollectionDurationGauge = factory.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ghcr_collection_duration_seconds",
			Help: "Duration of the last GHCR data collection in seconds",
		},
		[]string{"repo", "interval"},
	)

	baseRegistry.AddMetricInfo("ghcr_collection_duration_seconds", "Duration of the last GHCR data collection in seconds", []string{"repo", "interval"})

	ghcr.CollectionTimestampGauge = factory.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ghcr_collection_timestamp",
			Help: "Unix timestamp of the last GHCR data collection",
		},
		[]string{"repo", "interval"},
	)

	baseRegistry.AddMetricInfo("ghcr_collection_timestamp", "Unix timestamp of the last GHCR data collection", []string{"repo", "interval"})

	return ghcr
}
