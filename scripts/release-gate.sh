#!/usr/bin/env bash
set -euo pipefail

project_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
cd "${project_root}"

tests/quality-gate.sh
go mod vendor
git diff --exit-code -- go.mod go.sum vendor
tests/publisher-contract.sh
tests/publisher-payload.sh
tests/action-wrapper.sh
tests/release-package.sh
tests/release-workflow.sh
tests/native-comparison.sh
tests/quickstart-clean.sh
