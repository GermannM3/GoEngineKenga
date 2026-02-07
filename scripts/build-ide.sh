#!/bin/bash
# Build GoEngineKenga IDE (Tauri + React + Monaco)
# On Linux: produces .deb, .rpm, .AppImage
# Usage: ./scripts/build-ide.sh [version]

set -e

VERSION="${1:-0.1.0}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IDE_DIR="$(dirname "$SCRIPT_DIR")/ide"

echo "Building GoEngineKenga IDE v$VERSION"
echo "Working directory: $IDE_DIR"

cd "$IDE_DIR"

# Install deps if needed
if [ ! -d "node_modules" ]; then
    echo "Installing npm dependencies..."
    npm install
fi

# Update version
sed -i "s/\"version\": \"[^\"]*\"/\"version\": \"$VERSION\"/" package.json
sed -i "s/\"version\": \"[^\"]*\"/\"version\": \"$VERSION\"/" src-tauri/tauri.conf.json

# Build
echo "Running tauri build..."
npm run tauri build

# Output
BUNDLE_DIR="$IDE_DIR/src-tauri/target/release/bundle"
if [ -d "$BUNDLE_DIR" ]; then
    echo ""
    echo "Build complete! Installers:"
    find "$BUNDLE_DIR" -type f \( -name "*.deb" -o -name "*.rpm" -o -name "*.AppImage" \) 2>/dev/null | while read f; do
        echo "  $f"
    done
fi
