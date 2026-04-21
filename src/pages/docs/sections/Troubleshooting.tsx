import { DocsLayout } from "@/components/docs/DocsLayout";
import { Callout } from "@/components/docs/Callout";
import { CodeBlock } from "@/components/docs/CodeBlock";
import { findSection } from "../nav";

const section = findSection("troubleshooting")!;

export default function Troubleshooting() {
  return (
    <DocsLayout
      slug={section.slug}
      title={section.title}
      eyebrow={section.eyebrow}
      description={section.description}
      toc={[
        { id: "port-53", label: "Port 53 on WSL / Linux" },
        { id: "router-fallback", label: "Router fallback DNS" },
        { id: "health", label: "Health checks" },
        { id: "logs", label: "Where the logs live" },
        { id: "submodules", label: "Submodule headaches" },
      ]}
    >
      <h2 id="port-53">Port 53 on WSL / Linux</h2>
      <p>
        This is the single biggest source of first-hour confusion, especially
        on Windows. On WSL2, binding host port 53 collides with{" "}
        <code>systemd-resolved</code> and the Windows DNS client, so{" "}
        <code>dig @localhost example.com</code> from the host will often
        fail even when HydraDNS is serving queries correctly inside the
        container. The container is fine. Host networking is lying to you.
      </p>
      <p>
        Skip the host port and talk to the container directly:
      </p>
      <CodeBlock language="bash">{`docker exec hydradns-core dig @127.0.0.1 -p 1053 example.com`}</CodeBlock>
      <p>
        On bare-metal Linux, binding port 53 needs{" "}
        <code>CAP_NET_BIND_SERVICE</code> or root. Docker handles that for
        you via the port mapping in <code>docker-compose.yml</code>, which is
        one of the reasons the shipped setup runs in a container by default.
      </p>
      <Callout variant="tip">
        <code>docker exec ... dig</code> is the most reliable smoke test on
        any platform because it bypasses host networking entirely. If that
        command returns an answer, your data plane is healthy, and whatever
        you&apos;re seeing from the host is a networking problem, not a
        HydraDNS problem.
      </Callout>

      <h2 id="router-fallback">Router fallback DNS</h2>
      <p>
        This is the #1 &quot;it&apos;s not blocking anything!&quot; report.
        Many routers have a secondary DNS field and helpfully suggest filling
        it with <code>1.1.1.1</code> or <code>8.8.8.8</code>. Client devices
        treat the secondary as a fallback and will silently prefer whichever
        resolver answers faster, which means the first time Cloudflare or
        Google wins the race, every blocked domain comes straight back.
      </p>
      <p>
        The fix is boring and mandatory: set <strong>primary DNS only</strong>{" "}
        to your HydraDNS box, and leave the secondary field empty.
      </p>
      <Callout variant="danger" title="Primary only">
        <strong>Leave the secondary DNS field blank.</strong> If HydraDNS is
        down, you want the network to fail loudly so you notice and fix it,
        not silently bypass every filter you configured. A fallback to
        Cloudflare or Google defeats the entire point of running a local
        resolver.
      </Callout>

      <h2 id="health">Health checks</h2>
      <p>
        When something feels off, work down this list in order. Each check
        isolates a different layer of the stack.
      </p>
      <table>
        <thead>
          <tr>
            <th>Check</th>
            <th>Command</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>Core is up</td>
            <td>
              <code>curl http://localhost:8080/health</code>
            </td>
          </tr>
          <tr>
            <td>Engine state</td>
            <td>
              <code>hydra status</code> or dashboard overview
            </td>
          </tr>
          <tr>
            <td>DNS actually resolving</td>
            <td>
              <code>
                docker exec hydradns-core dig @127.0.0.1 -p 1053 example.com
              </code>
            </td>
          </tr>
          <tr>
            <td>Dashboard reachable</td>
            <td>
              <code>curl http://localhost:3000</code>
            </td>
          </tr>
          <tr>
            <td>Container health</td>
            <td>
              <code>docker compose ps</code>
            </td>
          </tr>
        </tbody>
      </table>
      <p>
        Two bits of colour that help when reading the results.{" "}
        <code>/health</code> returns <code>200</code> as long as the process
        is alive and SQLite opens cleanly, so a green check there rules out
        total process failure but says nothing about DNS behaviour. The
        dashboard overview is where you catch intent-vs-reality drift if the
        gRPC link between the two planes has hiccuped: the engine reads as
        &quot;on&quot; in the DB but the data plane reports it as off, or
        vice versa.
      </p>

      <h2 id="logs">Where the logs live</h2>
      <p>
        Three places, depending on what you&apos;re chasing.
      </p>
      <ul>
        <li>
          <strong>Structured app logs.</strong>{" "}
          <code>docker compose logs -f core</code> (or <code>ui</code>,{" "}
          <code>landing</code>, <code>scanner</code>). Core emits JSON lines
          with <code>level</code>, <code>ts</code>, <code>msg</code>, and{" "}
          <code>component</code> fields, so piping into <code>jq</code> works
          well for filtering by subsystem.
        </li>
        <li>
          <strong>Per-query logs.</strong> The dashboard&apos;s Query Logs
          tab, the <code>hydra logs</code> CLI command, or the{" "}
          <code>DNSQuery</code> table in SQLite directly if you want to run
          ad-hoc SQL. Every resolution lands here with latency, outcome, and
          which rule (if any) matched.
        </li>
        <li>
          <strong>Blocklist fetch status.</strong> The dashboard&apos;s
          Blocklists tab shows last-fetched timestamp, ETag, and error per
          source. That&apos;s where you catch a list URL that started
          returning 404 or a source whose ETag has quietly stopped changing.
        </li>
      </ul>

      <h2 id="submodules">Submodule headaches</h2>
      <p>
        Each <code>apps/*</code> directory is its own Git repo. If someone
        cloned the parent without <code>--recursive</code>, those directories
        will be empty and Docker builds will fail with missing{" "}
        <code>Dockerfile</code> errors. Fix:
      </p>
      <CodeBlock language="bash">{`# from the repo root
git submodule update --init --recursive`}</CodeBlock>
      <p>
        If a submodule shows as dirty in the parent <code>git status</code>,
        it usually means you&apos;ve committed inside the submodule but
        haven&apos;t bumped the parent pointer to match. Confirm from the
        submodule:
      </p>
      <CodeBlock language="bash">{`cd apps/<service>
git status
git log --oneline -5`}</CodeBlock>
      <p>
        Then from the parent repo, stage the new submodule commit and commit
        it to pin the pointer:
      </p>
      <CodeBlock language="bash">{`git add apps/<service>
git commit -m "bump <service> submodule"`}</CodeBlock>
      <Callout variant="note">
        <code>make update</code> automates the common case: it runs{" "}
        <code>git submodule update --remote --merge</code> to pull the latest
        upstream commit on each submodule&apos;s tracked branch. Use it when
        you just want everything on current HEADs without manual cd-and-pull.
      </Callout>
      <p>
        Still stuck? The <a href="/docs/architecture">Architecture</a> page
        covers the control-plane / data-plane split if a symptom feels like
        drift between the two, and the{" "}
        <a href="/docs/installation">Installation</a> guide has the full
        Docker and Raspberry Pi bring-up in case something in the initial
        setup needs redoing.
      </p>
    </DocsLayout>
  );
}
