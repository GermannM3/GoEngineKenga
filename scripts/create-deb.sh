#!/bin/bash

# Create Debian package for GoEngineKenga
# Usage: ./create-deb.sh -version 1.0.0

set -e

VERSION="1.0.0"
ARCH="amd64"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -version|--version)
            VERSION="$2"
            shift 2
            ;;
        -arch|--arch)
            ARCH="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [-version VERSION] [-arch ARCH]"
            exit 1
            ;;
    esac
done

PACKAGE_NAME="goenginekenga"
DEB_DIR="deb-package"
DEB_NAME="${PACKAGE_NAME}_${VERSION}_${ARCH}.deb"

echo "Creating Debian package $DEB_NAME"

# Clean previous build
rm -rf "$DEB_DIR"
mkdir -p "$DEB_DIR"

# Create directory structure
mkdir -p "$DEB_DIR/DEBIAN"
mkdir -p "$DEB_DIR/usr/bin"
mkdir -p "$DEB_DIR/usr/share/doc/$PACKAGE_NAME"
mkdir -p "$DEB_DIR/usr/share/$PACKAGE_NAME/examples"

# Copy binaries
if [ -f "dist/kenga-editor-linux-$ARCH" ]; then
    cp "dist/kenga-editor-linux-$ARCH" "$DEB_DIR/usr/bin/goenginekenga-editor"
    cp "dist/kenga-linux-$ARCH" "$DEB_DIR/usr/bin/goenginekenga-cli"
else
    echo "Error: Binaries not found in dist/"
    exit 1
fi

# Copy documentation
cp README.md "$DEB_DIR/usr/share/doc/$PACKAGE_NAME/"
cp BUILD.md "$DEB_DIR/usr/share/doc/$PACKAGE_NAME/" 2>/dev/null || true

# Copy examples
cp -r samples/* "$DEB_DIR/usr/share/$PACKAGE_NAME/examples/" 2>/dev/null || true

# Create control file
cat > "$DEB_DIR/DEBIAN/control" << EOF
Package: $PACKAGE_NAME
Version: $VERSION
Section: games
Priority: optional
Architecture: $ARCH
Depends: libc6 (>= 2.17)
Installed-Size: 0
Maintainer: GoEngineKenga Team <team@goenginekenga.org>
Description: Modern game engine written in Go
 A powerful, lightweight game engine built with Go programming language.
 Features include ECS architecture, WebGPU rendering, WASM scripting,
 physics simulation, and visual editor.
EOF

# Create postinst script
cat > "$DEB_DIR/DEBIAN/postinst" << 'EOF'
#!/bin/bash
set -e

# Create desktop entry
cat > /usr/share/applications/goenginekenga.desktop << EOL
[Desktop Entry]
Name=GoEngineKenga
Comment=Modern game engine written in Go
Exec=/usr/bin/goenginekenga-editor
Icon=goenginekenga
Terminal=false
Type=Application
Categories=Development;Game;
EOL

chmod +x /usr/share/applications/goenginekenga.desktop

# Update desktop database
if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database
fi
EOF

chmod 755 "$DEB_DIR/DEBIAN/postinst"

# Create prerm script
cat > "$DEB_DIR/DEBIAN/prerm" << 'EOF'
#!/bin/bash
set -e

# Remove desktop entry
rm -f /usr/share/applications/goenginekenga.desktop

# Update desktop database
if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database
fi
EOF

chmod 755 "$DEB_DIR/DEBIAN/prerm"

# Set permissions
chmod 755 "$DEB_DIR/usr/bin/goenginekenga-editor"
chmod 755 "$DEB_DIR/usr/bin/goenginekenga-cli"
chmod 644 "$DEB_DIR/usr/share/doc/$PACKAGE_NAME/"*

# Calculate installed size (in KB)
INSTALLED_SIZE=$(du -sk "$DEB_DIR" | cut -f1)
sed -i "s/^Installed-Size:.*/Installed-Size: $INSTALLED_SIZE/" "$DEB_DIR/DEBIAN/control"

# Build package
dpkg-deb --build "$DEB_DIR" "$DEB_NAME"

echo "Debian package created: $DEB_NAME"

# Clean up
rm -rf "$DEB_DIR"

# Create checksum
if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$DEB_NAME" > "${DEB_NAME}.sha256"
    echo "Checksum created: ${DEB_NAME}.sha256"
fi