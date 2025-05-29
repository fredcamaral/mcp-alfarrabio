# Multi-stage Docker build inspired by HashiCorp's Terraform MCP
# This follows HashiCorp's containerization best practices for security and efficiency

# Build stage - Use Debian-based Go for glibc compatibility
FROM golang:1.24-bookworm AS builder

# Install build dependencies including C compiler for CGO
RUN apt-get update && apt-get install -y \
    git \
    gcc \
    libc6-dev \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user for build
RUN groupadd -g 1001 mcpuser && \
    useradd -u 1001 -g mcpuser -s /bin/sh mcpuser

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

# Production stage - Using distroless for minimal attack surface
FROM gcr.io/distroless/base-debian12:nonroot

# Copy necessary files from builder
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder --chown=nonroot:nonroot /build/mcp-memory-server /app/

# Copy configuration templates
COPY --chown=nonroot:nonroot configs/docker/ /app/config/

# Expose ports
EXPOSE 9080 8081 8082

# Note: Health checks should be handled by orchestration layer (k8s, docker-compose)

# Set labels following OCI standards
LABEL \
    org.opencontainers.image.title="Claude Vector Memory MCP Server" \
    org.opencontainers.image.description="Intelligent conversation memory server for Claude MCP" \
    org.opencontainers.image.version="1.0.0" \
    org.opencontainers.image.vendor="fredcamaral" \
    org.opencontainers.image.licenses="MIT" \
    org.opencontainers.image.source="https://github.com/fredcamaral/mcp-memory" \
    org.opencontainers.image.documentation="https://github.com/fredcamaral/mcp-memory/blob/main/README.md"

# Volumes should be defined in docker-compose or k8s manifests

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