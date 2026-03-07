#!/bin/bash

set -e

echo "🔨 Building videoslicer..."
go build -o bin/videoslicer ./cmd/server

echo "✅ Build successful!"

echo "🧪 Running unit tests..."
go test -v ./pkg/... ./internal/...

echo "🎯 Running functional tests..."
go test -v ./test -run "TestFFmpeg|TestStorage|TestWorkflow"

echo "🏗️  Building Docker image..."
docker build -t videoslicer:test .

echo "🎉 All tests passed! Ready for deployment."

echo ""
echo "📋 Summary:"
echo "- ✅ Code compiles successfully"
echo "- ✅ All unit tests pass" 
echo "- ✅ Functional tests pass"
echo "- ✅ Docker image builds"
echo ""
echo "🚀 To run the service:"
echo "1. Start PostgreSQL service"
echo "2. Set environment variables (see .env.example)"
echo "3. Run: ./bin/videoslicer"
echo ""
echo "📚 API Documentation:"
echo "- POST /v1/tasks - Upload video"
echo "- GET /v1/tasks/{id} - Check status" 
echo "- GET /v1/tasks/{id}/result - Download result"
echo "- GET /health - Health check"