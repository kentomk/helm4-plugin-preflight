#!/bin/sh
set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
tool_root=$(mktemp -d)
cleanup() {
  chmod -R u+w "$tool_root" 2>/dev/null || true
  rm -rf "$tool_root"
}
trap cleanup EXIT HUP INT TERM

if [ "$(uname -s)" != Linux ] || [ "$(uname -m)" != aarch64 ]; then
  echo 'publisher gate currently supports the Linux aarch64 broker host' >&2
  exit 1
fi

fetch_verified() {
  artifact_url=$1
  artifact_path=$2
  expected_sha=$3
  status=$(curl -sS -L -o "$artifact_path" -w '%{http_code}' "$artifact_url")
  case "$status" in
    200) ;;
    401|403|429)
      echo "tool read blocked with HTTP $status" >&2
      exit 1
      ;;
    *)
      echo "unexpected tool HTTP status $status" >&2
      exit 1
      ;;
  esac
  actual_sha=$(sha256sum "$artifact_path" | cut -d ' ' -f 1)
  [ "$actual_sha" = "$expected_sha" ] || {
    echo "tool checksum mismatch: $(basename "$artifact_path")" >&2
    exit 1
  }
}

fetch_verified https://go.dev/dl/go1.26.5.linux-arm64.tar.gz "$tool_root/go.tar.gz" fe4789e92b1f33358680864bbe8704289e7bb5fc207d80623c308935bd696d49
fetch_verified https://ziglang.org/download/0.16.0/zig-aarch64-linux-0.16.0.tar.xz "$tool_root/zig.tar.xz" ea4b09bfb22ec6f6c6ceac57ab63efb6b46e17ab08d21f69f3a48b38e1534f17
fetch_verified https://get.helm.sh/helm-v4.2.2-linux-arm64.tar.gz "$tool_root/helm.tar.gz" 78803142087a0069fa4b50d3f32a84d3ef25c14d1ee8a40fbccf86a6216d2f36

tar -xzf "$tool_root/go.tar.gz" -C "$tool_root"
tar -xJf "$tool_root/zig.tar.xz" -C "$tool_root"
tar -xzf "$tool_root/helm.tar.gz" -C "$tool_root"

PATH="$tool_root/go/bin:$PATH" \
CC="$tool_root/zig-aarch64-linux-0.16.0/zig cc" \
HELM4_BINARY="$tool_root/linux-arm64/helm" \
  "$project_root/scripts/release-gate.sh"
