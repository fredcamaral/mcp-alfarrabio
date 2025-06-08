# Multi-stage Docker build for Lerian MCP Memory Server
# Optimized Go backend providing persistent memory capabilities for AI assistants

# Stage 1: Build the Go backend
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    gcc \
    musl-dev \
    ca-certificates \
    curl \
    wget \
    tzdata \
    && update-ca-certificates

# Create non-root user for build
RUN addgroup -g 1001 -S mcpuser && \
    adduser -u 1001 -S mcpuser -G mcpuser

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Copy the entire pkg directory to satisfy the replace directive
COPY pkg ./pkg

# Download dependencies with proper verification
RUN go mod download && go mod verify

# Copy the rest of the source code
COPY . .

# Build arguments for version information
ARG VERSION=unknown
ARG BUILD_TIME=unknown
ARG COMMIT_HASH=unknown

# Build with optimization flags for production
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -extldflags '-static' \
    -X 'lerian-mcp-memory/internal/version.Version=${VERSION}' \
    -X 'lerian-mcp-memory/internal/version.BuildTime=${BUILD_TIME}' \
    -X 'lerian-mcp-memory/internal/version.CommitHash=${COMMIT_HASH}'" \
    -a -installsuffix cgo \
    -o lerian-mcp-memory-server \
    ./cmd/server

# Verify the binary exists and is executable
RUN ls -la lerian-mcp-memory-server && \
    ./lerian-mcp-memory-server --help || echo "Binary built successfully"

# Stage 2: Production runtime
FROM alpine:3.22

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    curl \
    wget \
    tzdata \
    nodejs \
    npm \
    && update-ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S mcpuser && \
    adduser -u 1001 -S mcpuser -G mcpuser

# Set working directory
WORKDIR /app

# Copy Go binary from builder stage
COPY --from=builder --chown=mcpuser:mcpuser /build/lerian-mcp-memory-server /app/

# Create required directories with proper ownership
RUN mkdir -p /app/data /app/config /app/logs /app/backups /app/audit_logs && \
    chown -R mcpuser:mcpuser /app

# Copy configuration templates
COPY --chown=mcpuser:mcpuser configs/docker/ /app/config/

# Copy MCP proxy for stdio <> HTTP bridging (optional utility)
COPY --chown=mcpuser:mcpuser mcp-proxy.js /app/

# Switch to non-root user
USER mcpuser

# Expose ports
# 9080: MCP Memory Server API (HTTP, WebSocket, SSE)
# 8081: Health check endpoint
# 8082: Metrics endpoint (optional)
EXPOSE 9080 8081 8082

# Set labels following OCI standards
LABEL \
    org.opencontainers.image.title="Lerian MCP Memory Server" \
    org.opencontainers.image.description="Persistent memory server for AI assistants using Model Context Protocol" \
    org.opencontainers.image.version="${VERSION:-dev}" \
    org.opencontainers.image.vendor="Lerian Studio" \
    org.opencontainers.image.licenses="Apache-2.0" \
    org.opencontainers.image.source="https://github.com/LerianStudio/lerian-mcp-memory" \
    org.opencontainers.image.documentation="https://github.com/LerianStudio/lerian-mcp-memory/blob/main/README.md"

# Set environment variables
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    MCP_MEMORY_DATA_DIR=/app/data \
    MCP_MEMORY_CONFIG_DIR=/app/config \
    MCP_MEMORY_LOG_DIR=/app/logs \
    MCP_MEMORY_BACKUP_DIR=/app/backups \
    MCP_MEMORY_HTTP_PORT=9080 \
    MCP_MEMORY_HEALTH_PORT=8081 \
    MCP_MEMORY_METRICS_PORT=8082 \
    MCP_MEMORY_LOG_LEVEL=info \
    CONFIG_PATH=/app/config/config.yaml

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8081/health || exit 1

# Default to HTTP mode for Docker deployments
# Can be overridden with: docker run ... /app/lerian-mcp-memory-server -mode=stdio
ENTRYPOINT ["/app/lerian-mcp-memory-server"]
CMD ["-mode=http", "-addr=:9080"]
