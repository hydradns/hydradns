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
    title: "Block at the DNS Layer",
    desc: "Malware, phishing, ads, and trackers get dropped before a connection is ever made. Nothing to install on the client.",
    highlight: false,
  },
  {
    icon: Zap,
    title: "Sub-Millisecond Decisions",
    desc: "A Bloom filter sits in front of the policy engine, so most clean queries skip the lookup entirely and go straight to the upstream resolver.",
    highlight: true,
  },
  {
    icon: Lock,
    title: "Private by Default",
    desc: "Client IPs are hashed with HMAC-SHA256 before anything hits disk. No raw IPs in the logs, no phone-home, no third parties.",
    highlight: false,
  },
  {
    icon: BarChart3,
    title: "Real-Time Dashboard",
    desc: "Queries, block rates, top domains, categories, and a 24-hour trend in one view. Built on a single API call so it stays quick.",
    highlight: false,
  },
  {
    icon: ListFilter,
    title: "Rules That Fit Your Network",
    desc: "Exact domains, regex, priorities. Allow, block, redirect, or just log. Edits apply on the fly, no restarts.",
    highlight: false,
  },
  {
    icon: FileText,
    title: "Bring Your Own Blocklists",
    desc: "Pull from the usual lists over HTTP with ETag caching. Hosts files, domain lists, and adblock formats all work.",
    highlight: false,
  },
  {
    icon: Container,
    title: "One Command to Run",
    desc: "docker compose up and you're live. Happy on a Raspberry Pi, a NAS, or a small VM. Multi-arch, non-root.",
    highlight: false,
  },
  {
    icon: Plug,
    title: "A REST API for Everything",
    desc: "17 endpoints cover the dashboard, DNS engine, policies, blocklists, and analytics. Script it, graph it, glue it into your stack.",
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
