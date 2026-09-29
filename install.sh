#!/bin/sh
# Install script for lhcli.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/aiShuiJiaoDeXioShou/lhcli/main/install.sh | sh
#
# Environment variables:
#   VERSION       release tag to install (default: latest)
#   INSTALL_DIR   target directory (default: /usr/local/bin)

set -eu

REPO="aiShuiJiaoDeXioShou/lhcli"
BINARY="lhcli"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${VERSION:-latest}"

err() {
	echo "error: $*" >&2
	exit 1
}

need_cmd() {
	command -v "$1" >/dev/null 2>&1 || err "required command not found: $1"
}

need_cmd uname
need_cmd curl
need_cmd tar
need_cmd mktemp

os="$(uname -s)"
case "$os" in
	Linux) os="linux" ;;
	Darwin) os="darwin" ;;
	*) err "unsupported operating system: $os (try installing manually from GitHub Releases)" ;;
esac

arch="$(uname -m)"
case "$arch" in
	x86_64 | amd64) arch="amd64" ;;
	arm64 | aarch64) arch="arm64" ;;
	*) err "unsupported architecture: $arch" ;;
esac

if [ "$VERSION" = "latest" ]; then
	VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
		| grep '"tag_name"' \
		| head -n1 \
		| sed -E 's/.*"([^"]+)".*/\1/')"
	[ -n "$VERSION" ] || err "failed to resolve the latest release version"
fi

ver="${VERSION#v}"
archive="${BINARY}_${ver}_${os}_${arch}.tar.gz"
url="https://github.com/${REPO}/releases/download/${VERSION}/${archive}"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "Downloading ${url}"
curl -fsSL "$url" -o "${tmp}/${archive}"

tar -xzf "${tmp}/${archive}" -C "${tmp}"
[ -f "${tmp}/${BINARY}" ] || err "archive did not contain a ${BINARY} binary"

if [ -w "$INSTALL_DIR" ]; then
	mv "${tmp}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
	echo "Installing to ${INSTALL_DIR} (may require your password)"
	sudo mv "${tmp}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

chmod +x "${INSTALL_DIR}/${BINARY}"
echo "Installed ${BINARY} ${VERSION} to ${INSTALL_DIR}/${BINARY}"
"${INSTALL_DIR}/${BINARY}" version || true
