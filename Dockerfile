# Multi-stage Docker build for MCP Memory Server with WebUI
# Builds both Go backend and Next.js frontend in optimized production setup

# Stage 1: Build the Go backend
FROM golang:1.24-alpine AS go-builder

# Install build dependencies
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

# Stage 2: Build the Next.js frontend
FROM node:20-alpine AS frontend-builder

# Set working directory
WORKDIR /frontend

# Copy package files
COPY web-ui/package.json web-ui/package-lock.json* ./

# Install dependencies
RUN npm ci --only=production

# Copy frontend source
COPY web-ui/ ./

# Set build environment variables
ENV NEXT_TELEMETRY_DISABLED=1
ENV NODE_ENV=production

# Build the frontend
RUN npm run build

# Stage 3: Production runtime with both backend and frontend
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    nodejs \
    npm \
    ca-certificates \
    curl \
    tzdata \
    && update-ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S mcpuser && \
    adduser -u 1001 -S mcpuser -G mcpuser

# Set working directory
WORKDIR /app

# Copy Go binary from builder stage
COPY --from=go-builder --chown=mcpuser:mcpuser /build/mcp-memory-server /app/

# Copy Next.js build from frontend builder
COPY --from=frontend-builder --chown=mcpuser:mcpuser /frontend/.next/standalone /app/frontend/
COPY --from=frontend-builder --chown=mcpuser:mcpuser /frontend/.next/static /app/frontend/.next/static
COPY --from=frontend-builder --chown=mcpuser:mcpuser /frontend/public /app/frontend/public

# Create required directories with proper ownership
RUN mkdir -p /app/data /app/config /app/logs /app/backups /app/audit_logs /app/docs && \
    chown -R mcpuser:mcpuser /app

# Copy configuration templates
COPY --chown=mcpuser:mcpuser configs/docker/ /app/config/

# Copy MCP proxy for stdio <> HTTP bridging
COPY --chown=mcpuser:mcpuser mcp-proxy.js /app/

# Copy startup script
COPY --chown=mcpuser:mcpuser <<'EOF' /app/start.sh
#!/bin/sh
set -e

# Start the Go backend in the background
echo "Starting MCP Memory Server..."
/app/mcp-memory-server -mode=http -addr=:9080 &
BACKEND_PID=$!

# Start the Next.js frontend in the background
echo "Starting WebUI..."
cd /app/frontend && node server.js &
FRONTEND_PID=$!

# Function to handle shutdown
shutdown() {
    echo "Shutting down..."
    kill $BACKEND_PID $FRONTEND_PID 2>/dev/null || true
    wait $BACKEND_PID $FRONTEND_PID 2>/dev/null || true
    exit 0
}

# Set up signal handlers
trap shutdown SIGTERM SIGINT

# Wait for both processes
wait $BACKEND_PID $FRONTEND_PID
EOF

RUN chmod +x /app/start.sh

# Switch to non-root user
USER mcpuser

# Expose ports
# 9080: MCP Memory Server API
# 3000: WebUI (Next.js)
# 8081: Health check
# 8082: Metrics
EXPOSE 9080 3000 8081 8082

# Set labels following OCI standards
LABEL \
    org.opencontainers.image.title="MCP Memory Server with WebUI" \
    org.opencontainers.image.description="Intelligent conversation memory server with web interface" \
    org.opencontainers.image.version="VERSION_PLACEHOLDER" \
    org.opencontainers.image.vendor="fredcamaral" \
    org.opencontainers.image.licenses="Apache-2.0" \
    org.opencontainers.image.source="https://github.com/LerianStudio/mcp-memory"

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
    CONFIG_PATH=/app/config/config.yaml \
    NEXT_PUBLIC_API_URL=http://localhost:9080 \
    NEXT_PUBLIC_GRAPHQL_URL=http://localhost:9080/graphql \
    NEXT_PUBLIC_WS_URL=ws://localhost:9080/ws \
    NODE_ENV=production

# Start both services
ENTRYPOINT ["/app/start.sh"]