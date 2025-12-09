# Quick Start Guide

Get ghcr-exporter running in minutes!

## Option 1: Docker (Recommended)

### 1. Pull the image
```bash
docker pull ghcr.io/d0ugal/ghcr-exporter:v2.11.12
```

### 2. Create a configuration file
```bash
# Copy the example configuration to create your own config
cp config.example.yaml config.yaml

# Edit the configuration for your environment
nano config.yaml
```

### 3. Run the container
```bash
docker run -d \
  --name ghcr-exporter \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  ghcr.io/d0ugal/ghcr-exporter:v2.11.12
```

### 4. Verify it's working
```bash
# Check the health endpoint
curl http://localhost:8080/health

# View metrics
curl http://localhost:8080/metrics
```

## Option 2: From Source

### 1. Clone the repository
```bash
git clone https://github.com/d0ugal/ghcr-exporter.git
cd ghcr-exporter
```

### 2. Build and run
```bash
# Build the application
make build

# Run with default configuration
./ghcr-exporter
```

## Option 3: Docker Compose

### 1. Create docker-compose.yml
```yaml
version: '3.8'
services:
  ghcr-exporter:
    image: ghcr.io/d0ugal/ghcr-exporter:v2.11.12
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/app/config.yaml:ro
    restart: unless-stopped
```

### 2. Run with Docker Compose
```bash
docker-compose up -d
```

## Configuration Examples

### Basic Configuration
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

### GitHub Token Setup
1. Go to GitHub Settings → Developer settings → Personal access tokens
2. Generate a new token with `read:packages`, `read:org`, and `read:user` scopes
3. Add the token to your configuration file

## Next Steps

1. **Configure for your environment** - Edit `config.yaml` with your GitHub token and packages
2. **Set up monitoring** - Add the metrics endpoint to your Prometheus configuration
3. **Create dashboards** - Build Grafana dashboards using the available metrics
4. **Set up alerts** - Configure alerts for package activity and collection failures

## Troubleshooting

### Common Issues

**GitHub Token Issues**
- Ensure your GitHub token has the required scopes: `read:packages`, `read:org`, `read:user`
- Check that the token is valid and not expired

**Configuration Errors**
- Check YAML syntax in your config file
- Validate package names and owners exist on GitHub

**API Rate Limits**
- GitHub API has rate limits; the exporter handles this gracefully
- Consider reducing collection frequency if needed

### Getting Help

- **Documentation**: See [README.md](README.md) for detailed documentation
- **Issues**: Report bugs and request features on [GitHub](https://github.com/d0ugal/ghcr-exporter/issues)
- **Examples**: Check [config.example.yaml](config.example.yaml) for more configuration examples
