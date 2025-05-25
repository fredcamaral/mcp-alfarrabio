# Multi-stage Docker build inspired by HashiCorp's Terraform MCP
# This follows HashiCorp's containerization best practices for security and efficiency

# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies including C compiler for CGO
RUN apk add --no-cache \
    git \
    gcc \
    musl-dev \
    ca-certificates \
    tzdata \
    && update-ca-certificates

# Create non-root user for build
RUN addgroup -g 1001 -S mcpuser && \
    adduser -u 1001 -S mcpuser -G mcpuser

# Set working directory
WORKDIR /build

# Copy go mod files first
COPY go.mod go.sum ./

# Copy the entire pkg directory to satisfy the replace directive
COPY pkg ./pkg

# Now download dependencies (skip verify for local packages)
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application with CGO enabled for chroma-go
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags='-w -s' \
    -a \
    -o mcp-memory-server \
    ./cmd/server

# Verify the binary
RUN ls -la mcp-memory-server

# Production stage
FROM alpine:3.19

# Install runtime dependencies including Node.js for MCP proxy
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    nodejs \
    npm \
    && update-ca-certificates \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1001 -S mcpuser && \
    adduser -u 1001 -S mcpuser -G mcpuser

# Create necessary directories with proper permissions
RUN mkdir -p /app/data /app/config /app/logs /app/backups && \
    chown -R mcpuser:mcpuser /app

# Copy binary from builder stage
COPY --from=builder --chown=mcpuser:mcpuser /build/mcp-memory-server /app/

# Copy MCP proxy script
COPY --chown=mcpuser:mcpuser mcp-proxy.js /app/

# Copy configuration templates
COPY --chown=mcpuser:mcpuser configs/docker/ /app/config/

# Set working directory
WORKDIR /app

# Switch to non-root user
USER mcpuser

# Create health check script
RUN echo '#!/bin/sh\ncurl -f http://localhost:9080/health || exit 1' > /app/healthcheck.sh && \
    chmod +x /app/healthcheck.sh

# Expose ports
EXPOSE 9080 8081 8082

# Add health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD /app/healthcheck.sh

# Set labels following OCI standards
LABEL \
    org.opencontainers.image.title="Claude Vector Memory MCP Server" \
    org.opencontainers.image.description="Intelligent conversation memory server for Claude MCP" \
    org.opencontainers.image.version="1.0.0" \
    org.opencontainers.image.vendor="fredcamaral" \
    org.opencontainers.image.licenses="MIT" \
    org.opencontainers.image.source="https://github.com/fredcamaral/mcp-memory" \
    org.opencontainers.image.documentation="https://github.com/fredcamaral/mcp-memory/blob/main/README.md"

# Define volumes for persistent data
VOLUME ["/app/data", "/app/logs", "/app/backups"]

# Set environment variables
ENV MCP_MEMORY_DATA_DIR=/app/data \
    MCP_MEMORY_CONFIG_DIR=/app/config \
    MCP_MEMORY_LOG_DIR=/app/logs \
    MCP_MEMORY_BACKUP_DIR=/app/backups \
    MCP_MEMORY_HTTP_PORT=8080 \
    MCP_MEMORY_HEALTH_PORT=8081 \
    MCP_MEMORY_METRICS_PORT=8082 \
    MCP_MEMORY_LOG_LEVEL=info

# Define the command
ENTRYPOINT ["/app/mcp-memory-server"]
CMD ["-mode=http", "-addr=:9080"]