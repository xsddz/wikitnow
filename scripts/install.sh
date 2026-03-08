#!/bin/bash

# wikitnow install script
# https://github.com/xsddz/wikitnow

set -e

REPO="xsddz/wikitnow"
BIN_NAME="wikitnow"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/usr/local/etc/wikitnow"
CONFIG_URL="https://raw.githubusercontent.com/$REPO/main/internal/configs/default_ignore"

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
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP_FILE" "$INSTALL_DIR/$BIN_NAME"
else
    echo "Requires sudo privileges to install to $INSTALL_DIR"
    sudo mv "$TMP_FILE" "$INSTALL_DIR/$BIN_NAME"
fi

# Install default system config (only if not already present, to preserve user edits)
if [ ! -f "$CONFIG_DIR/ignore" ]; then
    echo "Installing default config to $CONFIG_DIR/ignore..."
    if [ -w "$(dirname $CONFIG_DIR)" ]; then
        mkdir -p "$CONFIG_DIR"
        curl -fsSL -o "$CONFIG_DIR/ignore" "$CONFIG_URL"
    else
        sudo mkdir -p "$CONFIG_DIR"
        sudo bash -c "curl -fsSL '$CONFIG_URL' > '$CONFIG_DIR/ignore'"
    fi
    echo "   Default ignore rules installed. Feel free to customize $CONFIG_DIR/ignore."
else
    echo "   System config $CONFIG_DIR/ignore already exists, keeping existing file."
fi

echo "✅ $BIN_NAME installed successfully!"
echo "   Binary:        $INSTALL_DIR/$BIN_NAME"
echo "   System config: $CONFIG_DIR/ignore"
echo ""
echo "Run '$BIN_NAME -h' to get started."
