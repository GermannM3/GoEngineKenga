#!/bin/bash
# Create AppImage for GoEngineKenga (portable Linux)
# Requires: appimagetool, or use docker with linuxdeploy
# Usage: ./create-appimage.sh -version 1.0.0

set -e

VERSION="1.0.0"
ARCH="x86_64"

while [[ $# -gt 0 ]]; do
    case $1 in
        -version|--version) VERSION="$2"; shift 2 ;;
        -arch|--arch) ARCH="$2"; shift 2 ;;
        *) echo "Unknown: $1"; exit 1 ;;
    esac
done

APPIMAGE_DIR="AppDir"
rm -rf "$APPIMAGE_DIR"
mkdir -p "$APPIMAGE_DIR/usr/bin"
mkdir -p "$APPIMAGE_DIR/usr/share/applications"
mkdir -p "$APPIMAGE_DIR/usr/share/goenginekenga/examples"

# Build
if [ ! -f "dist/kenga-linux-amd64" ]; then
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "dist/kenga-linux-amd64" ./cmd/kenga
fi
CGO_ENABLED=1 go build -o "dist/kenga-editor-linux-amd64" ./cmd/kenga-editor 2>/dev/null || true

cp "dist/kenga-linux-amd64" "$APPIMAGE_DIR/usr/bin/kenga"
[ -f "dist/kenga-editor-linux-amd64" ] && cp "dist/kenga-editor-linux-amd64" "$APPIMAGE_DIR/usr/bin/kenga-editor"
chmod +x "$APPIMAGE_DIR/usr/bin/"*
cp -r samples/* "$APPIMAGE_DIR/usr/share/goenginekenga/examples/" 2>/dev/null || true

# Desktop entry
cat > "$APPIMAGE_DIR/usr/share/applications/goenginekenga.desktop" << EOF
[Desktop Entry]
Name=GoEngineKenga
Comment=Game engine (Go)
Exec=kenga-editor
Icon=goenginekenga
Terminal=false
Type=Application
Categories=Development;Game;
EOF

# AppRun
cat > "$APPIMAGE_DIR/AppRun" << 'EOF'
#!/bin/bash
HERE="$(dirname "$(readlink -f "$0")")"
export PATH="$HERE/usr/bin:$PATH"
exec "$HERE/usr/bin/kenga-editor" "$@"
EOF
chmod +x "$APPIMAGE_DIR/AppRun"

# Icon (placeholder - create 64x64 PNG if needed)
# echo "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==" | base64 -d > "$APPIMAGE_DIR/goenginekenga.png" 2>/dev/null || true

echo "AppDir prepared. To create AppImage, install appimagetool and run:"
echo "  appimagetool AppDir GoEngineKenga-${VERSION}-${ARCH}.AppImage"
echo ""
echo "Or download from: https://github.com/AppImage/AppImageKit/releases"
if command -v appimagetool &>/dev/null; then
    appimagetool "$APPIMAGE_DIR" "GoEngineKenga-${VERSION}-${ARCH}.AppImage"
    echo "Created: GoEngineKenga-${VERSION}-${ARCH}.AppImage"
fi
