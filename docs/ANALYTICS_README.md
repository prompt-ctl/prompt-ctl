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
- **Dataset:** `promptctl_enhance`. Table name in SQL: `promptctl_enhance`. Columns: `timestamp`, `_sample_interval`, `index1`, `blob1`…`blob20`, `double1`…`double20`. No PII.

**Quick reference — writing events**

1. **Binding** (in `worker/wrangler.toml`): `[[analytics_engine_datasets]]` with `binding = "ENHANCE_ANALYTICS"`, `dataset = "promptctl_enhance"`.
2. **In the Worker:** call `env.ENHANCE_ANALYTICS.writeDataPoint({ blobs, doubles, indexes })`. Keep field order the same for every event. You don’t need to await it; it runs in the background.
3. **Example:** `env.ENHANCE_ANALYTICS.writeDataPoint({ blobs: ['ok', '/enhance'], doubles: [intentLength, 1], indexes: [''] })`. Use `indexes: [crypto.randomUUID()]` if you want sampling per event; we use `['']` for aggregate stats.

**Schema (what the enhance worker writes):**

| Event type   | blob1        | blob2     | double1      | double2 |
|-------------|--------------|-----------|--------------|--------|
| Request     | status       | path      | intent length| 1      |
| Rating      | `"rating"`   | 1–5       | intent length| 1      |
| Feedback err| `feedback_error` | `d1_insert` | 0       | 1      |

**How to run queries:**

1. **Create an API token:** [Cloudflare API Tokens](https://dash.cloudflare.com/profile/api-tokens) → Create Custom Token → Permissions: **Account \| Account Analytics \| Read**. Copy the token.
2. **Call the SQL API** (replace `ACCOUNT_ID` and `API_TOKEN`):

```bash
curl "https://api.cloudflare.com/client/v4/accounts/ACCOUNT_ID/analytics_engine/sql" \
  --header "Authorization: Bearer API_TOKEN" \
  --data "SELECT * FROM promptctl_enhance WHERE timestamp > NOW() - INTERVAL '1' DAY LIMIT 10"
```

3. **Example queries** (use `SUM(_sample_interval)` for counts when sampling may apply):

```sql
-- Requests last 24h by status (ok vs error)
SELECT blob1 AS status, blob2 AS path, SUM(_sample_interval) AS n
FROM promptctl_enhance
WHERE timestamp > NOW() - INTERVAL '1' DAY AND blob1 IN ('ok', 'error')
GROUP BY blob1, blob2;

-- Error rate last 7 days
SELECT
  SUM(CASE WHEN blob1 = 'error' THEN _sample_interval ELSE 0 END) * 1.0 / NULLIF(SUM(_sample_interval), 0) AS error_rate,
  SUM(_sample_interval) AS total
FROM promptctl_enhance
WHERE timestamp > NOW() - INTERVAL '7' DAY AND blob1 IN ('ok', 'error');

-- Ratings distribution (1–5)
SELECT blob2 AS rating, SUM(_sample_interval) AS n
FROM promptctl_enhance
WHERE blob1 = 'rating' AND timestamp > NOW() - INTERVAL '30' DAY
GROUP BY blob2
ORDER BY blob2;
```

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
