#!/bin/bash

# Build script for FinanceAPI
# Usage: ./build.sh [version]

VERSION=${1:-"dev"}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "local")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "Building FinanceAPI..."
echo "  Version: $VERSION"
echo "  Commit: $COMMIT"
echo "  Time: $BUILD_TIME"

# Build with ldflags to inject build info
go build -ldflags "
    -X main.BuildVersion=$VERSION
    -X main.BuildTime=$BUILD_TIME
    -X main.BuildCommit=$COMMIT
" -o financeapi .

echo "✅ Build complete: financeapi"

# Show file info
ls -lh financeapi