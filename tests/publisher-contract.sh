#!/usr/bin/env bash
set -euo pipefail

project_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
cd "${project_root}"

jq -e '
  .schemaVersion == 2 and (.action == "create" or .action == "update") and .owner == "kento-matsuki" and
  .name == "helm4-plugin-preflight" and
  (.description | type == "string" and length >= 20 and length <= 160) and
  (.topics | type == "array" and length >= 1 and length <= 10 and index("kento-oss") != null) and
  .candidateId == "20260719T061433Z-0a3a" and
  (.targetUsers | length >= 10 and length <= 500) and
  (.jobToBeDone | length >= 10 and length <= 1000) and
  (.distributionPath | length >= 10 and length <= 500) and
  (.successMetric | length >= 10 and length <= 500) and
  (.reviewAfterDays == 1) and .opportunityScore == 79 and
  (.demandEvidence | type == "array" and length >= 3 and
    all((.url | startswith("https://")) and (.kind | test("^[a-z][a-z0-9-]{2,49}$")) and (.independenceKey | length >= 3))) and
  ((.demandEvidence | map(.independenceKey | ascii_downcase) | unique | length) >= 3) and
  ((.demandEvidence | map(.kind) | unique | length) >= 2) and
  (.alternatives | type == "array" and length >= 3 and
    all((.url | startswith("https://")) and .tested == true and (.gap | length >= 10))) and
  .duplicateSearch.completed == true and (.duplicateSearch.summary | length >= 20) and
  (.differentiation | length >= 20) and
  .testCommand == "scripts/publisher-gate.sh" and .license == "Apache-2.0" and
  (.commitMessage | length >= 10 and length <= 120)
' publish-request.json >/dev/null

jq -e --slurpfile request publish-request.json '
  .schemaVersion == 1 and .candidateId == $request[0].candidateId and
  .owner == "kento-matsuki" and .author == "@kento-matsuki" and
  .automatedAgent == true and
  (.createdBy | test("Matsuki Kento") and test("@kento-matsuki") and test("AI|automated"; "i"))
' .kento-oss.json >/dev/null

grep -Eq '^## (Installation|Install|Getting Started)\b' README.md
grep -Eq '^## Quick[[:space:]]*start\b' README.md
grep -q 'Matsuki Kento' README.md
grep -q '@kento-matsuki' README.md
grep -Eiq 'AI|automated' README.md
grep -Eq 'uses: actions/checkout@[0-9a-f]{40}([[:space:]]|$)' .github/workflows/ci.yml
grep -Eq 'uses: actions/setup-go@[0-9a-f]{40}([[:space:]]|$)' .github/workflows/ci.yml
! grep -Eq 'uses: actions/(checkout|setup-go)@v[0-9]' .github/workflows/ci.yml
grep -Eq '^- uses: actions/checkout@[0-9a-f]{40}([[:space:]]|$)' README.md
grep -Eq '^- uses: actions/setup-go@[0-9a-f]{40}([[:space:]]|$)' README.md
grep -Eq '^- uses: kento-matsuki/helm4-plugin-preflight@[0-9a-f]{40}([[:space:]]|$)' README.md
! grep -Eq '^- uses: (actions/(checkout|setup-go)|kento-matsuki/helm4-plugin-preflight)@v[0-9]' README.md
