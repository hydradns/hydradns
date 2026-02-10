import { useScrollAnimation } from "@/hooks/useScrollAnimation";
import { Shield, Lock, Zap, BarChart3, ListFilter, FileText, Container, Plug } from "lucide-react";

const features = [
  { icon: Shield, title: "DNS-Layer Blocking", desc: "Blocks malware, phishing, ads, and trackers at the DNS level. Threats are stopped before they ever reach your devices." },
  { icon: Lock, title: "Privacy First", desc: "Client IPs are anonymized with HMAC-SHA256 before they touch disk. Your browsing data stays yours. Always." },
  { icon: Zap, title: "Sub-Millisecond Decisions", desc: "Bloom filter–accelerated policy engine with lock-free reads. 99.99% of clean queries bypass the filter with near-zero overhead." },
  { icon: BarChart3, title: "Real-Time Dashboard", desc: "See total queries, block rates, top domains, category breakdowns, and 24-hour trends. One API call powers the whole view." },
  { icon: ListFilter, title: "Flexible Policy Engine", desc: "Create rules with exact domains, regex patterns, and priorities. Allow, block, redirect, or log. Hot-reload without downtime." },
  { icon: FileText, title: "Multiple Blocklist Sources", desc: "Sync from popular blocklists via URL with ETag-based caching. Supports hosts files, domain lists, and adblock formats." },
  { icon: Container, title: "One Command Deploy", desc: "docker-compose up and you're protected. Runs on a Raspberry Pi, a NAS, or a cloud VM. Multi-arch, non-root containers." },
  { icon: Plug, title: "Full REST API", desc: "17 endpoints for dashboard, DNS engine, policies, blocklists, and analytics. Build your own integrations or automate everything." },
];

export function FeaturesSection() {
  const ref = useScrollAnimation();

  return (
    <section id="features" className="py-24 lg:py-32" ref={ref}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center mb-16">
          <p className="text-sm font-semibold uppercase tracking-widest text-accent mb-4 fade-in-up">Features</p>
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-foreground fade-in-up">
            Everything you need.<br />
            <span className="text-gradient">Nothing you don't.</span>
          </h2>
        </div>

        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-5">
          {features.map((f, i) => (
            <div
              key={i}
              className="glass rounded-xl p-6 hover:border-primary/30 transition-all duration-300 hover:glow-primary fade-in-up group"
              style={{ transitionDelay: `${i * 50}ms` }}
            >
              <f.icon className="h-8 w-8 text-primary mb-4 group-hover:text-accent transition-colors" />
              <h3 className="text-lg font-semibold text-foreground mb-2">{f.title}</h3>
              <p className="text-sm text-muted-foreground leading-relaxed">{f.desc}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
