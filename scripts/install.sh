#!/usr/bin/env bash
# install.sh — install cloudres from GitHub releases.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/mars-base/cloudres/main/scripts/install.sh | bash
#
#   # Install a specific version:
#   curl -fsSL https://raw.githubusercontent.com/mars-base/cloudres/main/scripts/install.sh | bash -s v1.0.0
#
#   # Install to a custom directory:
#   curl -fsSL https://raw.githubusercontent.com/mars-base/cloudres/main/scripts/install.sh | INSTALL_DIR=~/bin bash

set -euo pipefail

REPO="mars-base/cloudres"
DEFAULT_TAG="latest"
TAG="${1:-$DEFAULT_TAG}"

# --- detect os / arch ------------------------------------------------
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)
        echo "unsupported architecture: $ARCH"
        exit 1
        ;;
esac

case "$OS" in
    linux)   BINARY="cloudres-linux-${ARCH}" ;;
    darwin)  BINARY="cloudres-darwin-${ARCH}" ;;
    *)
        echo "unsupported OS: $OS"
        exit 1
        ;;
esac

# --- resolve version --------------------------------------------------
if [ "$TAG" = "latest" ]; then
    DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BINARY}"
else
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${BINARY}"
fi

if [ -z "$DOWNLOAD_URL" ]; then
    echo "could not find download URL for ${BINARY} (tag: ${TAG})"
    exit 1
fi

# --- install ----------------------------------------------------------
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "→ downloading ${BINARY} ..."
curl -fsSL "$DOWNLOAD_URL" -o "${TMP_DIR}/cloudres"

chmod +x "${TMP_DIR}/cloudres"

if [ -w "$INSTALL_DIR" ]; then
    mv "${TMP_DIR}/cloudres" "${INSTALL_DIR}/cloudres"
else
    sudo mv "${TMP_DIR}/cloudres" "${INSTALL_DIR}/cloudres"
fi

echo "✓ cloudres installed to ${INSTALL_DIR}/cloudres"
cloudres version
