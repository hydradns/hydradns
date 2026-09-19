"use client"

import { useEffect, useMemo, useState } from "react"
import {
  Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList,
  BreadcrumbPage, BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import { Separator } from "@/components/ui/separator"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { allowDomain, blockDomain, getQueryLogs } from "@/lib/api"
import type { QueryLogEntry, QueryLogPage } from "@/lib/types"
import {
  Search, ChevronLeft, ChevronRight, RefreshCw, ShieldCheck, Ban,
  Check, Loader2,
} from "lucide-react"

const PAGE_SIZE = 50
const DEBOUNCE_MS = 300

type RowAction = "allow" | "block"

type RowResult = { type: "ok" | "err"; text: string }

function kindOf(log: QueryLogEntry): "block" | "flagged" | "allow" {
  if (log.action?.toLowerCase() === "block") return "block"
  if (log.is_suspicious) return "flagged"
  return "allow"
}

function rowClasses(log: QueryLogEntry) {
  const base = "border-b border-background hover:bg-white/5 transition-colors"
  switch (kindOf(log)) {
    case "block":
      return `${base} border-l-4 border-l-hydra-red bg-hydra-red/5`
    case "flagged":
      // suspicious rows highlighted
      return `${base} border-l-4 border-l-hydra-amber bg-hydra-amber/5`
    default:
      return base
  }
}

function actionBadge(log: QueryLogEntry) {
  const kind = kindOf(log)
  const map = {
    block: { label: "BLOCKED", cls: "bg-hydra-red/10 text-hydra-red border-hydra-red/20" },
    flagged: { label: "FLAGGED", cls: "bg-hydra-amber/10 text-hydra-amber border-hydra-amber/20" },
    allow: { label: "ALLOWED", cls: "bg-hydra-green/10 text-hydra-green border-hydra-green/20" },
  } as const
  const { label, cls } = map[kind]
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-[10px] font-bold border ${cls}`}>
      {label}
    </span>
  )
}

export default function LogsPage() {
  const [data, setData] = useState<QueryLogPage | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Filters
  const [domain, setDomain] = useState("")
  const [client, setClient] = useState("")
  const [action, setAction] = useState("all")
  const [suspiciousOnly, setSuspiciousOnly] = useState(false)
  const [start, setStart] = useState("")
  const [end, setEnd] = useState("")
  const [page, setPage] = useState(1)

  // Refresh nonce so the manual refresh button re-triggers the fetch effect.
  const [nonce, setNonce] = useState(0)

  // Per-row one-click action feedback.
  const [pendingKey, setPendingKey] = useState<string | null>(null)
  const [rowResults, setRowResults] = useState<Record<number, RowResult>>({})

  // Reset to the first page whenever a filter (not the page itself) changes.
  const onFilterChange =
    <T,>(setter: (v: T) => void) =>
    (v: T) => {
      setter(v)
      setPage(1)
    }

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    const handle = setTimeout(() => {
      getQueryLogs({
        domain: domain || undefined,
        client: client || undefined,
        action,
        suspicious: suspiciousOnly || undefined,
        start: start ? new Date(start).toISOString() : undefined,
        end: end ? new Date(end).toISOString() : undefined,
        page,
        page_size: PAGE_SIZE,
      })
        .then((d) => {
          if (cancelled) return
          setData(d)
          setError(null)
        })
        .catch((e) => {
          if (cancelled) return
          setError(e instanceof Error ? e.message : "Failed to load query logs")
        })
        .finally(() => {
          if (!cancelled) setLoading(false)
        })
    }, DEBOUNCE_MS)
    return () => {
      cancelled = true
      clearTimeout(handle)
    }
  }, [domain, client, action, suspiciousOnly, start, end, page, nonce])

  const items = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const startIdx = total === 0 ? 0 : (page - 1) * PAGE_SIZE + 1
  const endIdx = Math.min(page * PAGE_SIZE, total)

  const hasFilters = useMemo(
    () => Boolean(domain || client || suspiciousOnly || start || end || action !== "all"),
    [domain, client, suspiciousOnly, start, end, action],
  )

  const handleRowAction = async (log: QueryLogEntry, act: RowAction) => {
    const key = `${log.id}:${act}`
    setPendingKey(key)
    setRowResults((prev) => {
      const next = { ...prev }
      delete next[log.id]
      return next
    })
    try {
      if (act === "allow") await allowDomain(log.domain)
      else await blockDomain(log.domain)
      setRowResults((prev) => ({
        ...prev,
        [log.id]: { type: "ok", text: act === "allow" ? "Allowed" : "Blocked" },
      }))
    } catch (e) {
      setRowResults((prev) => ({
        ...prev,
        [log.id]: { type: "err", text: e instanceof Error ? e.message : "Action failed" },
      }))
    } finally {
      setPendingKey(null)
    }
  }

  const actionPills: { label: string; value: string }[] = [
    { label: "All", value: "all" },
    { label: "Blocked", value: "block" },
    { label: "Allowed", value: "allow" },
  ]

  const filterInput =
    "bg-card border border-border rounded-lg px-3 py-2 text-sm text-foreground focus:outline-none focus:border-primary transition-colors"

  return (
    <>
      {/* Header */}
      <header className="flex flex-wrap gap-3 min-h-20 py-4 shrink-0 items-center border-b border-border">
        <div className="flex flex-1 items-center gap-2">
          <SidebarTrigger className="-ms-1" />
          <div className="max-lg:hidden lg:contents">
            <Separator orientation="vertical" className="me-2 data-[orientation=vertical]:h-4" />
            <Breadcrumb>
              <BreadcrumbList>
                <BreadcrumbItem className="hidden md:block">
                  <BreadcrumbLink href="/dashboard">Dashboard</BreadcrumbLink>
                </BreadcrumbItem>
                <BreadcrumbSeparator className="hidden md:block" />
                <BreadcrumbItem>
                  <BreadcrumbPage>Query Logs</BreadcrumbPage>
                </BreadcrumbItem>
              </BreadcrumbList>
            </Breadcrumb>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              aria-label="Filter by domain"
              placeholder="Filter domains..."
              className="pl-10 w-72 bg-card border-border focus-visible:ring-primary"
              value={domain}
              onChange={(e) => onFilterChange(setDomain)(e.target.value)}
            />
          </div>
          <Button
            variant="outline"
            size="sm"
            className="gap-2"
            onClick={() => setNonce((n) => n + 1)}
            aria-label="Refresh"
          >
            <RefreshCw className="h-4 w-4" />
            Refresh
          </Button>
        </div>
      </header>

      <div className="flex flex-1 flex-col gap-6 py-6">
        {error && (
          <div className="rounded-md border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
            {error}
          </div>
        )}

        {/* Filter Bar */}
        <div className="bg-card/30 border border-border rounded-xl p-3 flex flex-wrap items-center gap-4">
          <div className="flex items-center gap-2" role="group" aria-label="Action filter">
            {actionPills.map((pill) => (
              <button
                key={pill.value}
                onClick={() => onFilterChange(setAction)(pill.value)}
                aria-pressed={action === pill.value}
                className={`px-4 py-1.5 rounded-full text-sm font-medium transition-all ${
                  action === pill.value
                    ? "bg-primary text-primary-foreground"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                {pill.label}
              </button>
            ))}
          </div>

          <Input
            aria-label="Filter by client IP"
            placeholder="Client IP..."
            className={`${filterInput} w-40`}
            value={client}
            onChange={(e) => onFilterChange(setClient)(e.target.value)}
          />

          <label className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <span className="uppercase tracking-wider font-bold">From</span>
            <input
              type="datetime-local"
              aria-label="Start time"
              className={filterInput}
              value={start}
              onChange={(e) => onFilterChange(setStart)(e.target.value)}
            />
          </label>
          <label className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <span className="uppercase tracking-wider font-bold">To</span>
            <input
              type="datetime-local"
              aria-label="End time"
              className={filterInput}
              value={end}
              onChange={(e) => onFilterChange(setEnd)(e.target.value)}
            />
          </label>

          <label className="flex items-center gap-2 text-sm text-muted-foreground cursor-pointer select-none ml-auto">
            <input
              type="checkbox"
              className="accent-hydra-amber h-4 w-4"
              checked={suspiciousOnly}
              onChange={(e) => onFilterChange(setSuspiciousOnly)(e.target.checked)}
            />
            Suspicious only
          </label>
        </div>

        {/* Data Table */}
        <div className="bg-card rounded-xl border border-border overflow-hidden">
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow className="border-b border-background bg-background/50 hover:bg-background/50">
                  {["Timestamp", "Domain", "Client IP", "Action", "Reason"].map((h) => (
                    <TableHead
                      key={h}
                      className="px-4 py-4 text-[11px] font-bold uppercase tracking-wider text-muted-foreground"
                    >
                      {h}
                    </TableHead>
                  ))}
                  <TableHead className="px-4 py-4 text-[11px] font-bold uppercase tracking-wider text-muted-foreground text-right">
                    Quick Action
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody className="text-sm">
                {items.length > 0 ? (
                  items.map((log) => {
                    const result = rowResults[log.id]
                    const allowPending = pendingKey === `${log.id}:allow`
                    const blockPending = pendingKey === `${log.id}:block`
                    const busy = allowPending || blockPending
                    return (
                      <TableRow key={log.id} className={rowClasses(log)}>
                        <TableCell className="px-4 py-3 text-muted-foreground text-xs">
                          {new Date(log.timestamp).toLocaleString()}
                        </TableCell>
                        <TableCell className="px-4 py-3 font-mono text-foreground">
                          {log.domain}
                        </TableCell>
                        <TableCell className="px-4 py-3 font-mono text-muted-foreground text-xs">
                          {log.client_ip}
                        </TableCell>
                        <TableCell className="px-4 py-3">{actionBadge(log)}</TableCell>
                        <TableCell className="px-4 py-3 text-xs text-muted-foreground">
                          {log.threat_reason || log.detection_method || "—"}
                        </TableCell>
                        <TableCell className="px-4 py-3">
                          <div className="flex items-center justify-end gap-2">
                            {result && (
                              <span
                                className={`text-[11px] font-bold ${
                                  result.type === "ok" ? "text-hydra-green" : "text-hydra-red"
                                }`}
                              >
                                {result.text}
                              </span>
                            )}
                            <Button
                              variant="outline"
                              size="sm"
                              disabled={busy}
                              onClick={() => handleRowAction(log, "allow")}
                              aria-label={`Allow ${log.domain}`}
                              className="h-7 gap-1 text-xs text-hydra-green hover:text-hydra-green"
                            >
                              {allowPending ? (
                                <Loader2 className="h-3 w-3 animate-spin" />
                              ) : (
                                <Check className="h-3 w-3" />
                              )}
                              Allow
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              disabled={busy}
                              onClick={() => handleRowAction(log, "block")}
                              aria-label={`Block ${log.domain}`}
                              className="h-7 gap-1 text-xs text-hydra-red hover:text-hydra-red"
                            >
                              {blockPending ? (
                                <Loader2 className="h-3 w-3 animate-spin" />
                              ) : (
                                <Ban className="h-3 w-3" />
                              )}
                              Block
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    )
                  })
                ) : (
                  <TableRow>
                    <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                      {loading
                        ? "Loading query logs..."
                        : hasFilters
                          ? "No matching entries"
                          : "No query logs yet"}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>

          {/* Pagination Footer */}
          <div className="px-6 py-4 flex items-center justify-between border-t border-background bg-background/30">
            <p className="text-xs text-muted-foreground">
              Showing{" "}
              <span className="text-foreground font-bold">
                {startIdx}-{endIdx}
              </span>{" "}
              of <span className="text-foreground font-bold">{total.toLocaleString()}</span> queries
            </p>
            <div className="flex items-center gap-2">
              <span className="text-xs text-muted-foreground">
                Page {page} / {totalPages}
              </span>
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1 || loading}
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                className="text-xs font-bold gap-1"
              >
                <ChevronLeft className="h-3 w-3" />
                Previous
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={page >= totalPages || loading}
                onClick={() => setPage((p) => p + 1)}
                className="text-xs font-bold gap-1 border-primary/20 text-primary hover:bg-primary/10"
              >
                Next
                <ChevronRight className="h-3 w-3" />
              </Button>
            </div>
          </div>
        </div>

        <p className="flex items-center gap-2 text-[11px] text-muted-foreground">
          <ShieldCheck className="h-3.5 w-3.5 text-primary" />
          One-click actions create a high-priority policy scoped to the single domain.
        </p>
      </div>
    </>
  )
}
