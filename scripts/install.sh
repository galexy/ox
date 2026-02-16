#!/bin/bash
# SageOx (ox) installer script
# Usage: curl -sSL https://raw.githubusercontent.com/sageox/ox/main/scripts/install.sh | bash

set -e

REPO="sageox/ox"
BINARY="ox"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

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

case "$OS" in
    darwin|linux)
        ;;
    *)
        echo "Unsupported OS: $OS"
        echo "For Windows, download manually from https://github.com/$REPO/releases"
        exit 1
        ;;
esac

# Get latest version
echo "Fetching latest version..."
VERSION=$(curl -sI "https://github.com/$REPO/releases/latest" | grep -i "^location:" | sed 's/.*tag\///' | tr -d '\r\n')

if [ -z "$VERSION" ]; then
    echo "Failed to get latest version"
    exit 1
fi

echo "Installing $BINARY $VERSION for $OS/$ARCH..."

# Download URL
ARCHIVE="sageox_${VERSION#v}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$VERSION/$ARCHIVE"

# Create temp directory
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# Download archive and checksums
echo "Downloading $URL..."
curl -sL "$URL" -o "$TMP_DIR/$ARCHIVE"
curl -sL "https://github.com/$REPO/releases/download/$VERSION/checksums.txt" -o "$TMP_DIR/checksums.txt"

# Verify checksum
echo "Verifying checksum..."
cd "$TMP_DIR"
if command -v sha256sum &> /dev/null; then
    sha256sum -c checksums.txt --ignore-missing --quiet
elif command -v shasum &> /dev/null; then
    shasum -a 256 -c checksums.txt --ignore-missing --quiet 2>/dev/null
else
    echo "Warning: no sha256sum or shasum available, skipping verification"
fi
cd - > /dev/null

echo "Extracting..."
tar -xzf "$TMP_DIR/$ARCHIVE" -C "$TMP_DIR"

# Install
echo "Installing to $INSTALL_DIR/$BINARY..."
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP_DIR/$BINARY" "$INSTALL_DIR/$BINARY"
else
    sudo mv "$TMP_DIR/$BINARY" "$INSTALL_DIR/$BINARY"
fi

chmod +x "$INSTALL_DIR/$BINARY"

echo ""
echo "Successfully installed $BINARY $VERSION"
echo "Run '$BINARY --help' to get started"
