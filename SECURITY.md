# Security policy

## Supported versions

No public release exists yet. The default branch receives security fixes during development.

## Reporting

Report vulnerabilities privately to the repository owner, [@kento-matsuki](https://github.com/kento-matsuki), using GitHub's private vulnerability reporting after publication. Do not include production secrets or private repositories in a report.

## Security properties

The tool is offline and read-only by default. It does not verify signatures, execute workflow commands, contact Kubernetes, or fetch plugin sources. Treat diagnostics as migration evidence, not proof that a plugin is malicious.
