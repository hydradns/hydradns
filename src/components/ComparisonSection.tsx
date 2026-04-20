import { useScrollAnimation } from "@/hooks/useScrollAnimation";
import { Check, X } from "lucide-react";

type CellValue = boolean | string;

type Row = {
  feature: string;
  hydra: CellValue;
  pihole: CellValue;
  nextdns: CellValue;
  adguard: CellValue;
};

const rows: Row[] = [
  { feature: "Open Source", hydra: "GPL-3.0", pihole: true, nextdns: false, adguard: true },
  { feature: "Self-Hosted", hydra: true, pihole: true, nextdns: false, adguard: true },
  { feature: "Policy Engine", hydra: "Priority + Regex", pihole: false, nextdns: "Limited", adguard: "Limited" },
  { feature: "IP Anonymization", hydra: "HMAC-SHA256", pihole: false, nextdns: false, adguard: false },
  { feature: "Architecture", hydra: "Microservices + gRPC", pihole: "Monolith", nextdns: "Cloud SaaS", adguard: "Monolith" },
  { feature: "Language", hydra: "Go", pihole: "PHP + Shell", nextdns: "N/A", adguard: "Go" },
  { feature: "Hot-Reload Policies", hydra: true, pihole: false, nextdns: false, adguard: false },
  { feature: "REST API", hydra: "17 endpoints", pihole: false, nextdns: true, adguard: "Limited" },
  { feature: "Bloom Filter Engine", hydra: true, pihole: false, nextdns: false, adguard: false },
  { feature: "Your Data Stays Local", hydra: true, pihole: true, nextdns: false, adguard: true },
];

export function ComparisonSection() {
  const ref = useScrollAnimation();

  return (
    <section id="comparison" className="relative py-28 lg:py-36" ref={ref}>
      <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center mb-14">
          <p className="eyebrow fade-in-up">HOW WE COMPARE</p>
          <h2 className="mt-4 text-4xl sm:text-5xl lg:text-6xl font-bold text-foreground fade-in-up leading-[1.1]">
            Built <span className="text-gradient">different.</span>
          </h2>
        </div>

        <div className="relative rounded-xl overflow-hidden bg-surface-container border border-outline-variant/30 glow-primary fade-in-up">
          <div className="overflow-x-auto">
            <table className="w-full min-w-[760px] text-left border-collapse">
              <thead>
                <tr className="border-b border-outline-variant/30 bg-surface-container-low/60">
                  <HeaderCell>Feature</HeaderCell>
                  <HeaderCell highlight>
                    <div className="absolute top-0 inset-x-0 h-[3px] bg-gradient-to-r from-brand-teal to-brand-teal-bright" />
                    <div className="flex items-center gap-2">
                      <span className="text-brand-teal">HydraDNS</span>
                      <span className="hidden sm:inline px-1.5 py-0.5 rounded-full bg-brand-teal/15 border border-brand-teal/30 font-mono text-[9px] tracking-wider text-brand-teal">
                        RECOMMENDED
                      </span>
                    </div>
                  </HeaderCell>
                  <HeaderCell>Pi-hole</HeaderCell>
                  <HeaderCell>NextDNS</HeaderCell>
                  <HeaderCell>AdGuard Home</HeaderCell>
                </tr>
              </thead>
              <tbody>
                {rows.map((row, i) => (
                  <tr
                    key={i}
                    className="border-b border-outline-variant/10 last:border-b-0 hover:bg-surface-container-high/30 transition-colors"
                  >
                    <td className="px-5 py-4 text-sm font-medium text-foreground">
                      {row.feature}
                    </td>
                    <td className="relative px-5 py-4 text-center bg-brand-teal/[0.04]">
                      <Cell value={row.hydra} tone="hydra" />
                    </td>
                    <td className="px-5 py-4 text-center">
                      <Cell value={row.pihole} />
                    </td>
                    <td className="px-5 py-4 text-center">
                      <Cell value={row.nextdns} />
                    </td>
                    <td className="px-5 py-4 text-center">
                      <Cell value={row.adguard} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </section>
  );
}

function HeaderCell({
  children,
  highlight,
}: {
  children: React.ReactNode;
  highlight?: boolean;
}) {
  return (
    <th
      className={`relative px-5 py-5 font-headline text-xs uppercase tracking-widest font-semibold ${
        highlight ? "bg-brand-teal/[0.04] text-brand-teal" : "text-muted-foreground"
      }`}
    >
      {children}
    </th>
  );
}

function Cell({ value, tone }: { value: CellValue; tone?: "hydra" | "default" }) {
  const isHydra = tone === "hydra";

  if (value === true) {
    return (
      <span className="inline-flex items-center justify-center">
        <Check
          className={`h-4 w-4 ${isHydra ? "text-brand-teal" : "text-brand-green"}`}
          strokeWidth={2.75}
        />
      </span>
    );
  }
  if (value === false) {
    return (
      <span className="inline-flex items-center justify-center">
        <X className="h-4 w-4 text-brand-red/70" strokeWidth={2.25} />
      </span>
    );
  }
  if (isHydra) {
    return (
      <span className="inline-flex items-center px-2 py-0.5 rounded-full font-mono text-xs text-brand-teal bg-brand-teal/15 border border-brand-teal/30">
        {value}
      </span>
    );
  }
  return <span className="font-mono text-xs text-muted-foreground">{value}</span>;
}
