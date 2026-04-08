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
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Input } from "@/components/ui/input"
import { getQueryLogs } from "@/lib/api"
import type { QueryLogEntry } from "@/lib/types"

const actionBadge: Record<string, "default" | "destructive" | "secondary" | "outline"> = {
  allow: "default",
  block: "destructive",
  redirect: "secondary",
  flagged: "outline",
}

export default function LogsPage() {
  const [logs, setLogs] = useState<QueryLogEntry[]>([])
  const [filter, setFilter] = useState("")
  const [error, setError] = useState<string | null>(null)

  const fetchData = () => {
    getQueryLogs().then((d) => { setLogs(d); setError(null) }).catch((e) => setError(e.message))
  }

  useEffect(() => {
    fetchData()
    const interval = setInterval(fetchData, 10000)
    return () => clearInterval(interval)
  }, [])

  const filtered = filter
    ? logs.filter(
        (l) =>
          l.domain.toLowerCase().includes(filter.toLowerCase()) ||
          l.client_ip.includes(filter) ||
          l.action.includes(filter.toLowerCase())
      )
    : logs

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
                  <BreadcrumbPage>Query Logs</BreadcrumbPage>
                </BreadcrumbItem>
              </BreadcrumbList>
            </Breadcrumb>
          </div>
        </div>
        <Button variant="outline" size="sm" onClick={fetchData}>
          Refresh
        </Button>
      </header>

      <div className="flex flex-1 flex-col gap-4 py-4 md:gap-6 md:py-6">
        {error && (
          <div className="rounded-md border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
            {error}
          </div>
        )}
        <div className="flex items-center gap-4">
          <Input
            placeholder="Filter by domain, IP, or action..."
            className="max-w-sm"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
          />
          <Badge variant="outline">{filtered.length} entries</Badge>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Recent DNS Queries</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Domain</TableHead>
                  <TableHead>Client IP</TableHead>
                  <TableHead>Action</TableHead>
                  <TableHead>Threat</TableHead>
                  <TableHead>Time</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filtered.length > 0 ? (
                  filtered.map((log) => (
                    <TableRow key={log.id} className={log.is_suspicious ? "bg-destructive/5" : ""}>
                      <TableCell className="font-mono text-sm">{log.domain}</TableCell>
                      <TableCell className="font-mono text-sm text-muted-foreground">{log.client_ip}</TableCell>
                      <TableCell>
                        <Badge variant={actionBadge[log.action] || "outline"}>
                          {log.action}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        {log.is_suspicious ? (
                          <span className="text-xs" title={log.threat_reason || ""}>
                            <Badge variant="destructive" className="text-xs">
                              {Math.round(log.threat_score * 100)}%
                            </Badge>
                            <span className="ml-1 text-muted-foreground">
                              {log.detection_method}
                            </span>
                          </span>
                        ) : (
                          <span className="text-xs text-muted-foreground">—</span>
                        )}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {new Date(log.timestamp).toLocaleString()}
                      </TableCell>
                    </TableRow>
                  ))
                ) : (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center text-muted-foreground py-8">
                      {filter ? "No matching entries" : "No query logs yet"}
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
