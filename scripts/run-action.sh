#!/usr/bin/env bash
set -euo pipefail

: "${GITHUB_ACTION_PATH:?GITHUB_ACTION_PATH is required}"
: "${RUNNER_TEMP:?RUNNER_TEMP is required}"

root=${H4P_INPUT_ROOT:-.}
format=${H4P_INPUT_FORMAT:-text}
binary="${RUNNER_TEMP}/helm4-plugin-preflight"

(
  cd "${GITHUB_ACTION_PATH}"
  go build -mod=vendor -trimpath -o "${binary}" ./cmd/helm4-plugin-preflight
)

args=(check --root "${root}" --format "${format}")
if [[ -n "${H4P_INPUT_HELM_PLUGINS:-}" ]]; then
  args+=(--helm-plugins "${H4P_INPUT_HELM_PLUGINS}")
fi

exec "${binary}" "${args[@]}"
