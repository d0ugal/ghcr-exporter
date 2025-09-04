# Changelog

All notable changes to ghcr-exporter will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0](https://github.com/d0ugal/ghcr-exporter/compare/v1.1.2...v1.2.0) (2025-09-04)


### Features

* change default logging format to JSON ([1bff3b4](https://github.com/d0ugal/ghcr-exporter/commit/1bff3b47eddcea8b7500a09c9fdcd4fde228415f))
* enable global automerge in Renovate config ([9b27884](https://github.com/d0ugal/ghcr-exporter/commit/9b2788410e52f044f5c2b975afe8e2d0cafa6672))

## [1.1.2](https://github.com/d0ugal/ghcr-exporter/compare/v1.1.1...v1.1.2) (2025-09-03)


### Bug Fixes

* pin Alpine version to 3.22.1 for consistency ([96b1e6b](https://github.com/d0ugal/ghcr-exporter/commit/96b1e6bcd0d4e5d2672ca24229efb865323eaf93))

## [1.1.1](https://github.com/d0ugal/ghcr-exporter/compare/v1.1.0...v1.1.1) (2025-09-03)


### Bug Fixes

* pass version information to Docker build in CI workflow ([44ffcc0](https://github.com/d0ugal/ghcr-exporter/commit/44ffcc0a09f6001aa76de6e3eab9f022ec712eca))

## [1.1.0](https://github.com/d0ugal/ghcr-exporter/compare/v1.0.0...v1.1.0) (2025-09-01)


### Features

* implement actual download statistics scraping from package pages ([40ff36d](https://github.com/d0ugal/ghcr-exporter/commit/40ff36d2cd95dae2dbead9422e3cc1c06509fce0))

## 1.0.0 (2025-08-28)


### Bug Fixes

* resolve all linting issues ([ac5de74](https://github.com/d0ugal/ghcr-exporter/commit/ac5de74f74239bc1ce06bdc392bca9e2bab69cbb))

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
