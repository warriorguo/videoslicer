.PHONY: build run test clean deps docker-build docker-run setup-db

# Build the application
build:
	go build -o bin/videoslicer ./cmd/server

# Run the application
run:
	go run ./cmd/server

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf tasks/

# Download dependencies
deps:
	go mod download
	go mod tidy

# Setup database (requires PostgreSQL running)
setup-db:
	createdb videoslicer || true
	psql -d videoslicer -f internal/database/schema.sql

# Docker build
docker-build:
	docker build -t videoslicer:latest .

# Docker run with compose
docker-run:
	docker-compose up -d

# Install development dependencies
dev-deps:
	go install github.com/air-verse/air@latest

# Run with hot reload (requires air)
dev:
	air

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Check if required tools are available
check-deps:
	@which ffmpeg > /dev/null || (echo "FFmpeg is required but not installed" && exit 1)
	@which ffprobe > /dev/null || (echo "FFprobe is required but not installed" && exit 1)
	@echo "All required dependencies are available"