#!/bin/bash
# Build GoEngineKenga for Android (ebitenmobile bind)
# Requires: ANDROID_HOME (e.g. ~/Library/Android/sdk), ebitenmobile
#
#	go install github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile@latest
#	export ANDROID_HOME=~/Library/Android/sdk
#
# Output: mobile/kenga.aar

set -e
cd "$(dirname "$0")/.."

ebitenmobile bind -target android -javapkg com.goenginekenga -o mobile/kenga.aar ./mobile
echo "Android AAR: mobile/kenga.aar"
