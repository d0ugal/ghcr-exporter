# Changelog

All notable changes to ghcr-exporter will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Features

* Initial release of ghcr-exporter
* Collects package download statistics from GitHub Container Registry (GHCR)
* Tracks package version counts and last published timestamps
* Supports both user and organization packages
* Prometheus metrics endpoint with health checks
* Docker support with non-root user
* Graceful error handling and retry logic
* Web UI with metrics information
* GitHub API integration with authentication

### Metrics

* `ghcr_exporter_info` - Information about the exporter
* `ghcr_package_downloads_total` - Version count (proxy for package activity)
* `ghcr_package_last_published_timestamp` - Last published timestamp
* `ghcr_collection_duration_seconds` - Collection duration
* `ghcr_collection_success_total` - Successful collections
* `ghcr_collection_failed_total` - Failed collections
* `ghcr_collection_interval_seconds` - Collection interval
* `ghcr_collection_timestamp` - Last collection timestamp
