# Twin in Disguise - Multi-Architecture Docker Image
# Supports: linux/amd64, linux/arm64, and other platforms
# Copyright 2025 Matt Ho
# Licensed under the Apache License, Version 2.0

# Build stage
FROM golang:1.24-alpine AS builder

# Build arguments for target platform (auto-detected by Docker)
ARG TARGETOS
ARG TARGETARCH

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary for target platform
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build \
    -ldflags="-w -s" \
    -o twin-in-disguise \
    ./cmd/twin-in-disguise

# Runtime stage
FROM alpine:latest

# Install CA certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 app && \
    adduser -D -u 1000 -G app app

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/twin-in-disguise .

# Change ownership
RUN chown -R app:app /app

# Switch to non-root user
USER app

# Expose default port
EXPOSE 8080

# Set default environment variables
ENV PORT=8080

# Environment variables that can be set at runtime:
# - GEMINI_API_KEY: Required - Your Google Gemini API key
# - PORT: Optional - HTTP server port (default: 8080)
# - VERBOSE: Optional - Enable verbose logging
# - DEBUG: Optional - Enable debug logging
#
# Example usage:
# docker run -e GEMINI_API_KEY="your-key" -e PORT=8080 -e DEBUG=true -p 8080:8080 twin-in-disguise

# Run the binary
ENTRYPOINT ["/app/twin-in-disguise"]
