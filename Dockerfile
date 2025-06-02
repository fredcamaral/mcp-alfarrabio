# Multi-stage Docker build for production deployment
# Follows security best practices with Alpine Linux base images

# Build stage - Alpine Go image for smaller footprint and security
FROM golang:1.24-alpine AS builder

# Install build dependencies including C compiler for CGO
RUN apk add --no-cache \
    git \
    gcc \
    musl-dev \
    ca-certificates \
    tzdata \
    && update-ca-certificates

# Create non-root user for build (Alpine syntax)
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

# Build with optimization flags for production
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o mcp-memory-server \
    ./cmd/server

# Verify the binary exists and is executable
RUN ls -la mcp-memory-server

# Production stage - Alpine with Node.js for mcp-proxy.js
FROM alpine:3.19

# Install runtime dependencies (Node.js and basic utilities)
RUN apk add --no-cache \
    nodejs \
    npm \
    ca-certificates \
    curl \
    tzdata \
    && update-ca-certificates

# Create non-root user (Alpine syntax)
RUN addgroup -g 1001 -S mcpuser && \
    adduser -u 1001 -S mcpuser -G mcpuser

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder --chown=mcpuser:mcpuser /build/mcp-memory-server /app/

# Create required directories with proper ownership
RUN mkdir -p /app/data /app/config /app/logs /app/backups /app/audit_logs /app/docs && \
    chown -R mcpuser:mcpuser /app

# Copy configuration templates
COPY --chown=mcpuser:mcpuser configs/docker/ /app/config/

# Copy MCP proxy for stdio <> HTTP bridging
COPY --chown=mcpuser:mcpuser mcp-proxy.js /app/

# Switch to non-root user
USER mcpuser

# Expose ports (MCP server, health, metrics)
EXPOSE 9080 8081 8082

# Note: Health checks should be handled by orchestration layer (k8s, docker-compose)

# Set labels following OCI standards
LABEL \
    org.opencontainers.image.title="Claude Vector Memory MCP Server" \
    org.opencontainers.image.description="Intelligent conversation memory server for Claude MCP" \
    org.opencontainers.image.version="VERSION_PLACEHOLDER" \
    org.opencontainers.image.vendor="fredcamaral" \
    org.opencontainers.image.licenses="Apache-2.0" \
    org.opencontainers.image.source="https://github.com/LerianStudio/mcp-memory" \
    org.opencontainers.image.documentation="https://github.com/LerianStudio/mcp-memory/blob/main/README.md"

# Volumes should be defined in docker-compose or k8s manifests

# Set consistent environment variables
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

# Define the command
ENTRYPOINT ["/app/mcp-memory-server"]
CMD ["-mode=http", "-addr=:9080"]
