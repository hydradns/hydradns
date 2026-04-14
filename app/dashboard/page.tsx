"use client"

import { useEffect, useMemo, useState } from "react"
import Link from "next/link"
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import { Separator } from "@/components/ui/separator"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { SectionCards } from "@/components/section-cards"
import { getDashboardSummary, getDnsEngineStatus, getQueryLogs } from "@/lib/api"
import type { DashboardSummary, DnsEngineStatus, QueryLogEntry } from "@/lib/types"
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts"
import { ArrowRight, RefreshCw } from "lucide-react"

// ---- helpers ----

function formatTime(ts: string): string {
  try {
    const d = new Date(ts)
    return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" })
  } catch {
    return ts
  }
}

function threatColor(score: number): string {
  if (score >= 0.7) return "bg-red-500"
  if (score >= 0.3) return "bg-amber-500"
  return "bg-green-500"
}

function threatTextColor(score: number): string {
  if (score >= 0.7) return "text-red-400"
  if (score >= 0.3) return "text-amber-400"
  return "text-slate-500"
}

function actionBadge(action: string, isSuspicious: boolean) {
  if (isSuspicious) {
    return (
      <span className="px-2 py-0.5 rounded text-[10px] font-bold uppercase bg-amber-500/10 text-amber-500 border border-amber-500/20">
        Flagged
      </span>
    )
  }
  const lower = action.toLowerCase()
  if (lower === "block" || lower === "blocked" || lower === "refuse" || lower === "refused") {
    return (
      <span className="px-2 py-0.5 rounded text-[10px] font-bold uppercase bg-red-500/10 text-red-500 border border-red-500/20">
        Blocked
      </span>
    )
  }
  return (
    <span className="px-2 py-0.5 rounded text-[10px] font-bold uppercase bg-green-500/10 text-green-500 border border-green-500/20">
      Allowed
    </span>
  )
}

/** Build hourly buckets from query logs for the area chart. */
function buildChartData(logs: QueryLogEntry[]) {
  const buckets: Record<number, { allowed: number; blocked: number }> = {}
  for (let h = 0; h < 24; h++) {
    buckets[h] = { allowed: 0, blocked: 0 }
  }
  for (const log of logs) {
    try {
      const hour = new Date(log.timestamp).getHours()
      const a = log.action.toLowerCase()
      if (a === "block" || a === "blocked" || a === "refuse" || a === "refused") {
        buckets[hour].blocked++
      } else {
        buckets[hour].allowed++
      }
    } catch {
      // skip malformed timestamps
    }
  }
  return Array.from({ length: 24 }, (_, h) => ({
    hour: `${String(h).padStart(2, "0")}:00`,
    allowed: buckets[h].allowed,
    blocked: buckets[h].blocked,
  }))
}

/** Aggregate top blocked domains from query logs. */
function topBlockedDomains(logs: QueryLogEntry[], max = 5) {
  const counts: Record<string, number> = {}
  for (const log of logs) {
    const a = log.action.toLowerCase()
    if (a === "block" || a === "blocked" || a === "refuse" || a === "refused") {
      counts[log.domain] = (counts[log.domain] ?? 0) + 1
    }
  }
  const sorted = Object.entries(counts)
    .sort(([, a], [, b]) => b - a)
    .slice(0, max)
  const peak = sorted[0]?.[1] ?? 1
  return sorted.map(([domain, count]) => ({
    domain,
    count,
    pct: Math.round((count / peak) * 100),
  }))
}

// ---- component ----

