import { DocsLayout } from "@/components/docs/DocsLayout";
import { Callout } from "@/components/docs/Callout";
import { Screenshot } from "@/components/docs/Screenshot";
import { Pill } from "@/components/docs/Pill";
import { findSection } from "../nav";
import overviewImg from "@/assets/dashboard/overview.png";
import policiesImg from "@/assets/dashboard/policies.png";
import queryLogsImg from "@/assets/dashboard/query-logs.png";
import blocklistsImg from "@/assets/dashboard/blocklists.png";
import metricsImg from "@/assets/dashboard/metrics.png";

const section = findSection("dashboard")!;

export default function Dashboard() {
  return (
    <DocsLayout
      slug={section.slug}
      title={section.title}
      eyebrow={section.eyebrow}
      description={section.description}
      toc={[
        { id: "tour", label: "Tour" },
        { id: "engine-toggle", label: "Engine toggle" },
        { id: "blocklists", label: "Blocklists" },
        { id: "policies", label: "Policies" },
        { id: "query-logs", label: "Query logs" },
        { id: "metrics", label: "Metrics" },
      ]}
    >
      <h2 id="tour">Tour</h2>
      <p>
        The dashboard at <code>http://localhost:3000</code> is where you see
        what&apos;s happening and change how HydraDNS behaves. It talks to the
        control plane API on <code>:8080</code>.
      </p>
      <Screenshot
        src={overviewImg}
        alt="HydraDNS dashboard overview tab showing query totals, block rate, engine status, top blocked domains, and recent activity"
        caption="HydraDNS dashboard · overview tab"
      />
      <p>
        The overview leads with total queries, the blocked-vs-allowed split,
        and a headline block-rate percentage. Underneath, the engine status
        row tells you whether the data plane is actively filtering, and the
        right-hand panels surface the top blocked domains and a live tail of
        recent activity. Everything on this page is read-only, the controls
        live in the tabs below.
      </p>

      <h2 id="engine-toggle">Engine toggle</h2>
      <p>
        The engine switch at the top of the dashboard is the master kill
        switch. When it&apos;s <Pill variant="teal">ON</Pill>, every query
        runs through the full pipeline: blocklist, policies, upstream. When
        it&apos;s <Pill variant="muted">OFF</Pill>, queries still flow to
        your upstream resolvers and resolve normally, but no blocking
        happens. That&apos;s the first thing to try when a site breaks: flip
        the engine off, retry, and you instantly know whether HydraDNS was
        the cause.
      </p>
      <p>
        Toggling the switch writes desired state to SQLite, then fires a
        gRPC <code>SetAcceptQueries</code> call to the data plane so the
        runtime matches the database.
      </p>
      <Callout variant="tip" title="Intent vs reality">
        The status row always shows both the state you asked for and the
        state the data plane reports. If they disagree, something upstream
        is wedged, usually a data plane that hasn&apos;t reconnected to
        gRPC yet. The split makes drift visible instead of silent.
      </Callout>

      <h2 id="blocklists">Blocklists</h2>
      <p>
        The blocklists tab is where you wire in community-maintained threat
        and ad lists. Add a source by pasting a hosts-format URL, Steven
        Black&apos;s unified list is the go-to starter, and HydraDNS
        fetches, parses, and snapshots it in a single database transaction.
        After that, sources auto-refresh every six hours by default,
        tunable with the <code>BLOCKLIST_UPDATE_INTERVAL</code> environment
        variable.
      </p>
      <Screenshot
        src={blocklistsImg}
        alt="HydraDNS blocklists tab listing four active sources: Steven Black unified, OISD Big, URLhaus Malware, AdGuard Tracking, each with format and status"
        caption="Blocklists tab · curated community feeds wired in"
      />
      <p>
        Each row in the list shows the source name, URL, parsed domain
        count, last-updated timestamp, and status:{" "}
        <Pill variant="teal">active</Pill> if the last fetch succeeded,{" "}
        <Pill variant="amber">failed</Pill> otherwise. Three formats are
        supported out of the box: <code>hosts</code>,{" "}
        <code>domain-list</code>, and <code>ads-list</code>, picked by the{" "}
        <code>format</code> field when you add the source. See{" "}
        <a href="/docs/policies-and-blocklists">
          policies and blocklists
        </a>{" "}
        for the full reference.
      </p>

      <h2 id="policies">Policies</h2>
      <p>
        The policies tab is where you add your own rules on top of
        blocklists. Blocklists handle the long tail of known-bad domains,
        policies handle the handful of rules that are specific to your
        network.
      </p>
      <Screenshot
        src={policiesImg}
        alt="HydraDNS policies tab listing user-defined rules with action, domains, and priority"
        caption="Policies tab · user-defined rules layered over blocklists"
      />
      <p>
        Each policy has an <code>id</code>, an action (
        <Pill variant="amber">BLOCK</Pill>,{" "}
        <Pill variant="teal">ALLOW</Pill>,{" "}
        <Pill variant="sky">REDIRECT</Pill>), a list of domains, and a
        priority. Higher priority wins on ties, and{" "}
        <strong>ALLOW overrides BLOCK</strong> at the same priority. The
        classic use case is whitelisting a specific subdomain of an
        advertising network that a work tool actually needs to function,
        without tearing the whole network out of your blocklist. For the
        deep dive on action semantics, regex support, and priority
        ordering, see{" "}
        <a href="/docs/policies-and-blocklists">
          policies and blocklists
        </a>
        .
      </p>

      <h2 id="query-logs">Query logs</h2>
      <p>
        The query logs tab is the live firehose of every DNS request the
        data plane has seen, newest first. It&apos;s the first place to
        look when a site doesn&apos;t load.
      </p>
      <Screenshot
        src={queryLogsImg}
        alt="HydraDNS query log table with timestamp, client IP, domain, record type, action, upstream, and latency"
        caption="Live query log · searchable and filterable"
      />
      <p>
        Each row carries the timestamp, client IP, queried domain, record
        type (<code>A</code>, <code>AAAA</code>, <code>CNAME</code>, and
        friends), the action taken (
        <Pill variant="teal">allowed</Pill>,{" "}
        <Pill variant="amber">blocked</Pill>,{" "}
        <Pill variant="muted">refused</Pill>), which upstream resolver
        answered, and the end-to-end latency in milliseconds. The search
        box filters against any field. When a site misbehaves, type the
        domain in, and you immediately see whether HydraDNS refused it,
        which policy matched, or whether the upstream itself was slow.
      </p>
      <Callout variant="warning" title="Log retention">
        The <code>DNSQuery</code> table grows unbounded today. For a home
        setup that&apos;s fine for months, but for a long-running
        production deployment you&apos;ll want a rotation job until
        built-in retention ships.
      </Callout>

      <h2 id="metrics">Metrics</h2>
      <Screenshot
        src={metricsImg}
        alt="HydraDNS DNS Engine tab showing engine status, p50 p95 p99 latency, query totals, error rate, and configured upstream resolvers"
        caption="DNS Engine tab · engine state, latency percentiles, upstream resolvers"
      />
      <p>
        The DNS Engine tab turns the query stream into performance numbers.
        You get latency percentiles at <code>p50</code>, <code>p95</code>,
        and <code>p99</code>, total queries and error rate over a 300s
        rolling window, and the list of configured upstream resolvers so
        you can see at a glance which ones the data plane is talking to.
      </p>
      <p>
        A headline <strong>performance grade</strong> from{" "}
        <Pill variant="teal">good</Pill> through <Pill variant="amber">bad</Pill>{" "}
        summarises the whole picture, calibrated to what a typical home
        network should feel like. The most useful moment to check metrics
        is right after adding a new blocklist, the grade will tell you
        whether the extra filtering tanked your latency or passed
        unnoticed.
      </p>
    </DocsLayout>
  );
}
