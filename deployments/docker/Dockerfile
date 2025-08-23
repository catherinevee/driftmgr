# Multi-stage Docker build for minimal production image
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o driftmgr \
    ./cmd/main.go

# Production stage
FROM scratch

# Copy CA certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /app/driftmgr /driftmgr

# Create non-root user
USER 65534:65534

# Set entrypoint
ENTRYPOINT ["/driftmgr"]

# Default command
CMD ["--help"]

# Metadata
LABEL org.opencontainers.image.title="Terraform Import Helper"
LABEL org.opencontainers.image.description="A professional tool for discovering and importing cloud resources into Terraform"
LABEL org.opencontainers.image.vendor="Catherine Vee"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.source="https://github.com/catherinevee/driftmgr"
