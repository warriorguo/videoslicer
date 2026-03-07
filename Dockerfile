# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code (bust cache on every build)
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-X main.buildTime=$(date -u +%Y%m%d%H%M%S)" -o videoslicer ./cmd/server

# Runtime stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ffmpeg ca-certificates

# Create app user
RUN addgroup -g 1001 -S videoslicer && \
    adduser -u 1001 -S videoslicer -G videoslicer

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/videoslicer .

# Create tasks directory
RUN mkdir -p /app/tasks && chown -R videoslicer:videoslicer /app

# Switch to app user
USER videoslicer

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./videoslicer"]