import { useScrollAnimation } from "@/hooks/useScrollAnimation";

const stats = [
  { value: "17+", label: "API Endpoints" },
  { value: "<1ms", label: "Policy Decisions" },
  { value: "9", label: "Blocklist Categories" },
  { value: "0", label: "Client IPs Stored" },
];

export function StatsSection() {
  const ref = useScrollAnimation();

  return (
    <section className="relative py-16" ref={ref}>
      {/* Thin teal accent line above */}
      <div className="absolute top-0 left-1/2 -translate-x-1/2 w-1/2 h-px bg-gradient-to-r from-transparent via-brand-teal/50 to-transparent" />
      <div className="absolute top-0 left-1/2 -translate-x-1/2 w-1/3 h-24 bg-brand-teal/[0.04] blur-3xl pointer-events-none" />

      <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 md:gap-0 rounded-xl bg-surface-container border border-outline-variant/30 p-8 md:p-12 fade-in-up">
          {stats.map((stat, i) => (
            <div
              key={i}
              className={`text-center flex flex-col items-center justify-center ${
                i > 0 ? "md:border-l md:border-outline-variant/20" : ""
              }`}
            >
              <p className="font-headline text-4xl md:text-5xl font-bold tracking-tightest text-gradient">
                {stat.value}
              </p>
              <p className="mt-2 font-mono text-[11px] uppercase tracking-widest text-muted-foreground">
                {stat.label}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
