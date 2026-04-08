"use client"

import { useEffect, useState } from "react"
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
import { Button } from "@/components/ui/button"
import { SectionCards } from "@/components/section-cards"
import { getDashboardSummary, getDnsEngineStatus } from "@/lib/api"
import type { DashboardSummary, DnsEngineStatus } from "@/lib/types"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

export default function DashboardPage() {
  const [summary, setSummary] = useState<DashboardSummary | null>(null)
  const [engine, setEngine] = useState<DnsEngineStatus | null>(null)
  const [error, setError] = useState<string | null>(null)

  const fetchData = async () => {
    try {
      const [s, e] = await Promise.all([
        getDashboardSummary(),
        getDnsEngineStatus(),
      ])
      setSummary(s)
      setEngine(e)
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to connect to API")
    }
  }

  useEffect(() => {
    fetchData()
    const interval = setInterval(fetchData, 5000)
    return () => clearInterval(interval)
  }, [])

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
                  <BreadcrumbPage>Dashboard</BreadcrumbPage>
                </BreadcrumbItem>
              </BreadcrumbList>
            </Breadcrumb>
          </div>
        </div>
        <Button variant="outline" size="sm" onClick={fetchData}>
          Refresh
        </Button>
      </header>

      <div className="@container/main flex flex-1 flex-col gap-4 py-4 md:gap-6 md:py-6">
        {error && (
          <div className="rounded-md border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
            {error} — Is the API running on {process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"}?
          </div>
        )}

        <SectionCards data={summary} />

        <div className="grid gap-4 @xl/main:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>DNS Engine</CardTitle>
            </CardHeader>
            <CardContent className="flex items-center gap-4">
              {engine ? (
                <>
                  <Badge variant={engine.accepting_queries ? "default" : "destructive"}>
                    {engine.accepting_queries ? "Running" : "Stopped"}
                  </Badge>
                  <span className="text-sm text-muted-foreground">
                    {engine.enabled ? "Enabled" : "Disabled"} in config
                  </span>
                  {engine.last_error && (
                    <span className="text-sm text-destructive">{engine.last_error}</span>
                  )}
                </>
              ) : (
                <span className="text-sm text-muted-foreground">Loading...</span>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Quick Stats</CardTitle>
            </CardHeader>
            <CardContent>
              {summary ? (
                <div className="grid grid-cols-2 gap-2 text-sm">
                  <div className="text-muted-foreground">Redirected</div>
                  <div className="font-medium tabular-nums">{summary.redirected_queries.toLocaleString()}</div>
                  <div className="text-muted-foreground">Block Rate</div>
                  <div className="font-medium tabular-nums">{summary.block_rate_percent.toFixed(2)}%</div>
                </div>
              ) : (
                <span className="text-sm text-muted-foreground">Loading...</span>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </>
  )
}
