"use client"

import { Lock, ShieldOff, ShieldCheck, UserX } from "lucide-react"
import type { BypassAttempt, BypassAttemptsData, BypassProtocol } from "@/lib/types"

const PROTOCOL_LABEL: Record<BypassProtocol, string> = {
  doh: "DoH",
  dot: "DoT",
  doq: "DoQ",
}

const PROTOCOL_TITLE: Record<BypassProtocol, string> = {
  doh: "DNS over HTTPS",
  dot: "DNS over TLS",
  doq: "DNS over QUIC",
}

function protocolLabel(p: BypassProtocol): string {
  return PROTOCOL_LABEL[p] ?? p.toUpperCase()
}

function formatTime(ts: string): string {
  const d = new Date(ts)
  if (Number.isNaN(d.getTime())) return ts
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
}

function StatusBadge({ blocked }: { blocked: boolean }) {
  if (blocked) {
    return (
      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-bold uppercase bg-hydra-green/10 text-hydra-green border border-hydra-green/20">
        <ShieldCheck className="h-3 w-3" />
        Blocked
      </span>
    )
  }
  return (
    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-bold uppercase bg-hydra-red/10 text-hydra-red border border-hydra-red/20">
      <ShieldOff className="h-3 w-3" />
      Allowed
    </span>
  )
}

export function BypassPanel({ data }: { data: BypassAttemptsData | null }) {
  const attempts: BypassAttempt[] = data?.attempts ?? []
  const totalAttempts = data?.total_attempts ?? 0
  const uniqueClients = data?.unique_clients ?? 0

  return (
    <section
      aria-label="Encrypted-DNS bypass attempts"
      className="bg-card rounded-xl border border-border/20 overflow-hidden"
    >
      <div className="px-6 py-5 border-b border-border/20 flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-hydra-amber/10 border border-hydra-amber/20 text-hydra-amber">
            <Lock className="h-4 w-4" />
          </span>
          <div>
            <h3 className="font-headline font-bold text-lg text-foreground">
              Encrypted-DNS Bypass Attempts
            </h3>
            <p className="text-xs text-muted-foreground mt-0.5">
              Who tried to evade filtering with DoH / DoT / DoQ
            </p>
          </div>
        </div>
        <div className="flex items-center gap-5 text-right">
          <div>
            <div className="font-headline text-xl font-bold text-foreground tabular-nums">
              {totalAttempts.toLocaleString()}
            </div>
            <div className="text-[10px] uppercase tracking-wider text-muted-foreground font-bold">
              Attempts
            </div>
          </div>
          <div>
            <div className="font-headline text-xl font-bold text-foreground tabular-nums">
              {uniqueClients.toLocaleString()}
            </div>
            <div className="text-[10px] uppercase tracking-wider text-muted-foreground font-bold">
              Clients
            </div>
          </div>
        </div>
      </div>

      {attempts.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-2 px-6 py-10 text-center">
          <span className="flex h-10 w-10 items-center justify-center rounded-full bg-hydra-green/10 text-hydra-green">
            <ShieldCheck className="h-5 w-5" />
          </span>
          <p className="text-sm font-semibold text-foreground">No bypass attempts</p>
          <p className="text-xs text-muted-foreground max-w-xs">
            No client has tried to reach an encrypted-DNS resolver directly.
          </p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-left">
            <thead>
              <tr className="bg-muted/30 text-[10px] uppercase tracking-widest text-muted-foreground font-bold">
                <th className="px-6 py-4">Client</th>
                <th className="px-6 py-4">Protocol</th>
                <th className="px-6 py-4">Target Resolver</th>
                <th className="px-6 py-4 text-right">Attempts</th>
                <th className="px-6 py-4">Last Seen</th>
                <th className="px-6 py-4">Result</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border/10 text-sm">
              {attempts.map((a, i) => (
                <tr
                  key={`${a.client_ip}-${a.protocol}-${i}`}
                  className="hover:bg-muted/10 transition-colors"
                >
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-2">
                      <UserX className="h-4 w-4 text-muted-foreground shrink-0" />
                      <div className="min-w-0">
                        {a.client_name && (
                          <div className="font-medium text-foreground truncate">{a.client_name}</div>
                        )}
                        <div className="font-mono text-xs text-muted-foreground">{a.client_ip}</div>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    <span
                      title={PROTOCOL_TITLE[a.protocol] ?? a.protocol}
                      className="inline-block px-2 py-0.5 rounded text-[10px] font-bold uppercase bg-hydra-blue/10 text-hydra-blue border border-hydra-blue/20"
                    >
                      {protocolLabel(a.protocol)}
                    </span>
                  </td>
                  <td className="px-6 py-4 font-mono text-xs text-slate-300 truncate max-w-[220px]">
                    {a.target}
                  </td>
                  <td className="px-6 py-4 text-right tabular-nums text-foreground font-semibold">
                    {a.attempts.toLocaleString()}
                  </td>
                  <td className="px-6 py-4 text-muted-foreground font-mono text-xs whitespace-nowrap">
                    {formatTime(a.last_attempt)}
                  </td>
                  <td className="px-6 py-4">
                    <StatusBadge blocked={a.blocked} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  )
}
