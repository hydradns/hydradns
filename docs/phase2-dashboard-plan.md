# Phase 2: Dashboard Implementation Plan

**Date:** 2026-04-07
**Status:** In Progress

## Goal

Non-technical user can see and control HydraDNS entirely from the browser.

## Architecture Decisions

1. **Client components with native fetch** — No SWR/React Query to keep deps minimal. useEffect + useState for data fetching with manual refresh.
2. **Dashboard layout** — Extract sidebar into `dashboard/layout.tsx` so all sub-pages share it.
3. **Sheet component for forms** — No dialog installed; use Sheet (slide-over panel) for create forms.
4. **Existing patterns** — Reuse Card, Table, Badge, Button, Input, Select from shadcn/ui. Keep @remixicon icons.
5. **API client** — Single `lib/api.ts` with typed functions per endpoint. Base URL from `NEXT_PUBLIC_API_URL`.

## Files to Create/Modify

### New files
- `lib/api.ts` — API client with typed fetch functions
- `lib/types.ts` — TypeScript interfaces matching API response shapes
- `app/dashboard/layout.tsx` — Shared sidebar + header layout
- `app/dashboard/dns/page.tsx` — DNS engine status, toggle, metrics, resolvers
- `app/dashboard/blocklists/page.tsx` — Blocklist source list + add + delete
- `app/dashboard/policies/page.tsx` — Policy list + add + delete
- `app/dashboard/logs/page.tsx` — Query log table

### Modified files
- `app/dashboard/page.tsx` — Rewrite: DNS stats overview (total/blocked/allowed/rate)
- `components/app-sidebar.tsx` — Fix nav URLs from `#` to real routes
- `components/section-cards.tsx` — Replace financial KPIs with DNS stats

### Deleted files
- Chart components (chart-01 through chart-06, chart-area-interactive, charts-extra) — financial mock data
- `components/action-buttons.tsx` — date picker for financial data
- `components/date-picker.tsx` — not needed for DNS dashboard
- `components/data-table.tsx` — overly complex DnD table, will write simpler ones
- `app/dashboard/data.json` — mock business data

## API Endpoints Used

| Page | Endpoints |
|---|---|
| Dashboard home | GET /dashboard/summary |
| DNS Engine | GET /dns/engine, POST /dns/engine, GET /dns/metrics, GET /dns/resolvers |
| Blocklists | GET /blocklists, POST /blocklists, DELETE /blocklists/:id |
| Policies | GET /policies, POST /policies, DELETE /policies/:id |
| Logs | GET /analytics/audits |

## Execution Order

1. Create API client + types
2. Create dashboard layout (extract sidebar)
3. Update sidebar nav routes
4. Rewrite dashboard home page with real stats
5. Create DNS engine page
6. Create blocklists page
7. Create policies page
8. Create logs page
9. Clean up unused components
10. Build and test
