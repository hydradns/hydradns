import { DocsLayout } from "@/components/docs/DocsLayout";
import { Callout } from "@/components/docs/Callout";
import { CodeBlock } from "@/components/docs/CodeBlock";
import { findSection } from "../nav";

const section = findSection("cli")!;

export default function Cli() {
  return (
    <DocsLayout
      slug={section.slug}
      title={section.title}
      eyebrow={section.eyebrow}
      description={section.description}
      toc={[
        { id: "install", label: "Install + auth" },
        { id: "reference", label: "Command reference" },
        { id: "scripting", label: "Scripting tips" },
        { id: "mcp", label: "Conversational control" },
      ]}
    >
      <h2 id="install">Install and authenticate</h2>
      <p>
        <code>hydra</code> is a Go + Cobra binary that wraps the control plane
        REST API. Build it from the monorepo:
      </p>
      <CodeBlock language="bash">{`cd apps/cli && go build -o hydra .
./hydra --help`}</CodeBlock>
      <p>
        Two environment variables tell it where to connect and how to
        authenticate:
      </p>
      <ul>
        <li>
          <code>HYDRA_API_URL</code> (default <code>http://localhost:8080</code>)
          where the control plane is reachable.
        </li>
        <li>
          <code>HYDRA_TOKEN</code> the bearer token printed by the first-boot
          setup wizard. You can also re-copy it from the dashboard profile
          page.
        </li>
      </ul>
      <CodeBlock language="bash">{`export HYDRA_API_URL=http://localhost:8080
export HYDRA_TOKEN=abc123...
hydra status`}</CodeBlock>
      <Callout variant="tip">
        You can ship <code>hydra</code> to a laptop and point{" "}
        <code>HYDRA_API_URL</code> at your Pi&apos;s LAN IP on port 8080 to
        drive the box remotely. Same binary, same commands, no SSH required.
      </Callout>

      <h2 id="reference">Command reference</h2>
      <table>
        <thead>
          <tr>
            <th>Command</th>
            <th>What it does</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>
              <code>hydra status</code>
            </td>
            <td>Desired + actual engine state and query stats</td>
          </tr>
          <tr>
            <td>
              <code>hydra engine enable</code> / <code>disable</code>
            </td>
            <td>Master switch for DNS blocking</td>
          </tr>
          <tr>
            <td>
              <code>hydra block &lt;domain&gt;</code>
            </td>
            <td>Create a BLOCK policy for a single domain</td>
          </tr>
          <tr>
            <td>
              <code>hydra policies</code> /{" "}
              <code>policies delete &lt;id&gt;</code>
            </td>
            <td>List or remove policies</td>
          </tr>
          <tr>
            <td>
              <code>hydra blocklists</code>
            </td>
            <td>List loaded blocklist sources</td>
          </tr>
          <tr>
            <td>
              <code>hydra blocklists add --id X --name &quot;Y&quot; --url Z</code>
            </td>
            <td>Add a new source</td>
          </tr>
          <tr>
            <td>
              <code>hydra logs</code>
            </td>
            <td>Tail recent DNS queries</td>
          </tr>
          <tr>
            <td>
              <code>hydra metrics</code>
            </td>
            <td>Latency percentiles and performance grade</td>
          </tr>
          <tr>
            <td>
              <code>hydra mcp</code>
            </td>
            <td>Start the MCP JSON-RPC server on stdio</td>
          </tr>
        </tbody>
      </table>
      <p>A typical end-to-end workflow looks like this:</p>
      <CodeBlock language="bash">{`hydra status                               # confirm engine is on
hydra block ads.facebook.com               # one-off block
hydra blocklists add \\
  --id stevenblack \\
  --name "Steven Black unified" \\
  --url https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts
hydra metrics                              # verify latency didn't regress`}</CodeBlock>

      <h2 id="scripting">Scripting tips</h2>
      <ul>
        <li>
          Every command exits non-zero on error and prints JSON on stdout
          where applicable, so you can pipe straight into <code>jq</code>{" "}
          without parsing text.
        </li>
        <li>
          <code>HYDRA_API_URL</code> + <code>HYDRA_TOKEN</code> plus the CLI
          make it easy to put HydraDNS behaviors in a cronjob, systemd timer,
          or Ansible playbook. Everything the dashboard does is reachable from
          a shell.
        </li>
        <li>
          For quick diagnostics, filter flags on <code>hydra logs</code> (for
          example a <code>--since 10m</code> window) are documented in{" "}
          <code>--help</code>. Always check the live help output for the
          current flag set.
        </li>
      </ul>

      <h2 id="mcp">Next: conversational control</h2>
      <p>
        If you&apos;d rather say &quot;block all social media&quot; in English
        to an agent than wire up shell scripts, jump to{" "}
        <a href="/docs/mcp">MCP</a>. Same operations, no flags to memorize.
      </p>
    </DocsLayout>
  );
}
