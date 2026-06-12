# Security Policy

## Supported Versions

Only the latest release receives security fixes.

## Reporting a Vulnerability

Please report vulnerabilities privately via
[GitHub's private vulnerability reporting](https://github.com/jc00ke/harvest/security/advisories/new)
rather than opening a public issue.

You should receive a response within a week. Please include steps to
reproduce and the version affected.

## Verifying Releases

Every release artifact carries a [build provenance attestation](https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations/using-artifact-attestations-to-establish-provenance-for-builds)
created by the release workflow. Verify a downloaded archive with the
[GitHub CLI](https://cli.github.com/):

```bash
gh attestation verify harvest_linux_amd64.tar.gz -R jc00ke/harvest
```

Or verify an entire release at once:

```bash
mise run release:verify v0.1.0
```

## Credential Storage

`harvest` stores your Harvest Account ID and Access Token in the operating
system keyring (macOS Keychain, Linux Secret Service, Windows Credential
Manager) — never in plain-text config files.
