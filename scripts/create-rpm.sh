#!/bin/bash
# Create RPM package for GoEngineKenga (Fedora/RHEL/openSUSE)
# Usage: ./create-rpm.sh -version 1.0.0 -arch amd64

set -e

VERSION="1.0.0"
ARCH="amd64"

while [[ $# -gt 0 ]]; do
    case $1 in
        -version|--version) VERSION="$2"; shift 2 ;;
        -arch|--arch) ARCH="$2"; shift 2 ;;
        *) echo "Unknown: $1"; exit 1 ;;
    esac
done

PACKAGE="goenginekenga"
RPM_DIR="rpm-build"
mkdir -p "$RPM_DIR/BUILD" "$RPM_DIR/RPMS" "$RPM_DIR/SOURCES" "$RPM_DIR/SPECS"

# Build if dist missing
if [ ! -f "dist/kenga-linux-$ARCH" ]; then
    echo "Building..."
    CGO_ENABLED=0 GOOS=linux GOARCH=$ARCH go build -o "dist/kenga-linux-$ARCH" ./cmd/kenga
    CGO_ENABLED=1 go build -o "dist/kenga-editor-linux-$ARCH" ./cmd/kenga-editor 2>/dev/null || true
fi

# Tarball for RPM
TARBALL="$RPM_DIR/SOURCES/${PACKAGE}-${VERSION}.tar.gz"
mkdir -p "tmp-rpm/${PACKAGE}-${VERSION}/usr/bin"
mkdir -p "tmp-rpm/${PACKAGE}-${VERSION}/usr/share/doc/${PACKAGE}"
mkdir -p "tmp-rpm/${PACKAGE}-${VERSION}/usr/share/${PACKAGE}/examples"
cp "dist/kenga-linux-$ARCH" "tmp-rpm/${PACKAGE}-${VERSION}/usr/bin/kenga"
[ -f "dist/kenga-editor-linux-$ARCH" ] && cp "dist/kenga-editor-linux-$ARCH" "tmp-rpm/${PACKAGE}-${VERSION}/usr/bin/kenga-editor"
cp README.md "tmp-rpm/${PACKAGE}-${VERSION}/usr/share/doc/${PACKAGE}/" 2>/dev/null || true
cp -r samples/* "tmp-rpm/${PACKAGE}-${VERSION}/usr/share/${PACKAGE}/examples/" 2>/dev/null || true
tar -czf "$TARBALL" -C tmp-rpm "${PACKAGE}-${VERSION}"
rm -rf tmp-rpm

# Spec file
cat > "$RPM_DIR/SPECS/${PACKAGE}.spec" << EOF
Name: $PACKAGE
Version: $VERSION
Release: 1%{?dist}
Summary: GoEngineKenga game engine
License: MIT
URL: https://github.com/GermannM3/GoEngineKenga
Source0: ${PACKAGE}-${VERSION}.tar.gz
BuildArch: $ARCH

%description
Game engine and IDE written in Go. ECS, 3D render, physics, particles, WebSocket API for CAD.

%prep
%setup -q

%install
cp -r usr %{buildroot}

%files
/usr/bin/kenga
/usr/share/doc/%{name}
/usr/share/%{name}

%changelog
* $(date +"%a %b %d %Y") GoEngineKenga Team - $VERSION
- Initial package
EOF

[ -f "dist/kenga-editor-linux-$ARCH" ] && sed -i '/\/usr\/bin\/kenga/a /usr/bin/kenga-editor' "$RPM_DIR/SPECS/${PACKAGE}.spec"

rpmbuild --define "_topdir $(pwd)/$RPM_DIR" -bb "$RPM_DIR/SPECS/${PACKAGE}.spec"
mv "$RPM_DIR/RPMS/$ARCH/"*.rpm .
echo "Created: $(ls -1 *.rpm 2>/dev/null | head -1)"
rm -rf "$RPM_DIR"
