import { useScrollAnimation } from "@/hooks/useScrollAnimation";
import {
  Shield,
  Lock,
  Zap,
  BarChart3,
  ListFilter,
  FileText,
  Container,
  Plug,
  ArrowRight,
} from "lucide-react";

const features = [
  {
    icon: Shield,
    title: "DNS-Layer Blocking",
    desc: "Blocks malware, phishing, ads, and trackers at the DNS level. Threats are stopped before they ever reach your devices.",
    highlight: false,
  },
  {
    icon: Zap,
    title: "Sub-Millisecond Decisions",
    desc: "Bloom filter–accelerated policy engine with lock-free reads. 99.99% of clean queries bypass the filter with near-zero overhead.",
    highlight: true,
  },
  {
    icon: Lock,
    title: "Privacy First",
    desc: "Client IPs are anonymized with HMAC-SHA256 before they touch disk. Your browsing data stays yours. Always.",
    highlight: false,
  },
  {
    icon: BarChart3,
    title: "Real-Time Dashboard",
    desc: "See total queries, block rates, top domains, category breakdowns, and 24-hour trends. One API call powers the whole view.",
    highlight: false,
  },
  {
    icon: ListFilter,
    title: "Flexible Policy Engine",
    desc: "Create rules with exact domains, regex patterns, and priorities. Allow, block, redirect, or log. Hot-reload without downtime.",
    highlight: false,
  },
  {
    icon: FileText,
    title: "Multiple Blocklist Sources",
    desc: "Sync from popular blocklists via URL with ETag-based caching. Supports hosts files, domain lists, and adblock formats.",
    highlight: false,
  },
  {
    icon: Container,
    title: "One Command Deploy",
    desc: "docker-compose up and you're protected. Runs on a Raspberry Pi, a NAS, or a cloud VM. Multi-arch, non-root containers.",
    highlight: false,
  },
  {
    icon: Plug,
    title: "Full REST API",
    desc: "17 endpoints for dashboard, DNS engine, policies, blocklists, and analytics. Build your own integrations or automate everything.",
    highlight: false,
  },
];

export function FeaturesSection() {
  const ref = useScrollAnimation();

  return (
    <section id="features" className="relative py-28 lg:py-36" ref={ref}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center mb-16">
          <p className="eyebrow fade-in-up">FEATURES</p>
          <h2 className="mt-4 text-4xl sm:text-5xl lg:text-6xl font-bold text-foreground fade-in-up leading-[1.1]">
            Everything you need.
            <br />
            <span className="text-gradient">Nothing you don't.</span>
          </h2>
        </div>

        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-5">
          {features.map((f, i) => (
            <FeatureCard key={i} feature={f} delay={i * 50} />
          ))}
          <ViewAllCard delay={features.length * 50} />
        </div>
      </div>
    </section>
  );
}

function FeatureCard({
  feature,
  delay,
}: {
  feature: (typeof features)[number];
  delay: number;
}) {
  const { icon: Icon, title, desc, highlight } = feature;
  const base =
    "group relative rounded-xl bg-surface-container p-7 border transition-all duration-300 fade-in-up";
  const state = highlight
    ? "border-brand-teal/30 glow-primary"
    : "border-outline-variant/25 hover:border-brand-teal/40 hover:bg-surface-container-high/80";

  return (
    <div className={`${base} ${state}`} style={{ transitionDelay: `${delay}ms` }}>
      {highlight && (
        <span className="absolute top-5 right-5 px-2 py-0.5 rounded-full bg-brand-teal/15 border border-brand-teal/30 font-mono text-[10px] tracking-wider text-brand-teal">
          FAST PATH
        </span>
      )}
      <div className="mb-5 inline-flex h-11 w-11 items-center justify-center rounded-lg bg-brand-teal/10 border border-brand-teal/20">
        <Icon className="h-5 w-5 text-brand-teal" strokeWidth={2} />
      </div>
      <h3 className="font-headline text-lg font-bold text-foreground mb-2">{title}</h3>
      <p className="text-sm text-muted-foreground leading-relaxed">{desc}</p>
    </div>
  );
}

function ViewAllCard({ delay }: { delay: number }) {
  return (
    <a
      href="https://docs.hydradns.app"
      target="_blank"
      rel="noopener noreferrer"
      className="group relative rounded-xl border border-dashed border-outline-variant/40 bg-surface-container-low/40 p-7 flex flex-col items-center justify-center text-center transition-all hover:border-brand-teal/40 hover:bg-surface-container/60 fade-in-up"
      style={{ transitionDelay: `${delay}ms` }}
    >
      <div className="mb-3 inline-flex h-11 w-11 items-center justify-center rounded-lg bg-surface-container-high/60 border border-outline-variant/30 group-hover:border-brand-teal/40 transition-colors">
        <ArrowRight className="h-5 w-5 text-muted-foreground group-hover:text-brand-teal transition-colors" />
      </div>
      <p className="font-mono text-sm text-muted-foreground group-hover:text-brand-teal transition-colors">
        View all features →
      </p>
      <p className="mt-1 text-xs text-muted-foreground/60">Docs · API · CLI · MCP</p>
    </a>
  );
}
