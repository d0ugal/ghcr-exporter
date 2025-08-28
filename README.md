# GHCR Exporter

A Prometheus exporter for GitHub Container Registry (GHCR) metrics.

## Features

- Collects package download statistics from GitHub Container Registry (GHCR)
- Tracks package version counts and last published timestamps
- Monitors collection performance and success rates
- Supports both user and organization packages
- Prometheus metrics endpoint with health checks

## Configuration

Create a `config.yaml` file:

```yaml
server:
  host: "0.0.0.0"
  port: 8080

logging:
  level: "info"
  format: "text"

metrics:
  collection:
    default_interval: "60s"

# GitHub API configuration
github:
  token: "your_github_token_here"

packages:
  filesystem-exporter:
    owner: "d0ugal"
    repo: "filesystem-exporter"
  
  mqtt-exporter:
    owner: "d0ugal"
    repo: "mqtt-exporter"
  
  home-assistant:
    owner: "home-assistant"
    repo: "home-assistant"
```

## Building

```bash
make build
```

## Running

```bash
./ghcr-exporter -config config.yaml
```

## Metrics

The exporter provides the following metrics:

- `ghcr_exporter_info` - Information about the exporter
- `ghcr_package_downloads_total` - Version count (proxy for package activity)
- `ghcr_package_last_published_timestamp` - Last published timestamp
- `ghcr_collection_duration_seconds` - Collection duration
- `ghcr_collection_success_total` - Successful collections
- `ghcr_collection_failed_total` - Failed collections

## Development

```bash
# Run tests
make test

# Format code
make fmt

# Lint code
make lint
```

## Docker Deployment

The exporter is configured to run in Docker with:
- Health checks
- Non-root user
- Configurable via volume mount

```bash
# Using Docker Compose
docker-compose -f docker-compose.example.yml up -d

# Using Docker directly
docker run -d \
  --name ghcr-exporter \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  ghcr.io/d0ugal/ghcr-exporter:latest
```

## Quick Start

See [QUICKSTART.md](QUICKSTART.md) for detailed setup instructions.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
