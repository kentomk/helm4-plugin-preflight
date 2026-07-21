#!/usr/bin/env bash
set -euo pipefail

project_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
cd "${project_root}"

go test -mod=vendor ./...
go test -mod=vendor -race ./...
go vet -mod=vendor ./...
test -z "$(gofmt -l .)"

dependency_modules=$(
  go list -mod=vendor -deps -f '{{with .Module}}{{.Path}} {{.Version}}{{end}}' ./... |
    sort -u |
    sed '/^github\.com\/kento-matsuki\/helm4-plugin-preflight $/d'
)
test "${dependency_modules}" = 'go.yaml.in/yaml/v3 v3.0.4'
grep -Fxq '# go.yaml.in/yaml/v3 v3.0.4' vendor/modules.txt
grep -Fxq 'go.yaml.in/yaml/v3 v3.0.4 h1:tfq32ie2Jv2UxXFdLJdh3jXuOzWiL1fo0bu/FbuKpbc=' go.sum
printf '%s  %s\n' \
  d18f6323b71b0b768bb5e9616e36da390fbd39369a81807cca352de4e4e6aa0b \
  vendor/go.yaml.in/yaml/v3/LICENSE |
  sha256sum --check -
printf '%s  %s\n' \
  f6c2dd3a67b576eafb89b80200b8b1627230bf3821a0c14cb99a22ac19107d00 \
  vendor/go.yaml.in/yaml/v3/NOTICE |
  sha256sum --check -
grep -Fq 'go.yaml.in/yaml/v3 v3.0.4' THIRD_PARTY_LICENSES.md
grep -Fq 'MIT License' THIRD_PARTY_LICENSES.md
grep -Fq 'Apache License, Version 2.0' THIRD_PARTY_LICENSES.md
grep -Fq 'go.yaml.in/yaml/v3' NOTICE

unsafe_path=0
while IFS= read -r -d '' tracked_path; do
  case "${tracked_path}" in
    .env | .env.* | */.env | */.env.* | id_rsa | */id_rsa | credentials | */credentials | \
      .npmrc | */.npmrc | *.pem | *.key | *.p12 | *.pfx)
      printf 'credential-like tracked path: %s\n' "${tracked_path}" >&2
      unsafe_path=1
      ;;
  esac
done < <(git ls-files -z)
test "${unsafe_path}" -eq 0

private_key_prefix='BEGIN (RSA|OPENSSH|EC|DSA) PRIVATE'
private_key_suffix=' KEY'
aws_prefix='(AKIA|ASIA)'
aws_suffix='[0-9A-Z]{16}'
github_classic_prefix='gh[pousr]_'
github_classic_suffix='[A-Za-z0-9]{30,}'
github_fine_prefix='github_'
github_fine_middle='pat_'
github_fine_suffix='[A-Za-z0-9_]{20,}'
secret_pattern="${private_key_prefix}${private_key_suffix}|${aws_prefix}${aws_suffix}|${github_classic_prefix}${github_classic_suffix}|${github_fine_prefix}${github_fine_middle}${github_fine_suffix}"
if git grep -I -n -E "${secret_pattern}" -- .; then
  echo 'high-confidence credential material found in tracked content' >&2
  exit 1
fi
