"use client"

import { useEffect, useState } from "react"
import {
  Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList,
  BreadcrumbPage, BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import { Separator } from "@/components/ui/separator"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { Badge } from "@/components/ui/badge"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { getResolvers } from "@/lib/api"
import type { Resolver } from "@/lib/types"
import { Info, Server } from "lucide-react"

// Resolvers are read-only here: the control plane has no create/update/
// delete route for /dns/resolvers (see lib/api.ts). They're configured
// via configs/config.yaml's upstream_resolvers list, not the dashboard.
export default function ResolversPage() {
  const [resolvers, setResolvers] = useState<Resolver[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loaded, setLoaded] = useState(false)

  useEffect(() => {
    getResolvers()
      .then((r) => { setResolvers(r); setError(null) })
      .catch((e) => setError(e.message))
      .finally(() => setLoaded(true))
  }, [])

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
                  <BreadcrumbPage>Resolvers</BreadcrumbPage>
                </BreadcrumbItem>
              </BreadcrumbList>
            </Breadcrumb>
          </div>
        </div>
      </header>

      <div className="flex flex-1 flex-col gap-6 py-6">
        <div>
          <h2 className="font-headline text-3xl font-bold tracking-tight">Resolvers</h2>
          <p className="text-muted-foreground mt-1 text-sm">
            The upstream DNS resolvers HydraDNS forwards allowed queries to.
          </p>
        </div>

        <div className="flex items-start gap-3 rounded-lg border border-border bg-card p-4 text-sm text-muted-foreground">
          <Info className="h-4 w-4 mt-0.5 shrink-0 text-[#00D4AA]" />
          <p>
            Resolvers are configured in <code className="font-mono text-xs">configs/config.yaml</code>
            {" "}(<code className="font-mono text-xs">dataplane.upstream_resolvers</code>) and restart-applied.
            There&apos;s no add/edit/delete here yet.
          </p>
        </div>

        {error && (
          <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
            {error}
          </div>
        )}

        <div className="overflow-hidden rounded-xl border border-border bg-card">
          <div className="flex items-center gap-2 border-b border-border px-6 py-4">
            <Server className="h-5 w-5 text-[#00D4AA]" />
            <h3 className="font-headline font-bold text-foreground">Upstream Resolvers</h3>
            <span className="ml-auto text-xs text-muted-foreground">
              {resolvers.length} configured
            </span>
          </div>
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead className="px-6 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">Name</TableHead>
                  <TableHead className="px-6 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">Address</TableHead>
                  <TableHead className="px-6 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">Protocol</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {resolvers.length > 0 ? (
                  resolvers.map((r) => (
                    <TableRow key={r.id} className="hover:bg-background/30">
                      <TableCell className="px-6 py-4 text-sm font-medium text-foreground">
                        {r.name || r.address}
                      </TableCell>
                      <TableCell className="px-6 py-4 font-mono text-xs text-muted-foreground">
                        {r.address}
                      </TableCell>
                      <TableCell className="px-6 py-4">
                        <Badge variant="outline" className="text-[10px] font-bold uppercase">
                          {r.protocol}
                        </Badge>
                      </TableCell>
                    </TableRow>
                  ))
                ) : (
                  <TableRow>
                    <TableCell colSpan={3} className="py-10 text-center text-sm text-muted-foreground">
                      {loaded ? "No resolvers configured" : "Loading resolvers..."}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </div>
      </div>
    </>
  )
}
