import { DocsLayout } from "@/components/docs/DocsLayout";
import { Callout } from "@/components/docs/Callout";
import { CodeBlock } from "@/components/docs/CodeBlock";
import { findSection } from "../nav";

const section = findSection("mcp")!;

export default function Mcp() {
  return (
    <DocsLayout
      slug={section.slug}
      title={section.title}
      eyebrow={section.eyebrow}
      description={section.description}
      toc={[
        { id: "what", label: "What MCP gives you" },
        { id: "setup", label: "Claude Code setup" },
        { id: "tools", label: "Available tools" },
        { id: "prompts", label: "Example prompts" },
      ]}
    >
      <h2 id="what">What MCP gives you</h2>
      <p>
        MCP (Model Context Protocol) is a JSON-RPC 2.0 interface that lets an
        AI agent call tools on an external system. The <code>hydra</code>{" "}
        binary ships with an MCP server that exposes DNS firewall operations
        as tools. An assistant with MCP access can read your DNS state and
        mutate it in response to natural language. &quot;Block everything
        ByteDance owns.&quot; &quot;Why did <code>youtube.com</code> fail to
        load at 3pm?&quot; &quot;Add OISD to my blocklists and restart the
        engine.&quot; The agent picks the right tool calls for you.
      </p>
      <p>
        Why this matters for a DNS firewall: managing one is a lot of small,
        repetitive decisions. Which subdomain to exempt, which list to add,
        why a client got a refusal. MCP moves the friction from GUI and CLI
        to conversation, which is especially nice on mobile and for batch
        work. &quot;Block these 40 social media domains&quot; becomes one
        grouped policy instead of 40 round trips.
      </p>

      <h2 id="setup">Claude Code setup</h2>
      <p>
        Add HydraDNS to Claude Code&apos;s MCP config. The key pieces are the
        absolute path to your <code>hydra</code> binary and the same two env
        vars the CLI uses:
      </p>
      <CodeBlock language="json">{`{
  "mcpServers": {
    "hydradns": {
      "command": "/absolute/path/to/hydra",
      "args": ["mcp"],
      "env": {
        "HYDRA_API_URL": "http://localhost:8080",
        "HYDRA_TOKEN": "your-admin-token"
      }
    }
  }
}`}</CodeBlock>
      <p>
        Restart Claude Code. The server registers 9 tools that the agent can
        call on demand.
      </p>
      <Callout variant="tip">
        The exact same config block works with Gemini CLI. Drop it into
        Gemini&apos;s equivalent MCP config file and restart. Both clients
        speak plain JSON-RPC over stdio.
      </Callout>

      <h2 id="tools">Available tools</h2>
      <table>
        <thead>
          <tr>
            <th>Tool</th>
            <th>Input</th>
            <th>Effect</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>
              <code>get_status</code>
            </td>
            <td>none</td>
            <td>Engine state + query stats</td>
          </tr>
          <tr>
            <td>
              <code>toggle_engine</code>
            </td>
            <td>
              <code>{`{enabled: bool}`}</code>
            </td>
            <td>Turn the DNS engine on or off</td>
          </tr>
          <tr>
            <td>
              <code>block_domain</code>
            </td>
            <td>
              <code>{`{domain: string}`}</code>
            </td>
            <td>Create a BLOCK policy</td>
          </tr>
          <tr>
            <td>
              <code>unblock_domain</code>
            </td>
            <td>
              <code>{`{domain: string}`}</code>
            </td>
            <td>Delete matching BLOCK policies</td>
          </tr>
          <tr>
            <td>
              <code>create_policy</code>
            </td>
            <td>
              <code>{`{id, action, domains[], priority}`}</code>
            </td>
            <td>Create a grouped multi-domain policy in one call</td>
          </tr>
          <tr>
            <td>
              <code>list_policies</code>
            </td>
            <td>none</td>
            <td>All active policies</td>
          </tr>
          <tr>
            <td>
              <code>list_blocklists</code>
            </td>
            <td>none</td>
            <td>All blocklist sources + status</td>
          </tr>
          <tr>
            <td>
              <code>get_query_logs</code>
            </td>
            <td>
              <code>{`{limit?: number}`}</code>
            </td>
            <td>Recent DNS queries</td>
          </tr>
          <tr>
            <td>
              <code>get_metrics</code>
            </td>
            <td>none</td>
            <td>Latency percentiles + grade</td>
          </tr>
        </tbody>
      </table>
      <Callout variant="warning" title="Prefer create_policy for batches">
        When the user asks to block a group (&quot;block all social
        media&quot;), the agent should make{" "}
        <strong>one <code>create_policy</code> call</strong> with an array of
        domains, not N separate <code>block_domain</code> calls. The backend
        groups them into a single policy, the UI stays clean, and latency is
        much better. This is guidance from real runs, not a theoretical
        preference.
      </Callout>

      <h2 id="prompts">Example prompts</h2>
      <p>Things people have actually asked their agent:</p>
      <ul>
        <li>
          &quot;Block all Meta properties including WhatsApp and Instagram as
          a single policy called <code>block-meta</code>.&quot;
        </li>
        <li>
          &quot;Show me every domain that got refused in the last hour,
          grouped by client IP.&quot;
        </li>
        <li>
          &quot;My TV is trying to reach <code>samsungacr.com</code> what list
          is blocking it and can you make an exception just for my Plex
          server&apos;s IP?&quot;
        </li>
        <li>
          &quot;Add Steven Black, OISD Big, and URLhaus as blocklists and
          restart the engine.&quot;
        </li>
      </ul>
    </DocsLayout>
  );
}
