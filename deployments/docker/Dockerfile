# Multi-stage build for DriftMgr
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Set working directory
WORKDIR /build

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy all source code
COPY . .

# Build all binaries
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=v3.0.0" -o driftmgr ./cmd/driftmgr && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=v3.0.0" -o driftmgr-server ./cmd/server && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=v3.0.0" -o validate-discovery ./cmd/validate

# Final stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata git

# Create non-root user
RUN addgroup -g 1000 driftmgr && \
    adduser -D -u 1000 -G driftmgr driftmgr

# Create required directories
RUN mkdir -p /etc/driftmgr /var/log/driftmgr /var/lib/driftmgr /home/driftmgr/.driftmgr && \
    chown -R driftmgr:driftmgr /etc/driftmgr /var/log/driftmgr /var/lib/driftmgr /home/driftmgr/.driftmgr

# Copy binaries from builder
COPY --from=builder /build/driftmgr /usr/local/bin/driftmgr
COPY --from=builder /build/driftmgr-server /usr/local/bin/driftmgr-server
COPY --from=builder /build/validate-discovery /usr/local/bin/validate-discovery

# Copy configuration files if they exist
COPY --from=builder /build/configs/*.yaml /etc/driftmgr/ 2>/dev/null || true

# Make binaries executable
RUN chmod +x /usr/local/bin/driftmgr /usr/local/bin/driftmgr-server /usr/local/bin/validate-discovery

# Switch to non-root user
USER driftmgr
WORKDIR /home/driftmgr

# Set environment variables
ENV DRIFTMGR_CONFIG=/etc/driftmgr/production.yaml
ENV DRIFTMGR_LOG_DIR=/var/log/driftmgr
ENV DRIFTMGR_DATA_DIR=/var/lib/driftmgr

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health/live || exit 1

# Default to running the server
ENTRYPOINT ["/usr/local/bin/driftmgr-server"]
CMD ["--host", "0.0.0.0", "--port", "8080"]