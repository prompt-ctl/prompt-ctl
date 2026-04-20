# Security Policy

## Supported Versions

Security fixes are provided for the latest minor release line.

## Reporting a Vulnerability

Please do not open public issues for security vulnerabilities.

- Preferred: GitHub Security Advisory for `prompt-ctl/promptctl`
- Fallback: open a private report to the maintainer and include:
  - impact summary
  - reproduction steps
  - affected version(s)
  - suggested mitigation (if known)

## Secrets and Telemetry

- `promptctl` CLI is designed to work locally and offline.
- Optional cloud feedback/rating events are opt-in only via:
  - `PROMPTCTL_CLOUD_ENABLED=1`
  - `PROMPTCTL_CLOUD_URL=https://...`
- Without opt-in, feedback/rating data stays local under `~/.promptctl/`.

## Hardening and Scanning

- CI includes secret scanning for repository changes.
- Do not commit API keys, private keys, `.env` files, or credentials.
