# BrixaScaler Makefile
# Usage: make <target>

.PHONY: help install test benchmark run clean docker-up docker-down

# Default target
help:
	@echo "BrixaScaler Development Commands"
	@echo "================================"
	@echo "  make install     - Install dependencies"
	@echo "  make test        - Run unit tests"
	@echo "  make benchmark  - Run performance benchmarks"
	@echo "  make run        - Run the scaler locally"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make docker-up  - Start Docker stack"
	@echo "  make docker-down - Stop Docker stack"
	@echo ""

# Install dependencies
install:
	@echo "Installing dependencies..."
	cd integration/go && go mod download
	cd integration && npm install

# Run unit tests
test:
	@echo "Running unit tests..."
	cd integration/go && go test -v -cover ./...

# Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	cd integration/go && go test -bench=. -benchmem -count=5 ./...

# Run with custom config
run:
	@echo "Starting BrixaScaler..."
	cd integration/go && go run server.go

# Development mode (with demo flag)
dev:
	@echo "Starting BrixaScaler in development mode..."
	cd integration/go && DEMO_MODE=true go run server.go

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf integration/go/scaler
	rm -rf data/
	rm -rf *.test

# Docker commands
docker-up:
	@echo "Starting Docker stack..."
	docker-compose up -d
	@echo ""
	@echo "Services:"
	@echo "  - RPC:        http://localhost:8545"
	@echo "  - WebSocket:  ws://localhost:8546"
	@echo "  - Metrics:    http://localhost:9090/metrics"
	@echo "  - Grafana:    http://localhost:3001 (admin/admin)"
	@echo ""
	@echo "Run 'make docker-logs' to view logs"

docker-down:
	@echo "Stopping Docker stack..."
	docker-compose down

docker-logs:
	docker-compose logs -f

docker-logs-scaler:
	docker-compose logs -f scaler

# Build Docker image
docker-build:
	docker-compose build scaler

# Full stack test (starts, tests, stops)
full-test: docker-up
	sleep 5
	make test
	make benchmark
	docker-down

# Quick health check
health:
	@curl -s http://localhost:9090/stats | jq '.tps' 2>/dev/null || echo "Service not running"