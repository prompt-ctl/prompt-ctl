# Testing

## Automated tests

- **Go:** `go test ./...` (config, prompt enhance/enhanceclient, cmd).
- **Worker:** From `worker/`, run `npm test` or `node --test src/index.test.js` (contract and validation tests with mocked `env.AI`).

## Manual / E2E checklist

Use this checklist for smoke and integration checks (not automated).

| Scenario | How to verify |
| -------- | ------------- |
| **Default (LLM)** | Run `promptctl create "analyze my startup idea"` with no env. Expect a structured prompt from the hosted Worker (or rule-based fallback if Worker is down). |
| **Rule-based** | Run `PROMPTCTL_ENHANCE=rule promptctl create "review my code"`. Expect same structure, no network. |
| **Fallback** | Set `PROMPTCTL_ENHANCE_URL` to an invalid URL or a server returning 500. Run `promptctl create "test"`. Expect CLI to fall back to rule-based and print a prompt. |
| **Worker deploy** | Deploy with `wrangler deploy` from `worker/`. Then `curl -X POST https://<your-worker>/enhance -H "Content-Type: application/json" -d '{"intent":"test"}'`. Expect 200 and a `prompt` in the JSON body. |
| **Brew** | After a release, run `brew install oleg-koval/tap/promptctl` and run one `promptctl create "smoke"` (manual smoke). |
