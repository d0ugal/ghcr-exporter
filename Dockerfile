# Build stage
FROM golang:1.25.2-alpine AS builder

WORKDIR /app

# Install git and ca-certificates for go mod download
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with version information
# Accept build args for version info, fall back to git describe if not provided
ARG VERSION
ARG COMMIT
ARG BUILD_DATE

RUN VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")} && \
    COMMIT=${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")} && \
    BUILD_DATE=${BUILD_DATE:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")} && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-s -w \
        -X ghcr-exporter/internal/version.Version=$VERSION \
        -X ghcr-exporter/internal/version.Commit=$COMMIT \
        -X ghcr-exporter/internal/version.BuildDate=$BUILD_DATE" \
    -o ghcr-exporter ./cmd

# Final stage
FROM alpine:3.22.2

RUN apk --no-cache add ca-certificates wget

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/ghcr-exporter .

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Change ownership of the app directory
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./ghcr-exporter", "-config", "/app/config.yaml"]
