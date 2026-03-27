# ═══════════════════════════════════════════════════════════════
# BrixaScaler - Production Docker Image
# ═══════════════════════════════════════════════════════════════

# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /build

# Copy go mod files
COPY integration/go/go.mod integration/go/go.sum ./
RUN go mod download

# Copy source
COPY integration/go/ .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o brixascaler .

# ═══════════════════════════════════════════════════════════════
# Production stage - Minimal distroless image
# ═══════════════════════════════════════════════════════════════

FROM gcr.io/distroless/base-debian12:nonroot

# Create non-root user
RUN adduser --disabled-password --gecos "" --shell /bin/false brixascaler

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/brixascaler .

# Create config directory
RUN mkdir -p /app/config && chown brixascaler:brixascaler /app/config

# Use read-only filesystem
VOLUME ["/app/config"]

# Switch to non-root user
USER brixascaler

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

# Expose ports
EXPOSE 8080 9090

# Run as read-only with no shell access
ENTRYPOINT ["/app/brixascaler"]
