# Analytics: where to read what

Single guide for founder: user activity and satisfaction across GA4, Cloudflare Analytics Engine, and D1.

## Overview

| What | Where | How to read |
|------|--------|--------------|
| **Behavioral events (site + CLI)** | GA4 | [Google Analytics](https://analytics.google.com/) → property G-DQBN89S2FZ |
| **Request metrics + ratings** | Cloudflare Analytics Engine | SQL API or dashboard; dataset `promptctl_enhance` |
| **Feedback (free text)** | D1 | `wrangler d1 execute` or Cloudflare dashboard → D1 |

---

## GA4

- **Link:** [Google Analytics](https://analytics.google.com/) → select property **G-DQBN89S2FZ**.
- **Site:** Events are sent when the user has accepted cookies. Full event list and funnel: **promptctl-site** repo → `docs/ANALYTICS_EVENTS.md`.
- **CLI:** Events are sent only when `PROMPTCTL_GA4_SECRET` is set (Measurement Protocol). Events: `onboarding_started`, `onboarding_completed`, `onboarding_skipped`, `model_selected`, `prompt_created`, `prompt_saved`, `prompt_rated` (params: `rating`, optional `intent_length`).
- **How to read:** Reports → Engagement → Events; or Explorations to build funnels (e.g. onboarding → prompt_created → prompt_rated).

---

## Analytics Engine

- **Link:** [Analytics Engine SQL API](https://developers.cloudflare.com/analytics/analytics-engine/sql-api). Optional: [Grafana](https://developers.cloudflare.com/analytics/analytics-engine/grafana/) for dashboards.
- **Dataset:** `promptctl_enhance`. Schema: blobs (e.g. status, path; or `"rating"` and rating value), doubles (intent length, count). No PII.
- **Example queries** (run via SQL API or dashboard; adapt to your account/dataset name and column names per Cloudflare docs):
  - **Requests last 24h by status:** filter by time range and group by first blob (ok vs error).
  - **Error rate last 7 days:** count where blob0 = 'error' / total count.
  - **Ratings by day:** filter blob0 = 'rating', group by date, sum count.
  - **Ratings distribution (1–5):** filter blob0 = 'rating', group by blob1 (rating value).
- **How to run:** Use the SQL API with an API token (see Cloudflare docs), or open the Analytics Engine in the Cloudflare dashboard and run SQL there.

---

## D1 (feedback)

- **Database:** `promptctl-analytics`. **Table:** `feedback` (columns: `id`, `received_at`, `text`, `source`).
- **How to read:**
  - **CLI:** From promptctl repo root (with wrangler and D1 set up):
    ```bash
    cd worker && wrangler d1 execute promptctl-analytics --remote --command "SELECT id, received_at, source, substr(text, 1, 80) AS preview FROM feedback ORDER BY received_at DESC LIMIT 10"
    ```
  - **Dashboard:** Cloudflare dashboard → D1 → select `promptctl-analytics` → Execute SQL; run the same query or browse the table.

---

## Future dashboard

A future internal dashboard can read from **D1** (feedback), **Analytics Engine SQL** (ratings and request metrics), and the **GA4 API** (behavioral events) to show activity and satisfaction in one place.
