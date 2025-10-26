package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"ghcr-exporter/internal/collectors"
	"ghcr-exporter/internal/config"
	"ghcr-exporter/internal/metrics"
	"ghcr-exporter/internal/version"

	"github.com/d0ugal/promexporter/app"
	"github.com/d0ugal/promexporter/logging"
	promexporter_metrics "github.com/d0ugal/promexporter/metrics"
)

func main() {
	// Parse command line flags
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&showVersion, "v", false, "Show version information")

	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to configuration file")
	flag.Parse()

	// Show version if requested
	if showVersion {
		fmt.Printf("ghcr-exporter %s\n", version.Version)
		fmt.Printf("Commit: %s\n", version.Commit)
		fmt.Printf("Build Date: %s\n", version.BuildDate)
		os.Exit(0)
	}

	// Check if we should use environment variables
	configFromEnv := os.Getenv("GHCR_EXPORTER_CONFIG_FROM_ENV") == "true"

	// Load configuration
	var (
		cfg *config.Config
		err error
	)

	if configFromEnv {
		cfg, err = config.LoadConfig("", true)
	} else {
		// Use environment variable if config flag is not provided
		if configPath == "" {
			if envConfig := os.Getenv("CONFIG_PATH"); envConfig != "" {
				configPath = envConfig
			} else {
				configPath = "config.yaml"
			}
		}

		cfg, err = config.LoadConfig(configPath, false)
	}

	if err != nil {
		slog.Error("Failed to load configuration", "error", err, "path", configPath)
		os.Exit(1)
	}

	// Configure logging using promexporter
	logging.Configure(&logging.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
	})

	// Initialize metrics registry using promexporter
	metricsRegistry := promexporter_metrics.NewRegistry("ghcr_exporter_info")

	// Add custom metrics to the registry
	ghcrRegistry := metrics.NewGHCRRegistry(metricsRegistry)

	// Create collector
	ghcrCollector := collectors.NewGHCRCollector(cfg, ghcrRegistry)

	// Create and run application using promexporter
	application := app.New("GHCR Exporter").
		WithConfig(&cfg.BaseConfig).
		WithMetrics(metricsRegistry).
		WithCollector(ghcrCollector).
		WithVersionInfo(version.Version, version.Commit, version.BuildDate).
		Build()

	if err := application.Run(); err != nil {
		slog.Error("Application failed", "error", err)
		os.Exit(1)
	}
}
