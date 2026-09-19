import logo from "@/assets/hydradns-logo.png";

const productLinks = [
  { label: "Features", href: "#features" },
  { label: "How It Works", href: "#how-it-works" },
  { label: "Comparison", href: "#comparison" },
  { label: "Changelog", href: "https://github.com/hydradns/hydradns/releases" },
];

const communityLinks = [
  { label: "GitHub", href: "https://github.com/hydradns/hydradns" },
  { label: "Documentation", href: "https://docs.hydradns.app" },
  { label: "Contributing", href: "https://github.com/hydradns/hydradns/blob/main/CONTRIBUTING.md" },
  { label: "Security", href: "https://github.com/hydradns/hydradns/security" },
  { label: "License (GPL-3.0)", href: "https://github.com/hydradns/hydradns/blob/main/LICENSE" },
];

export function Footer() {
  return (
    <footer className="bg-[hsl(var(--surface-container-lowest))] border-t border-outline-variant/20 py-16">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="grid md:grid-cols-3 gap-10 md:gap-8">
          {/* Brand */}
          <div className="space-y-4">
            <div className="flex items-center gap-3">
              <img src={logo} alt="HydraDNS" className="h-9 w-9 rounded-full" />
              <span className="font-headline text-[15px] font-bold tracking-tighter uppercase text-foreground">
                HydraDNS
              </span>
            </div>
            <p className="text-sm text-muted-foreground leading-relaxed max-w-xs">
              © 2026 HydraDNS. DNS security for people who want their network back.
            </p>
            <p className="font-mono text-[11px] uppercase tracking-widest text-muted-foreground/70">
              DNS-layer · Self-hosted · Zero telemetry
            </p>
          </div>

          {/* Product */}
          <FooterColumn title="Product" links={productLinks} />
          {/* Community */}
          <FooterColumn title="Community" links={communityLinks} />
        </div>

        <div className="mt-14 pt-6 border-t border-outline-variant/15 flex flex-col md:flex-row items-center justify-between gap-3">
          <p className="font-mono text-xs text-muted-foreground/70">
            GPL-3.0 · Built in the open
          </p>
          <p className="font-headline italic text-sm text-muted-foreground/70">
            Made with Go, gRPC, and paranoia.
          </p>
        </div>
      </div>
    </footer>
  );
}

function FooterColumn({
  title,
  links,
}: {
  title: string;
  links: { label: string; href: string }[];
}) {
  return (
    <div>
      <p className="eyebrow mb-4">{title}</p>
      <ul className="space-y-2.5">
        {links.map((link) => (
          <li key={link.href}>
            <a
              href={link.href}
              target={link.href.startsWith("http") ? "_blank" : undefined}
              rel={link.href.startsWith("http") ? "noopener noreferrer" : undefined}
              className="text-sm text-muted-foreground hover:text-brand-sky transition-colors"
            >
              {link.label}
            </a>
          </li>
        ))}
      </ul>
    </div>
  );
}
