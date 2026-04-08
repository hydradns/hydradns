"use client"

import { IconShieldCheck, IconShieldX, IconActivity, IconPercentage } from "@tabler/icons-react"

import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import type { DashboardSummary } from "@/lib/types"

export function SectionCards({ data }: { data: DashboardSummary | null }) {
  const total = data?.total_queries ?? 0
  const blocked = data?.blocked_queries ?? 0
  const allowed = data?.allowed_queries ?? 0
  const rate = data?.block_rate_percent ?? 0

  return (
    <div className="*:data-[slot=card]:from-primary/5 *:data-[slot=card]:to-card dark:*:data-[slot=card]:bg-card grid grid-cols-1 gap-4 *:data-[slot=card]:bg-gradient-to-t *:data-[slot=card]:shadow-xs @xl/main:grid-cols-2 @5xl/main:grid-cols-4">
      <Card className="@container/card">
        <CardHeader>
          <CardTitle>Total Queries</CardTitle>
          <IconActivity className="size-5 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold tabular-nums @[250px]/card:text-3xl">
            {total.toLocaleString()}
          </div>
        </CardContent>
        <CardFooter className="text-sm text-muted-foreground">
          All DNS queries processed
        </CardFooter>
      </Card>

      <Card className="@container/card">
        <CardHeader>
          <CardTitle>Blocked</CardTitle>
          <IconShieldX className="size-5 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold tabular-nums @[250px]/card:text-3xl">
            {blocked.toLocaleString()}
          </div>
        </CardContent>
        <CardFooter className="text-sm text-muted-foreground">
          <Badge variant="destructive" className="text-xs">Blocked</Badge>
          <span className="ml-2">queries refused</span>
        </CardFooter>
      </Card>

      <Card className="@container/card">
        <CardHeader>
          <CardTitle>Allowed</CardTitle>
          <IconShieldCheck className="size-5 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold tabular-nums @[250px]/card:text-3xl">
            {allowed.toLocaleString()}
          </div>
        </CardContent>
        <CardFooter className="text-sm text-muted-foreground">
          Forwarded to upstream
        </CardFooter>
      </Card>

      <Card className="@container/card">
        <CardHeader>
          <CardTitle>Block Rate</CardTitle>
          <IconPercentage className="size-5 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold tabular-nums @[250px]/card:text-3xl">
            {rate.toFixed(1)}%
          </div>
        </CardContent>
        <CardFooter className="text-sm text-muted-foreground">
          Of total queries blocked
        </CardFooter>
      </Card>
    </div>
  )
}
