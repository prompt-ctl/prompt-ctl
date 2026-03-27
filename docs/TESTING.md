# Testing

## Automated tests

- **Go:** `go test ./...` (config, prompt enhance/enhanceclient, cmd).
- **Worker:** From `worker/`, run `npm test` or `node --test src/index.test.js` (contract and validation tests with mocked `env.AI`).

## Manual / E2E checklist

Use this checklist for smoke and integration checks (not automated).

| Scenario | How to verify |
| -------- | ------------- |
| **Default (rule-based)** | Run `promptctl create "analyze my startup idea"` with no env. Expect a structured prompt generated offline. |
| **Rule-based explicit** | Run `PROMPTCTL_ENHANCE=rule promptctl create "review my code"`. Expect same structure, no network. |
| **Custom backend** | Set `PROMPTCTL_ENHANCE_URL` to a custom endpoint. Run `promptctl create "test"`. Expect CLI to use that URL. |
| **Brew** | After a release, run `brew install promptctl` and run one `promptctl create "smoke"` (manual smoke). |
