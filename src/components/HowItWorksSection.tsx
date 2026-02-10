import { useScrollAnimation } from "@/hooks/useScrollAnimation";
import { Terminal, Network, BarChart3 } from "lucide-react";

const steps = [
  {
    icon: Terminal,
    title: "Deploy",
    desc: "Run docker-compose up --build on any machine. HydraDNS starts a DNS server and admin API in seconds.",
  },
  {
    icon: Network,
    title: "Point Your DNS",
    desc: "Set your router's DNS to your HydraDNS server. Every device on the network is now protected — phones, laptops, IoT, everything.",
  },
  {
    icon: BarChart3,
    title: "Monitor & Control",
    desc: "Use the dashboard to see real-time queries, manage blocklists, create policies, and track what's happening on your network.",
  },
];

export function HowItWorksSection() {
  const ref = useScrollAnimation();

  return (
    <section id="how-it-works" className="py-24 lg:py-32" ref={ref}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center mb-16">
          <p className="text-sm font-semibold uppercase tracking-widest text-accent mb-4 fade-in-up">How It Works</p>
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-foreground fade-in-up">
            Three steps. Five minutes.<br />
            <span className="text-gradient">Full protection.</span>
          </h2>
        </div>

        <div className="grid md:grid-cols-3 gap-8 relative">
          {/* Connecting line (desktop) */}
          <div className="hidden md:block absolute top-16 left-[16.67%] right-[16.67%] h-px border-t-2 border-dashed border-border" />

          {steps.map((step, i) => (
            <div key={i} className="text-center relative fade-in-up" style={{ transitionDelay: `${i * 150}ms` }}>
              <div className="inline-flex items-center justify-center w-14 h-14 rounded-2xl bg-primary/10 border border-primary/20 mb-6 relative z-10">
                <step.icon className="h-7 w-7 text-primary" />
              </div>
              <div className="absolute top-3 left-1/2 -translate-x-1/2 w-8 h-8 rounded-full bg-primary/5 blur-xl" />
              <p className="text-xs font-bold uppercase tracking-widest text-muted-foreground mb-2">Step {i + 1}</p>
              <h3 className="text-xl font-bold text-foreground mb-3">{step.title}</h3>
              <p className="text-sm text-muted-foreground leading-relaxed max-w-xs mx-auto">{step.desc}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
