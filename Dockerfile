# ===========================================
# Antigravity Lite - Optimized Dockerfile
# Multi-stage build for minimal image size
# ===========================================

# Stage 1: Build
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev git

# Copy dependency files first (better layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with optimizations
RUN CGO_ENABLED=1 GOOS=linux go build \
    -a -ldflags '-s -w -linkmode external -extldflags "-static"' \
    -o antigravity-lite .

# Stage 2: Runtime
FROM alpine:3.19

WORKDIR /app

# Install minimal runtime dependencies
RUN apk --no-cache add ca-certificates tzdata wget && \
    rm -rf /var/cache/apk/*

# Create non-root user for security
RUN addgroup -g 1000 antigravity && \
    adduser -u 1000 -G antigravity -s /bin/sh -D antigravity

# Copy binary and default config
COPY --from=builder --chown=antigravity:antigravity /app/antigravity-lite .
COPY --from=builder --chown=antigravity:antigravity /app/config.yaml ./config.yaml.default

# Create data directory with correct permissions
RUN mkdir -p /app/data && chown -R antigravity:antigravity /app

# Switch to non-root user
USER antigravity

# Expose port
EXPOSE 8045

# Set timezone
ENV TZ=Asia/Shanghai

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8045/health || exit 1

# Entrypoint with config fallback
CMD ["sh", "-c", "if [ ! -f /app/config.yaml ]; then cp /app/config.yaml.default /app/config.yaml; fi && ./antigravity-lite"]
