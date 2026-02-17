# promptctl — Ship it release

**For immediate release**

We’re building promptctl in public. Here’s what just shipped.

---

## What is promptctl?

promptctl is a CLI that structures every prompt for maximum signal and minimum waste. It works with Claude, GPT-5, Groq, DeepSeek and our own hosted model (codename **Atlas**). The goal: stop wasting money on bad prompts.

Savings are real but they’re not one number. Cheap models get spammed; expensive ones get more care. So we dropped the fake “67% everywhere” and switched to **tiered baselines**: ~55% for premium models (Opus, GPT-4), ~64% for mid-tier (Sonnet, GPT-4o), ~71% for cheap/fast (Haiku, Groq, DeepSeek). The landing page and the CLI both use the same math.

We also stopped talking in cents. “$0.03 saved per call” is noise. So we added **annual projections**: at 30 calls/day you’re in the **$500–1,200/year** range; at 100 calls/day (teams/CI), **$1,500–3,000**. That’s the number that actually changes behavior.

---

## What’s in this release

- **Landing page.** Shorter hero copy, cookie bar out of the way (bottom-right), clear primary CTA (Try vs Install). Terminal block and cost table are easier to scan. “Verified across 10 models” stands out. We shipped reviewer feedback instead of sitting on it.

- **Mac app (DMG).** There wasn’t one. Now the release pipeline builds a Mac app stub and a DMG on every tag and uploads it to [promptctl-releases](https://github.com/oleg-koval/promptctl-releases). “Download for Mac” points at a real file.

- **Atlas.** Our deployed LLM has a codename: **Atlas**. It’s in the models list; you can set it as default with `promptctl models --set`. It uses `PROMPTCTL_LLM_URL` (and optional `PROMPTCTL_API_KEY`). We don’t run a default completion endpoint yet—that’s on the roadmap.

- **Models list.** No more jargon-only table. We added a one-line explanation of what the default model is for, clearer column headers (In/Out $ per 1M tokens, max context), and a direct “Change default: promptctl models --set.”

- **Analytics (build in public).** We track Try button clicks (inline, CTA, nav), install/copy actions (brew, npm, etc.), and Google OAuth logins—so we know what’s used and what’s not. Events are documented in the repo.

- **Config.** You can remove or update API keys: `promptctl config --provider=anthropic --api-key=remove` or `--api-key=sk-ant-new...`. No more “where do I edit the file?”

- **Create (enhance) flow.** Rate the output 1–5; if you rate below 3 we offer **one free retry per day**. Ratings are stored locally and sent to the enhance worker for a future daily digest to hello@prompt-ctl.com. The hosted enhancer uses Workers AI (Llama 3.2 1B)—your API key is never used for `create`, only for `send` and `cost`.

- **Launch.cab.** We’re featured on [launch.cab](https://launch.cab/product/promptctl). Badge is on the hero.

- **Docs and ops.** WRANGLER_ENV_VARS.md lists all worker env/secret names. worker-alerts cron trigger fixed for current Wrangler. Scripts and loops for listing secrets tolerate missing workers.

---

## Why “ship it”

This is the batch we wanted out before calling this a release: honest savings math, annual numbers that matter, a Mac build that exists, and a landing page that doesn’t feel like a marketing cosplay. We’re not hiding the fact that Atlas has no default URL yet or that the daily ratings email is “documented, not wired.” Building in public means shipping the truth.

---

**Links**

- Site: [prompt-ctl.com](https://prompt-ctl.com)
- Repo: [github.com/oleg-koval/promptctl](https://github.com/oleg-koval/promptctl)
- Try: [prompt-ctl.com](https://prompt-ctl.com) → Try promptctl
- Featured: [launch.cab/product/promptctl](https://launch.cab/product/promptctl)

**Contact:** hello@prompt-ctl.com
