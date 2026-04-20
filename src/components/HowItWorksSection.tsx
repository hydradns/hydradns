import { useScrollAnimation } from "@/hooks/useScrollAnimation";
import { Terminal, Router, BarChart3, ChevronRight } from "lucide-react";

const steps = [
  {
    icon: Terminal,
    num: "01",
    title: "Deploy",
    desc: (
      <>
        Run{" "}
        <code className="font-mono text-xs text-brand-sky bg-surface-container-lowest px-1.5 py-0.5 rounded">
          docker compose up -d
        </code>{" "}
        on any machine. HydraDNS starts a DNS server and admin API in seconds.
      </>
    ),
  },
  {
    icon: Router,
    num: "02",
    title: "Point Your DNS",
    desc: (
      <>
        Set your router's DNS to your HydraDNS server. Every device on the network is now
        protected — phones, laptops, IoT, everything.
      </>
    ),
  },
  {
    icon: BarChart3,
    num: "03",
    title: "Monitor & Control",
    desc: (
      <>
        Use the dashboard to see real-time queries, manage blocklists, create policies,
        and track exactly what's happening on your network.
      </>
    ),
  },
];

export function HowItWorksSection() {
  const ref = useScrollAnimation();

  return (
    <section id="how-it-works" className="relative py-28 lg:py-36" ref={ref}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center mb-16">
          <p className="eyebrow fade-in-up">HOW IT WORKS</p>
          <h2 className="mt-4 text-4xl sm:text-5xl lg:text-6xl font-bold text-foreground fade-in-up leading-[1.1]">
            Three steps. Five minutes.
            <br />
            <span className="text-gradient">Full protection.</span>
          </h2>
        </div>

        <div className="relative max-w-5xl mx-auto">
          {/* Dashed connector */}
          <div className="hidden md:block absolute top-[50px] left-[12%] right-[12%] h-px border-t-2 border-dashed border-outline-variant/40 z-0">
            <ChevronRight className="absolute -top-[11px] left-[33%] h-5 w-5 text-brand-teal/60" />
            <ChevronRight className="absolute -top-[11px] left-[66%] h-5 w-5 text-brand-teal/60" />
          </div>

          <div className="grid md:grid-cols-3 gap-6 md:gap-8 relative z-10">
            {steps.map((s, i) => (
              <div
                key={i}
                className="relative rounded-xl bg-surface-container border border-outline-variant/25 p-8 text-center hover:border-brand-teal/30 hover:bg-surface-container-high/70 transition-all fade-in-up"
                style={{ transitionDelay: `${i * 120}ms` }}
              >
                <div className="mx-auto w-[72px] h-[72px] rounded-full bg-brand-teal/10 border border-brand-teal/20 flex items-center justify-center mb-6 relative">
                  <s.icon className="h-7 w-7 text-brand-teal" strokeWidth={1.75} />
                  <span className="absolute -top-1 -right-1 w-4 h-4 rounded-full bg-brand-teal shadow-[0_0_12px_rgba(0,212,170,0.6)]" />
                </div>
                <p className="font-mono text-xs font-semibold tracking-widest text-brand-teal mb-2">
                  STEP {s.num}
                </p>
                <h3 className="font-headline text-xl font-bold text-foreground mb-3">
                  {s.title}
                </h3>
                <p className="text-sm text-muted-foreground leading-relaxed">{s.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
