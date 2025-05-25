# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build examples
RUN go build -o /bin/echo-server ./examples/echo-server
RUN go build -o /bin/calculator ./examples/calculator
RUN go build -o /bin/file-manager ./examples/file-manager
RUN go build -o /bin/weather-service ./examples/weather-service

# Runtime stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates

# Copy binaries
COPY --from=builder /bin/echo-server /bin/echo-server
COPY --from=builder /bin/calculator /bin/calculator
COPY --from=builder /bin/file-manager /bin/file-manager
COPY --from=builder /bin/weather-service /bin/weather-service

# Default to echo server
CMD ["/bin/echo-server"]