import { DocsLayout } from "@/components/docs/DocsLayout";
import { Callout } from "@/components/docs/Callout";
import { CodeBlock } from "@/components/docs/CodeBlock";
import { Steps, Step } from "@/components/docs/Steps";
import { findSection } from "../nav";

const section = findSection("getting-started")!;

export default function GettingStarted() {
  return (
    <DocsLayout
      slug={section.slug}
      title={section.title}
      eyebrow={section.eyebrow}
      description={section.description}
      toc={[
        { id: "what-is-hydradns", label: "What is HydraDNS?" },
        { id: "quick-start", label: "60-second quick start" },
        { id: "point-your-router", label: "Point your router" },
        { id: "whats-next", label: "What's next" },
      ]}
    >
      <h2 id="what-is-hydradns">What is HydraDNS?</h2>
      <p>
        Every app, website, and smart device starts by asking a DNS server
        &quot;where do I find this hostname?&quot; HydraDNS sits between your
        devices and the internet, answers the lookups you allow, and refuses
        the ones you don&apos;t. Ads, trackers, malware domains, adult
        content, whatever category you pick: the lookup fails, the connection
        never opens, the payload never loads.
      </p>
      <p>
        It&apos;s self-hosted and runs on your own hardware, so nothing about
        your network leaves your network. A Go data plane handles DNS on UDP
        and TCP, a Gin control plane exposes a REST API, and a Next.js
        dashboard lets you manage policies and blocklists from a browser.
        Point your router at the machine running HydraDNS and every device on
        your Wi-Fi gets the same protection, including the ones you can&apos;t
        install software on.
      </p>
      <p>
        If you&apos;ve used <strong>Pi-hole</strong>, the mental model is
        similar: local DNS sinkhole, one box for the whole LAN. Compared to{" "}
        <strong>NextDNS</strong>, HydraDNS keeps queries on-premises instead
        of routing them through a third-party resolver. You trade a bit of
        setup for full custody of your DNS traffic.
      </p>

      <h2 id="quick-start">60-second quick start</h2>
      <p>
        You need Docker and Git. That&apos;s it. The stack ships as a
        multi-repo monorepo wired together with submodules.
      </p>
      <Steps>
        <Step title="Clone with submodules">
          <p>
            The <code>--recursive</code> flag pulls every service at once:
          </p>
          <CodeBlock language="bash">{`git clone --recursive https://github.com/hydradns/hydradns.git
cd hydradns`}</CodeBlock>
        </Step>
        <Step title="Start the stack">
          <p>
            Brings up the DNS engine, control API, and dashboard on the{" "}
            <code>hydra-net</code> bridge network:
          </p>
          <CodeBlock language="bash">{`docker compose up -d`}</CodeBlock>
        </Step>
        <Step title="Open the dashboard">
          <p>
            Visit <code>http://localhost:3000</code>. The first-boot setup
            wizard creates your admin password and lets you pick a few
            starter blocklists (ads, trackers, malware).
          </p>
        </Step>
        <Step title="Verify DNS is working">
          <p>
            From another shell, send a query straight into the container:
          </p>
          <CodeBlock language="bash">{`docker exec hydradns-core dig @127.0.0.1 -p 1053 example.com`}</CodeBlock>
          <p>
            You should see an A record in the answer section. If you do,
            resolution is live.
          </p>
        </Step>
      </Steps>
      <Callout variant="warning" title="WSL2 users">
        On WSL2, binding host port 53 is flaky because of systemd-resolved
        and the Windows DNS client. The <code>docker exec</code> smoke test
        above talks to the container directly on port 1053 and sidesteps the
        whole mess. Use it as your ground truth while developing.
      </Callout>

      <h2 id="point-your-router">Point your router at HydraDNS</h2>
      <p>
        To protect every device on your network, set your router&apos;s DNS
        server to the IP of the machine running HydraDNS. Phones, TVs, game
        consoles, smart bulbs: they all pick up router-assigned DNS through
        DHCP, so this one change filters everything without touching
        individual devices.
      </p>
      <p>
        The setting is usually under <strong>WAN</strong>,{" "}
        <strong>Internet</strong>, or <strong>LAN / DHCP</strong> in your
        router admin page. Replace the ISP-provided DNS with your HydraDNS
        box&apos;s LAN IP (something like <code>192.168.1.50</code>), save,
        and reboot the router or renew DHCP leases.
      </p>
      <Callout variant="danger" title="Do not set a secondary DNS">
        Many routers have a second DNS field and helpfully suggest filling
        it with <code>1.1.1.1</code> or <code>8.8.8.8</code>. Don&apos;t.
        Client devices treat secondary DNS as a fallback and will silently
        prefer the faster responder, which means every blocked domain comes
        back the moment Cloudflare or Google answers first. Set the primary
        to HydraDNS and <strong>leave the secondary blank</strong>. If
        HydraDNS goes down, you want the network to fail loudly, not bypass
        your filtering.
      </Callout>

      <h2 id="whats-next">What&apos;s next</h2>
      <p>Three directions from here, pick whichever fits your style:</p>
      <ul>
        <li>
          <a href="/docs/dashboard">Dashboard Guide</a> - walk through every
          screen: engine toggle, blocklists, policies, query logs, metrics.
        </li>
        <li>
          <a href="/docs/policies-and-blocklists">Policies &amp; Blocklists</a>{" "}
          - how the 3-step pipeline decides block vs allow, and how to write
          your own rules.
        </li>
        <li>
          <a href="/docs/cli">CLI Reference</a> - the{" "}
          <code>hydra</code> binary for driving the control plane from the
          terminal.
        </li>
      </ul>
    </DocsLayout>
  );
}
