#!/bin/sh
set -eu
version="${1:-v1.0.1}"
install_dir="${CODEX_HANDOFF_INSTALL_DIR:-$HOME/.local/bin}"
repo="zhiaiyrb/codex-lan-handoff"
os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in darwin) os=darwin;; linux) os=linux;; *) echo "Unsupported OS: $os" >&2; exit 1;; esac
case "$(uname -m)" in x86_64|amd64) arch=amd64;; arm64|aarch64) arch=arm64;; *) echo "Unsupported architecture" >&2; exit 1;; esac
asset="codex-lan-handoff_${os}_${arch}.tar.gz"
base="https://github.com/$repo/releases/download/$version"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM
curl -fL "$base/$asset" -o "$tmp/$asset"
curl -fL "$base/checksums.txt" -o "$tmp/checksums.txt"
expected=$(awk -v n="$asset" '$2 == n { print $1; exit }' "$tmp/checksums.txt")
[ -n "$expected" ] || { echo "Checksum not found" >&2; exit 1; }
if command -v sha256sum >/dev/null 2>&1; then actual=$(sha256sum "$tmp/$asset" | awk '{print $1}'); else actual=$(shasum -a 256 "$tmp/$asset" | awk '{print $1}'); fi
[ "$actual" = "$expected" ] || { echo "Checksum mismatch" >&2; exit 1; }
mkdir -p "$install_dir"
tar -xzf "$tmp/$asset" -C "$install_dir"
chmod 755 "$install_dir/codex-lan-handoff"
echo "Installed $install_dir/codex-lan-handoff. Ensure $install_dir is in PATH, then run: codex-lan-handoff setup"
