#!/usr/bin/env bash
set -euo pipefail

project_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
test_temp=$(mktemp -d)
trap 'rm -rf "${test_temp}"' EXIT

helm_binary=${HELM4_BINARY:-}
if [[ -z ${helm_binary} ]]; then
  case $(uname -m) in
    x86_64 | amd64)
      helm_arch=amd64
      helm_sha256=9adafecab4d406853bba163a70e9f104f47dbbf65ce24b7653bae7e36150bcb6
      ;;
    aarch64 | arm64)
      helm_arch=arm64
      helm_sha256=78803142087a0069fa4b50d3f32a84d3ef25c14d1ee8a40fbccf86a6216d2f36
      ;;
    *)
      echo "unsupported native Helm test architecture: $(uname -m)" >&2
      exit 2
      ;;
  esac
  helm_archive="${test_temp}/helm-v4.2.2-linux-${helm_arch}.tar.gz"
  curl --fail --silent --show-error --location \
    "https://get.helm.sh/helm-v4.2.2-linux-${helm_arch}.tar.gz" \
    --output "${helm_archive}"
  printf '%s  %s\n' "${helm_sha256}" "${helm_archive}" | sha256sum --check -
  tar -C "${test_temp}" -xzf "${helm_archive}"
  helm_binary="${test_temp}/linux-${helm_arch}/helm"
fi
test -x "${helm_binary}"
"${helm_binary}" version --short | grep -q '^v4\.2\.2+'

export HELM_CONFIG_HOME="${test_temp}/helm-config"
export HELM_CACHE_HOME="${test_temp}/helm-cache"
export HELM_DATA_HOME="${test_temp}/helm-data"
export HELM_PLUGINS="${test_temp}/installed-plugins"
cp -R "${project_root}/testdata/installed-legacy/plugins" "${HELM_PLUGINS}"

"${helm_binary}" plugin list >"${test_temp}/plugin-list.out"
grep -Eq 'legacy-example.*legacy.*unknown' "${test_temp}/plugin-list.out"

set +e
"${helm_binary}" plugin verify "${HELM_PLUGINS}/legacy" \
  >"${test_temp}/directory-verify.out" 2>"${test_temp}/directory-verify.err"
directory_verify_exit=$?
set -e
test "${directory_verify_exit}" -eq 1
grep -q 'directory verification not supported' "${test_temp}/directory-verify.err"

mkdir -p "${test_temp}/unsigned-source/legacy-example"
cp "${HELM_PLUGINS}/legacy/plugin.yaml" \
  "${test_temp}/unsigned-source/legacy-example/plugin.yaml"
tar -C "${test_temp}/unsigned-source" -czf "${test_temp}/legacy-example.tgz" \
  legacy-example

export HELM_PLUGINS="${test_temp}/install-target"
mkdir -p "${HELM_PLUGINS}"
set +e
"${helm_binary}" plugin install "${test_temp}/legacy-example.tgz" \
  >"${test_temp}/verified-install.out" 2>"${test_temp}/verified-install.err"
verified_install_exit=$?
set -e
test "${verified_install_exit}" -eq 1
grep -q 'no provenance file (.prov) found' "${test_temp}/verified-install.err"

"${helm_binary}" plugin install "${test_temp}/legacy-example.tgz" --verify=false \
  >"${test_temp}/bypass-install.out" 2>"${test_temp}/bypass-install.err"
grep -q 'Installed plugin: legacy-example' "${test_temp}/bypass-install.out"

plugin_metadata="${project_root}/testdata/mixed-repository/plugins/legacy/plugin.yaml"
! grep -Eq '^apiVersion:' "${plugin_metadata}"
! grep -Eq '^type:' "${plugin_metadata}"
grep -Fq -- '--verify=false' \
  "${project_root}/testdata/mixed-repository/.github/workflows/deploy.yml"

go -C "${project_root}" build -mod=vendor -trimpath \
  -o "${test_temp}/helm4-plugin-preflight" ./cmd/helm4-plugin-preflight
set +e
"${test_temp}/helm4-plugin-preflight" check \
  --root "${project_root}/testdata/mixed-repository" \
  --helm-plugins "${project_root}/testdata/mixed-repository/plugins" \
  --format json >"${test_temp}/integrated-report.json"
preflight_exit=$?
set -e
test "${preflight_exit}" -eq 1
jq -e '
  [.diagnostics[].ruleId] == ["H4P001", "H4P003", "H4P004"] and
  ([.diagnostics[].path] | index(".github/workflows/deploy.yml") != null) and
  ([.diagnostics[].path] | index("plugins/legacy/plugin.yaml") != null)
' "${test_temp}/integrated-report.json" >/dev/null
