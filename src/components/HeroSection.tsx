import { Github, BookOpen } from "lucide-react";
import { ParticleBackground } from "./ParticleBackground";

const chartData = [30, 45, 38, 52, 48, 65, 58, 72, 68, 85, 78, 90, 82, 75, 88, 92, 85, 78, 95, 88, 80, 72, 68, 60];

export function HeroSection() {
  return (
    <section className="relative min-h-screen flex items-center justify-center overflow-hidden pt-16">
      <ParticleBackground />

      {/* Gradient overlays */}
      <div className="absolute inset-0 bg-gradient-to-b from-transparent via-background/50 to-background pointer-events-none" />

      <div className="relative z-10 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-20">
        <div className="grid lg:grid-cols-2 gap-12 lg:gap-16 items-center">
          {/* Left content */}
          <div className="text-center lg:text-left">
            <h1 className="text-4xl sm:text-5xl lg:text-6xl font-bold tracking-tight text-foreground leading-[1.1]">
              Block threats before
              <br />
              <span className="text-gradient">they ever connect.</span>
            </h1>
            <p className="mt-6 text-lg text-muted-foreground max-w-xl mx-auto lg:mx-0 leading-relaxed">
              HydraDNS is an open-source DNS security gateway that protects every device on your network — without installing anything on them. Self-hosted. Private. Blazing fast.
            </p>
            <div className="mt-8 flex flex-wrap gap-4 justify-center lg:justify-start">
              <a
                href="https://github.com/lopster568/HydraDNS"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-2 px-6 py-3 rounded-lg bg-primary text-primary-foreground font-medium hover:bg-primary/90 transition-colors glow-primary"
              >
                <Github className="h-5 w-5" />
                Get Started — It's Free
              </a>
              <a
                href="https://github.com/lopster568/HydraDNS#readme"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-2 px-6 py-3 rounded-lg border border-border text-foreground font-medium hover:bg-muted/50 transition-colors"
              >
                <BookOpen className="h-5 w-5" />
                Read the Docs
              </a>
            </div>
            <p className="mt-6 text-sm text-muted-foreground">
              Open source · GPL-3.0 · Self-hosted · No data leaves your network
            </p>
          </div>

          {/* Right: Dashboard mockup */}
          <div className="flex justify-center lg:justify-end">
            <div className="glass glow-primary rounded-2xl p-6 w-full max-w-md animate-float">
              <div className="flex items-center gap-2 mb-4">
                <div className="w-3 h-3 rounded-full bg-destructive/80" />
                <div className="w-3 h-3 rounded-full bg-[hsl(45,93%,47%)]/80" />
                <div className="w-3 h-3 rounded-full bg-[hsl(142,71%,45%)]/80" />
                <span className="ml-2 text-xs text-muted-foreground">HydraDNS Dashboard</span>
              </div>

              <div className="grid grid-cols-3 gap-3 mb-4">
                <div className="bg-muted/50 rounded-lg p-3">
                  <p className="text-xs text-muted-foreground">Total Queries</p>
                  <p className="text-xl font-bold text-foreground">124,847</p>
                </div>
                <div className="bg-muted/50 rounded-lg p-3">
                  <p className="text-xs text-muted-foreground">Blocked</p>
                  <p className="text-xl font-bold text-destructive">18,293</p>
                </div>
                <div className="bg-muted/50 rounded-lg p-3">
                  <p className="text-xs text-muted-foreground">Block Rate</p>
                  <p className="text-xl font-bold text-accent">14.6%</p>
                </div>
              </div>

              {/* Mini chart */}
              <div className="bg-muted/30 rounded-lg p-3">
                <p className="text-xs text-muted-foreground mb-2">24h Query Volume</p>
                <svg viewBox="0 0 240 60" className="w-full h-12">
                  <defs>
                    <linearGradient id="chartGrad" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor="hsl(217, 91%, 60%)" stopOpacity="0.3" />
                      <stop offset="100%" stopColor="hsl(217, 91%, 60%)" stopOpacity="0" />
                    </linearGradient>
                  </defs>
                  <path
                    d={`M0,${60 - chartData[0] * 0.6} ${chartData.map((v, i) => `L${(i / (chartData.length - 1)) * 240},${60 - v * 0.6}`).join(" ")} L240,60 L0,60 Z`}
                    fill="url(#chartGrad)"
                  />
                  <path
                    d={`M0,${60 - chartData[0] * 0.6} ${chartData.map((v, i) => `L${(i / (chartData.length - 1)) * 240},${60 - v * 0.6}`).join(" ")}`}
                    fill="none"
                    stroke="hsl(217, 91%, 60%)"
                    strokeWidth="2"
                  />
                </svg>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
