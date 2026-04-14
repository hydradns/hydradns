"use client"

import { useEffect, useState, useMemo } from "react"
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
import { getQueryLogs } from "@/lib/api"
import type { QueryLogEntry } from "@/lib/types"
import {
  Search, Download, ChevronLeft, ChevronRight, ShieldCheck, ShieldOff,
  Gauge, AlertTriangle, RefreshCw,
} from "lucide-react"

type ActionFilter = "all" | "block" | "allow" | "flagged"

const ITEMS_PER_PAGE = 50

function classifyAction(log: QueryLogEntry): "block" | "allow" | "flagged" {
  if (log.action === "block") return "block"
  if (log.is_suspicious) return "flagged"
  return "allow"
}

function domainColorClass(kind: "block" | "allow" | "flagged") {
  switch (kind) {
    case "block": return "text-hydra-red"
    case "flagged": return "text-hydra-amber"
    default: return "text-hydra-blue"
  }
}

function actionBadge(kind: "block" | "allow" | "flagged") {
  switch (kind) {
    case "block":
      return (
        <span className="inline-flex items-center px-2 py-0.5 rounded text-[10px] font-bold bg-hydra-red/10 text-hydra-red border border-hydra-red/20">
          BLOCKED
        </span>
      )
    case "flagged":
      return (
        <span className="inline-flex items-center px-2 py-0.5 rounded text-[10px] font-bold bg-hydra-amber/10 text-hydra-amber border border-hydra-amber/20">
          FLAGGED
        </span>
      )
    default:
      return (
        <span className="inline-flex items-center px-2 py-0.5 rounded text-[10px] font-bold bg-hydra-green/10 text-hydra-green border border-hydra-green/20">
          ALLOWED
        </span>
      )
  }
}

function threatScoreBar(score: number) {
  const pct = Math.max(1, Math.round(score * 100))
  let barColor = "bg-hydra-green"
  if (score >= 0.7) barColor = "bg-gradient-to-r from-hydra-amber to-hydra-red"
  else if (score >= 0.3) barColor = "bg-hydra-amber"

  return (
    <div className="w-24 h-1.5 bg-background rounded-full overflow-hidden">
      <div className={`h-full ${barColor}`} style={{ width: `${pct}%` }} />
    </div>
  )
}

function rowClasses(kind: "block" | "allow" | "flagged") {
  switch (kind) {
    case "block":
      return "border-b border-background hover:bg-white/5 transition-colors border-l-4 border-l-hydra-red bg-hydra-red/5"
    case "flagged":
      return "border-b border-background hover:bg-white/5 transition-colors border-l-4 border-l-hydra-amber bg-hydra-amber/5"
    default:
      return "border-b border-background hover:bg-white/5 transition-colors"
  }
}

