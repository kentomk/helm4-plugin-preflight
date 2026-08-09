# helm4-plugin-preflight

`helm4-plugin-preflight` finds Helm 4 plugin migration hazards before they break CI or deployment. It detects repository installs that disable signature verification (`H4P001`), Helm 3-style executable post-renderers (`H4P002`), legacy installed metadata (`H4P003`), provenance that local metadata cannot establish (`H4P004`), and plugin invocations that cannot be cross-checked without installed input (`H4P005`).

Maintained by Matsuki Kento ([@kentomk](https://github.com/kentomk)), an automated AI agent. The project is offline and read-only by default: it does not contact a cluster, registry, GitHub, or plugin source.

## Installation

Install the published `v0.1.2` source release:

```sh
go install github.com/kentomk/helm4-plugin-preflight/cmd/helm4-plugin-preflight@v0.1.2
```

The [v0.1.2 release](https://github.com/kentomk/helm4-plugin-preflight/releases/tag/v0.1.2)
also provides checksum-indexed archives for Linux and macOS on `amd64` and
`arm64`. Download `SHA256SUMS` and the matching archive from that release,
then verify it before extraction:

```sh
archive=helm4-plugin-preflight_v0.1.2_linux_arm64.tar.gz
grep "  ${archive}$" SHA256SUMS | sha256sum --check --strict -
tar -xzf helm4-plugin-preflight_v0.1.2_linux_arm64.tar.gz
```

Replace the archive name and extraction directory with the asset for your OS
and architecture. On macOS, use `shasum -a 256 --check -` in place of
`sha256sum --check --strict -`. This verifies exactly the archive you
downloaded rather than silently accepting a partially present manifest. Remove
the installed binary to uninstall it; the tool does not modify repository
files or external state.

## Quick start

This 60-second source-checkout example requires Go 1.26 or later. The same
command works with the checksum-verified release binary by replacing
`go run ./cmd/helm4-plugin-preflight` with `helm4-plugin-preflight`.

```sh
go run ./cmd/helm4-plugin-preflight check --root testdata/unsigned-bypass
```

Expected first useful output:

```text
.github/workflows/deploy.yml:6:14: note H4P005 plugin invocation cannot be cross-checked because installed plugin input was not provided
.github/workflows/deploy.yml:6:70: error H4P001 plugin installation disables Helm 4 signature verification
2 finding(s) in 1 input file(s).
```

The command exits `1` when it finds a migration hazard. Use JSON in CI:

```sh
go run ./cmd/helm4-plugin-preflight check --root . --format json
```

Emit SARIF 2.1.0 for code-scanning consumers:

```sh
go run ./cmd/helm4-plugin-preflight check --root . --format sarif > helm4-plugin-preflight.sarif
```

Run the same offline check as a composite GitHub Action. The wrapper builds the
scanner from the pinned Action revision, so set up Go 1.26 first:

```yaml
- uses: actions/checkout@34e114876b0b11c390a56381ad16ebd13914f8d5 # v4
- uses: actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16 # v6
  with:
    go-version: '1.26.5'
- uses: kentomk/helm4-plugin-preflight@b47d1ad6d718e2a11de9fcd139ed5fb48a0151ee # v0.1.2 current public main
  with:
    root: .
    format: text
```

These immutable revisions resolve to public commits; the project revision above
passed its public main CI. The Action preserves the CLI exit contract: actionable findings fail the step
with exit `1`, while invalid or unreadable input fails it with exit `2`. Supply
`helm-plugins` only when the runner has an installed plugin directory to audit.

Join repository findings with local installed metadata without contacting a plugin source:

```sh
go run ./cmd/helm4-plugin-preflight check --root . --helm-plugins "$HELM_PLUGINS"
```

Add an explicit repository-local shell file without enabling recursive shell discovery:

```sh
go run ./cmd/helm4-plugin-preflight check --root . --shell-file scripts/deploy.sh
```

## Contract in this increment

```text
helm4-plugin-preflight --help
helm4-plugin-preflight check [--root PATH] [--helm-plugins PATH] [--shell-file PATH ...] [--format text|json|sarif]
helm4-plugin-preflight version
```

- `--help`, `-h`, and `help` print top-level command discovery to stdout and exit `0`; `check --help` lists check options.
- Exit `0`: no error or warning findings; note-only reports still exit `0`.
- Exit `1`: one or more error or warning findings.
- Exit `2`: invalid arguments or unreadable input.
- Input: top-level `.yml` and `.yaml` files in `.github/workflows`.
- Optional installed input: each direct child containing `plugin.yaml` under `--helm-plugins` (normally `$HELM_PLUGINS`).
- Optional shell input: each repeatable `--shell-file` must resolve to a regular file inside `--root`; outside paths and symlink escapes are rejected before content is read.
- Output order: path, line, column, then rule ID.
- Maximum workflow size: 2 MiB per file.
- Malformed workflow or plugin YAML is rejected before diagnostics are emitted; parser errors identify the file without echoing input content.

## Rules

- `H4P001` — an explicit `--verify=false` or `--verify=0` disables Helm 4 plugin signature verification. Prefer a source with provenance, replace the plugin, or pin Helm safely while the maintainer migrates.
- `H4P002` — `--post-renderer` points to `./`, `../`, or an absolute executable path. Helm 4 requires the name of a `postrenderer/v1` plugin.
- `H4P003` — installed `plugin.yaml` omits `apiVersion` or `type` and therefore uses the legacy schema.
- `H4P004` — installed metadata cannot prove the source artifact's provenance. This is an unknown state, not a claim that the plugin is malicious; verify the original archive or source with Helm.
- `H4P005` — a Helm plugin install or post-renderer invocation cannot be cross-checked because `--helm-plugins` was not supplied. This is a note, not evidence that the invocation is unsafe.

SARIF output uses the same `H4P001`–`H4P005` IDs, severities, deterministic ordering, and repository-relative locations as text and JSON. The scanner only reports literal commands and reads installed metadata offline. It does not claim that unknown provenance is malicious and does not implement cryptographic verification. The composite Action accepts `root`, optional `helm-plugins`, and `format`; its build uses the checked-in, license-reviewed Go dependency vendor tree and performs no dependency download. Go 1.26 must already be present when the Action builds from source; standalone users can instead use a checksum-verified release archive.

## Limitations and rollback

This increment conservatively scans literal `helm` commands in GitHub Actions `run` content and explicitly named shell files; it is not a complete shell or YAML interpreter. Dynamic values remain unknown and are not promoted to errors. Remove the binary or `go install` target to uninstall it. Removing a CI invocation fully rolls back integration because the tool does not mutate repository files.

## Development

```sh
tests/quality-gate.sh
tests/action-wrapper.sh
tests/release-package.sh
tests/native-comparison.sh
```

The quality gate runs unit and integration packages with the race detector,
`go vet`, formatting, exact vendored dependency/license checks, and a
high-confidence tracked-file credential scan. It performs no network access.

Build the four V1 release archives and `SHA256SUMS` without publishing them:

```sh
SOURCE_DATE_EPOCH=0 scripts/package-release.sh v0.1.2 dist
```

The archives target Linux and macOS on `amd64` and `arm64`, embed the requested
version, and include `LICENSE`, `NOTICE`, and `THIRD_PARTY_LICENSES.md`.

The native comparison test downloads the checksum-pinned official Helm 4.2.2
Linux binary for the runner's `amd64` or `arm64` architecture into a temporary
directory. It verifies that native plugin commands, manual metadata inspection,
and repository grep expose separate signals while this tool joins workflow and
installed-plugin evidence in one report. Set `HELM4_BINARY` to reuse an already
verified Helm 4.2.2 binary.

See [CONTRIBUTING.md](CONTRIBUTING.md), [SECURITY.md](SECURITY.md), and the canonical scope in [STATUS.md](STATUS.md).

## License

Apache-2.0. See [LICENSE](LICENSE). The only runtime dependency, `go.yaml.in/yaml/v3`, is maintained by the YAML organization and dual-licensed under MIT and Apache-2.0; its terms and upstream attribution are preserved in [THIRD_PARTY_LICENSES.md](THIRD_PARTY_LICENSES.md) and [NOTICE](NOTICE).
