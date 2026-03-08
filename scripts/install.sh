#!/bin/bash

# wikitnow install script
# https://github.com/xsddz/wikitnow

set -e

REPO="xsddz/wikitnow"
BIN_NAME="wikitnow"
INSTALL_DIR="/usr/local/bin"

# Detect OS and architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

echo "Installing $BIN_NAME for $OS/$ARCH..."

# Define latest binary URL (assuming GitHub Releases format matching Makefile)
DOWNLOAD_URL="https://github.com/$REPO/releases/latest/download/${BIN_NAME}-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
    DOWNLOAD_URL="${DOWNLOAD_URL}.exe"
fi

TMP_FILE="$(mktemp)"

# Download binary
echo "Downloading from $DOWNLOAD_URL..."
if ! curl -fsSL -o "$TMP_FILE" "$DOWNLOAD_URL"; then
    echo "Error: Failed to download $BIN_NAME. Are you sure the release exists?"
    rm -f "$TMP_FILE"
    exit 1
fi

chmod +x "$TMP_FILE"

# Install binary
echo "Installing to $INSTALL_DIR/$BIN_NAME..."
if [ -w "$INSTALL_DIR" ] || sudo mkdir -p "$INSTALL_DIR" 2>/dev/null; then
    if [ -w "$INSTALL_DIR" ]; then
        mkdir -p "$INSTALL_DIR"
        mv "$TMP_FILE" "$INSTALL_DIR/$BIN_NAME"
    else
        sudo mkdir -p "$INSTALL_DIR"
        echo "Requires sudo privileges to install to $INSTALL_DIR"
        sudo mv "$TMP_FILE" "$INSTALL_DIR/$BIN_NAME"
    fi
else
    echo "Error: Cannot create $INSTALL_DIR"
    rm -f "$TMP_FILE"
    exit 1
fi

# Install system-level default ignore rules
SYSTEM_CONFIG_DIR="/usr/local/etc/wikitnow"
SYSTEM_IGNORE_FILE="$SYSTEM_CONFIG_DIR/ignore"

echo "Installing system-level default ignore rules to $SYSTEM_IGNORE_FILE..."
if [ -w "$SYSTEM_CONFIG_DIR" ] 2>/dev/null || sudo mkdir -p "$SYSTEM_CONFIG_DIR" 2>/dev/null; then
    if [ -w "$SYSTEM_CONFIG_DIR" ]; then
        mkdir -p "$SYSTEM_CONFIG_DIR"
        (cd "$SYSTEM_CONFIG_DIR" && "$INSTALL_DIR/$BIN_NAME" config init-ignore --force)
    else
        sudo mkdir -p "$SYSTEM_CONFIG_DIR"
        (cd "$SYSTEM_CONFIG_DIR" && sudo "$INSTALL_DIR/$BIN_NAME" config init-ignore --force)
    fi
    echo "   System ignore config: $SYSTEM_IGNORE_FILE"
else
    echo "   Warning: Cannot create $SYSTEM_CONFIG_DIR, skipping system-level config."
fi

echo "✅ $BIN_NAME installed successfully!"
echo "   Binary:        $INSTALL_DIR/$BIN_NAME"
echo "   System config: $SYSTEM_IGNORE_FILE"
echo ""
echo "Run '$BIN_NAME -h' to get started."
echo "To customize ignore rules, run: $BIN_NAME config init-ignore"
