# syntax=docker/dockerfile:1

# ==============================================
# Build Stage
# ==============================================
FROM golang:1.26.2-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go module files first (for better caching)
COPY go.mod go.sum ./

# Download dependencies
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && \
    go mod verify

# Copy source code
COPY . .

# Build arguments
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

# Set build flags
ENV CGO_ENABLED=0
ENV GOOS=${TARGETOS}
ENV GOARCH=${TARGETARCH}

# Build the binary
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -v \
    -o nexus-hub \
    -ldflags=" \
        -s -w \
        -X main.Version=${VERSION} \
        -X main.GitCommit=${GIT_COMMIT} \
        -X main.BuildDate=${BUILD_DATE} \
    " \
    .

# ==============================================
# Final Stage
# ==============================================
FROM alpine:3.20

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    && rm -rf /var/cache/apk/*

# Set timezone
ENV TZ=UTC

# Create non-root user
RUN addgroup -g 1000 nexus && \
    adduser -D -u 1000 -G nexus nexus

# Create necessary directories
RUN mkdir -p /app/data /app/gcode_vault /app/uploads && \
    chown -R nexus:nexus /app

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/nexus-hub /usr/local/bin/nexus-hub

# Make binary executable
RUN chmod +x /usr/local/bin/nexus-hub

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/api/health || exit 1

# Switch to non-root user
USER nexus

# Volume for persistent data
VOLUME ["/app/data", "/app/gcode_vault"]

# Labels
LABEL org.opencontainers.image.source="https://github.com/asfrm/nexus-hub-core" \
      org.opencontainers.image.description="Nexus Hub Core - 3D Printing Farm Orchestration" \
      org.opencontainers.image.licenses="MIT"

# Entry point
ENTRYPOINT ["/usr/local/bin/nexus-hub"]

# Default command
CMD ["-debug"]
