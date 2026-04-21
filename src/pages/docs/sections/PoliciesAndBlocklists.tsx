import { DocsLayout } from "@/components/docs/DocsLayout";
import { Callout } from "@/components/docs/Callout";
import { CodeBlock } from "@/components/docs/CodeBlock";
import { Pill } from "@/components/docs/Pill";
import { findSection } from "../nav";

const section = findSection("policies-and-blocklists")!;

export default function PoliciesAndBlocklists() {
  return (
    <DocsLayout
      slug={section.slug}
      title={section.title}
      eyebrow={section.eyebrow}
      description={section.description}
      toc={[
        { id: "pipeline", label: "The 3-step pipeline" },
        { id: "policies", label: "Writing policies" },
        { id: "blocklists", label: "Adding blocklists" },
        { id: "which-to-use", label: "Which should I use?" },
      ]}
    >
      <h2 id="pipeline">The 3-step query pipeline</h2>
      <p>
        Every DNS query that hits HydraDNS runs through the same three stages.
        Each stage can short-circuit the request, so a decision is made as
        early as possible and the query never spends cycles it doesn&apos;t
        need to.
      </p>
      <ol>
        <li>
          <strong>Blocklist check.</strong>{" "}
          <Pill variant="amber">Hardest block</Pill> If the domain appears in
          any loaded blocklist snapshot, the engine responds{" "}
          <code>REFUSED</code> immediately and exits. No policy evaluation, no
          upstream call.
        </li>
        <li>
          <strong>Policy evaluation.</strong>{" "}
          <Pill variant="sky">O(1) negative lookup</Pill> A Bloom filter does a
          fast negative check. On a possible match, the engine performs an
          exact domain comparison against the atomic{" "}
          <code>PolicySnapshot</code>. If multiple policies match, priority
          wins; ties break lexicographically by policy ID.
        </li>
        <li>
          <strong>Upstream forward.</strong>{" "}
          <Pill variant="teal">With failover</Pill> Pool-per-resolver with
          failover across every configured upstream. Each resolver gets a 5s
          timeout and 2 retries before the next one takes over.
        </li>
      </ol>
      <Callout variant="note">
        Domain names are normalized before any lookup: lowercased and the
        trailing dot is stripped. <code>EXAMPLE.COM.</code> becomes{" "}
        <code>example.com</code> so your rules match regardless of how the
        client phrased the question.
      </Callout>

      <h2 id="policies">Writing your own policies</h2>
      <p>
        Policies live in <code>configs/policies.json</code> as an array of
        objects. Each policy has the following shape:
      </p>
      <ul>
        <li>
          <code>id</code> <Pill variant="muted">string</Pill> unique identifier
        </li>
        <li>
          <code>action</code> one of <code>BLOCK</code>, <code>ALLOW</code>, or{" "}
          <code>REDIRECT</code>
        </li>
        <li>
          <code>domains</code> array of exact hostnames to match
        </li>
        <li>
          <code>regexes</code> <Pill variant="muted">optional</Pill> array of
          regex patterns
        </li>
        <li>
          <code>priority</code> integer, higher wins on conflict
        </li>
      </ul>
      <p>
        Because priority-and-lex ordering runs before blocklist refusal can
        override them, an <code>ALLOW</code> policy reliably beats a{" "}
        <code>BLOCK</code> from a list. That&apos;s the classic whitelist use
        case: let a broad blocklist cover Facebook while a narrow{" "}
        <code>ALLOW</code> policy keeps <code>graph.facebook.com</code>{" "}
        reachable for a work integration.
      </p>
      <CodeBlock language="json" title="configs/policies.json">{`[
  {
    "id": "block-social",
    "action": "BLOCK",
    "priority": 50,
    "domains": ["facebook.com", "instagram.com", "twitter.com", "x.com"]
  },
  {
    "id": "allow-work-facebook",
    "action": "ALLOW",
    "priority": 100,
    "domains": ["graph.facebook.com"]
  }
]`}</CodeBlock>
      <Callout variant="warning" title="Regex and wildcards are parsed, not evaluated">
        The engine compiles and validates <code>regexes</code> and wildcard
        patterns on load, so a bad pattern fails fast. It does{" "}
        <strong>not</strong> yet evaluate them at query time. Stick to exact
        hostnames in <code>domains</code> for anything you need enforced
        today.
      </Callout>

      <h2 id="blocklists">Adding blocklist sources</h2>
      <p>
        Blocklists do the heavy lifting for common categories. Instead of
        hand-writing 40,000 ad domains, you point HydraDNS at a curated feed
        and it keeps itself current.
      </p>
      <p>
        Sources are fetched with <code>ETag</code> support, so a{" "}
        <code>304 Not Modified</code> skips the download entirely. Every
        payload gets a SHA-256 checksum for integrity, and the whole snapshot
        (entries plus metadata) is persisted inside a single database
        transaction. You never see a half-loaded list.
      </p>
      <p>
        Three parsers ship out of the box. The <code>format</code> field on
        the <code>BlocklistSource</code> picks which one runs:
      </p>
      <ul>
        <li>
          <code>hosts</code> classic <code>/etc/hosts</code> format
        </li>
        <li>
          <code>domain-list</code> one domain per line
        </li>
        <li>
          <code>ads-list</code> AdBlock-style rules
        </li>
      </ul>
      <p>A few recommended starters:</p>
      <table>
        <thead>
          <tr>
            <th>List</th>
            <th>URL</th>
            <th>Covers</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>Steven Black unified</td>
            <td>
              <code>
                https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts
              </code>
            </td>
            <td>Ads, malware, trackers</td>
          </tr>
          <tr>
            <td>OISD Big</td>
            <td>
              <code>https://big.oisd.nl/</code>
            </td>
            <td>High-quality general purpose</td>
          </tr>
          <tr>
            <td>URLhaus</td>
            <td>
              <code>https://urlhaus.abuse.ch/downloads/hostfile/</code>
            </td>
            <td>Active malware distribution</td>
          </tr>
        </tbody>
      </table>
      <p>
        Refresh cadence defaults to 6 hours. Override it with the{" "}
        <code>BLOCKLIST_UPDATE_INTERVAL</code> env var (accepts Go duration
        strings like <code>1h</code>, <code>30m</code>, <code>12h</code>).
      </p>

      <h2 id="which-to-use">Which should I use?</h2>
      <p>Two tools, two jobs. Pick by intent:</p>
      <ul>
        <li>
          <strong>Use a blocklist</strong> when you want broad category
          coverage maintained by someone else. Millions of ad domains, curated
          malware feeds, tracker lists. You subscribe, they update, you
          benefit.
        </li>
        <li>
          <strong>Use a policy</strong> when you want a specific behavior on a
          specific domain: whitelist an exception a list got wrong,
          force-block a domain no list caught, or redirect a hostname to a
          local server.
        </li>
      </ul>
      <p>
        Both shapes are easy to automate. See the{" "}
        <a href="/docs/cli">CLI Reference</a> for scripting from the terminal,
        or the <a href="/docs/mcp">MCP</a> guide for agent-driven management
        through Claude Code or Gemini.
      </p>
    </DocsLayout>
  );
}
