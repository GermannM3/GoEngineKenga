#!/bin/bash

# GoEngineKenga Release Build Script
# Usage: ./build-release.sh -version 1.0.0

set -e

VERSION="1.0.0"
CLEAN=false
TEST=false
NO_ARCHIVE=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -version|--version)
            VERSION="$2"
            shift 2
            ;;
        -clean|--clean)
            CLEAN=true
            shift
            ;;
        -test|--test)
            TEST=true
            shift
            ;;
        -no-archive|--no-archive)
            NO_ARCHIVE=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [-version VERSION] [-clean] [-test] [-no-archive]"
            exit 1
            ;;
    esac
done

echo "Building GoEngineKenga Release v$VERSION"

# Clean previous builds
if [ "$CLEAN" = true ]; then
    echo "Cleaning previous builds..."
    rm -rf dist
fi

# Create dist directory
mkdir -p dist

# Run tests if requested
if [ "$TEST" = true ]; then
    echo "Running tests..."
    go test ./...
fi

# Build targets
TARGETS=(
    "windows:amd64"
    "windows:386"
    "linux:amd64"
    "linux:386"
    "darwin:amd64"
    "darwin:arm64"
)

LDFLAGS="-X main.version=$VERSION -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"

for target in "${TARGETS[@]}"; do
    IFS=':' read -r goos goarch <<< "$target"
    echo "Building for $goos/$goarch..."

    GOOS=$goos GOARCH=$goarch CGO_ENABLED=0 go build $LDFLAGS -o "dist/kenga-editor-$goos-$goarch" ./cmd/kenga-editor
    GOOS=$goos GOARCH=$goarch CGO_ENABLED=0 go build $LDFLAGS -o "dist/kenga-$goos-$goarch" ./cmd/kenga

    # Add .exe extension for Windows
    if [ "$goos" = "windows" ]; then
        mv "dist/kenga-editor-$goos-$goarch" "dist/kenga-editor-$goos-$goarch.exe"
        mv "dist/kenga-$goos-$goarch" "dist/kenga-$goos-$goarch.exe"
    fi
done

# Create archives
if [ "$NO_ARCHIVE" = false ]; then
    echo "Creating release archives..."

    for target in "${TARGETS[@]}"; do
        IFS=':' read -r goos goarch <<< "$target"

        if [ "$goos" = "windows" ]; then
            ext="exe"
            archive_name="GoEngineKenga-$VERSION-$goos-$goarch.zip"
            # Create zip archive (requires zip utility)
            if command -v zip >/dev/null 2>&1; then
                zip -r "dist/$archive_name" dist/kenga-editor-$goos-$goarch.exe dist/kenga-$goos-$goarch.exe README.md
            fi
        else
            archive_name="GoEngineKenga-$VERSION-$goos-$goarch.tar.gz"
            tar -czf "dist/$archive_name" -C dist kenga-editor-$goos-$goarch kenga-$goos-$goarch README.md
        fi

        if [ -f "dist/$archive_name" ]; then
            echo "Created $archive_name"
        fi
    done
fi

# Create checksums
echo "Creating checksums..."
cd dist
find . -type f -exec sha256sum {} \; > SHA256SUMS
cd ..

echo "Release build completed!"
echo "Files created in dist/ directory:"
ls -la dist/