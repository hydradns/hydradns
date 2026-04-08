# Phase 2: Dashboard Implementation

**Date:** 2026-04-07
**Status:** Complete

## Context

Phase 1 delivered a working core with real API endpoints. Phase 2 connects the Next.js UI to those endpoints so users can manage HydraDNS from the browser.

## What Was Built

### API Client Layer
- `lib/api.ts` — typed fetch wrapper for all 11 API endpoints, uses `NEXT_PUBLIC_API_URL` env var
- `lib/types.ts` — 15 TypeScript interfaces matching exact API response shapes

### Dashboard Layout
- `app/dashboard/layout.tsx` — shared sidebar + content area, all sub-pages inherit it
- Updated sidebar nav routes from `#` to real paths (`/dashboard/dns`, `/dashboard/blocklists`, etc.)

### Pages (5 total)

| Page | Route | Features |
|---|---|---|
| **Dashboard Home** | `/dashboard` | 4 KPI cards (total/blocked/allowed/rate), engine status badge, auto-refresh every 5s |
| **DNS Engine** | `/dashboard/dns` | Engine status + enable/disable toggle, query metrics (total/errors/rate), latency (p50/p95/p99) with grade badge, upstream resolvers table |
| **Blocklists** | `/dashboard/blocklists` | Source list table with domain count, inline add form (ID/name/URL/format/category), delete with cascade, stats badges |
| **Policies** | `/dashboard/policies` | Policy table with action badges + domain chips, inline create form (domains via textarea), delete, active/inactive counts |
| **Query Logs** | `/dashboard/logs` | Full query log table, filter by domain/IP/action, action badges (allow/block/redirect), auto-refresh every 10s |

### Components Modified
- `section-cards.tsx` — replaced financial KPIs with DNS stats (total queries, blocked, allowed, block rate)
- `app-sidebar.tsx` — updated routes, removed unused icon imports

### Cleanup (12 files removed)
- 6 financial chart components (chart-01 through chart-06)
- chart-area-interactive, charts-extra (custom tooltip)
- action-buttons, date-picker (financial UI)
- data-table (complex DnD table — replaced with simple tables per page)
- data.json (mock business data)

## Architecture Decisions

| Decision | Why |
|---|---|
| Native fetch + useEffect, not SWR/React Query | Keep deps minimal, no new packages needed |
| Shared dashboard layout | All pages get sidebar without repeating it |
| Inline forms, not modals | No dialog component installed; sheet exists but inline is simpler |
| Auto-refresh with setInterval | Simple real-time feel without WebSockets |
| textarea for domain input | Easier than multi-input for bulk domains |

## Verification

- `npx next build` — compiles successfully, all 5 routes generate
- `npx eslint` — 0 errors, 0 warnings
- No dangling imports to deleted components

## What Was NOT Done (deferred)

- Settings page (no backend for it yet)
- Real-time WebSocket streaming for logs
- Edit/update for policies and blocklists (only create/delete)
- Pagination for logs table
- Dark/light theme toggle (hardcoded dark)
- User authentication

## Lessons Learned

1. **Extract layout early** — Moving SidebarProvider to a layout.tsx immediately made all sub-pages cleaner. Should have been the structure from the start.
2. **Type the API responses first** — Writing `lib/types.ts` before any page code caught mismatches between API field names and what I expected.
3. **Delete aggressively** — 12 files of financial mock data were noise. Removing them made the codebase honest about what it actually does.
4. **CORS must match** — The control plane CORS is hardcoded to `localhost:3000`. This will need env config before any non-local deployment.
