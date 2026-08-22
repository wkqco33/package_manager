#!/usr/bin/env sh
# Install the latest ppm release on Linux or macOS.
# Usage: PPM_VERSION=v1.0.0 PPM_INSTALL_DIR="$HOME/.local/bin" ./install.sh
set -eu

REPOSITORY="wkqco33/package_manager"
INSTALL_DIR=${PPM_INSTALL_DIR:-"$HOME/.local/bin"}
VERSION=${PPM_VERSION:-latest}

fail() {
    printf 'ppm install: %s\n' "$1" >&2
    exit 1
}

command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v tar >/dev/null 2>&1 || fail "tar is required"
command -v uname >/dev/null 2>&1 || fail "uname is required"

case "$(uname -s)" in
Linux) OS=linux ;;
Darwin) OS=darwin ;;
*) fail "unsupported operating system: $(uname -s) (use a GitHub Release asset instead)" ;;
esac

case "$(uname -m)" in
x86_64 | amd64) ARCH=amd64 ;;
arm64 | aarch64) ARCH=arm64 ;;
*) fail "unsupported architecture: $(uname -m)" ;;
esac

# Only accept version tags that can safely be used as a URL path.
if [ "$VERSION" != latest ]; then
    case "$VERSION" in
    v[0-9]*) : ;;
    *) fail "PPM_VERSION must be a release tag such as v1.0.0" ;;
    esac
    RELEASE_PATH="releases/download/$VERSION"
else
    RELEASE_PATH="releases/latest/download"
fi

ASSET="ppm_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/$REPOSITORY/$RELEASE_PATH"
TMP_DIR=$(mktemp -d 2>/dev/null || mktemp -d -t ppm-install)
trap 'rm -rf "$TMP_DIR"' EXIT INT TERM

printf 'Downloading ppm (%s/%s, %s)...\n' "$OS" "$ARCH" "$VERSION"
curl --fail --silent --show-error --location --proto '=https' --tlsv1.2 \
    "$BASE_URL/$ASSET" -o "$TMP_DIR/$ASSET"
curl --fail --silent --show-error --location --proto '=https' --tlsv1.2 \
    "$BASE_URL/$ASSET.sha256" -o "$TMP_DIR/$ASSET.sha256"

# Verify before extracting or executing anything from the archive.
(
    cd "$TMP_DIR"
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum -c "$ASSET.sha256"
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 -c "$ASSET.sha256"
    else
        fail "sha256sum or shasum is required to verify the download"
    fi
)

tar -xzf "$TMP_DIR/$ASSET" -C "$TMP_DIR"
test -f "$TMP_DIR/ppm" || fail "release archive does not contain the ppm binary"

mkdir -p "$INSTALL_DIR"
install -m 0755 "$TMP_DIR/ppm" "$INSTALL_DIR/ppm"
printf 'Installed ppm to %s/ppm\n' "$INSTALL_DIR"
printf 'Run "%s/ppm version" to verify the installation.\n' "$INSTALL_DIR"
