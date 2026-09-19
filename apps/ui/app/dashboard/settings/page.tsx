"use client"

import { useEffect, useState } from "react"
import {
  Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList,
  BreadcrumbPage, BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import { Separator } from "@/components/ui/separator"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { getSettings, updateSettings } from "@/lib/api"
import type { Settings } from "@/lib/types"
import { Shield, Save, CheckCircle2, AlertTriangle } from "lucide-react"

const DEFAULT_SETTINGS: Settings = {
  engine_enabled: true,
  block_page_enabled: true,
  cache_enabled: true,
  cache_ttl_seconds: 300,
  upstream_timeout_ms: 2000,
  log_retention_days: 30,
}

export default function SettingsPage() {
  const [settings, setSettings] = useState<Settings>(DEFAULT_SETTINGS)
  const [loaded, setLoaded] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  useEffect(() => {
    getSettings()
      .then((s) => { setSettings(s); setError(null) })
      .catch((e) => setError(e.message))
      .finally(() => setLoaded(true))
  }, [])

  const setField = <K extends keyof Settings>(key: K, value: Settings[K]) => {
    setSettings((prev) => ({ ...prev, [key]: value }))
    setSaved(false)
  }

  const handleSave = async () => {
    setSaving(true)
    setError(null)
    setSaved(false)
    try {
      await updateSettings(settings)
      setSaved(true)
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to save settings")
    } finally {
      setSaving(false)
    }
  }

  const toggleRow = (
    key: keyof Settings,
    title: string,
    description: string,
  ) => (
    <div className="flex items-center justify-between gap-4 rounded-xl border border-border bg-card p-5">
      <div>
        <Label htmlFor={`setting-${key}`} className="text-sm font-semibold text-foreground">
          {title}
        </Label>
        <p className="text-xs text-muted-foreground mt-1">{description}</p>
      </div>
      <Switch
        id={`setting-${key}`}
        checked={Boolean(settings[key])}
        onCheckedChange={(v) => setField(key, v as Settings[typeof key])}
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
    <div className="flex items-center justify-between gap-4 rounded-xl border border-border bg-card p-5">
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
          onChange={(e) => setField(key, Number(e.target.value) as Settings[typeof key])}
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

        <Button
          size="sm"
          onClick={handleSave}
          disabled={saving || !loaded}
          className="bg-[#00D4AA] hover:bg-[#00BD98] text-[#1A1D23] font-semibold gap-2"
        >
          <Save className="h-4 w-4" />
          {saving ? "Saving..." : "Save Changes"}
        </Button>
      </header>

      <div className="flex flex-1 flex-col gap-6 py-6">
        <div>
          <h2 className="font-headline text-3xl font-bold tracking-tight">Settings</h2>
          <p className="text-muted-foreground mt-1 text-sm">
            Configure the DNS engine, caching, and data retention for this gateway.
          </p>
        </div>

        {error && (
          <div className="flex items-center gap-3 rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
            <AlertTriangle className="h-4 w-4 shrink-0" />
            {error}
          </div>
        )}

        {saved && (
          <div className="flex items-center gap-3 rounded-lg border border-green-500/40 bg-green-500/10 p-4 text-sm text-green-500">
            <CheckCircle2 className="h-4 w-4 shrink-0" />
            Settings saved.
          </div>
        )}

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
