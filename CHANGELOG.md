# Changelog

All notable changes will be documented here.

## Unreleased

- Align the copy-ready composite Action example with the current successful public main revision and reject the superseded pin.
- Pin the copy-ready composite Action example to the current successful public main and reject the superseded revision in the publisher contract.
- Make manual and broker-triggered release repairs check out the requested tag before building assets.
- Align the security policy with the published `v0.1.1` release and verify the
  public-release marker in the publisher contract.
- Pin the copy-ready Action example to the exact reviewed Go `1.26.5` patch
  used by CI and the release publisher, and reject the drifting `1.26.x` form.
- Align the CI and release workflows with the same exact Go patch so manual and
  repository-dispatch release repair cannot resolve a different toolchain.
- Add an owner-repairable release workflow that uploads the four reproducible archives and `SHA256SUMS`.
### Fixed

- Align the copy-ready composite Action pin with the current successful public
  main revision, while retaining the published v0.1.1 release contract.
- Keep the README install, archive, packaging, and immutable Action examples
  aligned with the latest published release, enforced by the publisher gate.
- Make top-level `--help`, `-h`, and `help` discover the CLI contract with exit `0`.
- Replace the unresolved `@v0` Action example and mutable setup references with publicly verified immutable commit SHAs.

### Added

- Self-contained publication gates for clean-archive quickstart, immutable CI
  dependencies, release payload limits, identity, demand, alternatives,
  distribution, and adoption-observability contracts.
- Offline race, vet, formatting, exact dependency/license integrity, and
  high-confidence tracked credential checks in one CI quality command.
- Offline composite GitHub Action wrapper with explicit root, installed plugin,
  and output-format inputs while preserving CLI exit codes.
- Local Action integration coverage for clean, actionable, and invalid runs.
- Reproducible, checksum-indexed release archives for Linux and macOS on
  `amd64` and `arm64`, including license and attribution files.
- Automated Helm 4.2.2 comparison covering native plugin inventory and
  verification, manual metadata, repository grep, and the integrated report.
- Offline GitHub Actions workflow discovery.
- `H4P001` detection for explicit plugin verification bypasses.
- `H4P002` detection for Helm 3-style executable post-renderer paths.
- Offline installed plugin metadata checks for legacy schema (`H4P003`) and unknown provenance (`H4P004`).
- Missing installed-input notes (`H4P005`) that do not fail CI.
- Repeatable, repository-root-confined `--shell-file` scanning with outside-path and symlink-escape rejection.
- Deterministic text and versioned JSON output with fixture-backed golden tests.
- SARIF 2.1.0 output with stable rule metadata, severity mapping, and repository-relative locations.
- Bounded YAML syntax validation for workflows and installed plugin metadata, with content-safe exit `2` failures.
