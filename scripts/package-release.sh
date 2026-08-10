#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: package-release.sh VERSION OUTPUT_DIR" >&2
  exit 2
fi

version=$1
output_dir=$2
if [[ ! ${version} =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
  echo "invalid release version: ${version}" >&2
  exit 2
fi

project_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
mkdir -p "${output_dir}"
output_dir=$(cd "${output_dir}" && pwd -P)
if find "${output_dir}" -mindepth 1 -print -quit | grep -q .; then
  echo "output directory must be empty: ${output_dir}" >&2
  exit 2
fi

release_temp=$(mktemp -d)
trap 'rm -rf "${release_temp}"' EXIT
source_date_epoch=${SOURCE_DATE_EPOCH:-0}
if command -v sha256sum >/dev/null 2>&1; then
  checksum_command=(sha256sum)
elif command -v shasum >/dev/null 2>&1; then
  checksum_command=(shasum -a 256)
else
  echo 'package release requires sha256sum or shasum' >&2
  exit 1
fi
if [[ ! ${source_date_epoch} =~ ^[0-9]+$ ]]; then
  echo "SOURCE_DATE_EPOCH must be a non-negative integer" >&2
  exit 2
fi

targets=(linux/amd64 linux/arm64 darwin/amd64 darwin/arm64)
for target in "${targets[@]}"; do
  os=${target%/*}
  arch=${target#*/}
  base="helm4-plugin-preflight_${version}_${os}_${arch}"
  archive_root="${release_temp}/${base}"
  mkdir -p "${archive_root}"

  (
    cd "${project_root}"
    CGO_ENABLED=0 GOOS=${os} GOARCH=${arch} go build \
      -mod=vendor \
      -trimpath \
      -buildvcs=false \
      -ldflags "-s -w -X main.version=${version}" \
      -o "${archive_root}/helm4-plugin-preflight" \
      ./cmd/helm4-plugin-preflight
  )
  cp "${project_root}/LICENSE" "${project_root}/NOTICE" \
    "${project_root}/THIRD_PARTY_LICENSES.md" "${archive_root}/"

  tar --sort=name \
    --mtime="@${source_date_epoch}" \
    --owner=0 --group=0 --numeric-owner \
    -C "${release_temp}" \
    -czf "${output_dir}/${base}.tar.gz" \
    "${base}"
  rm -rf "${archive_root}"
done

(
  cd "${output_dir}"
  LC_ALL=C "${checksum_command[@]}" ./*.tar.gz | sed 's#  \./#  #' > SHA256SUMS
)
