#!/usr/bin/env bash
set -euo pipefail

project_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
test_temp=$(mktemp -d)
trap 'rm -rf "${test_temp}"' EXIT

git -C "${project_root}" archive HEAD | tar -x -C "${test_temp}"
start_seconds=$(date +%s)
SOURCE_DATE_EPOCH=0 "${test_temp}/scripts/package-release.sh" v0.1.0 "${test_temp}/dist"
(
  cd "${test_temp}/dist"
  sha256sum --check SHA256SUMS
)
archive="${test_temp}/dist/helm4-plugin-preflight_v0.1.0_linux_arm64.tar.gz"
tar -xzf "${archive}" -C "${test_temp}"
binary="${test_temp}/helm4-plugin-preflight_v0.1.0_linux_arm64/helm4-plugin-preflight"
test "$("${binary}" version)" = 'helm4-plugin-preflight v0.1.0'
set +e
"${binary}" check --root "${test_temp}/testdata/unsigned-bypass" >"${test_temp}/quickstart.out"
quickstart_exit=$?
set -e
test "${quickstart_exit}" -eq 1
grep -q 'H4P001' "${test_temp}/quickstart.out"
elapsed_seconds=$(($(date +%s) - start_seconds))
((elapsed_seconds <= 300))
printf 'clean quickstart: %d seconds\n' "${elapsed_seconds}"
