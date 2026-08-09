#!/usr/bin/env bash
set -euo pipefail

project_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
cd "${project_root}"

jq -e '
  .schemaVersion == 2 and (.action == "create" or .action == "update") and .owner == "kentomk" and
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
  .owner == "kentomk" and .author == "@kentomk" and
  .automatedAgent == true and
  (.createdBy | test("Matsuki Kento") and test("@kentomk") and test("AI|automated"; "i"))
' .kento-oss.json >/dev/null

grep -Eq '^## (Installation|Install|Getting Started)\b' README.md
grep -Eq '^## Quick[[:space:]]*start\b' README.md
grep -q 'Matsuki Kento' README.md
grep -q '@kentomk' README.md
grep -Eiq 'AI|automated' README.md
grep -Fq 'supported public release is `v0.1.1`' SECURITY.md
! grep -Fq 'No public release exists yet' SECURITY.md
grep -Eq 'uses: actions/checkout@[0-9a-f]{40}([[:space:]]|$)' .github/workflows/ci.yml
grep -Eq 'uses: actions/setup-go@[0-9a-f]{40}([[:space:]]|$)' .github/workflows/ci.yml
! grep -Eq 'uses: actions/(checkout|setup-go)@v[0-9]' .github/workflows/ci.yml
grep -Eq '^- uses: actions/checkout@[0-9a-f]{40}([[:space:]]|$)' README.md
grep -Eq '^- uses: actions/setup-go@[0-9a-f]{40}([[:space:]]|$)' README.md
grep -Eq '^- uses: kentomk/helm4-plugin-preflight@[0-9a-f]{40}([[:space:]]|$)' README.md
! grep -Eq '^- uses: (actions/(checkout|setup-go)|kentomk/helm4-plugin-preflight)@v[0-9]' README.md
grep -Fq 'helm4-plugin-preflight --help' README.md
grep -Fq "go-version: '1.26.5'" README.md
! grep -Fq "go-version: '1.26.x'" README.md
grep -Fq "go-version: '1.26.5'" .github/workflows/ci.yml
grep -Fq "go-version: '1.26.5'" .github/workflows/release.yml
! grep -Fq "go-version: '1.26.x'" .github/workflows/ci.yml
! grep -Fq "go-version: '1.26.x'" .github/workflows/release.yml

published_release=v0.1.1
published_main=3c47201c1903c34a30425c688bf63bf16647ec79
grep -Fq "go install github.com/kentomk/helm4-plugin-preflight/cmd/helm4-plugin-preflight@${published_release}" README.md
grep -Fq "releases/tag/${published_release}" README.md
grep -Fq "helm4-plugin-preflight_${published_release}_linux_arm64.tar.gz" README.md
grep -Fq "uses: kentomk/helm4-plugin-preflight@${published_main}" README.md
if grep -Fq 'kentomk/helm4-plugin-preflight@df37f769472f9baf99638e765e987ae39168bf93' README.md; then
  printf '%s\n' 'publisher contract: README still pins the superseded public Action revision' >&2
  exit 1
fi
! grep -Fq 'v0.1.0' README.md

help_output=$(go run ./cmd/helm4-plugin-preflight --help)
grep -Fq 'Usage:' <<<"${help_output}"
grep -Fq 'helm4-plugin-preflight check' <<<"${help_output}"
