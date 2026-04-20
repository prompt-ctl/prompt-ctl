# Open Source Launch Plan

This plan operationalizes promptctl's open-source distribution strategy for the current stage (v1.0.x).

## Goals

- Increase developer adoption through OSS distribution channels
- Improve trust via transparent code and clear boundaries
- Build contributor funnel for integrations and tooling
- Prepare monetization path with open-core boundaries

## Open-Core Boundaries

### Open source (this repo)

- `promptctl` CLI
- Prompt engine and YAML template format
- Prompt scoring and testing (`score`, `fix`, `test`, `experiment`)
- Provider adapters and examples
- Core docs and contributor workflow

### Planned commercial services (separate repos/services)

- Prompt registry service
- Cloud prompt versioning and sync
- Team collaboration and access control
- Usage analytics dashboard

## One-Week Execution Checklist

1. Distribution setup
- Create or finalize org/repo namespace (`prompt-ctl`)
- Ensure install surfaces are current:
  - Homebrew tap formula
  - `go install` path
  - release binaries and checksums

2. Repo hygiene
- Verify no secrets in git history and working tree
- Remove stale experiments and dead scripts
- Confirm `README`, `LICENSE`, `CONTRIBUTING`, and `docs/` are discoverable

3. Product positioning
- Keep README focused on:
  - problem statement
  - quick start
  - comparison table
  - prompt testing workflow (`promptctl test`)
  - open-core roadmap

4. Launch distribution
- Publish launch post:
  - Hacker News ("Show HN")
  - Reddit (`r/LocalLLaMA`, `r/MachineLearning`)
  - X / LinkedIn
- Include:
  - install command
  - quick demo
  - clear OSS + roadmap statement

5. Contributor activation
- Add "good first issue" labels and starter issues
- Prioritize contributions in:
  - provider integrations
  - template/testing improvements
  - editor integrations

## Positioning Notes

- Open source is the distribution engine.
- Cloud services are the monetization layer.
- Success metric for OSS launch is adoption and contribution velocity, not stars alone.
