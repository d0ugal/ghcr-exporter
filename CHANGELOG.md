# Changelog

All notable changes to ghcr-exporter will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0](https://github.com/d0ugal/ghcr-exporter/compare/v1.4.0...v2.0.0) (2025-09-13)


### ⚠ BREAKING CHANGES

* support repository discovery and improve config structure

### Features

* support repository discovery and improve config structure ([0892678](https://github.com/d0ugal/ghcr-exporter/commit/089267828c04843acc9c9e3aa5022608cbf30cd5))

## [1.4.0](https://github.com/d0ugal/ghcr-exporter/compare/v1.3.2...v1.4.0) (2025-09-12)


### Features

* replace latest docker tags with versioned variables for Renovate compatibility ([cc8892b](https://github.com/d0ugal/ghcr-exporter/commit/cc8892befabb1816b86d380d680523c6bd2fe532))

## [1.3.2](https://github.com/d0ugal/ghcr-exporter/compare/v1.3.1...v1.3.2) (2025-09-05)


### Bug Fixes

* **deps:** update module github.com/prometheus/client_golang to v1.23.2 ([b110b93](https://github.com/d0ugal/ghcr-exporter/commit/b110b9361a40c0b961c36c5ee106ddd879328788))
* **deps:** update module github.com/prometheus/client_golang to v1.23.2 ([c032b84](https://github.com/d0ugal/ghcr-exporter/commit/c032b84467b9d1e46a9410245216a66b9afc5cd4))

## [1.3.1](https://github.com/d0ugal/ghcr-exporter/compare/v1.3.0...v1.3.1) (2025-09-04)


### Bug Fixes

* **deps:** update module github.com/prometheus/client_golang to v1.23.1 ([2ea7a97](https://github.com/d0ugal/ghcr-exporter/commit/2ea7a970f11dd2550081a9ddb90d3c4071f1dc9c))
* **deps:** update module github.com/prometheus/client_golang to v1.23.1 ([ecc8a71](https://github.com/d0ugal/ghcr-exporter/commit/ecc8a7187129f05714f5b01328600924e784da6a))

## [1.3.0](https://github.com/d0ugal/ghcr-exporter/compare/v1.2.0...v1.3.0) (2025-09-04)


### Features

* update dev build versioning to use semver-compatible pre-release tags ([4821374](https://github.com/d0ugal/ghcr-exporter/commit/4821374450ad1cc4224523cf73bed9124a84c2a3))


### Bug Fixes

* **ci:** add v prefix to dev tags for consistent versioning ([edb79ed](https://github.com/d0ugal/ghcr-exporter/commit/edb79ede3743eaed97572f8be984e16aaa9e4c9e))
* use actual release version as base for dev tags instead of hardcoded 0.0.0 ([33539ee](https://github.com/d0ugal/ghcr-exporter/commit/33539ee91ba80a1ecb0364488eb06c1e0644d1db))
* use fetch-depth: 0 instead of fetch-tags for full git history ([4fbb734](https://github.com/d0ugal/ghcr-exporter/commit/4fbb734e4f6d21f5b6674efe1be9f281d5b6b7ee))
* use fetch-tags instead of fetch-depth for GitHub Actions ([d396654](https://github.com/d0ugal/ghcr-exporter/commit/d396654d2ee6d3645883a0b12b42f3c95a5ca124))

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
