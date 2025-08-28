package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"ghcr-exporter/internal/collectors"
	"ghcr-exporter/internal/config"
	"ghcr-exporter/internal/logging"
	"ghcr-exporter/internal/metrics"
	"ghcr-exporter/internal/server"
	"ghcr-exporter/internal/version"
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
		versionInfo := version.Get()
		fmt.Printf("ghcr-exporter %s\n", versionInfo.Version)
		fmt.Printf("Commit: %s\n", versionInfo.Commit)
		fmt.Printf("Build Date: %s\n", versionInfo.BuildDate)
		fmt.Printf("Go Version: %s\n", versionInfo.GoVersion)
		os.Exit(0)
	}

	// Use environment variable if config flag is not provided
	if configPath == "" {
		if envConfig := os.Getenv("CONFIG_PATH"); envConfig != "" {
			configPath = envConfig
		} else {
			configPath = "config.yaml"
		}
	}

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		slog.Error("Failed to load configuration", "error", err, "path", configPath)
		os.Exit(1)
	}

	// Configure logging
	logging.Configure(&logging.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
	})

	// Initialize metrics
	metricsRegistry := metrics.NewRegistry()

	// Set version info metric
	versionInfo := version.Get()
	metricsRegistry.VersionInfo.WithLabelValues(versionInfo.Version, versionInfo.Commit, versionInfo.BuildDate).Set(1)

	// Create collectors
	ghcrCollector := collectors.NewGHCRCollector(cfg, metricsRegistry)

	// Start collectors
	ctx, cancel := context.WithCancel(context.Background())

	ghcrCollector.Start(ctx)

	// Create and start server
	srv := server.New(cfg, metricsRegistry)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Shutting down gracefully...")
		cancel()

		if err := srv.Shutdown(); err != nil {
			slog.Error("Error during server shutdown", "error", err)
		}
	}()

	// Start server
	if err := srv.Start(); err != nil {
		slog.Error("Server failed", "error", err)
		cancel() // Cancel context before exiting
		os.Exit(1)
	}
}
