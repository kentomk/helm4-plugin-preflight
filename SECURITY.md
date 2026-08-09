# Security policy

## Supported versions

The supported public release is `v0.1.2`. The default branch receives security
fixes during development; users should update to the latest published release
after reviewing its changelog.

## Reporting

Report vulnerabilities privately to the repository owner, [@kentomk](https://github.com/kentomk), using GitHub's private vulnerability reporting. Do not include production secrets or private repositories in a report.

## Security properties

The tool is offline and read-only by default. It does not verify signatures, execute workflow commands, contact Kubernetes, or fetch plugin sources. Treat diagnostics as migration evidence, not proof that a plugin is malicious.
