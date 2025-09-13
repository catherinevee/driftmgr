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

# Fix line endings and validate syntax
RUN find . -name "*.go" -exec dos2unix {} \; 2>/dev/null || true
RUN go fmt ./...
RUN go vet ./...

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o driftmgr ./cmd/driftmgr

# Final stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 driftmgr && \
    adduser -D -u 1000 -G driftmgr driftmgr

# Create required directories
RUN mkdir -p /etc/driftmgr /var/log/driftmgr /var/lib/driftmgr && \
    chown -R driftmgr:driftmgr /etc/driftmgr /var/log/driftmgr /var/lib/driftmgr

# Copy binary from builder
COPY --from=builder /build/driftmgr /usr/local/bin/driftmgr

# Copy default configuration
COPY --from=builder /build/configs/production.yaml /etc/driftmgr/config.yaml

# Switch to non-root user
USER driftmgr

# Set environment variables
ENV DRIFTMGR_CONFIG=/etc/driftmgr/config.yaml
ENV DRIFTMGR_LOG_DIR=/var/log/driftmgr
ENV DRIFTMGR_DATA_DIR=/var/lib/driftmgr

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health/live || exit 1

# Run the application
ENTRYPOINT ["/usr/local/bin/driftmgr"]
CMD ["server", "--config", "/etc/driftmgr/config.yaml"]