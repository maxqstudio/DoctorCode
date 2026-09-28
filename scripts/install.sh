#!/usr/bin/env bash
set -euo pipefail

version=""
install_dir="${DOCTORCODE_INSTALL_DIR:-$HOME/.local/bin}"
source_dir=""
base_url="https://github.com/maxqstudio/DoctorCode/releases/download"

usage() {
  cat <<'EOF'
Usage: install.sh [--version VERSION] [--install-dir DIR] [--source-dir DIR] [--base-url URL]

Network installs require --version. --source-dir is intended for offline installs
and release acceptance; it auto-detects the single matching native archive.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version) version="${2:-}"; shift 2 ;;
    --install-dir) install_dir="${2:-}"; shift 2 ;;
    --source-dir) source_dir="${2:-}"; shift 2 ;;
    --base-url) base_url="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

case "$(uname -s)" in
  Linux) os="linux" ;;
  Darwin) os="darwin" ;;
  *) echo "unsupported operating system: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

if [[ -n "$source_dir" ]]; then
  source_dir="$(cd "$source_dir" && pwd)"
  shopt -s nullglob
  matches=("$source_dir"/doctorcode_*_"$os"_"$arch".tar.gz)
  shopt -u nullglob
  if [[ ${#matches[@]} -ne 1 ]]; then
    echo "expected exactly one native archive in $source_dir, found ${#matches[@]}" >&2
    exit 1
  fi
  archive="${matches[0]}"
  filename="$(basename "$archive")"
  checksum_file="$source_dir/checksums.txt"
  if [[ ! -f "$checksum_file" ]]; then
    echo "missing checksums.txt in $source_dir" >&2
    exit 1
  fi
  if [[ -z "$version" ]]; then
    prefix="doctorcode_"
    suffix="_${os}_${arch}.tar.gz"
    version="${filename#"$prefix"}"
    version="${version%"$suffix"}"
  fi
  cp "$archive" "$tmp/$filename"
  cp "$checksum_file" "$tmp/checksums.txt"
else
  if [[ -z "$version" ]]; then
    echo "--version is required for network installs" >&2
    exit 2
  fi
  filename="doctorcode_${version}_${os}_${arch}.tar.gz"
  curl -fsSL "$base_url/v$version/$filename" -o "$tmp/$filename"
  curl -fsSL "$base_url/v$version/checksums.txt" -o "$tmp/checksums.txt"
fi

expected="$(awk -v f="$filename" '$2 == f {print $1}' "$tmp/checksums.txt")"
if [[ -z "$expected" ]] || [[ "$(printf '%s\n' "$expected" | wc -l | tr -d ' ')" != "1" ]]; then
  echo "checksum entry missing or ambiguous for $filename" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$tmp/$filename" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "$tmp/$filename" | awk '{print $1}')"
else
  echo "sha256sum or shasum is required" >&2
  exit 1
fi

actual_lower="$(printf '%s' "$actual" | tr '[:upper:]' '[:lower:]')"
expected_lower="$(printf '%s' "$expected" | tr '[:upper:]' '[:lower:]')"
if [[ "$actual_lower" != "$expected_lower" ]]; then
  echo "SHA-256 mismatch for $filename" >&2
  exit 1
fi

extract_dir="$tmp/extract"
mkdir -p "$extract_dir"
tar -xzf "$tmp/$filename" -C "$extract_dir"
package_dir="$extract_dir/doctorcode_${version}_${os}_${arch}"
for binary in doctorcode doctorcode-mcp; do
  if [[ ! -f "$package_dir/$binary" ]]; then
    echo "release archive missing $binary" >&2
    exit 1
  fi
done

mkdir -p "$install_dir"
cp "$package_dir/doctorcode" "$install_dir/doctorcode"
cp "$package_dir/doctorcode-mcp" "$install_dir/doctorcode-mcp"
chmod 0755 "$install_dir/doctorcode" "$install_dir/doctorcode-mcp"

echo "DoctorCode $version installed to $install_dir"
echo "Add $install_dir to PATH if it is not already present."
