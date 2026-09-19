import { describe, expect, it } from "vitest"
import { render, screen } from "@testing-library/react"

import { HealthWidget, deriveHealthSignals, overallLevel } from "@/components/health-widget"
import type {
  Blocklist,
  BlocklistListData,
  DashboardSummary,
  DnsEngineStatus,
  DnsMetrics,
} from "@/lib/types"

// ---- fixtures ----

function makeBlocklist(overrides: Partial<Blocklist> = {}): Blocklist {
  return {
    id: "steven-black",
    name: "StevenBlack Hosts",
    url: "https://example.test/hosts",
    format: "hosts",
    category: "ads",
    domains_count: 120_000,
    enabled: true,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    ...overrides,
  }
}

const healthyEngine: DnsEngineStatus = {
  enabled: true,
  accepting_queries: true,
  last_error: "",
}

const healthyMetrics: DnsMetrics = {
  window_seconds: 60,
  queries: { total: 5000, errors: 2, error_rate: 0.0004 },
  latency_ms: { p50: 4, p95: 12, p99: 30 },
  grade: "good",
}

const healthyBlocklists: BlocklistListData = {
  total_blocklists: 1,
  total_domains: 120_000,
  active_lists: [makeBlocklist()],
}

const summary: DashboardSummary = {
  total_queries: 5000,
  blocked_queries: 342,
  allowed_queries: 4658,
  redirected_queries: 0,
  block_rate_percent: 6.84,
}

describe("HealthWidget (healthy state)", () => {
  it("renders green plain-language signals when everything is fine", () => {
    render(
      <HealthWidget
        engine={healthyEngine}
        metrics={healthyMetrics}
        blocklists={healthyBlocklists}
        summary={summary}
      />,
    )

    expect(screen.getByText("Filtering is on")).toBeInTheDocument()
    expect(screen.getByText("Lists are fresh")).toBeInTheDocument()
    expect(screen.getByText("Box is healthy")).toBeInTheDocument()
    expect(screen.getByText("342 threats blocked today")).toBeInTheDocument()
    // Overall headline pill
    expect(screen.getByText("All good")).toBeInTheDocument()
  })
})

describe("HealthWidget (unhealthy state)", () => {
  it("surfaces filtering-off, stale lists and a bad box, with an action-required headline", () => {
    const staleLists: BlocklistListData = {
      total_blocklists: 1,
      total_domains: 120_000,
      active_lists: [
        makeBlocklist({
          // 3 days old -> stale
          updated_at: new Date(Date.now() - 3 * 24 * 3_600_000).toISOString(),
        }),
      ],
    }

    render(
      <HealthWidget
        engine={{ enabled: false, accepting_queries: false, last_error: "" }}
        metrics={{ ...healthyMetrics, grade: "bad" }}
        blocklists={staleLists}
        summary={summary}
      />,
    )

    expect(screen.getByText("Filtering is off")).toBeInTheDocument()
    expect(screen.getByText("Lists need a refresh")).toBeInTheDocument()
    expect(screen.getByText("Box is struggling")).toBeInTheDocument()
    expect(screen.getByText("Action required")).toBeInTheDocument()
  })

  it("shows a pending headline while status is still loading", () => {
    render(
      <HealthWidget engine={null} metrics={null} blocklists={null} summary={null} />,
    )
    expect(screen.getByText("Checking…")).toBeInTheDocument()
  })
})

describe("deriveHealthSignals", () => {
  it("marks the overall level as the worst individual signal", () => {
    const signals = deriveHealthSignals({
      engine: { enabled: false, accepting_queries: false, last_error: "" },
      metrics: healthyMetrics,
      blocklists: healthyBlocklists,
      summary,
    })
    // filtering is down -> overall down
    expect(overallLevel(signals)).toBe("down")
  })

  it("is all-ok when every source is healthy", () => {
    const signals = deriveHealthSignals({
      engine: healthyEngine,
      metrics: healthyMetrics,
      blocklists: healthyBlocklists,
      summary,
    })
    expect(overallLevel(signals)).toBe("ok")
  })
})
