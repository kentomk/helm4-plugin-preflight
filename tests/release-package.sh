#!/usr/bin/env bash
set -euo pipefail

project_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
test_temp=$(mktemp -d)
trap 'rm -rf "${test_temp}"' EXIT

export SOURCE_DATE_EPOCH=1784555473
"${project_root}/scripts/package-release.sh" v0.1.0 "${test_temp}/first"
"${project_root}/scripts/package-release.sh" v0.1.0 "${test_temp}/second"

cmp "${test_temp}/first/SHA256SUMS" "${test_temp}/second/SHA256SUMS"
test "$(find "${test_temp}/first" -name '*.tar.gz' -type f | wc -l)" -eq 4
(
  cd "${test_temp}/first"
  sha256sum --check SHA256SUMS
)

archive="${test_temp}/first/helm4-plugin-preflight_v0.1.0_linux_arm64.tar.gz"
tar -xzf "${archive}" -C "${test_temp}"
release_root="${test_temp}/helm4-plugin-preflight_v0.1.0_linux_arm64"
test "$("${release_root}/helm4-plugin-preflight" version)" = \
  'helm4-plugin-preflight v0.1.0'
test -s "${release_root}/LICENSE"
test -s "${release_root}/NOTICE"
test -s "${release_root}/THIRD_PARTY_LICENSES.md"

set +e
"${project_root}/scripts/package-release.sh" not-a-version "${test_temp}/invalid" \
  >"${test_temp}/invalid.out" 2>"${test_temp}/invalid.err"
invalid_exit=$?
set -e
test "${invalid_exit}" -eq 2
grep -q 'invalid release version' "${test_temp}/invalid.err"
