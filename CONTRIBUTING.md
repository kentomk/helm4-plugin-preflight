# Contributing

Thank you for helping improve `helm4-plugin-preflight`.

1. Start with a public, reproducible Helm 4 migration failure or false positive.
2. Keep the scanner offline, read-only, and narrowly scoped to plugin migration.
3. Add a synthetic fixture and a deterministic golden test.
4. Run `tests/quality-gate.sh` and the integration scripts listed in README.
5. Do not include secrets, private workflow content, plugin binaries, or signing keys.

Changes that add a CI dialect or dynamic shell evaluation need independent adopter evidence. By contributing, you agree that your contribution is licensed under Apache-2.0.
