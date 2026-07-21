#!/usr/bin/env bash
set -euo pipefail

project_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
test_temp=$(mktemp -d)
trap 'rm -rf "${test_temp}"' EXIT

export GITHUB_ACTION_PATH=${project_root}
export RUNNER_TEMP=${test_temp}
export H4P_INPUT_FORMAT=text

export H4P_INPUT_ROOT="${project_root}/testdata/safe-post-renderer"
unset H4P_INPUT_HELM_PLUGINS
"${project_root}/scripts/run-action.sh" >"${test_temp}/safe.out"
grep -q 'note H4P005' "${test_temp}/safe.out"

export H4P_INPUT_HELM_PLUGINS="${project_root}/testdata/safe-v1-plugin/plugins"
set +e
"${project_root}/scripts/run-action.sh" >"${test_temp}/unknown.out" 2>"${test_temp}/unknown.err"
unknown_exit=$?
set -e
test "${unknown_exit}" -eq 1
grep -q 'warning H4P004' "${test_temp}/unknown.out"

export H4P_INPUT_ROOT="${project_root}/testdata/unsigned-bypass"
unset H4P_INPUT_HELM_PLUGINS
set +e
"${project_root}/scripts/run-action.sh" >"${test_temp}/unsafe.out" 2>"${test_temp}/unsafe.err"
unsafe_exit=$?
set -e
test "${unsafe_exit}" -eq 1
grep -q 'H4P001' "${test_temp}/unsafe.out"

export H4P_INPUT_FORMAT=unsupported
set +e
"${project_root}/scripts/run-action.sh" >"${test_temp}/invalid.out" 2>"${test_temp}/invalid.err"
invalid_exit=$?
set -e
test "${invalid_exit}" -eq 2
grep -q 'unsupported format' "${test_temp}/invalid.err"
