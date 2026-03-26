# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

# Copy Go module files
COPY integration/go/go.mod integration/go/go.sum ./
RUN go mod download

# Copy source code
COPY integration/go/ ./

# Build the binary
RUN CGO_ENABLED=1 go build -o scaler .

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates leveldb

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/scaler .

# Copy config and data
COPY config.yaml ./
RUN mkdir -p data

# Expose ports
EXPOSE 8545 8546 9090

# Run the scaler
ENTRYPOINT ["/app/scaler"]