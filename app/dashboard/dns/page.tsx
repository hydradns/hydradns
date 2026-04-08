"use client"

import { useEffect, useState } from "react"
import {
  Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList,
  BreadcrumbPage, BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import { Separator } from "@/components/ui/separator"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { getDnsEngineStatus, toggleDnsEngine, getDnsMetrics, getResolvers } from "@/lib/api"
import type { DnsEngineStatus, DnsMetrics, Resolver } from "@/lib/types"

const gradeColors: Record<string, string> = {
  excellent: "bg-green-500/15 text-green-500",
  good: "bg-blue-500/15 text-blue-500",
  degraded: "bg-yellow-500/15 text-yellow-500",
  bad: "bg-red-500/15 text-red-500",
  unknown: "bg-muted text-muted-foreground",
}

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

      <div className="flex flex-1 flex-col gap-4 py-4 md:gap-6 md:py-6">
        {error && (
          <div className="rounded-md border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
            {error}
          </div>
        )}
        {/* Engine Status */}
        <Card>
          <CardHeader>
            <CardTitle>Engine Status</CardTitle>
            <CardDescription>Control the DNS query processing engine</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-wrap items-center gap-4">
            {status ? (
              <>
                <Badge variant={status.accepting_queries ? "default" : "destructive"}>
                  {status.accepting_queries ? "Accepting Queries" : "Not Accepting"}
                </Badge>
                <Badge variant="outline">
                  {status.enabled ? "Enabled" : "Disabled"}
                </Badge>
                {status.last_error && (
                  <span className="text-sm text-destructive">{status.last_error}</span>
                )}
                <Button
                  variant={status.enabled ? "destructive" : "default"}
                  size="sm"
                  onClick={handleToggle}
                  disabled={toggling}
                >
                  {toggling ? "..." : status.enabled ? "Disable Engine" : "Enable Engine"}
                </Button>
              </>
            ) : (
              <span className="text-sm text-muted-foreground">Connecting to dataplane...</span>
            )}
          </CardContent>
        </Card>

        {/* Metrics */}
        <div className="grid gap-4 @xl:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Query Metrics</CardTitle>
              <CardDescription>
                {metrics ? `${metrics.window_seconds}s rolling window` : "Loading..."}
              </CardDescription>
            </CardHeader>
            <CardContent>
              {metrics ? (
                <div className="grid grid-cols-2 gap-3 text-sm">
                  <div className="text-muted-foreground">Total Queries</div>
                  <div className="font-medium tabular-nums">{metrics.queries.total.toLocaleString()}</div>
                  <div className="text-muted-foreground">Errors</div>
                  <div className="font-medium tabular-nums">{metrics.queries.errors.toLocaleString()}</div>
                  <div className="text-muted-foreground">Error Rate</div>
                  <div className="font-medium tabular-nums">{(metrics.queries.error_rate * 100).toFixed(2)}%</div>
                </div>
              ) : (
                <span className="text-sm text-muted-foreground">—</span>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Latency</CardTitle>
              {metrics && (
                <Badge className={gradeColors[metrics.grade] || ""} variant="outline">
                  {metrics.grade}
                </Badge>
              )}
            </CardHeader>
            <CardContent>
              {metrics ? (
                <div className="grid grid-cols-2 gap-3 text-sm">
                  <div className="text-muted-foreground">p50</div>
                  <div className="font-medium tabular-nums">{metrics.latency_ms.p50}ms</div>
                  <div className="text-muted-foreground">p95</div>
                  <div className="font-medium tabular-nums">{metrics.latency_ms.p95}ms</div>
                  <div className="text-muted-foreground">p99</div>
                  <div className="font-medium tabular-nums">{metrics.latency_ms.p99}ms</div>
                </div>
              ) : (
                <span className="text-sm text-muted-foreground">—</span>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Resolvers */}
        <Card>
          <CardHeader>
            <CardTitle>Upstream Resolvers</CardTitle>
            <CardDescription>Configured in config.yaml</CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Address</TableHead>
                  <TableHead>Protocol</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {resolvers.length > 0 ? (
                  resolvers.map((r) => (
                    <TableRow key={r.id}>
                      <TableCell className="font-mono">{r.address}</TableCell>
                      <TableCell>
                        <Badge variant="outline">{r.protocol}</Badge>
                      </TableCell>
                    </TableRow>
                  ))
                ) : (
                  <TableRow>
                    <TableCell colSpan={2} className="text-center text-muted-foreground">
                      No resolvers configured
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </div>
    </>
  )
}
