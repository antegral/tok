#!/usr/bin/env sh
# tok installer — downloads the latest GitHub release binary for the host.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/antegral/tok/main/install.sh | sh
#
# Env overrides:
#   INSTALL_DIR  destination directory   (default: $HOME/.local/bin)
#   VERSION      release tag to install  (default: latest)
#
# Examples:
#   curl -fsSL .../install.sh | INSTALL_DIR=/usr/local/bin sudo sh
#   curl -fsSL .../install.sh | VERSION=v1.0.0 sh

set -eu

REPO="antegral/tok"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${VERSION:-latest}"

err()  { echo "error: $*" >&2; exit 1; }
info() { echo ">> $*"; }

# --- detect host triplet ---------------------------------------------------
os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$os" in
    linux)
        case "$arch" in
            x86_64|amd64) triplet="linux-amd64" ;;
            aarch64|arm64) triplet="linux-arm64" ;;
            *) err "unsupported linux arch: $arch" ;;
        esac ;;
    darwin)
        case "$arch" in
            x86_64) triplet="darwin-amd64" ;;
            arm64)  triplet="darwin-arm64" ;;
            *) err "unsupported darwin arch: $arch" ;;
        esac ;;
    *) err "unsupported OS: $os (only linux/darwin amd64/arm64 are released)" ;;
esac
info "host: $triplet"

# --- pick downloader -------------------------------------------------------
if command -v curl >/dev/null 2>&1; then
    fetch()        { curl -fsSL "$1" -o "$2"; }
    fetch_stdout() { curl -fsSL "$1"; }
elif command -v wget >/dev/null 2>&1; then
    fetch()        { wget -qO "$2" "$1"; }
    fetch_stdout() { wget -qO- "$1"; }
else
    err "neither curl nor wget is available"
fi

# --- pick sha256 utility ---------------------------------------------------
if command -v sha256sum >/dev/null 2>&1; then
    sha_check() { sha256sum -c "$1" --ignore-missing >/dev/null; }
elif command -v shasum >/dev/null 2>&1; then
    sha_check() { shasum -a 256 -c "$1" --ignore-missing >/dev/null; }
else
    err "no sha256 utility (install coreutils on linux; ships on macOS)"
fi

# --- resolve version -------------------------------------------------------
if [ "$VERSION" = "latest" ]; then
    info "resolving latest release..."
    VERSION=$(fetch_stdout "https://api.github.com/repos/${REPO}/releases/latest" \
        | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' \
        | head -n1)
    [ -n "$VERSION" ] || err "could not determine latest version"
fi
info "version: $VERSION"

# --- download --------------------------------------------------------------
asset="tok-${VERSION}-${triplet}.tar.gz"
asset_url="https://github.com/${REPO}/releases/download/${VERSION}/${asset}"
checksum_url="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

info "downloading $asset"
fetch "$asset_url"    "$tmp/$asset"        || err "download failed: $asset_url"
fetch "$checksum_url" "$tmp/checksums.txt" || err "checksum download failed: $checksum_url"

info "verifying SHA-256"
( cd "$tmp" && sha_check checksums.txt ) || err "SHA-256 mismatch"

# --- extract ---------------------------------------------------------------
info "extracting"
tar -xzf "$tmp/$asset" -C "$tmp"
[ -f "$tmp/tok" ] || err "binary 'tok' not found in archive"

# --- install ---------------------------------------------------------------
mkdir -p "$INSTALL_DIR"
mv "$tmp/tok" "$INSTALL_DIR/tok"
chmod +x "$INSTALL_DIR/tok"
info "installed: $INSTALL_DIR/tok"

# --- PATH check ------------------------------------------------------------
case ":${PATH:-}:" in
    *":$INSTALL_DIR:"*) ;;
    *)
        echo ""
        echo "warning: $INSTALL_DIR is not in PATH"
        echo "         add to your shell rc:"
        echo "           export PATH=\"$INSTALL_DIR:\$PATH\""
        ;;
esac

# --- done ------------------------------------------------------------------
echo ""
info "done. try: tok openai/gpt-4o <file>"
