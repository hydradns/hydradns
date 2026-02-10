import { useScrollAnimation } from "@/hooks/useScrollAnimation";

const rows = [
  { feature: "Open Source", hydra: "GPL-3.0", pihole: true, nextdns: false, adguard: true },
  { feature: "Self-Hosted", hydra: true, pihole: true, nextdns: false, adguard: true },
  { feature: "Policy Engine", hydra: "Priority + Regex", pihole: false, nextdns: "Limited", adguard: "Limited" },
  { feature: "IP Anonymization", hydra: "Built-in HMAC", pihole: false, nextdns: false, adguard: false },
  { feature: "Architecture", hydra: "Microservices + gRPC", pihole: "Monolith", nextdns: "Cloud SaaS", adguard: "Monolith" },
  { feature: "Language", hydra: "Go", pihole: "PHP + Shell", nextdns: "—", adguard: "Go" },
  { feature: "Hot Reload Policies", hydra: true, pihole: false, nextdns: false, adguard: false },
  { feature: "REST API", hydra: "17 endpoints", pihole: false, nextdns: true, adguard: "Limited" },
  { feature: "Bloom Filter Engine", hydra: true, pihole: false, nextdns: false, adguard: false },
  { feature: "Your Data Stays Local", hydra: true, pihole: true, nextdns: false, adguard: true },
];

function CellValue({ value }: { value: boolean | string }) {
  if (value === true) return <span className="text-[hsl(142,71%,45%)]">✓</span>;
  if (value === false) return <span className="text-destructive/70">✗</span>;
  return <span className="text-sm">{value}</span>;
}

export function ComparisonSection() {
  const ref = useScrollAnimation();

  return (
    <section id="comparison" className="py-24 lg:py-32" ref={ref}>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center mb-16">
          <p className="text-sm font-semibold uppercase tracking-widest text-accent mb-4 fade-in-up">How We Compare</p>
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-foreground fade-in-up">
            Built <span className="text-gradient">different.</span>
          </h2>
        </div>

        <div className="overflow-x-auto fade-in-up">
          <table className="w-full min-w-[640px] text-sm">
            <thead>
              <tr className="border-b border-border">
                <th className="text-left py-4 px-4 text-muted-foreground font-medium">Feature</th>
                <th className="py-4 px-4 text-primary font-bold border-x border-primary/20 bg-primary/5">HydraDNS</th>
                <th className="py-4 px-4 text-muted-foreground font-medium">Pi-hole</th>
                <th className="py-4 px-4 text-muted-foreground font-medium">NextDNS</th>
                <th className="py-4 px-4 text-muted-foreground font-medium">AdGuard Home</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row, i) => (
                <tr key={i} className="border-b border-border/50 hover:bg-muted/20 transition-colors">
                  <td className="py-3 px-4 text-foreground font-medium">{row.feature}</td>
                  <td className="py-3 px-4 text-center border-x border-primary/20 bg-primary/5 font-medium text-foreground">
                    <CellValue value={row.hydra} />
                  </td>
                  <td className="py-3 px-4 text-center text-muted-foreground"><CellValue value={row.pihole} /></td>
                  <td className="py-3 px-4 text-center text-muted-foreground"><CellValue value={row.nextdns} /></td>
                  <td className="py-3 px-4 text-center text-muted-foreground"><CellValue value={row.adguard} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  );
}
