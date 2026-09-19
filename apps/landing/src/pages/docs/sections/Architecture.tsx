import { DocsLayout } from "@/components/docs/DocsLayout";
import { Callout } from "@/components/docs/Callout";
import { CodeBlock } from "@/components/docs/CodeBlock";
import { Pill } from "@/components/docs/Pill";
import { findSection } from "../nav";

const section = findSection("architecture")!;

export default function Architecture() {
  return (
    <DocsLayout
      slug={section.slug}
      title={section.title}
      eyebrow={section.eyebrow}
      description={section.description}
      toc={[
        { id: "planes", label: "Control plane vs data plane" },
        { id: "pipeline", label: "Query pipeline" },
        { id: "storage", label: "Storage & state" },
        { id: "auth", label: "Authentication" },
        { id: "api-envelope", label: "API envelope" },
      ]}
    >
      <h2 id="planes">Control plane vs data plane</h2>
      <p>
        HydraDNS splits cleanly into two processes with different jobs. The{" "}
        <strong>control plane</strong> is the source of truth: a Gin REST API
        on <Pill variant="teal">:8080</Pill>, a SQLite database, bearer-token
        auth, and the admin models that back the dashboard and CLI. The{" "}
        <strong>data plane</strong> is the hot path: a UDP and TCP DNS
        listener on <Pill variant="sky">:1053</Pill> (Docker maps host{" "}
        <code>:53</code> to this), blocklist and policy evaluation, and
        upstream forwarding with failover.
      </p>
      <p>
        The two talk to each other over gRPC on{" "}
        <Pill variant="muted">:50051</Pill>. When you toggle the engine or
        edit a policy, the control plane first persists the new desired state
        to SQLite, then calls the data plane (for example{" "}
        <code>SetAcceptQueries</code>) to reconcile runtime with intent.
        Status endpoints return both sides stitched together: the DB intent
        and the live gRPC-reported state, so any drift between what you asked
        for and what is actually running is visible in the UI.
      </p>
      <Callout variant="note">
        gRPC between the two planes currently uses{" "}
        <code>grpc.WithInsecure()</code>. That&apos;s fine for single-host
        Docker where the channel never leaves the <code>hydra-net</code>
        bridge, but TLS on this link is planned before multi-host deployments
        ship.
      </Callout>

      <h2 id="pipeline">Query pipeline</h2>
      <p>
        Every DNS query runs through the same three steps. The pipeline
        short-circuits as soon as it has a verdict, so the common case stays
        cheap.
      </p>
      <ol>
        <li>
          <strong>Blocklist check.</strong> All loaded blocklist entries live
          in a single atomically rebuilt snapshot. A shared Bloom filter
          fronts the hashmap to give O(1) negative lookups on cold domains.
          On match, the data plane responds <code>REFUSED</code> immediately
          and stops.
        </li>
        <li>
          <strong>Policy evaluation.</strong> A second Bloom filter fronts
          the policy index, then an exact-domain match against the{" "}
          <code>PolicySnapshot</code>. If multiple policies match, priority
          wins (higher first), with lexicographic policy id as the
          tiebreaker. Supported actions:{" "}
          <Pill variant="teal">BLOCK</Pill>{" "}
          <Pill variant="sky">ALLOW</Pill>{" "}
          <Pill variant="amber">REDIRECT</Pill>.
        </li>
        <li>
          <strong>Upstream forward.</strong> One connection pool per
          configured upstream resolver, with failover across the list. Each
          upstream gets a 5 second timeout and 2 retries before the data
          plane moves to the next one. Latency and outcome of every
          resolution land in the <code>DNSQuery</code> table.
        </li>
      </ol>
      <p>
        Domains are normalized before every lookup: lowercased and stripped
        of the trailing dot, so <code>EXAMPLE.COM.</code> and{" "}
        <code>example.com</code> hit the same entry.
      </p>
      <Callout variant="tip">
        Both the policy snapshot and the blocklist snapshot are{" "}
        <strong>atomic</strong>. Updates build a fresh structure off to the
        side and the active pointer swaps via <code>atomic.Pointer</code>, so
        queries in flight see either the old world or the new world, never a
        half-merged one.
      </Callout>

      <h2 id="storage">Storage &amp; state</h2>
      <p>
        Persistence is SQLite via the pure-Go <code>glebarez/sqlite</code>
        driver, which keeps the binary static-buildable with no CGo toolchain
        involved. The database runs in WAL mode for concurrent readers, and
        the pool is pinned to <code>MaxOpenConns=1</code> at the connection
        layer to keep GORM&apos;s single-writer assumption happy.
      </p>
      <p>GORM auto-migrates the full model set on startup:</p>
      <ul>
        <li>
          <code>Policy</code>, <code>DomainPolicy</code>, <code>Action</code>,{" "}
          <code>Category</code> - the filtering rule set
        </li>
        <li>
          <code>BlocklistSource</code>, <code>BlocklistSnapshot</code>,{" "}
          <code>BlocklistEntry</code> - blocklist ingestion and cached state
        </li>
        <li>
          <code>DNSQuery</code>, <code>Statistics</code>,{" "}
          <code>SystemState</code> - observability and runtime metadata
        </li>
        <li>
          <code>AdminCredential</code> - the single-admin auth record
        </li>
      </ul>
      <p>
        SQLite is deliberate, not a placeholder for Postgres. HydraDNS is a
        single-node home appliance. One file, zero ops, trivial backups (copy
        the <code>.db</code>), and the write volume from an average home
        network never comes close to needing a server-class database.
      </p>
      <Callout variant="note">
        The <code>DNSQuery</code> table grows unbounded today. Rotation and a
        retention window are on the roadmap; until then, a periodic{" "}
        <code>DELETE</code> or a fresh DB file will keep size in check on
        long-running deployments.
      </Callout>

      <h2 id="auth">Authentication</h2>
      <p>
        HydraDNS uses a single admin user model: an{" "}
        <code>AdminCredential</code> singleton in the database. On first boot
        the dashboard detects no admin exists, routes you through the setup
        wizard, and calls <code>POST /api/v1/auth/setup</code>, which
        bcrypt-hashes the password and returns a UUID bearer token. Later
        logins hit <code>POST /api/v1/auth/login</code> to exchange the
        password for a token. The dashboard stores the token in{" "}
        <code>localStorage</code> and a cookie (the cookie is there so SSR
        and middleware can see it).
      </p>
      <p>
        Every API route outside the auth bootstrap requires a valid bearer
        token. The unauthenticated surface is intentionally tiny:
      </p>
      <table>
        <thead>
          <tr>
            <th>Endpoint</th>
            <th>Auth</th>
            <th>Purpose</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>
              <code>GET /health</code>
            </td>
            <td>none</td>
            <td>Liveness</td>
          </tr>
          <tr>
            <td>
              <code>GET /api/v1/auth/status</code>
            </td>
            <td>none</td>
            <td>
              Returns <code>{"{setup_complete: bool}"}</code>
            </td>
          </tr>
          <tr>
            <td>
              <code>POST /api/v1/auth/setup</code>
            </td>
            <td>none</td>
            <td>First-boot admin creation</td>
          </tr>
          <tr>
            <td>
              <code>POST /api/v1/auth/login</code>
            </td>
            <td>none</td>
            <td>Exchange password for token</td>
          </tr>
          <tr>
            <td>
              <code>everything else</code>
            </td>
            <td>Bearer token</td>
            <td>Policies, blocklists, metrics, logs</td>
          </tr>
        </tbody>
      </table>

      <h2 id="api-envelope">API envelope</h2>
      <p>
        Every control plane JSON response wraps in the same envelope. This is
        the contract the dashboard, the <code>hydra</code> CLI, and the MCP
        server all rely on, so it stays uniform across every handler.
      </p>
      <CodeBlock language="json">{`{
  "status": "success",
  "data": { "engine": { "running": true } },
  "error": null
}`}</CodeBlock>
      <p>
        Errors come back with <code>&quot;status&quot;: &quot;error&quot;</code>,{" "}
        <code>data</code> set to <code>null</code>, and a human-readable{" "}
        <code>error</code> string. Clients can branch on{" "}
        <code>status</code> alone and only touch <code>data</code> on the
        success path.
      </p>
      <p>
        See <a href="/docs/cli">CLI Reference</a> for how the envelope is
        consumed in practice, and{" "}
        <a href="/docs/policies-and-blocklists">Policies &amp; Blocklists</a>{" "}
        for the full query pipeline from a user-facing angle.
      </p>
    </DocsLayout>
  );
}
