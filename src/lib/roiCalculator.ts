/*
 * ROI / cost-comparison math for the landing page calculator.
 *
 * HydraDNS is billed as a flat annual fee (₹15,000–18,000/yr) that does NOT
 * scale with device/seat count. Per-seat DNS-security competitors are priced
 * in USD per user per month, so their annual cost grows linearly with seats.
 * This module is pure (no React) so it can be unit-tested in isolation.
 */

/**
 * USD → INR assumption used for competitor pricing. Published competitor list
 * prices are in USD; this is a deliberately conservative round figure so the
 * comparison stays defensible. Surfaced in the UI next to the result.
 *
 * Checked 2026-07-24 against investing.com / Wise / BookMyForex spot rates
 * (~₹96.6–97.0/USD at the time). This is a spot-rate snapshot, not a peg —
 * review periodically and bump it so it doesn't silently go stale.
 */
export const USD_TO_INR = 97;

/**
 * HydraDNS flat annual price in INR. Top of the ₹15,000–18,000/yr range is used
 * so the savings shown are the conservative (smallest) case.
 */
export const HYDRADNS_ANNUAL_INR = 18000;

export interface Competitor {
  id: string;
  name: string;
  /**
   * Published list price, USD per user per month, for vendors that bill
   * linearly per seat. For block-billed vendors (see `billing` below) this
   * is a display-only approximation at the reference block size — the
   * actual cost is computed from `billing`, not this field.
   */
  perUserPerMonthUSD: number;
  /** Short label describing the tier the price is drawn from. */
  tier: string;
  /**
   * Optional non-linear billing model. When present, annual cost is
   * computed from this instead of `perUserPerMonthUSD × seats × 12`. Used
   * for vendors (e.g. NextDNS) that sell in fixed-size seat blocks rather
   * than per individual seat — cost jumps at each block boundary instead of
   * scaling smoothly.
   */
  billing?: {
    type: "block";
    seatsPerBlock: number;
    annualUSDPerBlock: number;
  };
}

/**
 * Approximate published list prices (USD / user / month). These are the
 * per-seat plans an Indian SMB / school / hospital would actually be quoted.
 *
 * Prices last checked 2026-07-24. Cisco Umbrella publishes no official price
 * list; $3.00 is the reseller/estimator midpoint for the Essentials tier,
 * not a vendor-confirmed figure.
 */
export const COMPETITORS: Competitor[] = [
  { id: "cisco-umbrella", name: "Cisco Umbrella", perUserPerMonthUSD: 3.0, tier: "DNS Security Essentials (est.)" },
  { id: "cloudflare", name: "Cloudflare Gateway", perUserPerMonthUSD: 7, tier: "Zero Trust Standard" },
  { id: "dnsfilter", name: "DNSFilter", perUserPerMonthUSD: 1.05, tier: "Core" },
  {
    id: "nextdns",
    name: "NextDNS",
    // Display-only approximation at the 50-seat block size (~$0.40/user/mo).
    // Real billing is NOT linear per seat — see `billing`.
    perUserPerMonthUSD: 0.4,
    tier: "Business",
    billing: { type: "block", seatsPerBlock: 50, annualUSDPerBlock: 199 },
  },
];

export interface CompetitorResult extends Competitor {
  /** Total annual cost in INR for the given seat count. */
  annualINR: number;
  /** competitor annual − HydraDNS flat, in INR (>= 0 when competitor is dearer). */
  savingsINR: number;
  /** How many times more the competitor costs vs the HydraDNS flat fee. */
  multiplier: number;
}

export interface RoiResult {
  /** Sanitised seat count actually used for the math. */
  seats: number;
  usdToInr: number;
  hydradnsAnnualINR: number;
  competitors: CompetitorResult[];
}

export interface RoiOptions {
  usdToInr?: number;
  hydradnsAnnualINR?: number;
  competitors?: Competitor[];
}

/** Annual INR cost for a single per-seat (linear) competitor. */
export function competitorAnnualINR(
  seats: number,
  perUserPerMonthUSD: number,
  usdToInr: number = USD_TO_INR,
): number {
  const s = sanitizeSeats(seats);
  return Math.round(s * perUserPerMonthUSD * 12 * usdToInr);
}

/**
 * Annual INR cost for a competitor, honoring its billing model. Defaults to
 * linear per-seat (`competitorAnnualINR`); for competitors with `billing`
 * set to a fixed-size seat block (e.g. NextDNS, sold in blocks of 50 seats
 * at a flat price per block), this is a step function of seats: cost only
 * increases at each block boundary, it does not scale smoothly per seat.
 */
export function competitorAnnualINRForCompetitor(
  seats: number,
  competitor: Pick<Competitor, "perUserPerMonthUSD" | "billing">,
  usdToInr: number = USD_TO_INR,
): number {
  const s = sanitizeSeats(seats);
  if (competitor.billing?.type === "block") {
    const blocks = Math.ceil(s / competitor.billing.seatsPerBlock);
    return Math.round(blocks * competitor.billing.annualUSDPerBlock * usdToInr);
  }
  return competitorAnnualINR(s, competitor.perUserPerMonthUSD, usdToInr);
}

/** Clamp seats to a non-negative integer; treat NaN / negative as 0. */
export function sanitizeSeats(seats: number): number {
  if (!Number.isFinite(seats) || seats <= 0) return 0;
  return Math.floor(seats);
}

/**
 * Given a seat/device count, return HydraDNS's flat annual cost plus each
 * competitor's per-seat annual cost, savings, and cost multiplier.
 */
export function calculateRoi(seats: number, options: RoiOptions = {}): RoiResult {
  const usdToInr = options.usdToInr ?? USD_TO_INR;
  const hydradnsAnnualINR = options.hydradnsAnnualINR ?? HYDRADNS_ANNUAL_INR;
  const competitors = options.competitors ?? COMPETITORS;
  const s = sanitizeSeats(seats);

  return {
    seats: s,
    usdToInr,
    hydradnsAnnualINR,
    competitors: competitors.map((c) => {
      const annualINR = competitorAnnualINRForCompetitor(s, c, usdToInr);
      return {
        ...c,
        annualINR,
        savingsINR: annualINR - hydradnsAnnualINR,
        multiplier: hydradnsAnnualINR > 0 ? annualINR / hydradnsAnnualINR : 0,
      };
    }),
  };
}

/** Format an INR amount as e.g. "₹5,80,800" (no paise). */
export function formatINR(amount: number): string {
  return new Intl.NumberFormat("en-IN", {
    style: "currency",
    currency: "INR",
    maximumFractionDigits: 0,
  }).format(Math.round(amount));
}
