#!/bin/bash
# Установщик GoEngineKenga для Linux — один скрипт, мастер в стиле современных программ
# Запуск: curl -fsSL https://.../install-linux.sh | sh   или   ./install-linux.sh
# Требует: собранные бинарники в dist/ или передать путь к архиву

set -e

VERSION="${VERSION:-1.0.0}"
INSTALL_PREFIX="${INSTALL_PREFIX:-/usr/local}"
BIN_DIR="$INSTALL_PREFIX/bin"
SHARE_DIR="$INSTALL_PREFIX/share/goenginekenga"
APPLICATIONS="$INSTALL_PREFIX/share/applications"

echo "=============================================="
echo "  GoEngineKenga $VERSION — установка"
echo "=============================================="
echo ""

# Определить архитектуру
ARCH=$(uname -m)
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    i686|i386) ARCH="386" ;;
    *) echo "Неизвестная архитектура: $ARCH"; exit 1 ;;
esac

# Путь к бинарникам: либо из текущей папки (dist/), либо из каталога скрипта
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-.}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DIST="$REPO_ROOT/dist"

if [ ! -f "$DIST/kenga-linux-$ARCH" ]; then
    echo "Ошибка: бинарники не найдены в $DIST/"
    echo "Сначала соберите проект: ./scripts/build-release.sh -Version $VERSION -NoArchive"
    exit 1
fi

echo "Будут установлены:"
echo "  - kenga (CLI)"
echo "  - kenga-editor"
echo "  - в каталог: $INSTALL_PREFIX"
echo ""

# Запрос прав при установке в системный каталог
NEED_SUDO=""
if [ ! -w "$INSTALL_PREFIX" ]; then
    echo "Требуются права администратора для записи в $INSTALL_PREFIX"
    NEED_SUDO="sudo"
fi

$NEED_SUDO mkdir -p "$BIN_DIR" "$SHARE_DIR/examples" "$APPLICATIONS"

$NEED_SUDO cp "$DIST/kenga-linux-$ARCH" "$BIN_DIR/kenga"
$NEED_SUDO cp "$DIST/kenga-editor-linux-$ARCH" "$BIN_DIR/kenga-editor"
$NEED_SUDO chmod 755 "$BIN_DIR/kenga" "$BIN_DIR/kenga-editor"

[ -d "$REPO_ROOT/samples" ] && $NEED_SUDO cp -r "$REPO_ROOT/samples/"* "$SHARE_DIR/examples/" 2>/dev/null || true
[ -f "$REPO_ROOT/README.md" ] && $NEED_SUDO cp "$REPO_ROOT/README.md" "$SHARE_DIR/" 2>/dev/null || true

# Desktop-файл для редактора
$NEED_SUDO tee "$APPLICATIONS/goenginekenga.desktop" > /dev/null << EOF
[Desktop Entry]
Name=GoEngineKenga Editor
Comment=Game engine and editor (Go)
Exec=$BIN_DIR/kenga-editor
Icon=applications-development
Terminal=false
Type=Application
Categories=Development;Game;
EOF

echo ""
echo "Установка завершена."
echo "  CLI:    $BIN_DIR/kenga"
echo "  Editor: $BIN_DIR/kenga-editor"
echo ""
if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
    echo "Добавьте в PATH для запуска из терминала:"
    echo "  export PATH=\"$BIN_DIR:\$PATH\""
    echo "или добавьте эту строку в ~/.profile / ~/.bashrc"
fi
echo ""