export default function DashboardPage() {
  const [summary, setSummary] = useState<DashboardSummary | null>(null)
  const [engine, setEngine] = useState<DnsEngineStatus | null>(null)
  const [logs, setLogs] = useState<QueryLogEntry[]>([])
  const [error, setError] = useState<string | null>(null)

  const fetchData = async () => {
    try {
      const [s, e, l] = await Promise.all([
        getDashboardSummary(),
        getDnsEngineStatus(),
        getQueryLogs(),
      ])
      setSummary(s)
      setEngine(e)
      setLogs(l)
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to connect to API")
    }
  }

  useEffect(() => {
    const interval = setInterval(fetchData, 5000)
    // eslint-disable-next-line react-hooks/set-state-in-effect -- initial fetch before interval kicks in
    void fetchData()
    return () => clearInterval(interval)
  }, [])

  const chartData = useMemo(() => buildChartData(logs), [logs])
  const topBlocked = useMemo(() => topBlockedDomains(logs), [logs])
  const recentLogs = useMemo(() => logs.slice(0, 10), [logs])
  const threatCount = useMemo(
    () => logs.filter((l) => l.is_suspicious).length,
    [logs],
  )

  return (
    <>
      {/* Header */}
      <header className="flex flex-wrap gap-3 min-h-16 py-3 shrink-0 items-center border-b border-border/40">
        <div className="flex flex-1 items-center gap-2">
          <SidebarTrigger className="-ms-1" />
          <div className="max-lg:hidden lg:contents">
            <Separator
              orientation="vertical"
              className="me-2 data-[orientation=vertical]:h-4"
            />
            <Breadcrumb>
              <BreadcrumbList>
                <BreadcrumbItem className="hidden md:block">
                  <BreadcrumbLink href="/dashboard">HydraDNS</BreadcrumbLink>
                </BreadcrumbItem>
                <BreadcrumbSeparator className="hidden md:block" />
                <BreadcrumbItem>
                  <BreadcrumbPage>Dashboard</BreadcrumbPage>
                </BreadcrumbItem>
              </BreadcrumbList>
            </Breadcrumb>
          </div>
        </div>

        <div className="flex items-center gap-4">
          {/* Protected status pill */}
          {engine?.accepting_queries && (
            <div className="flex items-center gap-2 px-3 py-1 bg-green-500/10 border border-green-500/20 rounded-full">
              <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
              <span className="text-[11px] font-bold text-green-500 uppercase tracking-wider">
                Protected
              </span>
            </div>
          )}
          {engine && !engine.accepting_queries && (
            <div className="flex items-center gap-2 px-3 py-1 bg-red-500/10 border border-red-500/20 rounded-full">
              <span className="w-2 h-2 bg-red-500 rounded-full" />
              <span className="text-[11px] font-bold text-red-500 uppercase tracking-wider">
                Unprotected
              </span>
            </div>
          )}
          <button
            onClick={fetchData}
            className="text-muted-foreground hover:text-primary transition-colors"
          >
            <RefreshCw className="size-4" />
          </button>
        </div>
      </header>

      <div className="@container/main flex flex-1 flex-col gap-6 py-6">
        {/* Error banner */}
        {error && (
          <div className="rounded-md border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
            {error} — Is the API running on{" "}
            {process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"}?
          </div>
        )}

        {/* Stat cards */}
        <SectionCards data={summary} threats={threatCount} />

        {/* Charts row */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Query Activity chart */}
          <div className="lg:col-span-2 bg-card p-6 rounded-xl border border-border/20">
            <div className="flex justify-between items-center mb-6">
              <h3 className="font-headline font-bold text-lg text-foreground">
                Query Activity
              </h3>
              <div className="flex gap-4">
                <div className="flex items-center gap-2">
                  <span className="w-3 h-3 rounded-full bg-[#00D4AA]" />
                  <span className="text-xs text-muted-foreground">Allowed</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="w-3 h-3 rounded-full bg-red-500" />
                  <span className="text-xs text-muted-foreground">Blocked</span>
                </div>
              </div>
            </div>
            <ResponsiveContainer width="100%" height={256}>
              <AreaChart data={chartData}>
                <defs>
                  <linearGradient id="fillAllowed" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#00D4AA" stopOpacity={0.3} />
                    <stop offset="100%" stopColor="#00D4AA" stopOpacity={0.02} />
                  </linearGradient>
                  <linearGradient id="fillBlocked" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#EF4444" stopOpacity={0.3} />
                    <stop offset="100%" stopColor="#EF4444" stopOpacity={0.02} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.05)" />
                <XAxis
                  dataKey="hour"
                  tick={{ fontSize: 10, fill: "#64748B" }}
                  tickLine={false}
                  axisLine={false}
                  interval={3}
                />
                <YAxis
                  tick={{ fontSize: 10, fill: "#64748B" }}
                  tickLine={false}
                  axisLine={false}
                  width={32}
                />
                <Tooltip
                  contentStyle={{
                    backgroundColor: "#22262E",
                    border: "1px solid rgba(255,255,255,0.1)",
                    borderRadius: "0.5rem",
                    fontSize: "12px",
                  }}
                  labelStyle={{ color: "#94A3B8" }}
                />
                <Area
                  type="monotone"
                  dataKey="allowed"
                  stroke="#00D4AA"
                  fill="url(#fillAllowed)"
                  strokeWidth={2}
                  stackId="1"
                />
                <Area
                  type="monotone"
                  dataKey="blocked"
                  stroke="#EF4444"
                  fill="url(#fillBlocked)"
                  strokeWidth={2}
                  stackId="1"
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>

          {/* Top Blocked Domains */}
          <div className="bg-card p-6 rounded-xl border border-border/20">
            <h3 className="font-headline font-bold text-lg text-foreground mb-6">
              Top Blocked Domains
            </h3>
            <div className="space-y-5">
              {topBlocked.length === 0 && (
                <p className="text-sm text-muted-foreground">No blocked domains yet.</p>
              )}
              {topBlocked.map(({ domain, count, pct }) => (
                <div key={domain} className="space-y-1.5">
                  <div className="flex justify-between text-xs">
                    <span className="font-mono text-slate-300 truncate mr-2">
                      {domain}
                    </span>
                    <span className="text-muted-foreground tabular-nums shrink-0">
                      {count.toLocaleString()}
                    </span>
                  </div>
                  <div className="h-2 bg-slate-700/50 rounded-full overflow-hidden">
                    <div
                      className="h-full bg-red-500 rounded-full transition-all duration-500"
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Recent Activity table */}
        <div className="bg-card rounded-xl border border-border/20 overflow-hidden">
          <div className="px-6 py-5 border-b border-border/20 flex justify-between items-center">
            <h3 className="font-headline font-bold text-lg text-foreground">
              Recent Activity
            </h3>
            <Link
              href="/dashboard/logs"
              className="text-xs font-semibold text-primary flex items-center gap-1 hover:underline"
            >
              View full logs
              <ArrowRight className="size-3" />
            </Link>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-left">
              <thead>
                <tr className="bg-muted/30 text-[10px] uppercase tracking-widest text-muted-foreground font-bold">
                  <th className="px-6 py-4">Time</th>
                  <th className="px-6 py-4">Domain</th>
                  <th className="px-6 py-4">Client</th>
                  <th className="px-6 py-4">Action</th>
                  <th className="px-6 py-4">Threat Score</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border/10 text-sm">
                {recentLogs.length === 0 && (
                  <tr>
                    <td
                      colSpan={5}
                      className="px-6 py-8 text-center text-muted-foreground"
                    >
                      No query logs yet.
                    </td>
                  </tr>
                )}
                {recentLogs.map((log) => (
                  <tr
                    key={log.id}
                    className="hover:bg-muted/10 transition-colors"
                  >
                    <td className="px-6 py-4 text-muted-foreground font-mono text-xs whitespace-nowrap">
                      {formatTime(log.timestamp)}
                    </td>
                    <td className="px-6 py-4 font-mono text-slate-200 truncate max-w-[200px]">
                      {log.domain}
                    </td>
                    <td className="px-6 py-4 text-muted-foreground font-mono text-xs">
                      {log.client_ip}
                    </td>
                    <td className="px-6 py-4">
                      {actionBadge(log.action, log.is_suspicious)}
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-2">
                        <div className="w-12 h-1.5 bg-slate-700 rounded-full overflow-hidden">
                          <div
                            className={`h-full ${threatColor(log.threat_score)} rounded-full`}
                            style={{
                              width: `${Math.max(log.threat_score * 100, 2)}%`,
                            }}
                          />
                        </div>
                        <span
                          className={`text-[10px] tabular-nums ${threatTextColor(log.threat_score)}`}
                        >
                          {log.threat_score.toFixed(2)}
                        </span>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </>
  )
}
