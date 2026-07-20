import { useMemo, useState } from "react";
import { useScrollAnimation } from "@/hooks/useScrollAnimation";
import { Input } from "@/components/ui/input";
import { Slider } from "@/components/ui/slider";
import {
  calculateRoi,
  formatINR,
  sanitizeSeats,
  HYDRADNS_ANNUAL_INR,
  USD_TO_INR,
} from "@/lib/roiCalculator";
import { TrendingDown } from "lucide-react";

const DEFAULT_SEATS = 50;
const MIN_SEATS = 1;
const MAX_SEATS = 1000;

export function RoiCalculatorSection() {
  const ref = useScrollAnimation();
  const [seats, setSeats] = useState<number>(DEFAULT_SEATS);

  const result = useMemo(() => calculateRoi(seats), [seats]);

  // Biggest per-seat competitor bill, used for the headline savings figure.
  const dearest = useMemo(
    () =>
      result.competitors.reduce(
        (max, c) => (c.annualINR > max.annualINR ? c : max),
        result.competitors[0],
      ),
    [result],
  );

  const handleInput = (raw: string) => {
    if (raw === "") {
      setSeats(0);
      return;
    }
    const parsed = Number.parseInt(raw, 10);
    setSeats(Number.isNaN(parsed) ? 0 : Math.min(parsed, MAX_SEATS));
  };

  const cleanSeats = sanitizeSeats(seats);

  return (
    <section id="roi-calculator" className="relative py-20 sm:py-28 lg:py-36" ref={ref}>
      <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center mb-14">
          <p className="eyebrow fade-in-up">ROI CALCULATOR</p>
          <h2 className="mt-4 text-4xl sm:text-5xl lg:text-6xl font-bold text-foreground fade-in-up leading-[1.1]">
            The flat-fee <span className="text-gradient">math.</span>
          </h2>
          <p className="mt-5 max-w-2xl mx-auto text-base sm:text-lg text-muted-foreground fade-in-up">
            Per-seat DNS security scales with your headcount. HydraDNS is one flat annual fee.
            Drag the count and watch the gap.
          </p>
        </div>

        <div className="grid lg:grid-cols-5 gap-6 lg:gap-8 items-start">
          {/* Input panel */}
          <div className="lg:col-span-2 rounded-xl bg-surface-container border border-outline-variant/30 p-7 sm:p-8 fade-in-up">
            <label
              htmlFor="roi-seats"
              className="block font-mono text-[11px] uppercase tracking-widest text-muted-foreground"
            >
              Devices / seats
            </label>

            <div className="mt-3 flex items-end gap-3">
              <Input
                id="roi-seats"
                type="number"
                inputMode="numeric"
                min={MIN_SEATS}
                max={MAX_SEATS}
                aria-label="Number of devices or seats"
                value={seats === 0 ? "" : seats}
                onChange={(e) => handleInput(e.target.value)}
                className="h-14 w-32 bg-surface-container-low text-3xl font-headline font-bold tracking-tightest"
              />
              <span className="pb-3 font-mono text-xs text-muted-foreground">
                {cleanSeats === 1 ? "device" : "devices"}
              </span>
            </div>

            <Slider
              className="mt-8"
              value={[Math.min(Math.max(cleanSeats, MIN_SEATS), MAX_SEATS)]}
              min={MIN_SEATS}
              max={MAX_SEATS}
              step={1}
              aria-label="Adjust device or seat count"
              onValueChange={(v) => setSeats(v[0])}
            />
            <div className="mt-2 flex justify-between font-mono text-[10px] text-muted-foreground">
              <span>{MIN_SEATS}</span>
              <span>{MAX_SEATS}+</span>
            </div>

            {/* HydraDNS flat result */}
            <div className="mt-8 rounded-lg bg-brand-teal/[0.06] border border-brand-teal/30 glow-primary p-5">
              <p className="font-mono text-[10px] uppercase tracking-widest text-brand-teal">
                HydraDNS · flat
              </p>
              <p
                className="mt-1 font-headline text-3xl sm:text-4xl font-bold tracking-tightest text-gradient"
                data-testid="roi-hydradns-annual"
              >
                {formatINR(result.hydradnsAnnualINR)}
              </p>
              <p className="mt-1 font-mono text-[11px] text-muted-foreground">
                per year · any number of devices
              </p>
            </div>

            <p className="mt-5 font-mono text-[10px] leading-relaxed text-muted-foreground">
              Competitor pricing is approximate published list price, converted at
              US$1 = ₹{USD_TO_INR}. HydraDNS flat fee shown is the top of the
              ₹15,000–18,000/yr range.
            </p>
          </div>

          {/* Comparison panel */}
          <div className="lg:col-span-3 fade-in-up">
            {/* Headline savings */}
            <div className="rounded-xl bg-surface-container border border-outline-variant/30 p-6 sm:p-7 flex items-center gap-4">
              <div className="shrink-0 h-11 w-11 rounded-lg bg-brand-green/10 border border-brand-green/30 flex items-center justify-center">
                <TrendingDown className="h-5 w-5 text-brand-green" strokeWidth={2.25} />
              </div>
              <div>
                <p className="font-mono text-[10px] uppercase tracking-widest text-muted-foreground">
                  Save up to
                </p>
                <p
                  className="font-headline text-2xl sm:text-3xl font-bold tracking-tightest text-brand-green"
                  data-testid="roi-max-savings"
                >
                  {formatINR(Math.max(dearest.savingsINR, 0))}
                  <span className="ml-2 text-sm font-normal text-muted-foreground">
                    / year vs {dearest.name}
                  </span>
                </p>
              </div>
            </div>

            {/* Per-competitor rows */}
            <div className="mt-4 rounded-xl bg-surface-container border border-outline-variant/30 overflow-hidden">
              {result.competitors.map((c, i) => (
                <div
                  key={c.id}
                  data-testid={`roi-row-${c.id}`}
                  className={`flex items-center justify-between gap-4 px-5 sm:px-6 py-4 ${
                    i > 0 ? "border-t border-outline-variant/10" : ""
                  }`}
                >
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-foreground truncate">{c.name}</p>
                    <p className="font-mono text-[10px] text-muted-foreground">
                      ${c.perUserPerMonthUSD}/user/mo · {c.tier}
                    </p>
                  </div>
                  <div className="text-right shrink-0">
                    <p
                      className="font-headline text-lg sm:text-xl font-bold tracking-tightest text-foreground"
                      data-testid={`roi-annual-${c.id}`}
                    >
                      {formatINR(c.annualINR)}
                    </p>
                    <p className="font-mono text-[10px] text-brand-amber">
                      {cleanSeats > 0 ? `${c.multiplier.toFixed(1)}× HydraDNS` : "—"}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
