"use client"

import { Shield, ShieldOff, Activity, AlertTriangle } from "lucide-react"
import type { DashboardSummary } from "@/lib/types"

interface SectionCardsProps {
  data: DashboardSummary | null
  threats?: number
}

export function SectionCards({ data, threats = 0 }: SectionCardsProps) {
  const total = data?.total_queries ?? 0
  const blocked = data?.blocked_queries ?? 0
  const allowed = data?.allowed_queries ?? 0
  const rate = data?.block_rate_percent ?? 0
  const allowRate = total > 0 ? ((allowed / total) * 100) : 0

  return (
    <div className="grid grid-cols-1 gap-4 @xl/main:grid-cols-2 @5xl/main:grid-cols-4">
      {/* Total Queries */}
      <div className="bg-card p-6 rounded-xl border-l-4 border-[#00D4AA] shadow-lg shadow-black/20">
        <div className="flex flex-col">
          <div className="flex items-center justify-between mb-1">
            <span className="text-muted-foreground text-xs font-headline uppercase tracking-widest">
              Total Queries
            </span>
            <Activity className="size-4 text-[#00D4AA]/60" />
          </div>
          <div className="flex items-baseline gap-2">
            <span className="text-3xl font-headline font-bold text-foreground leading-tight">
              {total.toLocaleString()}
            </span>
          </div>
          <div className="mt-4 flex items-center gap-2">
            <div className="flex-1 h-8 flex items-end gap-0.5">
              {[40, 60, 30, 80, 55, 45, 90, 70].map((h, i) => (
                <div
                  key={i}
                  className="w-full bg-[#00D4AA]/20 rounded-t-sm"
                  style={{ height: `${h}%` }}
                />
              ))}
            </div>
            <span className="text-[10px] text-muted-foreground font-medium whitespace-nowrap">
              Last 24h
            </span>
          </div>
        </div>
      </div>

      {/* Blocked */}
      <div className="bg-card p-6 rounded-xl border-l-4 border-red-500 shadow-lg shadow-black/20">
        <div className="flex flex-col">
          <div className="flex items-center justify-between mb-1">
            <span className="text-muted-foreground text-xs font-headline uppercase tracking-widest">
              Blocked
            </span>
            <ShieldOff className="size-4 text-red-500/60" />
          </div>
          <div className="flex items-baseline gap-2">
            <span className="text-3xl font-headline font-bold text-foreground leading-tight">
              {blocked.toLocaleString()}
            </span>
            <span className="text-red-500 text-sm font-semibold">
              {rate.toFixed(1)}%
            </span>
          </div>
          <div className="mt-4 h-1 w-full bg-slate-700 rounded-full overflow-hidden">
            <div
              className="h-full bg-red-500 transition-all duration-500"
              style={{ width: `${Math.min(rate, 100)}%` }}
            />
          </div>
          <span className="text-[10px] text-muted-foreground font-medium mt-2">
            Security filters active
          </span>
        </div>
      </div>

      {/* Allowed */}
      <div className="bg-card p-6 rounded-xl border-l-4 border-green-500 shadow-lg shadow-black/20">
        <div className="flex flex-col">
          <div className="flex items-center justify-between mb-1">
            <span className="text-muted-foreground text-xs font-headline uppercase tracking-widest">
              Allowed
            </span>
            <Shield className="size-4 text-green-500/60" />
          </div>
          <div className="flex items-baseline gap-2">
            <span className="text-3xl font-headline font-bold text-foreground leading-tight">
              {allowed.toLocaleString()}
            </span>
            <span className="text-green-500 text-sm font-semibold">
              {allowRate.toFixed(1)}%
            </span>
          </div>
          <div className="mt-4 h-1 w-full bg-slate-700 rounded-full overflow-hidden">
            <div
              className="h-full bg-green-500 transition-all duration-500"
              style={{ width: `${Math.min(allowRate, 100)}%` }}
            />
          </div>
          <span className="text-[10px] text-muted-foreground font-medium mt-2">
            Legitimate traffic
          </span>
        </div>
      </div>

      {/* Threats Detected */}
      <div className="bg-card p-6 rounded-xl border-l-4 border-amber-500 shadow-lg shadow-black/20">
        <div className="flex flex-col">
          <div className="flex items-center justify-between mb-1">
            <span className="text-muted-foreground text-xs font-headline uppercase tracking-widest">
              Threats Detected
            </span>
            <AlertTriangle className="size-4 text-amber-500/60" />
          </div>
          <div className="flex items-baseline gap-2">
            <span className="text-3xl font-headline font-bold text-foreground leading-tight">
              {threats.toLocaleString()}
            </span>
          </div>
          <div className="mt-4 flex items-center gap-2">
            <AlertTriangle className="size-5 text-amber-500" />
            <span className="text-[11px] text-amber-500/80 font-semibold uppercase tracking-wider">
              Suspicious domains
            </span>
          </div>
        </div>
      </div>
    </div>
  )
}
