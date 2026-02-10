import { useScrollAnimation } from "@/hooks/useScrollAnimation";

const stats = [
  { value: "17+", label: "API Endpoints" },
  { value: "< 1ms", label: "Policy Decisions" },
  { value: "9", label: "Blocklist Categories" },
  { value: "0", label: "Client IPs Stored" },
];

export function StatsSection() {
  const ref = useScrollAnimation();

  return (
    <section className="py-16" ref={ref}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="glass rounded-2xl p-8 lg:p-12 fade-in-up">
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-8">
            {stats.map((stat, i) => (
              <div key={i} className="text-center">
                <p className="text-3xl lg:text-4xl font-bold text-gradient">{stat.value}</p>
                <p className="mt-1 text-sm text-muted-foreground">{stat.label}</p>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
