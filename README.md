# GHCR Exporter

A Prometheus exporter for GitHub Container Registry (GHCR) metrics.

**Image**: `ghcr.io/d0ugal/ghcr-exporter:v2.11.1`

## Metrics

### Package Information
- `ghcr_exporter_info` - Information about the exporter
- `ghcr_package_version_count` - Total number of versions for a package
- `ghcr_package_downloads` - **Actual download count** scraped from package pages
- `ghcr_package_last_published_timestamp` - Last published timestamp

### Collection Metrics
- `ghcr_collection_duration_seconds` - Collection duration
- `ghcr_collection_success_total` - Successful collections
- `ghcr_collection_failed_total` - Failed collections

### Endpoints
- `GET /`: HTML dashboard with service status and metrics information
- `GET /metrics`: Prometheus metrics endpoint
- `GET /health`: Health check endpoint

## Quick Start

### Docker Compose

```yaml
version: '3.8'
services:
  ghcr-exporter:
    image: ghcr.io/d0ugal/ghcr-exporter:v2.11.1
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/app/config.yaml:ro
    restart: unless-stopped
```

1. Create a `config.yaml` file (see Configuration section)
2. Run: `docker-compose up -d`
3. Access metrics: `curl http://localhost:8080/metrics`

## Configuration

Create a `config.yaml` file to configure GitHub API and packages to monitor:

```yaml
server:
  host: "0.0.0.0"
  port: 8080

logging:
  level: "info"
  format: "json"

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

## Deployment

### Docker Compose (Environment Variables)

```yaml
version: '3.8'
services:
  ghcr-exporter:
    image: ghcr.io/d0ugal/ghcr-exporter:v2.11.1
    ports:
      - "8080:8080"
    environment:
      - GHCR_EXPORTER_GITHUB_TOKEN=your_github_token_here
      - GHCR_EXPORTER_PACKAGES=filesystem-exporter:d0ugal:filesystem-exporter,mqtt-exporter:d0ugal:mqtt-exporter
    restart: unless-stopped
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ghcr-exporter
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ghcr-exporter
  template:
    metadata:
      labels:
        app: ghcr-exporter
    spec:
      containers:
      - name: ghcr-exporter
        image: ghcr.io/d0ugal/ghcr-exporter:v2.11.1
        ports:
        - containerPort: 8080
        env:
        - name: GHCR_EXPORTER_GITHUB_TOKEN
          valueFrom:
            secretKeyRef:
              name: github-credentials
              key: token
        - name: GHCR_EXPORTER_PACKAGES
          value: "filesystem-exporter:d0ugal:filesystem-exporter,mqtt-exporter:d0ugal:mqtt-exporter"
```

## Prometheus Integration

Add to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'ghcr-exporter'
    static_configs:
      - targets: ['ghcr-exporter:8080']
```

## Important Note About Download Statistics

The `ghcr_package_downloads` metric provides **actual download counts** by scraping the package page HTML, which matches what you see on GitHub (e.g., "Total Downloads 176K"). This is different from version count, which only represents the number of different versions/tags available.

## Development

### Building

```bash
make build
```

### Testing

```bash
make test
```

### Linting

```bash
make lint
```

### Formatting

```bash
make fmt
```

## License

This project is licensed under the MIT License.