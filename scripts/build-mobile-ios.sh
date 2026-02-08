#!/bin/bash
# Build GoEngineKenga for iOS (ebitenmobile bind)
# Requires: Xcode, macOS, ebitenmobile
# Output: mobile/GoEngineKenga.xcframework

set -e
cd "$(dirname "$0")/.."

ebitenmobile bind -target ios -o mobile/GoEngineKenga.xcframework ./mobile
echo "iOS framework: mobile/GoEngineKenga.xcframework"
