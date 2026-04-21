import { DocsLayout } from "@/components/docs/DocsLayout";
import { Callout } from "@/components/docs/Callout";
import { CodeBlock } from "@/components/docs/CodeBlock";
import { findSection } from "../nav";

const section = findSection("installation")!;

export default function Installation() {
  return (
    <DocsLayout
      slug={section.slug}
      title={section.title}
      eyebrow={section.eyebrow}
      description={section.description}
      toc={[
        { id: "docker-compose", label: "Docker Compose" },
        { id: "raspberry-pi", label: "Raspberry Pi" },
        { id: "updating", label: "Updating" },
      ]}
    >
      <h2 id="docker-compose">Docker Compose (recommended)</h2>
      <p>
        The default path. A <code>docker-compose.yml</code> at the repo root
        orchestrates four services on a bridge network called{" "}
        <code>hydra-net</code>:
      </p>
      <ul>
        <li>
          <strong>core</strong> - the Go DNS engine (data plane) plus the
          Gin control API.
        </li>
        <li>
          <strong>ui</strong> - the Next.js dashboard on port{" "}
          <code>3000</code>.
        </li>
        <li>
          <strong>landing</strong> - the marketing and docs site on port{" "}
          <code>3001</code>.
        </li>
        <li>
          <strong>scanner</strong> - a background worker that probes the
          host&apos;s DNS config.
        </li>
      </ul>
      <p>The three commands you&apos;ll use most:</p>
      <CodeBlock language="bash">{`docker compose up -d          # start everything
docker compose logs -f core   # tail DNS engine logs
docker compose down           # stop`}</CodeBlock>
      <p>
        <strong>Port mapping:</strong> Docker binds host{" "}
        <code>:53</code> to the container&apos;s <code>:1053</code> for DNS
        (UDP and TCP), and exposes <code>:8080</code> for the control API and{" "}
        <code>:3000</code> for the dashboard. gRPC between control plane and
        data plane stays internal on the bridge network.
      </p>
      <Callout variant="note" title="Data persistence">
        SQLite lives in a named Docker volume, so your policies, blocklists,
        snapshots, and admin credentials survive{" "}
        <code>docker compose down</code> and container rebuilds. Only{" "}
        <code>docker compose down -v</code> (volume flag) will wipe state.
      </Callout>

      <h2 id="raspberry-pi">Raspberry Pi / always-on server</h2>
      <p>
        HydraDNS was designed to live on a Pi in the corner of the room. Flash
        Raspberry Pi OS Lite (64-bit), SSH in, and run the one-line
        installer:
      </p>
      <CodeBlock language="bash">{`curl -fsSL https://raw.githubusercontent.com/hydradns/hydradns/main/scripts/install.sh | bash`}</CodeBlock>
      <p>
        The script installs Docker if it&apos;s missing, clones the repo with
        submodules, writes a sensible <code>.env</code>, and brings the stack
        up. When it finishes, grab the Pi&apos;s LAN IP, point your
        router&apos;s primary DNS at it, and you&apos;re protecting the whole
        network.
      </p>
      <Callout variant="tip" title="Hardware sizing">
        A Pi 4 with 2GB of RAM is the practical floor; a Pi 5 gives you the
        best query latency and keeps 1ms cache hits realistic even under
        load. The blocklist engine holds several million entries comfortably
        in around <strong>200MB</strong> of RAM thanks to the Bloom filter
        and interned string table, so you&apos;ve got plenty of headroom for
        the rest of the stack.
      </Callout>

      <h2 id="updating">Updating</h2>
      <p>
        Each app under <code>apps/</code> is its own Git repository pulled in
        as a submodule. The root repo pins each submodule to a specific
        commit, so updates are a deliberate two-step: bump the submodules,
        then rebuild whatever changed.
      </p>
      <p>From the repo root:</p>
      <CodeBlock language="bash">{`make update        # git submodule update --remote --merge
make build-core    # rebuild core if its submodule moved
make restart-core  # swap the running container`}</CodeBlock>
      <p>
        The individual <code>build-*</code> and <code>restart-*</code>{" "}
        targets exist for every service: <code>core</code>, <code>ui</code>,{" "}
        <code>landing</code>, <code>scanner</code>. For a full-stack refresh
        with the published Docker images instead of local builds:
      </p>
      <CodeBlock language="bash">{`docker compose pull && docker compose up -d`}</CodeBlock>
      <Callout variant="note" title="Schema migrations are automatic">
        Running <code>make update</code> between versions is safe. GORM
        auto-migrates every model on control plane startup, so policies,
        blocklist snapshots, and admin credentials carry forward without
        manual SQL. If a migration ever needs intervention, release notes
        will call it out explicitly.
      </Callout>
    </DocsLayout>
  );
}
