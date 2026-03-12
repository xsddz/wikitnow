#!/bin/bash

# wikitnow install script
# https://github.com/xsddz/wikitnow

set -e

REPO="xsddz/wikitnow"
BIN_NAME="wikitnow"
INSTALL_DIR="/usr/local/bin"

# ── 卸载 ─────────────────────────────────────────────────────────────────────
if [ "$1" = "uninstall" ]; then
  echo "Uninstalling $BIN_NAME..."

  # 删除二进制
  BIN_PATH="$INSTALL_DIR/$BIN_NAME"
  if [ -f "$BIN_PATH" ]; then
    if [ -w "$INSTALL_DIR" ]; then
      rm -f "$BIN_PATH"
    else
      sudo rm -f "$BIN_PATH"
    fi
    echo "✅ Removed binary: $BIN_PATH"
  else
    echo "   Binary not found: $BIN_PATH (skipped)"
  fi

  # 删除家目录配置目录
  HOME_CONFIG="$HOME/.wikitnow"
  if [ -d "$HOME_CONFIG" ]; then
    rm -rf "$HOME_CONFIG"
    echo "✅ Removed config dir: $HOME_CONFIG"
  else
    echo "   Config dir not found: $HOME_CONFIG (skipped)"
  fi

  echo ""
  echo "$BIN_NAME uninstalled."
  exit 0
fi

# ── 安装 ─────────────────────────────────────────────────────────────────────

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

echo "✅ $BIN_NAME installed successfully!"
echo "   Binary: $INSTALL_DIR/$BIN_NAME"
echo ""
echo "Run '$BIN_NAME -h' to get started."
echo "To set up ignore rules, run: $BIN_NAME config init-ignore --dest ~/.wikitnow/ignore"
echo "To uninstall, run: curl -fsSL https://raw.githubusercontent.com/$REPO/main/scripts/install.sh | bash -s uninstall"
