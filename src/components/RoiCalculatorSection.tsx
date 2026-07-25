import { useMemo, useState } from "react";
import { useScrollAnimation } from "@/hooks/useScrollAnimation";
import { Input } from "@/components/ui/input";
import { Slider } from "@/components/ui/slider";
import { calculateRoi, formatINR, sanitizeSeats, USD_TO_INR } from "@/lib/roiCalculator";
import { Github, Users } from "lucide-react";

const DEFAULT_SEATS = 50;
const MIN_SEATS = 1;
const MAX_SEATS = 1000;

// Same destination as the site's other primary CTAs (Navbar "Get Started",
// Hero "Get the code") — the project has no separate contact/sales page yet,
// GitHub is where inquiries actually land.
const CTA_HREF = "https://github.com/hydradns/hydradns";

export function RoiCalculatorSection() {
  const ref = useScrollAnimation();
  const [seats, setSeats] = useState<number>(DEFAULT_SEATS);

  const result = useMemo(() => calculateRoi(seats), [seats]);

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
          <p className="eyebrow fade-in-up">COST CALCULATOR</p>
          <h2 className="mt-4 text-4xl sm:text-5xl lg:text-6xl font-bold text-foreground fade-in-up leading-[1.1]">
            See what per-seat <span className="text-gradient">costs you.</span>
          </h2>
          <p className="mt-5 max-w-2xl mx-auto text-base sm:text-lg text-muted-foreground fade-in-up">
            Per-seat DNS security scales with your headcount — every device you add raises the
            bill. HydraDNS is billed as one flat annual fee instead. Drag the count and watch
            the gap grow.
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

            {/* HydraDNS flat-fee positioning (no price published yet) */}
            <div className="mt-8 rounded-lg bg-brand-teal/[0.06] border border-brand-teal/30 glow-primary p-5">
              <p className="font-mono text-[10px] uppercase tracking-widest text-brand-teal">
                HydraDNS · flat
              </p>
              <p className="mt-1 font-headline text-2xl sm:text-3xl font-bold tracking-tightest text-gradient">
                One annual fee
              </p>
              <p className="mt-1 font-mono text-[11px] text-muted-foreground">
                same price at 10 devices or 1,000 — not per seat
              </p>
              <a
                href={CTA_HREF}
                target="_blank"
                rel="noopener noreferrer"
                className="btn-primary mt-4 inline-flex items-center gap-2 px-4 py-2 rounded-md text-sm font-semibold shadow-[0_0_20px_rgba(0,212,170,0.25)] hover:shadow-[0_0_28px_rgba(0,212,170,0.4)] transition-shadow"
              >
                <Github className="h-4 w-4" />
                Talk to us for pricing
              </a>
            </div>

            <p className="mt-5 font-mono text-[10px] leading-relaxed text-muted-foreground">
              Competitor pricing is approximate published list price, converted at
              US$1 = ₹{USD_TO_INR} (FX rate as of Jul 2026 — checked periodically,
              may drift). HydraDNS pricing isn't published yet — the rows below show what
              you'd pay elsewhere.
            </p>
          </div>

          {/* Comparison panel */}
          <div className="lg:col-span-3 fade-in-up">
            {/* Flat-fee vs per-seat positioning */}
            <div className="rounded-xl bg-surface-container border border-outline-variant/30 p-6 sm:p-7 flex items-center gap-4">
              <div className="shrink-0 h-11 w-11 rounded-lg bg-brand-green/10 border border-brand-green/30 flex items-center justify-center">
                <Users className="h-5 w-5 text-brand-green" strokeWidth={2.25} />
              </div>
              <div>
                <p className="font-mono text-[10px] uppercase tracking-widest text-muted-foreground">
                  Every row below grows with headcount
                </p>
                <p className="mt-1 text-base sm:text-lg font-medium text-foreground leading-snug">
                  HydraDNS doesn't. It's one flat fee — add devices for free.{" "}
                  <a
                    href={CTA_HREF}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-brand-teal underline underline-offset-2 hover:text-brand-teal/80 transition-colors"
                  >
                    Get a quote
                  </a>
                  .
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
                      {c.billing?.type === "block"
                        ? `$${c.billing.annualUSDPerBlock}/yr per ${c.billing.seatsPerBlock} seats · ${c.tier}`
                        : `$${c.perUserPerMonthUSD}/user/mo · ${c.tier}`}
                    </p>
                  </div>
                  <div className="text-right shrink-0">
                    <p
                      className="font-headline text-lg sm:text-xl font-bold tracking-tightest text-foreground"
                      data-testid={`roi-annual-${c.id}`}
                    >
                      {formatINR(c.annualINR)}
                    </p>
                    <p className="font-mono text-[10px] text-muted-foreground">/ year</p>
                  </div>
                </div>
              ))}
            </div>

            {result.competitors.some((c) => c.billing?.type === "block") && (
              <p className="mt-3 font-mono text-[10px] leading-relaxed text-muted-foreground">
                NextDNS Business is billed in blocks of 50 seats, not per seat — cost
                steps up at each 50-seat threshold rather than scaling smoothly like
                the other rows above.
              </p>
            )}
          </div>
        </div>
      </div>
    </section>
  );
}
