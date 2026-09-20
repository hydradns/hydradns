"use client"

import {
  Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList,
  BreadcrumbPage, BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import { Separator } from "@/components/ui/separator"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import type { Settings } from "@/lib/types"
import { Shield, Info } from "lucide-react"

// There is no GET/PATCH /settings route on the control plane (see
// lib/api.ts) — this page shows the intended shape of engine/caching/
// retention settings as a disabled preview rather than pretending changes
// here would save. Values are illustrative defaults, not read from the
// backend.
const PREVIEW_SETTINGS: Settings = {
  engine_enabled: true,
  block_page_enabled: true,
  cache_enabled: true,
  cache_ttl_seconds: 300,
  upstream_timeout_ms: 2000,
  log_retention_days: 30,
}

export default function SettingsPage() {
  const settings = PREVIEW_SETTINGS

  const toggleRow = (
    key: keyof Settings,
    title: string,
    description: string,
  ) => (
    <div className="flex items-center justify-between gap-4 rounded-xl border border-border bg-card p-5 opacity-60">
      <div>
        <Label htmlFor={`setting-${key}`} className="text-sm font-semibold text-foreground">
          {title}
        </Label>
        <p className="text-xs text-muted-foreground mt-1">{description}</p>
      </div>
      <Switch
        id={`setting-${key}`}
        checked={Boolean(settings[key])}
        disabled
        aria-label={title}
      />
    </div>
  )

  const numberRow = (
    key: "cache_ttl_seconds" | "upstream_timeout_ms" | "log_retention_days",
    title: string,
    description: string,
    unit: string,
  ) => (
    <div className="flex items-center justify-between gap-4 rounded-xl border border-border bg-card p-5 opacity-60">
      <div>
        <Label htmlFor={`setting-${key}`} className="text-sm font-semibold text-foreground">
          {title}
        </Label>
        <p className="text-xs text-muted-foreground mt-1">{description}</p>
      </div>
      <div className="flex items-center gap-2 shrink-0">
        <Input
          id={`setting-${key}`}
          type="number"
          value={settings[key]}
          disabled
          className="w-28 bg-background border-border rounded-lg text-right font-mono"
        />
        <span className="text-xs text-muted-foreground w-8">{unit}</span>
      </div>
    </div>
  )

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
                  <BreadcrumbPage>Settings</BreadcrumbPage>
                </BreadcrumbItem>
              </BreadcrumbList>
            </Breadcrumb>
          </div>
        </div>
      </header>

      <div className="flex flex-1 flex-col gap-6 py-6">
        <div>
          <h2 className="font-headline text-3xl font-bold tracking-tight">Settings</h2>
          <p className="text-muted-foreground mt-1 text-sm">
            Engine, caching, and data retention settings for this gateway.
          </p>
        </div>

        <div className="flex items-start gap-3 rounded-lg border border-border bg-card p-4 text-sm text-muted-foreground">
          <Info className="h-4 w-4 mt-0.5 shrink-0 text-[#00D4AA]" />
          <p>
            Not available yet — the control plane doesn&apos;t have a settings API to read or
            save these from the dashboard. The controls below preview the planned settings and
            are disabled.
          </p>
        </div>

        <section className="space-y-3">
          <div className="flex items-center gap-2 px-1">
            <Shield className="h-4 w-4 text-[#00D4AA]" />
            <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-widest">
              Engine
            </h3>
          </div>
          {toggleRow("engine_enabled", "DNS Engine", "Master switch for query processing. Disabling stops all filtering.")}
          {toggleRow("block_page_enabled", "Block Page", "Serve a branded block page instead of NXDOMAIN for blocked lookups.")}
        </section>

        <section className="space-y-3">
          <div className="flex items-center gap-2 px-1">
            <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-widest">
              Caching & Upstream
            </h3>
          </div>
          {toggleRow("cache_enabled", "Response Cache", "Cache upstream answers to speed up repeat lookups.")}
          {numberRow("cache_ttl_seconds", "Cache TTL", "How long cached answers are kept before refetching.", "sec")}
          {numberRow("upstream_timeout_ms", "Upstream Timeout", "Time to wait for an upstream resolver before failing over.", "ms")}
        </section>

        <section className="space-y-3">
          <div className="flex items-center gap-2 px-1">
            <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-widest">
              Data Retention
            </h3>
          </div>
          {numberRow("log_retention_days", "Query Log Retention", "How many days of query logs to keep before pruning.", "days")}
        </section>
      </div>
    </>
  )
}