export default function LogsPage() {
  const [logs, setLogs] = useState<QueryLogEntry[]>([])
  const [error, setError] = useState<string | null>(null)
  const [domainSearch, setDomainSearch] = useState("")
  const [actionFilter, setActionFilter] = useState<ActionFilter>("all")
  const [page, setPage] = useState(1)

  const fetchData = () => {
    getQueryLogs()
      .then((d) => { setLogs(d); setError(null) })
      .catch((e) => setError(e.message))
  }

  useEffect(() => {
    fetchData()
    const interval = setInterval(fetchData, 10000)
    return () => clearInterval(interval)
  }, [])

  const filtered = useMemo(() => {
    let result = logs
    if (domainSearch) {
      const q = domainSearch.toLowerCase()
      result = result.filter((l) => l.domain.toLowerCase().includes(q))
    }
    if (actionFilter !== "all") {
      result = result.filter((l) => classifyAction(l) === actionFilter)
    }
    return result
  }, [logs, domainSearch, actionFilter])

  const totalPages = Math.max(1, Math.ceil(filtered.length / ITEMS_PER_PAGE))
  const safePage = Math.min(page, totalPages)
  const paginated = filtered.slice(
    (safePage - 1) * ITEMS_PER_PAGE,
    safePage * ITEMS_PER_PAGE,
  )
  const startIdx = (safePage - 1) * ITEMS_PER_PAGE + 1
  const endIdx = Math.min(safePage * ITEMS_PER_PAGE, filtered.length)

  // Summary stats
  const blockedCount = logs.filter((l) => l.action === "block").length
  const flaggedCount = logs.filter((l) => l.is_suspicious).length
  const avgThreat = logs.length > 0
    ? logs.reduce((s, l) => s + l.threat_score, 0) / logs.length
    : 0
  const healthPct = logs.length > 0
    ? (((logs.length - blockedCount) / logs.length) * 100).toFixed(1)
    : "100.0"

  const filterPills: { label: string; value: ActionFilter }[] = [
    { label: "All Queries", value: "all" },
    { label: "Blocked", value: "block" },
    { label: "Allowed", value: "allow" },
    { label: "Flagged", value: "flagged" },
  ]

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
              placeholder="Filter domains..."
              className="pl-10 w-72 bg-card border-border focus-visible:ring-primary"
              value={domainSearch}
              onChange={(e) => { setDomainSearch(e.target.value); setPage(1) }}
            />
          </div>
          <Button variant="outline" size="sm" className="gap-2">
            <Download className="h-4 w-4" />
            Export
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
        <div className="bg-card/30 border border-border rounded-xl p-3 flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-2">
            {filterPills.map((pill) => (
              <button
                key={pill.value}
                onClick={() => { setActionFilter(pill.value); setPage(1) }}
                className={`px-4 py-1.5 rounded-full text-sm font-medium transition-all ${
                  actionFilter === pill.value
                    ? "bg-primary text-primary-foreground"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                {pill.label}
              </button>
            ))}
          </div>
          <button
            onClick={fetchData}
            className="p-1.5 text-muted-foreground hover:text-foreground transition-colors"
            title="Refresh"
          >
            <RefreshCw className="h-4 w-4" />
          </button>
        </div>

        {/* Data Table */}
        <div className="bg-card rounded-xl border border-border overflow-hidden">
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow className="border-b border-background bg-background/50 hover:bg-background/50">
                  <TableHead className="px-4 py-4 text-[11px] font-bold uppercase tracking-wider text-muted-foreground">Timestamp</TableHead>
                  <TableHead className="px-4 py-4 text-[11px] font-bold uppercase tracking-wider text-muted-foreground">Domain</TableHead>
                  <TableHead className="px-4 py-4 text-[11px] font-bold uppercase tracking-wider text-muted-foreground">Type</TableHead>
                  <TableHead className="px-4 py-4 text-[11px] font-bold uppercase tracking-wider text-muted-foreground">Client IP</TableHead>
                  <TableHead className="px-4 py-4 text-[11px] font-bold uppercase tracking-wider text-muted-foreground text-center">Action</TableHead>
                  <TableHead className="px-4 py-4 text-[11px] font-bold uppercase tracking-wider text-muted-foreground">Threat Score</TableHead>
                  <TableHead className="px-4 py-4 text-[11px] font-bold uppercase tracking-wider text-muted-foreground">Detection Method</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody className="text-sm">
                {paginated.length > 0 ? (
                  paginated.map((log) => {
                    const kind = classifyAction(log)
                    return (
                      <TableRow key={log.id} className={rowClasses(kind)}>
                        <TableCell className="px-4 py-3 text-muted-foreground text-xs">
                          {new Date(log.timestamp).toLocaleString()}
                        </TableCell>
                        <TableCell className={`px-4 py-3 font-mono ${domainColorClass(kind)}`}>
                          {log.domain}
                        </TableCell>
                        <TableCell className="px-4 py-3 text-muted-foreground text-xs font-bold">
                          A
                        </TableCell>
                        <TableCell className="px-4 py-3 font-mono text-muted-foreground text-xs">
                          {log.client_ip}
                        </TableCell>
                        <TableCell className="px-4 py-3 text-center">
                          {actionBadge(kind)}
                        </TableCell>
                        <TableCell className="px-4 py-3">
                          {threatScoreBar(log.threat_score)}
                        </TableCell>
                        <TableCell className="px-4 py-3 text-xs text-muted-foreground italic">
                          {log.detection_method || "\u2014"}
                        </TableCell>
                      </TableRow>
                    )
                  })
                ) : (
                  <TableRow>
                    <TableCell colSpan={7} className="text-center text-muted-foreground py-8">
                      {domainSearch || actionFilter !== "all" ? "No matching entries" : "No query logs yet"}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>

          {/* Pagination Footer */}
          <div className="px-6 py-4 flex items-center justify-between border-t border-background bg-background/30">
            <p className="text-xs text-muted-foreground">
              Showing <span className="text-foreground font-bold">{filtered.length > 0 ? startIdx : 0}-{endIdx}</span> of{" "}
              <span className="text-foreground font-bold">{filtered.length.toLocaleString()}</span> queries
            </p>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={safePage <= 1}
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                className="text-xs font-bold gap-1"
              >
                <ChevronLeft className="h-3 w-3" />
                Previous
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={safePage >= totalPages}
                onClick={() => setPage((p) => p + 1)}
                className="text-xs font-bold gap-1 border-primary/20 text-primary hover:bg-primary/10"
              >
                Next
                <ChevronRight className="h-3 w-3" />
              </Button>
            </div>
          </div>
        </div>

        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          <div className="bg-card p-4 rounded-xl border border-border">
            <div className="flex items-center gap-3 mb-2">
              <ShieldCheck className="h-5 w-5 text-primary" />
              <h3 className="text-xs font-bold text-muted-foreground uppercase tracking-wider">Health Index</h3>
            </div>
            <div className="text-2xl font-headline font-bold text-foreground">{healthPct}%</div>
            <div className="mt-2 h-1 w-full bg-background rounded-full overflow-hidden">
              <div className="h-full bg-primary" style={{ width: `${healthPct}%` }} />
            </div>
          </div>
          <div className="bg-card p-4 rounded-xl border border-border">
            <div className="flex items-center gap-3 mb-2">
              <ShieldOff className="h-5 w-5 text-hydra-red" />
              <h3 className="text-xs font-bold text-muted-foreground uppercase tracking-wider">Blocks (24h)</h3>
            </div>
            <div className="text-2xl font-headline font-bold text-foreground">{blockedCount.toLocaleString()}</div>
            <div className="text-[10px] text-hydra-red font-bold mt-1">
              {logs.length > 0
                ? `${((blockedCount / logs.length) * 100).toFixed(1)}% of total`
                : "No data"}
            </div>
          </div>
          <div className="bg-card p-4 rounded-xl border border-border">
            <div className="flex items-center gap-3 mb-2">
              <Gauge className="h-5 w-5 text-hydra-blue" />
              <h3 className="text-xs font-bold text-muted-foreground uppercase tracking-wider">Avg Threat</h3>
            </div>
            <div className="text-2xl font-headline font-bold text-foreground">{(avgThreat * 100).toFixed(1)}%</div>
            <div className="text-[10px] text-muted-foreground mt-1">Mean threat score</div>
          </div>
          <div className="bg-card p-4 rounded-xl border border-border">
            <div className="flex items-center gap-3 mb-2">
              <AlertTriangle className="h-5 w-5 text-hydra-amber" />
              <h3 className="text-xs font-bold text-muted-foreground uppercase tracking-wider">Active Threats</h3>
            </div>
            <div className="text-2xl font-headline font-bold text-foreground">{flaggedCount}</div>
            <div className="text-[10px] text-muted-foreground mt-1">
              {flaggedCount > 0 ? "Requires review" : "All clear"}
            </div>
          </div>
        </div>
      </div>
    </>
  )
}
