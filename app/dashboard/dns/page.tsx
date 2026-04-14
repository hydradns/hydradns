"use client"

import { useEffect, useState } from "react"
import {
  Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList,
  BreadcrumbPage, BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import { Separator } from "@/components/ui/separator"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { Badge } from "@/components/ui/badge"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { getDnsEngineStatus, toggleDnsEngine, getDnsMetrics, getResolvers } from "@/lib/api"
import type { DnsEngineStatus, DnsMetrics, Resolver } from "@/lib/types"
import {
  Shield, Server, Activity, Clock, Gauge,
  Zap, AlertTriangle, Network,
} from "lucide-react"

export default function DnsEnginePage() {
  const [status, setStatus] = useState<DnsEngineStatus | null>(null)
  const [metrics, setMetrics] = useState<DnsMetrics | null>(null)
  const [resolvers, setResolvers] = useState<Resolver[]>([])
  const [toggling, setToggling] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchAll = () => {
    getDnsEngineStatus().then((s) => { setStatus(s); setError(null) }).catch((e) => setError(e.message))
    getDnsMetrics().then(setMetrics).catch(() => {})
    getResolvers().then(setResolvers).catch(() => {})
  }

  useEffect(() => {
    fetchAll()
    const interval = setInterval(fetchAll, 5000)
    return () => clearInterval(interval)
  }, [])

  const handleToggle = async () => {
    if (!status) return
    setToggling(true)
    try {
      await toggleDnsEngine(!status.enabled)
      fetchAll()
    } finally {
      setToggling(false)
    }
  }

  const isActive = status?.enabled && status?.accepting_queries
  const gradeColors: Record<string, string> = {
    excellent: "text-green-400",
    good: "text-sky-400",
    degraded: "text-amber-400",
    bad: "text-red-400",
    unknown: "text-muted-foreground",
  }
  const gradeColor = metrics?.grade
    ? (gradeColors[metrics.grade] ?? "text-muted-foreground")
    : "text-muted-foreground"

  return (
    <>
      <header className="flex flex-wrap gap-3 min-h-20 py-4 shrink-0 items-center border-b">
        <div className="flex flex-1 items-center gap-2">
          <SidebarTrigger className="-ms-1" />
          <div className="max-lg:hidden lg:contents">
            <Separator orientation="vertical" className="me-2 data-[orientation=vertical]:h-4" />
            <Breadcrumb>
              <BreadcrumbList>
                <BreadcrumbItem className="hidden md:block">
                  <BreadcrumbLink href="/dashboard">Home</BreadcrumbLink>
                </BreadcrumbItem>
                <BreadcrumbSeparator className="hidden md:block" />
                <BreadcrumbItem>
                  <BreadcrumbPage>DNS Engine</BreadcrumbPage>
                </BreadcrumbItem>
              </BreadcrumbList>
            </Breadcrumb>
          </div>
        </div>
      </header>

      <div className="flex flex-1 flex-col gap-6 py-6">
        {error && (
          <div className="flex items-center gap-3 rounded-xl border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
            <AlertTriangle className="h-4 w-4 shrink-0" />
            {error}
          </div>
        )}

        {/* ── Hero Card: Engine Status ── */}
        <section className="relative overflow-hidden rounded-xl border border-border bg-card p-8">
          {/* Decorative glow */}
          {isActive && (
            <div className="pointer-events-none absolute -right-20 -top-20 h-64 w-64 rounded-full bg-primary/5 blur-3xl" />
          )}

          <div className="relative flex flex-col gap-8 md:flex-row md:items-center md:justify-between">
            {/* Left: icon + copy */}
            <div className="flex items-center gap-6">
              <div className={`flex h-16 w-16 shrink-0 items-center justify-center rounded-2xl border ${
                isActive
                  ? "border-primary/20 bg-primary/10"
                  : "border-destructive/20 bg-destructive/10"
              }`}>
                <Shield className={`h-8 w-8 ${isActive ? "text-primary" : "text-destructive"}`} />
              </div>

              <div>
                <div className="mb-1 flex flex-wrap items-center gap-3">
                  <h2 className="font-headline text-2xl font-bold text-foreground">
                    {status
                      ? isActive
                        ? "DNS Engine is Active"
                        : "DNS Engine is Offline"
                      : "Connecting..."}
                  </h2>
                  {status && (
                    <span className={`inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider ${
                      isActive
                        ? "border-green-500/20 bg-green-500/10 text-green-400"
                        : "border-red-500/20 bg-red-500/10 text-red-400"
                    }`}>
                      <span className={`h-2 w-2 rounded-full ${isActive ? "bg-green-400" : "bg-red-400"}`} />
                      {isActive ? "Online" : "Offline"}
                    </span>
                  )}
                </div>

                <p className="text-sm text-muted-foreground">
                  {isActive
                    ? "Protecting your network from malicious domains and ensuring fast resolution."
                    : status
                      ? "The engine is not currently processing DNS queries."
                      : "Connecting to dataplane..."}
                </p>

                {status?.last_error && (
                  <p className="mt-2 text-xs text-destructive">{status.last_error}</p>
                )}

                {status && metrics && (
                  <div className="mt-4 flex flex-wrap items-center gap-4 font-mono text-xs text-muted-foreground">
                    <span className="flex items-center gap-1.5">
                      <Clock className="h-3.5 w-3.5" />
                      Window: {metrics.window_seconds}s
                    </span>
                    <span className="flex items-center gap-1.5">
                      <Network className="h-3.5 w-3.5" />
                      {resolvers.length} resolver{resolvers.length !== 1 ? "s" : ""}
                    </span>
                  </div>
                )}
              </div>
            </div>

            {/* Right: Power toggle */}
            <div className="flex items-center gap-4 rounded-xl border border-border bg-background p-4">
              <span className="text-sm font-bold text-muted-foreground">Engine Power</span>
              <button
                onClick={handleToggle}
                disabled={toggling || !status}
                className={`relative inline-flex h-8 w-14 shrink-0 items-center rounded-full transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 ${
                  isActive ? "bg-primary" : "bg-muted"
                }`}
                role="switch"
                aria-checked={!!isActive}
                aria-label="Toggle DNS engine"
              >
                <span
                  className={`inline-block h-6 w-6 rounded-full bg-white shadow-lg transition-transform duration-200 ${
                    isActive ? "translate-x-7" : "translate-x-1"
                  }`}
                />
              </button>
            </div>
          </div>
        </section>

        {/* ── Two-column layout ── */}
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
          {/* Left: Upstream Resolvers */}
          <div className="overflow-hidden rounded-xl border border-border bg-card lg:col-span-7">
            <div className="flex items-center justify-between border-b border-background px-6 py-5">
              <h3 className="flex items-center gap-2 font-headline font-bold text-foreground">
                <Server className="h-5 w-5 text-primary" />
                Upstream Resolvers
              </h3>
            </div>

            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow className="border-background hover:bg-transparent">
                    <TableHead className="px-6 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">Name</TableHead>
                    <TableHead className="px-6 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">IP Address</TableHead>
                    <TableHead className="px-6 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">Protocol</TableHead>
                    <TableHead className="px-6 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">Status</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {resolvers.length > 0 ? (
                    resolvers.map((r, idx) => (
                      <TableRow key={r.id} className="border-background hover:bg-background/30">
                        <TableCell className="px-6 py-4">
                          <div className="flex items-center gap-3">
                            <span className={`h-2 w-2 rounded-full ${idx === 0 ? "bg-green-400" : "bg-muted-foreground/40"}`} />
                            <span className="text-sm font-medium text-foreground">{r.name || r.address}</span>
                          </div>
                        </TableCell>
                        <TableCell className="px-6 py-4 font-mono text-xs text-muted-foreground">
                          {r.address}
                        </TableCell>
                        <TableCell className="px-6 py-4">
                          <Badge variant="outline" className="text-[10px] font-bold uppercase">
                            {r.protocol}
                          </Badge>
                        </TableCell>
                        <TableCell className="px-6 py-4">
                          {idx === 0 ? (
                            <span className="rounded bg-green-500/10 px-2 py-1 text-[10px] font-bold uppercase text-green-400">
                              Active
                            </span>
                          ) : (
                            <span className="rounded bg-muted px-2 py-1 text-[10px] font-bold uppercase text-muted-foreground">
                              Standby
                            </span>
                          )}
                        </TableCell>
                      </TableRow>
                    ))
                  ) : (
                    <TableRow>
                      <TableCell colSpan={4} className="py-8 text-center text-muted-foreground">
                        No resolvers configured
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </div>
          </div>

          {/* Right: Engine Metrics */}
          <div className="space-y-6 lg:col-span-5">
            <div className="rounded-xl border border-border bg-card p-6">
              <div className="mb-6 flex items-center justify-between">
                <h3 className="flex items-center gap-2 font-headline font-bold text-foreground">
                  <Activity className="h-5 w-5 text-primary" />
                  Engine Metrics
                </h3>
                {metrics && (
                  <Badge variant="outline" className={`text-[10px] font-bold uppercase ${gradeColor}`}>
                    {metrics.grade}
                  </Badge>
                )}
              </div>

              {metrics ? (
                <div className="space-y-6">
                  {/* Total queries hero stat */}
                  <div className="rounded-lg border border-border bg-background p-4">
                    <div className="mb-2 flex items-center justify-between">
                      <span className="text-xs text-muted-foreground">Total Queries</span>
                      <span className="text-xs font-bold text-muted-foreground">
                        {metrics.window_seconds}s window
                      </span>
                    </div>
                    <div className="flex items-end gap-3">
                      <span className="font-headline text-3xl font-bold text-foreground">
                        {metrics.queries.total.toLocaleString()}
                      </span>
                    </div>
                  </div>

                  {/* 2x2 metric grid */}
                  <div className="grid grid-cols-2 gap-4">
                    <div className="rounded-lg border border-border border-l-2 border-l-sky-500 bg-background p-4">
                      <div className="mb-1 flex items-center gap-1.5">
                        <Gauge className="h-3 w-3 text-muted-foreground" />
                        <span className="text-[10px] uppercase tracking-wider text-muted-foreground">p50 Latency</span>
                      </div>
                      <span className="font-headline text-xl font-bold text-foreground">{metrics.latency_ms.p50}ms</span>
                    </div>

                    <div className="rounded-lg border border-border border-l-2 border-l-primary bg-background p-4">
                      <div className="mb-1 flex items-center gap-1.5">
                        <Gauge className="h-3 w-3 text-muted-foreground" />
                        <span className="text-[10px] uppercase tracking-wider text-muted-foreground">p95 Latency</span>
                      </div>
                      <span className="font-headline text-xl font-bold text-foreground">{metrics.latency_ms.p95}ms</span>
                    </div>

                    <div className="rounded-lg border border-border border-l-2 border-l-foreground bg-background p-4">
                      <div className="mb-1 flex items-center gap-1.5">
                        <Gauge className="h-3 w-3 text-muted-foreground" />
                        <span className="text-[10px] uppercase tracking-wider text-muted-foreground">p99 Latency</span>
                      </div>
                      <span className="font-headline text-xl font-bold text-foreground">{metrics.latency_ms.p99}ms</span>
                    </div>

                    <div className="rounded-lg border border-border border-l-2 border-l-amber-500 bg-background p-4">
                      <div className="mb-1 flex items-center gap-1.5">
                        <AlertTriangle className="h-3 w-3 text-muted-foreground" />
                        <span className="text-[10px] uppercase tracking-wider text-muted-foreground">Error Rate</span>
                      </div>
                      <span className="font-headline text-xl font-bold text-foreground">
                        {(metrics.queries.error_rate * 100).toFixed(2)}%
                      </span>
                    </div>
                  </div>

                  {/* Errors row */}
                  <div className="rounded-lg border border-border bg-background p-4">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <Zap className="h-4 w-4 text-primary" />
                        <span className="text-xs text-muted-foreground">Query Errors</span>
                      </div>
                      <span className="font-headline text-lg font-bold text-foreground">
                        {metrics.queries.errors.toLocaleString()}
                      </span>
                    </div>
                  </div>
                </div>
              ) : (
                <div className="flex h-40 items-center justify-center text-sm text-muted-foreground">
                  Waiting for metrics...
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </>
  )
}
