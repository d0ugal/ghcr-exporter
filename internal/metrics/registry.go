package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Registry holds all the metrics for the GHCR exporter
type Registry struct {
	// Version info metric
	VersionInfo *prometheus.GaugeVec

	// GHCR package metrics
	PackageDownloadsGauge     *prometheus.GaugeVec
	PackageLastPublishedGauge *prometheus.GaugeVec

	// Collection statistics
	CollectionFailedCounter  *prometheus.CounterVec
	CollectionSuccessCounter *prometheus.CounterVec
	CollectionIntervalGauge  *prometheus.GaugeVec
	CollectionDurationGauge  *prometheus.GaugeVec
	CollectionTimestampGauge *prometheus.GaugeVec

	// Metric information for UI
	metricInfo []MetricInfo
}

// MetricInfo contains information about a metric for the UI
type MetricInfo struct {
	Name         string
	Help         string
	Labels       []string
	ExampleValue string
}

// addMetricInfo adds metric information to the registry
func (r *Registry) addMetricInfo(name, help string, labels []string) {
	r.metricInfo = append(r.metricInfo, MetricInfo{
		Name:         name,
		Help:         help,
		Labels:       labels,
		ExampleValue: "",
	})
}

// NewRegistry creates a new metrics registry
func NewRegistry() *Registry {
	r := &Registry{}

	r.VersionInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ghcr_exporter_info",
			Help: "Information about the GHCR exporter",
		},
		[]string{"version", "commit", "build_date"},
	)
	r.addMetricInfo("ghcr_exporter_info", "Information about the GHCR exporter", []string{"version", "commit", "build_date"})

	r.PackageDownloadsGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ghcr_package_downloads",
			Help: "Total number of downloads for a GHCR package (using version count as proxy)",
		},
		[]string{"owner", "repo"},
	)
	r.addMetricInfo("ghcr_package_downloads", "Total number of downloads for a GHCR package (using version count as proxy)", []string{"owner", "repo"})

	r.PackageLastPublishedGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ghcr_package_last_published_timestamp",
			Help: "Timestamp of the last published version for a GHCR package",
		},
		[]string{"owner", "repo"},
	)
	r.addMetricInfo("ghcr_package_last_published_timestamp", "Timestamp of the last published version for a GHCR package", []string{"owner", "repo"})

	r.CollectionFailedCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ghcr_collection_failed_total",
			Help: "Total number of failed GHCR data collection attempts",
		},
		[]string{"repo", "interval"},
	)
	r.addMetricInfo("ghcr_collection_failed_total", "Total number of failed GHCR data collection attempts", []string{"repo", "interval"})

	r.CollectionSuccessCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ghcr_collection_success_total",
			Help: "Total number of successful GHCR data collection attempts",
		},
		[]string{"repo", "interval"},
	)
	r.addMetricInfo("ghcr_collection_success_total", "Total number of successful GHCR data collection attempts", []string{"repo", "interval"})

	r.CollectionIntervalGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ghcr_collection_interval_seconds",
			Help: "Collection interval in seconds",
		},
		[]string{"repo"},
	)
	r.addMetricInfo("ghcr_collection_interval_seconds", "Collection interval in seconds", []string{"repo"})

	r.CollectionDurationGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ghcr_collection_duration_seconds",
			Help: "Duration of GHCR data collection in seconds",
		},
		[]string{"repo", "interval"},
	)
	r.addMetricInfo("ghcr_collection_duration_seconds", "Duration of GHCR data collection in seconds", []string{"repo", "interval"})

	r.CollectionTimestampGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ghcr_collection_timestamp",
			Help: "Timestamp of the last collection",
		},
		[]string{"repo", "interval"},
	)
	r.addMetricInfo("ghcr_collection_timestamp", "Timestamp of the last collection", []string{"repo", "interval"})

	return r
}

// GetMetricsInfo returns information about all metrics for the UI
func (r *Registry) GetMetricsInfo() []MetricInfo {
	return r.metricInfo
}
