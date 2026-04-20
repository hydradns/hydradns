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
  Bot,
  ArrowRight,
} from "lucide-react";

type Badge = { label: string; tone: "teal" | "sky" };

type Feature = {
  icon: typeof Shield;
  title: string;
  desc: string;
  badge?: Badge;
};

const features: Feature[] = [
  {
    icon: Shield,
    title: "Block at the DNS Layer",
    desc: "Malware, phishing, ads, and trackers get dropped before a connection is ever made. Nothing to install on the client.",
  },
  {
    icon: Zap,
    title: "Sub-Millisecond Decisions",
    desc: "A Bloom filter sits in front of the policy engine, so most clean queries skip the lookup entirely and go straight to the upstream resolver.",
    badge: { label: "FAST PATH", tone: "teal" },
  },
  {
    icon: Lock,
    title: "Private by Default",
    desc: "Client IPs are hashed with HMAC-SHA256 before anything hits disk. No raw IPs in the logs, no phone-home, no third parties.",
  },
  {
    icon: BarChart3,
    title: "Real-Time Dashboard",
    desc: "Queries, block rates, top domains, categories, and a 24-hour trend in one view. Built on a single API call so it stays quick.",
  },
  {
    icon: ListFilter,
    title: "Rules That Fit Your Network",
    desc: "Exact domains, regex, priorities. Allow, block, redirect, or just log. Edits apply on the fly, no restarts.",
  },
  {
    icon: FileText,
    title: "Bring Your Own Blocklists",
    desc: "Pull from the usual lists over HTTP with ETag caching. Hosts files, domain lists, and adblock formats all work.",
  },
  {
    icon: Bot,
    title: "Run It by Asking",
    desc: "A built-in MCP server exposes every action to AI assistants. Block a domain, tune a policy, or audit traffic by asking Claude or ChatGPT in plain English. No rule syntax to memorize.",
    badge: { label: "AI-NATIVE", tone: "sky" },
  },
  {
    icon: Container,
    title: "One Command to Run",
    desc: "docker compose up and you're live. Happy on a mini-PC, a NAS, or a small VM. Multi-arch, non-root.",
  },
  {
    icon: Plug,
    title: "A REST API for Everything",
    desc: "17 endpoints cover the dashboard, DNS engine, policies, blocklists, and analytics. Script it, graph it, glue it into your stack.",
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
        </div>

        <div className="mt-10 text-center fade-in-up">
          <a
            href="https://docs.hydradns.app"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 font-mono text-sm text-muted-foreground hover:text-brand-teal transition-colors"
          >
            The full feature list lives in the docs
            <ArrowRight className="h-4 w-4" />
          </a>
        </div>
      </div>
    </section>
  );
}

function FeatureCard({ feature, delay }: { feature: Feature; delay: number }) {
  const { icon: Icon, title, desc, badge } = feature;
  const base =
    "group relative rounded-xl bg-surface-container p-7 border transition-all duration-300 fade-in-up";
  const state = badge
    ? badge.tone === "sky"
      ? "border-brand-sky/30 shadow-glow-sky"
      : "border-brand-teal/30 glow-primary"
    : "border-outline-variant/25 hover:border-brand-teal/40 hover:bg-surface-container-high/80";

  const badgeClasses =
    badge?.tone === "sky"
      ? "bg-brand-sky/15 border-brand-sky/30 text-brand-sky"
      : "bg-brand-teal/15 border-brand-teal/30 text-brand-teal";

  const iconTileClasses =
    badge?.tone === "sky"
      ? "bg-brand-sky/10 border-brand-sky/20"
      : "bg-brand-teal/10 border-brand-teal/20";

  const iconColor = badge?.tone === "sky" ? "text-brand-sky" : "text-brand-teal";

  return (
    <div className={`${base} ${state}`} style={{ transitionDelay: `${delay}ms` }}>
      {badge && (
        <span
          className={`absolute top-5 right-5 px-2 py-0.5 rounded-full border font-mono text-[10px] tracking-wider ${badgeClasses}`}
        >
          {badge.label}
        </span>
      )}
      <div
        className={`mb-5 inline-flex h-11 w-11 items-center justify-center rounded-lg border ${iconTileClasses}`}
      >
        <Icon className={`h-5 w-5 ${iconColor}`} strokeWidth={2} />
      </div>
      <h3 className="font-headline text-lg font-bold text-foreground mb-2">{title}</h3>
      <p className="text-sm text-muted-foreground leading-relaxed">{desc}</p>
    </div>
  );
}
