import { Github, BookOpen, Shield, Code, Scale, Server, Lock } from "lucide-react";
import { ParticleBackground } from "./ParticleBackground";

const sparklineData = [20, 35, 25, 45, 30, 60, 40, 75, 55, 90, 60, 78, 48, 82, 70, 88];

const trustBadges = [
  { icon: Code, label: "Open Source" },
  { icon: Scale, label: "GPL-3.0" },
  { icon: Server, label: "Self-hosted" },
  { icon: Lock, label: "No data leaves your network" },
];

export function HeroSection() {
  return (
    <section className="relative min-h-screen flex items-center justify-center overflow-hidden pt-24 pb-16">
      {/* Ambient radial glow + dot field */}
      <div className="absolute inset-0 hero-ambient pointer-events-none" />
      <ParticleBackground />
      <div className="absolute bottom-0 inset-x-0 h-40 bg-gradient-to-b from-transparent to-background pointer-events-none" />

      <div className="relative z-10 w-full max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="grid lg:grid-cols-2 gap-10 lg:gap-20 items-center">
          {/* Left content */}
          <div className="text-center lg:text-left">
            {/* Status pill */}
            <div className="inline-flex items-center gap-2.5 px-3.5 py-1.5 rounded-full bg-surface-container-low/80 border border-outline-variant/30">
              <span className="relative flex h-2 w-2">
                <span className="absolute inline-flex h-full w-full rounded-full bg-brand-teal opacity-75 animate-ping" />
                <span className="relative inline-flex h-2 w-2 rounded-full bg-brand-teal" />
              </span>
              <span className="font-mono text-xs text-muted-foreground tracking-wide">
                v1.0 · GPL-3.0 · Self-hosted
              </span>
            </div>

            {/* Headline */}
            <h1 className="mt-6 font-headline text-4xl sm:text-5xl lg:text-7xl font-bold tracking-tightest text-foreground leading-[1.05]">
              Block threats before
              <br />
              <span className="text-gradient">they ever connect.</span>
            </h1>

            <p className="mt-6 text-lg lg:text-xl text-muted-foreground max-w-xl mx-auto lg:mx-0 leading-relaxed">
              HydraDNS is an open-source DNS gateway that filters every device on your
              network without touching any of them.{" "}
              <span className="text-foreground/85">
                You run it. You own the logs.
              </span>
            </p>

            {/* CTAs */}
            <div className="mt-9 flex flex-wrap gap-4 justify-center lg:justify-start">
              <a
                href="https://github.com/hydradns/hydradns"
                target="_blank"
                rel="noopener noreferrer"
                className="btn-primary inline-flex items-center gap-2.5 px-7 py-3.5 rounded-md font-semibold shadow-[0_0_30px_rgba(0,212,170,0.25)] hover:shadow-[0_0_40px_rgba(0,212,170,0.4)] transition-shadow"
              >
                <Github className="h-5 w-5" />
                Get the code
              </a>
              <a
                href="https://docs.hydradns.app"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-2.5 px-7 py-3.5 rounded-md border border-outline-variant/40 bg-surface-container/30 text-foreground font-medium hover:bg-surface-container-high/60 transition-colors"
              >
                <BookOpen className="h-5 w-5 text-brand-sky" />
                Read the Docs
              </a>
            </div>

            {/* Trust badges */}
            <div className="mt-10 pt-8 border-t border-outline-variant/20 flex flex-wrap gap-x-7 gap-y-3 justify-center lg:justify-start">
              {trustBadges.map((b) => (
                <div
                  key={b.label}
                  className="inline-flex items-center gap-2 text-sm text-muted-foreground"
                >
                  <b.icon className="h-4 w-4 text-brand-teal" />
                  <span>{b.label}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Right: Dashboard preview window */}
          <div className="flex justify-center lg:justify-end">
            <div className="relative w-full max-w-md">
              {/* Glow halo */}
              <div className="absolute -inset-4 bg-brand-teal/10 blur-3xl rounded-3xl pointer-events-none" />

              <div className="relative rounded-xl bg-surface-container border border-outline-variant/40 overflow-hidden glow-primary-strong animate-float">
                {/* Window chrome */}
                <div className="flex items-center justify-between h-11 px-4 bg-surface-container-highest border-b border-outline-variant/20">
                  <div className="flex gap-2">
                    <span className="w-3 h-3 rounded-full bg-surface-variant/80 border border-outline-variant/30" />
                    <span className="w-3 h-3 rounded-full bg-surface-variant/80 border border-outline-variant/30" />
                    <span className="w-3 h-3 rounded-full bg-surface-variant/80 border border-outline-variant/30" />
                  </div>
                  <div className="flex items-center gap-1.5 px-2.5 py-1 rounded border border-outline-variant/30 bg-surface-container-lowest">
                    <span className="relative flex h-1.5 w-1.5">
                      <span className="absolute inline-flex h-full w-full rounded-full bg-brand-teal opacity-75 animate-ping" />
                      <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-brand-teal" />
                    </span>
                    <span className="font-mono text-[10px] font-semibold tracking-wider text-brand-teal">
                      LIVE
                    </span>
                  </div>
                </div>

                {/* Body */}
                <div className="p-5 space-y-4 bg-surface-container">
                  {/* Stat tiles */}
                  <div className="grid grid-cols-2 gap-3">
                    <StatTile label="Total Queries" value="124,847" tone="default" />
                    <StatTile label="Blocked" value="18,293" tone="danger" />
                  </div>

                  <div className="flex items-center justify-between p-4 rounded-md bg-surface-container-low border border-outline-variant/20">
                    <div>
                      <p className="text-[11px] uppercase tracking-wider text-muted-foreground font-mono">
                        Block Rate
                      </p>
                      <p className="mt-0.5 font-headline text-2xl font-bold text-brand-teal">
                        14.6%
                      </p>
                    </div>
                    <div className="relative">
                      <Shield className="h-10 w-10 text-brand-teal/20" strokeWidth={1.25} />
                      <Shield className="absolute inset-0 h-10 w-10 text-brand-teal/60" strokeWidth={1.25} />
                    </div>
                  </div>

                  {/* Sparkline */}
                  <div className="p-4 rounded-md bg-surface-container-lowest border border-outline-variant/20">
                    <p className="text-[11px] uppercase tracking-wider text-muted-foreground font-mono mb-3">
                      Query Volume · 24h
                    </p>
                    <div className="flex items-end gap-[3px] h-16">
                      {sparklineData.map((v, i) => (
                        <span
                          key={i}
                          className="flex-1 rounded-t-sm"
                          style={{
                            height: `${v}%`,
                            background: `linear-gradient(to top, rgba(0,212,170,${
                              0.2 + (v / 100) * 0.7
                            }), rgba(70,241,197,${0.4 + (v / 100) * 0.5}))`,
                          }}
                        />
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

type Tone = "default" | "danger";

function StatTile({ label, value, tone }: { label: string; value: string; tone: Tone }) {
  return (
    <div className="p-4 rounded-md bg-surface-container-low border border-outline-variant/20">
      <p className="text-[11px] uppercase tracking-wider text-muted-foreground font-mono">
        {label}
      </p>
      <p
        className={`mt-1 font-mono text-xl font-semibold ${
          tone === "danger" ? "text-brand-red" : "text-foreground"
        }`}
      >
        {value}
      </p>
    </div>
  );
}
