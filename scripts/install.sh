#!/bin/bash
set -e

# DriftMgr Installation Script
# This script downloads and installs the latest version of DriftMgr

REPO="catherinevee/driftmgr"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS and architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

# Map architecture names
case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Map OS names
case "$OS" in
    linux)
        OS="linux"
        ;;
    darwin)
        OS="darwin"
        ;;
    mingw*|msys*|cygwin*|windows*)
        OS="windows"
        ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

echo "Installing DriftMgr for $OS/$ARCH..."

# Get latest version
if [ -z "$VERSION" ]; then
    echo "Fetching latest version..."
    VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$VERSION" ]; then
        echo "Failed to get latest version. Please specify VERSION environment variable."
        exit 1
    fi
fi

echo "Installing version: $VERSION"

# Construct download URL
ARCHIVE_EXT="tar.gz"
if [ "$OS" = "windows" ]; then
    ARCHIVE_EXT="zip"
fi

DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/driftmgr-$VERSION-$OS-$ARCH.$ARCHIVE_EXT"
TEMP_DIR="$(mktemp -d)"

echo "Downloading from: $DOWNLOAD_URL"

# Download the archive
cd "$TEMP_DIR"
if command -v wget > /dev/null; then
    wget -q -O "driftmgr.$ARCHIVE_EXT" "$DOWNLOAD_URL"
elif command -v curl > /dev/null; then
    curl -sL -o "driftmgr.$ARCHIVE_EXT" "$DOWNLOAD_URL"
else
    echo "Error: wget or curl is required to download DriftMgr"
    exit 1
fi

# Extract the archive
echo "Extracting..."
if [ "$ARCHIVE_EXT" = "zip" ]; then
    unzip -q "driftmgr.$ARCHIVE_EXT"
else
    tar -xzf "driftmgr.$ARCHIVE_EXT"
fi

# Install binaries
echo "Installing to $INSTALL_DIR..."
BINARY_SUFFIX=""
if [ "$OS" = "windows" ]; then
    BINARY_SUFFIX=".exe"
fi

# Check if we need sudo
SUDO=""
if [ ! -w "$INSTALL_DIR" ] && [ "$EUID" -ne 0 ]; then
    SUDO="sudo"
fi

# Install main binary
if [ -f "driftmgr-$OS-$ARCH$BINARY_SUFFIX" ]; then
    $SUDO install -m 755 "driftmgr-$OS-$ARCH$BINARY_SUFFIX" "$INSTALL_DIR/driftmgr$BINARY_SUFFIX"
    echo "✅ Installed driftmgr to $INSTALL_DIR/driftmgr$BINARY_SUFFIX"
fi

# Install server binary
if [ -f "driftmgr-server-$OS-$ARCH$BINARY_SUFFIX" ]; then
    $SUDO install -m 755 "driftmgr-server-$OS-$ARCH$BINARY_SUFFIX" "$INSTALL_DIR/driftmgr-server$BINARY_SUFFIX"
    echo "✅ Installed driftmgr-server to $INSTALL_DIR/driftmgr-server$BINARY_SUFFIX"
fi

# Clean up
cd /
rm -rf "$TEMP_DIR"

echo ""
echo "Installation complete!"
echo ""
echo "To get started, run:"
echo "  driftmgr --help"
echo ""
echo "To start the server:"
echo "  driftmgr-server --port 8080"
echo ""

# Check if install dir is in PATH
if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo "⚠️  Warning: $INSTALL_DIR is not in your PATH"
    echo "Add it to your PATH by adding this line to your shell profile:"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
fi