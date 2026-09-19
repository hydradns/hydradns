"use client"

import { Fragment, useCallback, useEffect, useState } from "react"
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
import { getAuditEvents } from "@/lib/api"
import type { AuditEvent, AuditListData } from "@/lib/types"
import {
  Search, RefreshCw, ChevronLeft, ChevronRight, ChevronDown, History,
} from "lucide-react"

const PAGE_SIZE = 25

function actionBadge(action: string) {
  const a = action.toLowerCase()
  let cls = "bg-muted text-muted-foreground border border-border"
  if (a.includes("delete") || a.includes("revoke")) {
    cls = "bg-hydra-red/10 text-hydra-red border border-hydra-red/20"
  } else if (a.includes("create")) {
    cls = "bg-hydra-green/10 text-hydra-green border border-hydra-green/20"
  } else if (a.includes("update") || a.includes("rotate") || a.includes("edit")) {
    cls = "bg-hydra-blue/10 text-hydra-blue border border-hydra-blue/20"
  } else if (a.includes("login") || a.includes("auth")) {
    cls = "bg-hydra-teal/10 text-hydra-teal border border-hydra-teal/20"
  }
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider ${cls}`}>
      {action}
    </span>
  )
}

function JsonBlock({ label, value }: { label: string; value: Record<string, unknown> | null }) {
  return (
    <div className="flex-1 min-w-[180px]">
      <p className="text-[10px] text-muted-foreground uppercase tracking-widest font-bold mb-1">{label}</p>
      <pre className="text-[11px] font-mono bg-background border border-border rounded-md p-2.5 overflow-x-auto text-foreground/80 whitespace-pre-wrap break-words">
        {value ? JSON.stringify(value, null, 2) : "—"}
      </pre>
    </div>
  )
}

export default function AuditPage() {
  const [data, setData] = useState<AuditListData | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [actorInput, setActorInput] = useState("")
  const [actionInput, setActionInput] = useState("")
  const [filters, setFilters] = useState<{ actor: string; action: string }>({ actor: "", action: "" })
  const [expanded, setExpanded] = useState<string | null>(null)

  const fetchData = useCallback(() => {
    setLoading(true)
    getAuditEvents({
      page,
      page_size: PAGE_SIZE,
      ...(filters.actor ? { actor: filters.actor } : {}),
      ...(filters.action ? { action: filters.action } : {}),
    })
      .then((d) => { setData(d); setError(null) })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false))
  }, [page, filters])

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- fetchData flips the loading flag before awaiting the API
    fetchData()
  }, [fetchData])

  const applyFilters = () => {
    setPage(1)
    setFilters({ actor: actorInput.trim(), action: actionInput.trim() })
  }

  const events = data?.events ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const startIdx = total > 0 ? (page - 1) * PAGE_SIZE + 1 : 0
  const endIdx = Math.min(page * PAGE_SIZE, total)

  return (
    <>
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
                  <BreadcrumbPage>Audit Log</BreadcrumbPage>
                </BreadcrumbItem>
              </BreadcrumbList>
            </Breadcrumb>
          </div>
        </div>
        <button
          onClick={fetchData}
          className="p-1.5 text-muted-foreground hover:text-foreground transition-colors"
          title="Refresh"
        >
          <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
        </button>
      </header>

      <div className="flex flex-1 flex-col gap-6 py-6">
        <div>
          <h2 className="font-headline text-3xl font-bold tracking-tight">Audit Log</h2>
          <p className="text-muted-foreground mt-1 text-sm">
            Immutable trail of every administrative change on the gateway.
          </p>
        </div>

        {error && (
          <div className="rounded-md border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
            {error}
          </div>
        )}

        {/* Filter bar */}
        <div className="bg-card/30 border border-border rounded-xl p-3 flex flex-wrap items-center gap-3">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Filter by actor..."
              className="pl-10 w-56 bg-background border-border focus-visible:ring-primary"
              value={actorInput}
              onChange={(e) => setActorInput(e.target.value)}
              onKeyDown={(e) => { if (e.key === "Enter") applyFilters() }}
            />
          </div>
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Filter by action..."
              className="pl-10 w-56 bg-background border-border focus-visible:ring-primary"
              value={actionInput}
              onChange={(e) => setActionInput(e.target.value)}
              onKeyDown={(e) => { if (e.key === "Enter") applyFilters() }}
            />
          </div>
          <Button variant="outline" size="sm" onClick={applyFilters} className="gap-2">
            <Search className="h-4 w-4" />
            Search
          </Button>
        </div>

        {/* Table */}
        <div className="bg-card rounded-xl border border-border overflow-hidden">
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow className="border-b border-background bg-background/50 hover:bg-background/50">
                  <TableHead className="px-4 py-4 w-8" />
                  <TableHead className="px-4 py-4 text-[11px] font-bold uppercase tracking-wider text-muted-foreground">Timestamp</TableHead>
                  <TableHead className="px-4 py-4 text-[11px] font-bold uppercase tracking-wider text-muted-foreground">Actor</TableHead>
                  <TableHead className="px-4 py-4 text-[11px] font-bold uppercase tracking-wider text-muted-foreground">Action</TableHead>
                  <TableHead className="px-4 py-4 text-[11px] font-bold uppercase tracking-wider text-muted-foreground">Target</TableHead>
                  <TableHead className="px-4 py-4 text-[11px] font-bold uppercase tracking-wider text-muted-foreground">IP</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody className="text-sm">
                {events.length > 0 ? (
                  events.map((ev: AuditEvent) => {
                    const isOpen = expanded === ev.id
                    return (
                      <Fragment key={ev.id}>
                        <TableRow
                          className="border-b border-background hover:bg-white/5 transition-colors cursor-pointer"
                          onClick={() => setExpanded(isOpen ? null : ev.id)}
                        >
                          <TableCell className="px-4 py-3 text-muted-foreground">
                            <ChevronDown className={`h-4 w-4 transition-transform ${isOpen ? "rotate-180" : ""}`} />
                          </TableCell>
                          <TableCell className="px-4 py-3 text-muted-foreground text-xs whitespace-nowrap">
                            {new Date(ev.timestamp).toLocaleString()}
                          </TableCell>
                          <TableCell className="px-4 py-3 font-medium text-foreground">
                            {ev.actor}
                          </TableCell>
                          <TableCell className="px-4 py-3">
                            {actionBadge(ev.action)}
                          </TableCell>
                          <TableCell className="px-4 py-3 font-mono text-xs text-muted-foreground">
                            {ev.target_type ? <span className="text-foreground/60">{ev.target_type}/</span> : null}
                            {ev.target}
                          </TableCell>
                          <TableCell className="px-4 py-3 font-mono text-xs text-muted-foreground">
                            {ev.ip}
                          </TableCell>
                        </TableRow>
                        {isOpen && (
                          <TableRow className="border-b border-background bg-background/30 hover:bg-background/30">
                            <TableCell colSpan={6} className="px-6 py-4">
                              <div className="flex flex-wrap gap-4">
                                <JsonBlock label="Before" value={ev.before} />
                                <JsonBlock label="After" value={ev.after} />
                              </div>
                            </TableCell>
                          </TableRow>
                        )}
                      </Fragment>
                    )
                  })
                ) : (
                  <TableRow>
                    <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                      {loading
                        ? "Loading audit events..."
                        : filters.actor || filters.action
                          ? "No matching audit events"
                          : "No audit events recorded yet"}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>

          {/* Pagination footer */}
          <div className="px-6 py-4 flex items-center justify-between border-t border-background bg-background/30">
            <p className="text-xs text-muted-foreground">
              Showing <span className="text-foreground font-bold">{startIdx}-{endIdx}</span> of{" "}
              <span className="text-foreground font-bold">{total.toLocaleString()}</span> events
            </p>
            <div className="flex items-center gap-2">
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
              <span className="text-xs text-muted-foreground tabular-nums">
                Page {page} / {totalPages}
              </span>
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

        {/* Footer hint */}
        <div className="flex items-center gap-2 text-xs text-muted-foreground px-1">
          <History className="h-3.5 w-3.5" />
          Click any row to inspect the before/after snapshot for that change.
        </div>
      </div>
    </>
  )
}
