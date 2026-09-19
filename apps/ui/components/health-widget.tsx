"use client"

import {
  ShieldCheck,
  ShieldOff,
  ListChecks,
  HardDrive,
  ShieldAlert,
  type LucideIcon,
} from "lucide-react"
import type {
  DnsEngineStatus,
  DnsMetrics,
  BlocklistListData,
  DashboardSummary,
} from "@/lib/types"

// ---- health model ----
//
// The health widget speaks plain language for a non-technical owner. Every
// signal collapses raw status/metrics data into a single traffic-light level.

export type HealthLevel = "ok" | "warn" | "down" | "pending"

export interface HealthSignal {
  key: string
  title: string
  detail: string
  level: HealthLevel
}

export interface HealthWidgetData {
  engine: DnsEngineStatus | null
  metrics: DnsMetrics | null
  blocklists: BlocklistListData | null
  summary: DashboardSummary | null
}

// Enabled lists older than this are treated as stale (auto-refresh runs every 6h).
const FRESH_LIST_MAX_HOURS = 24

function hoursSince(iso: string): number {
  const t = new Date(iso).getTime()
  if (Number.isNaN(t)) return Infinity
  return (Date.now() - t) / 3_600_000
}

/** Is filtering actually on the wire? */
export function deriveFilteringSignal(engine: DnsEngineStatus | null): HealthSignal {
  if (!engine) {
    return {
      key: "filtering",
      title: "Filtering",
      detail: "Checking status…",
      level: "pending",
    }
  }
  const on = engine.enabled && engine.accepting_queries
  return {
    key: "filtering",
    title: on ? "Filtering is on" : "Filtering is off",
    detail: on
      ? "Every device on your network is protected."
      : engine.last_error || "Your network is not being filtered right now.",
    level: on ? "ok" : "down",
  }
}

/** Are the blocklists fresh enough to catch new threats? */
export function deriveListsSignal(blocklists: BlocklistListData | null): HealthSignal {
  if (!blocklists) {
    return {
      key: "lists",
      title: "Blocklists",
      detail: "Checking status…",
      level: "pending",
    }
  }
  const enabled = (blocklists.active_lists ?? []).filter((b) => b.enabled)
  if (enabled.length === 0) {
    return {
      key: "lists",
      title: "No filter lists",
      detail: "Add a blocklist so HydraDNS knows what to block.",
      level: "warn",
    }
  }
  const stale = enabled.filter(
    (b) => !b.updated_at || hoursSince(b.updated_at) > FRESH_LIST_MAX_HOURS,
  )
  if (stale.length > 0) {
    return {
      key: "lists",
      title: "Lists need a refresh",
      detail: `${stale.length} of ${enabled.length} lists are more than a day old.`,
      level: "warn",
    }
  }
  return {
    key: "lists",
    title: "Lists are fresh",
    detail: `${enabled.length} blocklist${enabled.length !== 1 ? "s" : ""} updated in the last 24 hours.`,
    level: "ok",
  }
}

/** Is the appliance itself healthy (resolving quickly, no errors)? */
export function deriveBoxSignal(
  engine: DnsEngineStatus | null,
  metrics: DnsMetrics | null,
): HealthSignal {
  if (engine?.last_error) {
    return {
      key: "box",
      title: "Box needs attention",
      detail: engine.last_error,
      level: "down",
    }
  }
  if (!metrics) {
    return {
      key: "box",
      title: "Box",
      detail: "Checking status…",
      level: "pending",
    }
  }
  switch (metrics.grade) {
    case "excellent":
    case "good":
      return {
        key: "box",
        title: "Box is healthy",
        detail: `Answering in about ${metrics.latency_ms.p95}ms.`,
        level: "ok",
      }
    case "degraded":
      return {
        key: "box",
        title: "Box is running slow",
        detail: `Responses are lagging (${metrics.latency_ms.p95}ms). Keep an eye on it.`,
        level: "warn",
      }
    case "bad":
      return {
        key: "box",
        title: "Box is struggling",
        detail: "The appliance is slow or dropping queries.",
        level: "down",
      }
    default:
      return {
        key: "box",
        title: "Box",
        detail: "Not enough traffic to judge yet.",
        level: "pending",
      }
  }
}

