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
        { id: "static-ip", label: "Static IP" },
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

      <h2 id="static-ip">Give the device a static IP</h2>
      <p>
        Your router forwards every DNS query on the network to the IP of the
        machine running HydraDNS. If that machine gets its address from DHCP,
        the router can hand it a different IP after a reboot or lease
        renewal, and DNS for the entire network silently breaks. Pin the IP{" "}
        <strong>before</strong> pointing the router at it.
      </p>
      <p>
        The cleanest fix is a <strong>DHCP reservation</strong> on the
        router: bind the machine&apos;s MAC address to a fixed IP under the
        router&apos;s DHCP / LAN settings (called{" "}
        <em>Address Reservation</em>, <em>Static Lease</em>, or{" "}
        <em>DHCP Binding</em> depending on the brand). It works the same
        regardless of OS and survives reinstalls. Find the MAC address with{" "}
        <code>ip link</code> (Linux), <code>ipconfig /all</code> (Windows),
        or System Settings &gt; Network (macOS).
      </p>
      <p>
        If your router doesn&apos;t support reservations, set the address on
        the device itself. Pick an IP outside the router&apos;s DHCP pool so
        it never gets assigned to another device.
      </p>
      <h3>Linux / Raspberry Pi OS</h3>
      <p>
        Raspberry Pi OS Bookworm and most modern distros use NetworkManager
        (list connection names with <code>nmcli con show</code>):
      </p>
      <CodeBlock language="bash">{`sudo nmcli con mod "Wired connection 1" \\
  ipv4.method manual \\
  ipv4.addresses 192.168.1.53/24 \\
  ipv4.gateway 192.168.1.1 \\
  ipv4.dns 1.1.1.1
sudo nmcli con up "Wired connection 1"`}</CodeBlock>
      <p>
        Older Raspberry Pi OS (Bullseye and earlier) uses{" "}
        <code>dhcpcd</code> instead: append the block below to{" "}
        <code>/etc/dhcpcd.conf</code> and reboot.
      </p>
      <CodeBlock language="bash">{`interface eth0
static ip_address=192.168.1.53/24
static routers=192.168.1.1
static domain_name_servers=1.1.1.1`}</CodeBlock>
      <h3>macOS</h3>
      <p>
        <strong>System Settings</strong> &gt; <strong>Network</strong> &gt;
        your connection &gt; <strong>Details</strong> &gt;{" "}
        <strong>TCP/IP</strong>, set <strong>Configure IPv4</strong> to{" "}
        <strong>Manually</strong>, then enter the IP, subnet mask{" "}
        <code>255.255.255.0</code>, and your router&apos;s IP as the gateway.
        Or from Terminal (service names via{" "}
        <code>networksetup -listallnetworkservices</code>):
      </p>
      <CodeBlock language="bash">{`sudo networksetup -setmanual "Ethernet" 192.168.1.53 255.255.255.0 192.168.1.1
sudo networksetup -setdnsservers "Ethernet" 1.1.1.1`}</CodeBlock>
      <h3>Windows</h3>
      <p>
        <strong>Settings</strong> &gt; <strong>Network &amp; Internet</strong>{" "}
        &gt; <strong>Ethernet</strong> (or your Wi-Fi network) &gt;{" "}
        <strong>IP assignment</strong> &gt; <strong>Edit</strong> &gt;{" "}
        <strong>Manual</strong>, turn on IPv4, and fill in the IP, prefix
        length <code>24</code>, gateway, and DNS. Or in an elevated
        PowerShell (adapter names via <code>Get-NetAdapter</code>):
      </p>
      <CodeBlock language="powershell">{`New-NetIPAddress -InterfaceAlias "Ethernet" -IPAddress 192.168.1.53 -PrefixLength 24 -DefaultGateway 192.168.1.1
Set-DnsClientServerAddress -InterfaceAlias "Ethernet" -ServerAddresses 1.1.1.1`}</CodeBlock>
      <Callout variant="tip" title="Point the device's own DNS elsewhere">
        Set the HydraDNS machine&apos;s own DNS server to a public resolver
        like <code>1.1.1.1</code>, not to itself. HydraDNS needs working DNS
        to download blocklists even while its container is restarting.
      </Callout>
      <p>
        Verify from another device on the network before touching the
        router: <code>ping</code> the new address and confirm the dashboard
        loads at <code>http://&lt;static-ip&gt;:3000</code>.
      </p>

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