/** How much bad traffic has been stopped. Reassuring, always green. */
export function deriveThreatsSignal(summary: DashboardSummary | null): HealthSignal {
  if (!summary) {
    return {
      key: "threats",
      title: "Threats blocked",
      detail: "Checking status…",
      level: "pending",
    }
  }
  const blocked = summary.blocked_queries
  return {
    key: "threats",
    title: `${blocked.toLocaleString()} threats blocked today`,
    detail:
      blocked > 0
        ? "HydraDNS is actively stopping bad domains."
        : "No threats yet — all clear.",
    level: "ok",
  }
}

export function deriveHealthSignals(data: HealthWidgetData): HealthSignal[] {
  return [
    deriveFilteringSignal(data.engine),
    deriveListsSignal(data.blocklists),
    deriveBoxSignal(data.engine, data.metrics),
    deriveThreatsSignal(data.summary),
  ]
}

/** Worst level across all signals, used for the overall headline. */
export function overallLevel(signals: HealthSignal[]): HealthLevel {
  if (signals.some((s) => s.level === "down")) return "down"
  if (signals.some((s) => s.level === "warn")) return "warn"
  if (signals.some((s) => s.level === "pending")) return "pending"
  return "ok"
}

// ---- presentation ----

const LEVEL_ICON: Record<string, LucideIcon> = {
  filtering: ShieldCheck,
  lists: ListChecks,
  box: HardDrive,
  threats: ShieldAlert,
}

function levelStyles(level: HealthLevel) {
  switch (level) {
    case "ok":
      return { dot: "bg-hydra-green", text: "text-hydra-green", tint: "bg-hydra-green/10 border-hydra-green/20" }
    case "warn":
      return { dot: "bg-hydra-amber", text: "text-hydra-amber", tint: "bg-hydra-amber/10 border-hydra-amber/20" }
    case "down":
      return { dot: "bg-hydra-red", text: "text-hydra-red", tint: "bg-hydra-red/10 border-hydra-red/20" }
    default:
      return { dot: "bg-muted-foreground/40", text: "text-muted-foreground", tint: "bg-muted/30 border-border" }
  }
}

const OVERALL_COPY: Record<HealthLevel, { label: string; help: string }> = {
  ok: { label: "All good", help: "Your network is protected." },
  warn: { label: "Needs attention", help: "Everything still works, but check the items below." },
  down: { label: "Action required", help: "Protection is affected — see the red items below." },
  pending: { label: "Checking…", help: "Reading live status from the box." },
}

export function HealthWidget({ engine, metrics, blocklists, summary }: HealthWidgetData) {
  const signals = deriveHealthSignals({ engine, metrics, blocklists, summary })
  const overall = overallLevel(signals)
  const overallStyle = levelStyles(overall)
  const copy = OVERALL_COPY[overall]

  return (
    <section
      aria-label="System health"
      className="bg-card rounded-xl border border-border/20 overflow-hidden"
    >
      <div className="px-6 py-5 border-b border-border/20 flex items-center justify-between gap-4">
        <div>
          <h3 className="font-headline font-bold text-lg text-foreground">System Health</h3>
          <p className="text-xs text-muted-foreground mt-0.5">{copy.help}</p>
        </div>
        <span
          className={`inline-flex items-center gap-2 rounded-full border px-3 py-1 text-[11px] font-bold uppercase tracking-wider ${overallStyle.tint} ${overallStyle.text}`}
        >
          <span
            className={`h-2 w-2 rounded-full ${overallStyle.dot} ${overall === "ok" ? "animate-pulse" : ""}`}
          />
          {copy.label}
        </span>
      </div>

      <ul className="grid grid-cols-1 sm:grid-cols-2 gap-px bg-border/20">
        {signals.map((signal) => {
          const style = levelStyles(signal.level)
          const Icon = LEVEL_ICON[signal.key] ?? (signal.level === "down" ? ShieldOff : ShieldCheck)
          return (
            <li key={signal.key} className="bg-card p-5 flex items-start gap-3">
              <span
                className={`mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border ${style.tint}`}
              >
                <Icon className={`h-4 w-4 ${style.text}`} />
              </span>
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <span className={`h-2 w-2 rounded-full shrink-0 ${style.dot}`} aria-hidden />
                  <span className="font-semibold text-sm text-foreground">{signal.title}</span>
                </div>
                <p className="text-xs text-muted-foreground mt-1 leading-relaxed">{signal.detail}</p>
              </div>
            </li>
          )
        })}
      </ul>
    </section>
  )
}
